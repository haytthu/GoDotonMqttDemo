package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

const rtuType = 5
const tcpType = 6
const pt100Type = 12
const registerSlots = 16

var registerWidths = []int{1, 2, 4, 4, 4, 4}
var hexLetters = regexp.MustCompile(`[a-fA-F]`)

func setting(collector, channel document, key string, fallback int) int {
	return integer(channel, key, integer(object(collector, "vibrationDefaults"), key, fallback))
}

func (a *app) validateChannels(collector document) {
	expectedTypes := map[int]string{0: "VIBRATION", 2: "PROCESS", 3: "TEMPERATURE", 10: "HUMIDITY"}
	addresses := make(map[[3]int]bool)
	physical := make(map[int]bool)
	for _, channel := range objects(collector, "channels") {
		number, kind, direction := integer(channel, "channelNo", 0), integer(channel, "channelType", 0), integer(channel, "direction", 0)
		id := field(channel, "pointId")
		point := entity(a.store, "points", id)
		expected := expectedTypes[kind]
		require(number >= 1 && number <= 256 && direction >= 0 && direction <= 3 && (kind == 0 || direction == 0) &&
			expected != "" && expected == field(point, "type"), "The channel number, direction and point type must match")
		if group := field(point, "axisGroupId"); group != "" {
			require(direction == 0 && group == id, "Bind each three-axis point to a single physical channel")
		}
		address := [3]int{kind, number, direction}
		require(!addresses[address], "A channel and direction are assigned more than once")
		addresses[address] = true
		if kind == 0 {
			single := direction == 0
			before, exists := physical[number]
			require(!exists || before == single, "Use a single-axis or a multi-axis binding per vibration channel")
			physical[number] = single
			freq := setting(collector, channel, "lowAccFreq", 3125)
			length := setting(collector, channel, "lowAccLength", 16384)
			plus := setting(collector, channel, "plusLength", 1024)
			low := setting(collector, channel, "velocityLowFreq", 2)
			high := setting(collector, channel, "velocityHighFreq", 1000)
			velocity := setting(collector, channel, "velocityLength", 4096)
			require(freq >= 1 && length >= 0 && plus >= 0 && int64(length)+int64(plus) <= 1048576 &&
				low >= 0 && high > max(low, 2) && velocity >= 0 && velocity <= 1048576 && (velocity == 0 || high <= freq),
				"Check the bands, sample counts and extra samples for low-frequency acceleration and velocity")
			if velocity > 0 {
				prepareVelocity(freq, float64(low), float64(high), velocity, length, plus)
			}
		}
	}
}

func (a *app) collectorSnapshot(collector document) document {
	snapshot := clone(collector)
	channels := []document{}
	for _, binding := range objects(snapshot, "channels") {
		point := entity(a.store, "points", field(binding, "pointId"))
		group := field(point, "axisGroupId")
		if group != "" && integer(binding, "channelType", -1) == 0 {
			points := a.pointGroup(group)
			require(len(points) == 3, "Cannot publish parameters because a three-axis measurement group is missing an axis")
			for _, point := range points {
				axis := clone(binding)
				axis["pointId"], axis["direction"], axis["threeAxis"] = field(point, "id"), integer(point, "direction", 0), true
				channels = append(channels, axis)
			}
		} else {
			delete(binding, "threeAxis")
			channels = append(channels, binding)
		}
	}
	snapshot["channels"] = channels
	return snapshot
}

func findChannel(collector document, kind, number, direction int) document {
	var equivalent document
	for _, channel := range objects(collector, "channels") {
		if integer(channel, "channelType", -1) != kind || integer(channel, "channelNo", -1) != number {
			continue
		}
		bound := integer(channel, "direction", 0)
		if bound == direction || kind != 0 {
			return channel
		}
		if bound == 0 && direction == 1 || flag(channel, "threeAxis") && bound == 1 && direction == 0 {
			equivalent = channel
		}
	}
	return equivalent
}

func sampleDirection(binding document, direction int) int {
	if direction == 0 && flag(binding, "threeAxis") {
		return 1
	}
	return direction
}

func newVibrationChannel(number int) document {
	return document{"ChNo": number, "SpeedNo": 0, "SensType": 0, "SensCoef": 10, "CnstCurSrcState": 0,
		"DevId": 1, "IsTem": 0, "IsRunning": 1, "SpeedDiffRate": 0, "PolePairsNum": 1,
		"VibParaList": []document{}, "LimitParaList": []document{}}
}

func deviceKey(id string) int {
	var hash int32
	for _, value := range utf16.Encode([]rune(id)) {
		hash = 31*hash + int32(value)
	}
	return int(hash&math.MaxInt32)%65535 + 1
}

func addWave(channel document, code, direction, low, high, length, plus int, unit string) {
	if length == 0 && code != 15 {
		return
	}
	channel["VibParaList"] = append(objects(channel, "VibParaList"), document{
		"Code": code, "Dir": direction, "LowFreq": low, "HighFreq": high, "Length": length, "PlusLength": plus, "Unit": unit,
	})
}

func alarmParameters(point document, direction int) []document {
	result := []document{}
	positions := make(map[[2]int]int)
	for _, alarm := range objects(point, "alarms") {
		axis := integer(alarm, "direction", 0)
		if integer(alarm, "type", 0) != 0 || axis != 0 && axis != direction {
			continue
		}
		code, level := integer(alarm, "code", 0), integer(alarm, "level", 3)
		key := [2]int{code, level}
		row := document{"Code": code, "Level": level, "Type": 0, "Limit": number(alarm, "limit", 0), "Dir": direction}
		if i, exists := positions[key]; exists {
			if axis != 0 {
				result[i] = row
			}
			continue
		}
		positions[key] = len(result)
		result = append(result, row)
	}
	return result
}

func (a *app) collectParameters(collector document) document {
	result := document{}
	for _, key := range []string{"VibChannelList", "TechChannelList", "SpeedChannelList", "TriggerList",
		"RtuChannelList", "ModbusTCPChannelList", "OutputChannelList", "StopLimitList"} {
		result[key] = []document{}
	}
	vibration := make(map[int]document)
	stopLimits := make(map[[3]int]float64)
	for _, channel := range objects(collector, "channels") {
		ch, kind, direction := integer(channel, "channelNo", 0), integer(channel, "channelType", 0), integer(channel, "direction", 0)
		pointID := field(channel, "pointId")
		point := entity(a.store, "points", pointID)
		if point != nil {
			upper := []document{}
			for _, alarm := range objects(point, "alarms") {
				if integer(alarm, "type", 0) == 0 {
					upper = append(upper, alarm)
				}
			}
			point["alarms"] = upper
			validateAlarms(point)
		}
		device := entity(a.store, "devices", field(point, "deviceId"))
		if kind == 2 {
			result["TechChannelList"] = append(objects(result, "TechChannelList"), document{
				"ChNo": ch, "SensCoef": number(channel, "sensitivity", 1), "Amendment": number(channel, "amendment", 0),
				"LimitParaList": alarmParameters(point, 0),
			})
		} else {
			wire := vibration[ch]
			if wire == nil {
				wire = newVibrationChannel(ch)
				vibration[ch] = wire
				result["VibChannelList"] = append(objects(result, "VibChannelList"), wire)
			}
			if kind == 0 {
				wire["SensCoef"], wire["SensType"], wire["DevId"] = number(channel, "sensitivity", 10), integer(channel, "sensorType", 0), deviceKey(field(point, "deviceId"))
				if direction == 0 || direction == 1 {
					addWave(wire, 1, direction, 0, setting(collector, channel, "longWaveFreq", 12500), setting(collector, channel, "longWaveLength", 131072), 0, "m/s^2")
					addWave(wire, 11, direction, 0, setting(collector, channel, "highFreq", 6250), setting(collector, channel, "highAccLength", 16384), 0, "m/s^2")
				}
				addWave(wire, 12, direction, 0, setting(collector, channel, "lowAccFreq", 3125), setting(collector, channel, "lowAccLength", 16384), setting(collector, channel, "plusLength", 1024), "m/s^2")
				if setting(collector, channel, "velocityLength", 4096) > 0 {
					addWave(wire, 15, direction, setting(collector, channel, "velocityLowFreq", 2), setting(collector, channel, "velocityHighFreq", 1000), 0, 0, "mm/s")
				}
			}
			if kind == 3 {
				wire["IsTem"] = 1
			}
			wire["LimitParaList"] = append(objects(wire, "LimitParaList"), alarmParameters(point, direction)...)
		}
		if kind == 0 && flag(device, "stopEnabled") {
			for _, rule := range objects(device, "stopLimits") {
				if pointID != field(rule, "pointId") || !supportsStop(point, rule) {
					continue
				}
				code, limit := integer(rule, "code", 107), number(rule, "limit", 0)
				key := [3]int{kind, ch, code}
				if before, exists := stopLimits[key]; exists {
					require(before == limit, "Set one start/stop threshold per collector channel and indicator")
					continue
				}
				stopLimits[key] = limit
				result["StopLimitList"] = append(objects(result, "StopLimitList"), document{
					"ChNo": ch, "ChType": kind, "Code": code, "StopLimit": limit, "StopLimitChangeRate": 0,
				})
			}
		}
	}
	appendExtensionParameters(collector, result, vibration)
	return result
}

func inRange(value any, key string, low, high int) int {
	result, err := strconv.ParseFloat(strings.TrimSpace(field(document{key: value}, key)), 64)
	require(err == nil && result == math.Trunc(result) && result >= float64(low) && result <= float64(high),
		fmt.Sprintf("%s must be an integer from %d to %d", key, low, high))
	return int(result)
}

func optionalRange(o document, key string, fallback, low, high int) int {
	if o[key] == nil {
		return fallback
	}
	return inRange(o[key], key, low, high)
}

func calibration(o document, key string, fallback float64) float64 {
	if strings.TrimSpace(field(o, key)) == "" {
		return fallback
	}
	value := number(o, key, fallback)
	require(finite(value), key+" must be a valid number")
	return value
}

func functionCode(group document) int {
	code := strings.TrimSpace(field(group, "functionCode"))
	base := 10
	if strings.HasPrefix(strings.ToLower(code), "0x") {
		code, base = code[2:], 16
	} else if hexLetters.MatchString(code) {
		base = 16
	}
	value, err := strconv.ParseInt(code, base, 32)
	require(err == nil && value >= 1 && value <= 255, "The function code must be an integer from 1 to 255. Hexadecimal is supported")
	return int(value)
}

func bindingID(binding document) string { return strings.TrimSpace(field(binding, "pointId")) }
func registerType(binding document) int {
	return inRange(binding["dataType"], "dataType", 0, len(registerWidths)-1)
}

func extensionPointIDs(collector document, key string) map[string]bool {
	ids := make(map[string]bool)
	for _, group := range objects(collector, key) {
		if key == "pt100" {
			ids[bindingID(group)] = true
			continue
		}
		for _, binding := range objects(group, "points") {
			if id := bindingID(binding); id != "" {
				ids[id] = true
			}
		}
	}
	return ids
}

func validateExtensions(collector document) {
	for _, key := range []string{"rtu", "modbusTcp"} {
		tcp := key == "modbusTcp"
		name, low, high := "RTU", 1, 247
		if tcp {
			name, low, high = "Modbus TCP", 0, 255
		}
		channels := make(map[int]bool)
		for _, group := range objects(collector, key) {
			ch := inRange(group["channelNo"], "channelNo", 1, 8)
			require(!channels[ch], fmt.Sprintf("%s channel %d is configured more than once", name, ch))
			channels[ch] = true
			inRange(group["deviceAddress"], "deviceAddress", low, high)
			functionCode(group)
			if tcp {
				mode := inRange(group["mode"], "mode", 0, 1)
				require(mode != 0 || strings.TrimSpace(field(group, "ipAddress")) != "", "Enter an IP address for Modbus TCP master mode")
				inRange(group["port"], "port", 1, 65535)
				inRange(group["status"], "status", 0, 1)
				inRange(group["startAddress"], "startAddress", 0, 65535)
			} else {
				inRange(group["registerAddress"], "registerAddress", 0, 65535)
				inRange(group["period"], "period", 1, math.MaxInt32)
			}
			bindings := objects(group, "points")
			require(len(bindings) > 0, fmt.Sprintf("Add a data position to %s channel %d", name, ch))
			slots := make(map[int]bool)
			for _, binding := range bindings {
				slot := inRange(binding["slot"], "slot", 1, registerSlots)
				require(!slots[slot], fmt.Sprintf("%s channel %d, position %d is assigned more than once", name, ch, slot))
				slots[slot] = true
				if tcp {
					if bindingID(binding) != "" {
						registerType(binding)
					}
					optionalRange(binding, "registerAddress", 0, 0, 65535)
					calibration(binding, "scale", 1)
					calibration(binding, "offset", 0)
				} else {
					require(bindingID(binding) != "", "Select a measurement point for the extension channel")
					registerType(binding)
					require(calibration(binding, "inputMin", 0) <= calibration(binding, "inputMax", 65535), "The lower range limit must not exceed the upper limit")
				}
			}
		}
	}
	positions := make(map[[2]int]bool)
	for _, binding := range objects(collector, "pt100") {
		ch := inRange(binding["channelNo"], "channelNo", 1, 45)
		slot := inRange(binding["subChannel"], "subChannel", 1, 8)
		key := [2]int{ch, slot}
		require(!positions[key], fmt.Sprintf("PT100 channel %d, position %d is assigned more than once", ch, slot))
		positions[key] = true
		require(bindingID(binding) != "", "Select a measurement point for the extension channel")
		calibration(binding, "sensitivity", 1)
		calibration(binding, "amendment", 0)
	}
}

func ordered(o document, key, sortKey string) []document {
	rows := append([]document{}, objects(o, key)...)
	sort.SliceStable(rows, func(i, j int) bool { return integer(rows[i], sortKey, 0) < integer(rows[j], sortKey, 0) })
	return rows
}

func slotsByNumber(group document) map[int]document {
	result := make(map[int]document)
	for _, binding := range objects(group, "points") {
		result[integer(binding, "slot", 0)] = binding
	}
	return result
}

func appendExtensionParameters(collector, wire document, vibration map[int]document) {
	rtu := []document{}
	for _, group := range ordered(collector, "rtu", "channelNo") {
		types, length := []int{}, 0
		for _, binding := range ordered(group, "points", "slot") {
			kind := registerType(binding)
			types, length = append(types, kind), length+registerWidths[kind]
		}
		rtu = append(rtu, document{"ChNo": integer(group, "channelNo", 0), "RTUDeviceAddr": integer(group, "deviceAddress", 0),
			"FunCode": functionCode(group), "RegAddr": integer(group, "registerAddress", 0), "RtuPeriod": integer(group, "period", 0),
			"DataLength": length, "DataType": types})
	}
	tcp := []document{}
	for _, group := range ordered(collector, "modbusTcp", "channelNo") {
		bindings := slotsByNumber(group)
		registers := make([]document, 0, registerSlots)
		for slot := 1; slot <= registerSlots; slot++ {
			binding := bindings[slot]
			kind := 0
			if bindingID(binding) != "" {
				kind = registerType(binding)
			}
			registers = append(registers, document{"RegAddr": optionalRange(binding, "registerAddress", 0, 0, 65535), "DataType": kind,
				"Coef_K": calibration(binding, "scale", 1), "Coef_B": calibration(binding, "offset", 0)})
		}
		tcp = append(tcp, document{"ChNo": integer(group, "channelNo", 0), "DeviceAddr": integer(group, "deviceAddress", 0),
			"IPAddr": strings.TrimSpace(field(group, "ipAddress")), "Port": integer(group, "port", 0), "Status": integer(group, "status", 0),
			"FunCode": functionCode(group), "Mode": integer(group, "mode", 0), "StartAddr": integer(group, "startAddress", 0), "ModbusTCPParaList": registers})
	}
	wire["RtuChannelList"], wire["ModbusTCPChannelList"] = rtu, tcp
	if bindings := ordered(collector, "pt100", "channelNo"); len(bindings) > 0 {
		for _, binding := range bindings {
			channel := integer(binding, "channelNo", 0)
			if vibration[channel] != nil {
				continue
			}
			row := newVibrationChannel(channel)
			row["SensCoef"], row["IsTem"] = 1, 1
			vibration[channel] = row
			wire["VibChannelList"] = append(objects(wire, "VibChannelList"), row)
		}
		wire["VibChannelList"] = ordered(wire, "VibChannelList", "ChNo")
	}
}

func findGroup(collector document, key string, ch int) document {
	for _, group := range objects(collector, key) {
		if integer(group, "channelNo", 0) == ch {
			return group
		}
	}
	return nil
}

func decodeBase64(encoded string) ([]byte, error) {
	if strings.HasSuffix(encoded, "=") {
		return base64.StdEncoding.DecodeString(encoded)
	}
	return base64.RawStdEncoding.DecodeString(encoded)
}

func decodeBytes(value any, key string) []byte {
	if encoded, ok := value.(string); ok {
		data, err := decodeBase64(encoded)
		require(err == nil, key+" must contain valid Base64")
		return data
	}
	require(value != nil, "Missing "+key)
	items := asArray(value, key)
	data := make([]byte, len(items))
	for i, item := range items {
		data[i] = byte(inRange(item, key, -128, 255))
	}
	return data
}

func readRegister(data []byte, kind int) float64 {
	switch kind {
	case 0:
		if data[0] != 0 {
			return 1
		}
		return 0
	case 1:
		return float64(int16(binary.BigEndian.Uint16(data)))
	case 2, 5:
		return float64(math.Float32frombits(binary.BigEndian.Uint32(data)))
	case 3:
		return float64(int32(binary.BigEndian.Uint32(data)))
	default:
		return float64(binary.BigEndian.Uint32(data))
	}
}

func decodeRegisters(collector, frame document, time string) []sample {
	ch := inRange(frame["ChNo"], "ChNo", 1, 8)
	kind := optionalRange(frame, "ChType", rtuType, rtuType, tcpType)
	tcp := kind == tcpType
	key := "rtu"
	if tcp {
		key = "modbusTcp"
	}
	group := findGroup(collector, key, ch)
	if group == nil {
		return nil
	}
	data := decodeBytes(frame["Values"], "Values")
	bindings := slotsByNumber(group)
	length, assigned := 0, 0
	for _, binding := range bindings {
		if bindingID(binding) != "" {
			length += registerWidths[registerType(binding)]
			assigned++
		}
	}
	full := tcp && len(data) == length+registerSlots-assigned
	require(len(data) == length || full, fmt.Sprintf("RtuData length does not match the configuration for channel %d", ch))
	result := []sample{}
	for slot := 1; slot <= registerSlots; slot++ {
		binding := bindings[slot]
		id := bindingID(binding)
		if id == "" {
			if full {
				data = data[1:]
			}
			continue
		}
		dataType := registerType(binding)
		value := readRegister(data, dataType)
		data = data[registerWidths[dataType]:]
		if dataType != 0 {
			if tcp {
				value = value*calibration(binding, "scale", 1) + calibration(binding, "offset", 0)
			} else {
				low, high := calibration(binding, "inputMin", 0), calibration(binding, "inputMax", 65535)
				value = value*((high-low)/65535) + low
			}
		}
		if finite(value) {
			result = append(result, sample{id, kind, ch, 0, slot, 0, value, time})
		}
	}
	return result
}

func resolveExtension(collector document, kind, ch, direction, sub, code int, value float64, time string) *sample {
	if kind == pt100Type {
		for _, binding := range objects(collector, "pt100") {
			if integer(binding, "channelNo", 0) == ch && integer(binding, "subChannel", 0) == sub {
				value = value*calibration(binding, "sensitivity", 1) + calibration(binding, "amendment", 0)
				if finite(value) {
					return &sample{bindingID(binding), kind, ch, 0, sub, code, value, time}
				}
				return nil
			}
		}
	}
	if sub > 0 && (kind == rtuType || kind == tcpType) {
		key := "rtu"
		if kind == tcpType {
			key = "modbusTcp"
		}
		binding := slotsByNumber(findGroup(collector, key, ch))[sub]
		if id := bindingID(binding); id != "" {
			return &sample{id, kind, ch, direction, sub, code, value, time}
		}
	}
	return nil
}
