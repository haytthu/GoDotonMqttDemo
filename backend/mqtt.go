package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type pendingCommand struct {
	collectorID, name, revision string
	sentAt                      time.Time
}

type incomingMessage struct {
	message    mqtt.Message
	connection uint64
}

func problemText(issue any) string {
	if err, ok := issue.(requestError); ok {
		return err.message
	}
	return fmt.Sprint(issue)
}

func (a *app) startMQTT(parent context.Context) func() {
	ctx, cancel := context.WithCancel(parent)
	messages := make(chan incomingMessage, 128)
	var connection atomic.Uint64
	options := mqtt.NewClientOptions().AddBroker(env("MQTT_BROKER", "tcp://127.0.0.1:1883")).
		SetClientID(env("MQTT_CLIENT_ID", "sg500z-demo")).SetUsername(env("MQTT_USERNAME", "")).SetPassword(env("MQTT_PASSWORD", "")).
		SetCleanSession(true).SetKeepAlive(60 * time.Second).SetAutoReconnect(true).SetConnectTimeout(5 * time.Second).
		SetConnectRetry(true).SetConnectRetryInterval(5 * time.Second).SetWriteTimeout(3 * time.Second)
	options.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		connection.Add(1)
		a.mu.Lock()
		clear(a.online)
		a.mu.Unlock()
		log.Printf("MQTT connection lost: %v", err)
	})
	options.SetOnConnectHandler(func(client mqtt.Client) {
		current := connection.Load()
		for ctx.Err() == nil && client.IsConnectionOpen() && current == connection.Load() {
			token := client.Subscribe(a.prefix+"/Online/#", 1, func(_ mqtt.Client, message mqtt.Message) {
				select {
				case messages <- incomingMessage{message, current}:
				case <-ctx.Done():
				}
			})
			if token.WaitTimeout(5*time.Second) && token.Error() == nil {
				a.mqttWork(ctx, func() {
					if current == connection.Load() {
						a.transmit("", "ServerStart", a.prefix+"/ServerStart", nil, nil)
						log.Printf("Subscribed to %s/Online/#", a.prefix)
					}
				})
				return
			}
			log.Printf("Could not subscribe to %s/Online/#: %v", a.prefix, token.Error())
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	})
	a.client = mqtt.NewClient(options)
	a.client.Connect()
	var worker sync.WaitGroup
	worker.Add(1)
	go func() {
		defer worker.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case message := <-messages:
				a.mqttWork(ctx, func() {
					if message.connection == connection.Load() {
						a.receive(message.message.Topic(), message.message.Payload())
					}
				})
			case now := <-ticker.C:
				a.mqttWork(ctx, func() {
					a.expireTransfers(now)
					for key, command := range a.pending {
						if now.Sub(command.sentAt) > 30*time.Second {
							delete(a.pending, key)
							a.commandResult(command, false, "等待设备回复超时，可在设备上线后重新发送")
						}
					}
				})
			}
		}
	}()
	return func() {
		cancel()
		a.client.Disconnect(250)
		worker.Wait()
	}
}

func (a *app) mqttWork(ctx context.Context, action func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	defer func() {
		if issue := recover(); issue != nil {
			log.Printf("MQTT processing failed: %s", problemText(issue))
		}
	}()
	if ctx.Err() == nil {
		action()
	}
}

func (a *app) connected() bool           { return a.client != nil && a.client.IsConnectionOpen() }
func (a *app) isOnline(code string) bool { return a.connected() && a.online[code] }

func (a *app) receive(topic string, bytes []byte) {
	remainder, ok := strings.CutPrefix(topic, a.prefix+"/")
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimRight(remainder, "/"), "/")
	if len(parts) < 3 || len(parts) > 5 || parts[0] != "Online" || parts[1] == "" {
		return
	}
	code, command := parts[1], parts[2]
	var status any
	if len(parts) == 5 {
		value, err := strconv.ParseInt(parts[4], 10, 32)
		if err != nil {
			return
		}
		status = int(value)
	}
	a.logMessage(code, "RX", command, topic, status, summary(command, bytes))
	if command == "Disconnect" {
		delete(a.online, code)
		return
	}
	if status != nil {
		key := code + "/" + command + "/" + parts[3]
		if pending, exists := a.pending[key]; exists {
			delete(a.pending, key)
			success := status == 1
			message := "设备已确认通知"
			if !success {
				message = fmt.Sprintf("设备返回状态 %d", status)
			}
			a.commandResult(pending, success, message)
		}
		return
	}
	if len(parts) != 4 {
		return
	}
	message := parts[3]
	defer func() {
		if issue := recover(); issue != nil {
			log.Printf("Could not process %s: %s", topic, problemText(issue))
			a.logMessage(code, "SYSTEM", command, topic, -3, problemText(issue))
			if command != "WaveDataPacket" && a.connected() {
				a.reply(code, command, message, -3, nil)
			}
		}
	}()
	a.deviceRequest(code, command, message, string(bytes))
}

func (a *app) deviceRequest(code, command, message, raw string) {
	switch command {
	case "Register":
		id := field(a.collectorByCode(code), "id")
		if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "null" || id == "" {
			a.reply(code, command, message, -4, nil)
			return
		}
		info := asObject(decodeJSON(raw))
		if integer(info, "DasType", 0) != 0 {
			a.reply(code, command, message, -4, nil)
			return
		}
		a.online[code] = true
		state := a.state(id)
		state["lastSeen"] = a.now()
		if info["Ver"] != nil {
			state["firmwareVersion"] = field(info, "Ver")
		}
		put(a.store, collectorStates, id, state)
		a.firmwareRegistered(id, field(info, "Ver"))
	case "SysPara":
		a.reply(code, command, message, 1, a.systemParameters(a.savedParameters(a.collectorID(code))))
		return
	case "CollectPara":
		id := a.collectorID(code)
		edition := a.savedParameters(id)
		if edition == nil {
			a.logMessage(code, "SYSTEM", command, "", -4, "No saved acquisition parameters are available")
			a.reply(code, command, message, -4, nil)
			return
		}
		a.reply(code, command, message, 1, edition["wire"])
		a.parametersDelivered(id, field(edition, "version"))
		return
	case "LubParams":
		a.reply(code, command, message, 1, []any{})
		return
	case "IndexData":
		a.indexData(a.collectorID(code), parseFrames(raw))
	case "RtuData", "ModbusTCPData":
		a.rtuData(a.collectorID(code), command, parseFrames(raw))
	case "ParamsResp":
		return
	case "DeviceState", "TimeSeqData", "TimeMarkData", "CheckData", "Location":
		a.collectorID(code)
	case "WaveData":
		a.legacyWave(code, a.collectorID(code), message, asObject(decodeJSON(raw)))
		return
	case "WaveDataIndex", "WaveDataPacket", "WaveDataConfirm":
		a.packetWave(code, a.collectorID(code), command, message, asObject(decodeJSON(raw)))
		return
	case "SensorInfo", "BatteryInfo":
		a.addReports(code, command, parseFrames(raw))
	case "LogFile":
		if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "null" {
			a.reply(code, command, message, 0, nil)
			return
		}
		frame := asObject(decodeJSON(raw))
		response := document{"PackIndex": integer(frame, "PackIndex", 0)}
		if frame["Md5Code"] != nil {
			response["Md5Code"] = field(frame, "Md5Code")
		}
		a.reply(code, command, message, a.fileAction(code, command, func() { a.acceptLog(code, frame) }), response)
		return
	case "UpdateFileList":
		a.reply(code, command, message, 1, a.availableFirmware(a.collectorID(code)))
		return
	case "UpdateFile":
		id := a.collectorID(code)
		frame := asObject(decodeJSON(raw))
		var data any
		status := a.fileAction(code, command, func() {
			if pack := a.firmwarePacket(id, frame); pack != nil {
				data = pack
			}
		})
		if integer(frame, "IsConfirm", 0) != 1 {
			a.reply(code, command, message, status, data)
		}
		return
	case "LubricateTaskList":
		tasks := []document{}
		for _, sn := range []string{"50000169", "50000170", "50000171"} {
			tasks = append(tasks, document{"Sn": sn, "Amount": 0, "Time": "2026-04-15T14:29:23", "Type": 0, "Period": 0, "Enable": 1})
		}
		a.reply(code, command, message, 1, tasks)
		return
	case "Event":
		asObject(decodeJSON(raw))
		return
	default:
		a.reply(code, command, message, -2, nil)
		return
	}
	a.reply(code, command, message, 1, nil)
}

func parseFrames(raw string) []document {
	value := decodeJSON(raw)
	require(value != nil, "Expected a JSON array")
	return objects(document{"frames": value}, "frames")
}

func (a *app) fileAction(code, command string, run func()) (status int) {
	defer func() {
		if issue := recover(); issue != nil {
			a.logMessage(code, "SYSTEM", command, "", 0, problemText(issue))
			status = 0
		}
	}()
	run()
	return 1
}

func (a *app) systemParameters(edition document) document {
	settings := object(object(edition, "collector"), "system")
	saved := "1970-01-01T00:00:00"
	if edition != nil {
		value := field(object(edition, "collector"), "updatedAt")
		if value == "" {
			value = field(edition, "version")
		}
		if normalized := a.sampleTime(value); normalized != "" {
			saved = normalized[:len(mqttLayout)]
		}
	}
	return document{"SysTime": a.nowMQTT(), "ParaTime": saved, "IndexPeriod": integer(settings, "indexPeriod", 60),
		"WavePeriod": integer(settings, "wavePeriod", 300), "LongWavePeriod": integer(settings, "longWavePeriod", 259200),
		"LongWaveOffset": integer(settings, "longWaveOffset", 0), "CollectType": 1, "OutType485": integer(settings, "outType485", 0),
		"TechPeriod": integer(settings, "techPeriod", 600), "RpmPeriod": integer(settings, "rpmPeriod", 600),
		"RtuPeriod": integer(settings, "rtuPeriod", 600), "UploadPeriod": integer(settings, "uploadPeriod", 300),
		"AlarmCount": integer(settings, "alarmCount", 0), "WaveCompress": integer(settings, "waveCompress", 1),
		"UserCustom": integer(settings, "userCustom", 0), "WlessIndexPeriod": 0, "WlessWavePeriod": 0, "CircleCount": 0,
		"CirclePointCount": 0, "UploadSegment": 0, "IsLub": 0, "BiasCheck": 0}
}

func (a *app) transmit(code, command, topic string, status any, bytes []byte) {
	if !a.connected() {
		reject(409, "The MQTT broker is disconnected")
	}
	token := a.client.Publish(topic, 1, false, bytes)
	if err := token.Error(); err != nil {
		reject(503, "The MQTT message could not be sent")
	}
	a.logMessage(code, "TX", command, topic, status, summary(command, bytes))
}

func (a *app) reply(code, command, message string, status int, payload any) {
	var bytes []byte
	if payload != nil {
		bytes = []byte(jsonText(payload))
	}
	topic := fmt.Sprintf("%s/%s/%s/%s/%d", a.prefix, code, command, message, status)
	a.transmit(code, command, topic, status, bytes)
}

func (a *app) issueCommand(code, command, revision string, payload []byte) {
	id := a.collectorID(code)
	a.messageID++
	if a.messageID > math.MaxInt32 {
		a.messageID = 1
	}
	key := fmt.Sprintf("%s/%s/%d", code, command, a.messageID)
	a.transmit(code, command, a.prefix+"/"+key, nil, payload)
	a.pending[key] = pendingCommand{id, command, revision, time.Now()}
}

func (a *app) commandResult(command pendingCommand, success bool, message string) {
	if command.name == "IssuePara" {
		state := "FAILED"
		if success {
			state = "ACKNOWLEDGED"
		}
		a.parameterState(command.collectorID, command.revision, state, message)
	} else if command.name == "IssueUpdateFile" {
		state := "PENDING"
		if success {
			state = "ACKNOWLEDGED"
		}
		a.upgradeNotification(command.collectorID, command.revision, state, message)
	}
}

func (a *app) issueParameters(code, revision string) {
	id := a.collectorID(code)
	if !a.isOnline(code) {
		a.parameterState(id, revision, "PENDING", "等待设备下次请求参数")
		return
	}
	a.parameterState(id, revision, "NOTIFIED", "已发送获取参数通知，等待设备拉取")
	defer func() {
		if issue := recover(); issue != nil {
			if err, ok := issue.(requestError); ok {
				a.parameterState(id, revision, "FAILED", err.message)
				return
			}
			panic(issue)
		}
	}()
	a.issueCommand(code, "IssuePara", revision, nil)
}

func (a *app) issueFirmware(collector, task document, files []document) {
	code, id := field(collector, "code"), field(collector, "id")
	if !a.isOnline(code) || !a.upgradeList && len(files) > 1 {
		return
	}
	revision := field(task, "id")
	if !a.upgradeNotification(id, revision, "NOTIFIED", "已发送升级通知，等待设备请求文件") {
		return
	}
	var payload any = files
	if !a.upgradeList {
		payload = files[0]
	}
	defer func() {
		if issue := recover(); issue != nil {
			if err, ok := issue.(requestError); ok {
				a.upgradeNotification(id, revision, "PENDING", err.message)
				return
			}
			panic(issue)
		}
	}()
	a.issueCommand(code, "IssueUpdateFile", revision, []byte(jsonText(payload)))
}
