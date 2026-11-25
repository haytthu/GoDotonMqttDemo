package main

import (
	"fmt"
	"math"
	"strings"
)

const collectorStates = "collector-status"

func (a *app) collectorID(code string) string {
	id := field(a.collectorByCode(code), "id")
	require(id != "", "This collector ID has not been configured")
	return id
}

func (a *app) state(id string) document {
	state := entity(a.store, collectorStates, id)
	if state == nil {
		state = document{}
	}
	return state
}

func (a *app) collectorStatus(id string) document {
	collector := a.record("collectors", id)
	state := a.state(id)
	state["online"], state["brokerConnected"] = a.isOnline(field(collector, "code")), a.connected()
	if state["parameterState"] == nil {
		state["parameterState"] = "DRAFT"
	}
	if latest := a.latestEdition(id); latest != nil {
		version := field(latest, "version")
		if version != field(state, "publishedVersion") {
			state["parameterState"] = "PENDING"
		}
		state["publishedVersion"] = version
	}
	if task := entity(a.store, upgrades, id); task != nil {
		for _, key := range []string{"upgradeState", "upgradeProgress", "upgradeMessage"} {
			state[key] = task[key]
		}
	}
	return state
}

func (a *app) parameterState(id, version, status, message string) {
	if id == "" || version == "" || version != field(a.latestEdition(id), "version") {
		return
	}
	state := a.state(id)
	if version == field(state, "publishedVersion") && field(state, "parameterState") == "PULLED" && status != "PULLED" {
		return
	}
	state["publishedVersion"], state["parameterState"], state["parameterMessage"] = version, status, message
	put(a.store, collectorStates, id, state)
}

func (a *app) parametersDelivered(id, version string) {
	execute(a.store, "UPDATE edition SET activated=COALESCE(activated,?) WHERE collector=? AND version=?", a.sampleTime(a.nowMQTT()), id, version)
	a.parameterState(id, version, "PULLED", "设备已获取采集参数")
}

func (a *app) logMessage(code, direction, command, topic string, status any, message string) {
	id := nullable(field(a.collectorByCode(code), "id"))
	time := a.now()
	if len(message) > 1200 {
		runes := []rune(message)
		message = string(runes[:min(1200, len(runes))])
	}
	entry := document{"time": time, "collectorId": id, "code": code, "direction": direction, "command": command,
		"topic": topic, "status": status, "message": strings.ToValidUTF8(message, "\uFFFD")}
	execute(a.store, "INSERT INTO mqtt_log(collector,time,body) VALUES(?,?,?)", id, time, jsonText(entry))
}

func (a *app) indexData(id string, frames []document) {
	results := []sample{}
	byTime := make(map[string]document)
	for _, frame := range frames {
		kind, ch := integer(frame, "ChType", 0), integer(frame, "ChNo", 0)
		times := asArray(frame["Time"], "Time")
		for _, item := range objects(frame, "Content") {
			dir, code := integer(item, "Dir", 0), integer(item, "Code", 0)
			sub := integer(item, "SubChNo", integer(frame, "SubChNo", 0))
			values := asArray(item["Values"], "Values")
			require(len(times) == len(values), "Metric timestamp and value counts do not match")
			for i, entry := range times {
				if entry == nil || values[i] == nil {
					continue
				}
				time := a.sampleTime(field(document{"time": entry}, "time"))
				value := number(document{"value": values[i]}, "value", math.NaN())
				if time == "" || !finite(value) {
					continue
				}
				collector, exists := byTime[time]
				if !exists {
					collector = object(a.editionAt(id, time), "collector")
					byTime[time] = collector
				}
				if metric := resolveExtension(collector, kind, ch, dir, sub, code, value, time); metric != nil {
					results = append(results, *metric)
					continue
				}
				binding := findChannel(collector, kind, ch, dir)
				if binding == nil && kind == 0 && code == 100 {
					binding = findChannel(collector, 3, ch, 0)
				}
				if binding != nil {
					results = append(results, sample{field(binding, "pointId"), kind, ch, sampleDirection(binding, dir), 0, code, value, time})
				}
			}
		}
	}
	a.addMetrics(id, results)
}

func (a *app) rtuData(id, command string, frames []document) {
	results := []sample{}
	for _, frame := range frames {
		if command == "ModbusTCPData" {
			if _, ok := frame["ChType"]; !ok {
				frame["ChType"] = tcpType
			}
		}
		if time := a.sampleTime(field(frame, "Time")); time != "" {
			collector := object(a.editionAt(id, time), "collector")
			results = append(results, decodeRegisters(collector, frame, time)...)
		}
	}
	a.addMetrics(id, results)
}

func (a *app) addReports(code, kind string, frames []document) {
	if len(frames) == 0 {
		return
	}
	timeField, identityField := "Clock", "SN"
	if kind == "BatteryInfo" {
		timeField, identityField = "Time", "ID"
	}
	received := a.now()
	reports := make([]document, 0, len(frames))
	for _, frame := range frames {
		channel := inRange(frame["ChNo"], "ChNo", 0, math.MaxInt32)
		time := a.sampleTime(field(frame, timeField))
		if time == "" {
			time = received
		}
		id := nameID(jsonText([]any{kind, code, channel, field(frame, identityField), time}))
		reports = append(reports, document{"id": id, "collectorCode": code, "channelNo": channel, "time": time, "receivedAt": received, "data": frame})
	}
	a.saveReports(kind, code, reports)
}

func (a *app) reports(kind, id string, limit int) []document {
	var code any
	if nullable(id) != nil {
		code = field(a.record("collectors", id), "code")
	}
	return query(a.store, "SELECT body FROM collector_report WHERE kind=? AND (? IS NULL OR collector_code=?) ORDER BY time DESC,rowid DESC LIMIT ?", kind, code, code, limit)
}

func summary(command string, payload []byte) string {
	switch command {
	case "WaveData", "WaveDataPacket", "LogFile", "UpdateFile":
		return fmt.Sprintf("%d bytes", len(payload))
	default:
		return string(payload)
	}
}
