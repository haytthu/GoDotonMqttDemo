package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

const rootID = "root"
const threeAxisType = "VIBRATION_3AXIS"

var collectorCodePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,63}$`)

func checkKind(kind string) {
	switch kind {
	case "organizations", "devices", "points", "collectors":
	default:
		reject(404, "Record not found")
	}
}

func (a *app) record(kind, id string) document {
	checkKind(kind)
	result := entity(a.store, kind, id)
	if result == nil {
		reject(404, "Record not found")
	}
	return result
}

func (a *app) referenced(kind, id, message string) document {
	result := entity(a.store, kind, id)
	require(result != nil, message)
	return result
}

func (a *app) used(kind, key, id string) bool {
	for _, row := range a.entities(kind) {
		if field(row, key) == id {
			return true
		}
	}
	return false
}

func (a *app) pointUsed(id string) bool {
	for _, collector := range a.entities("collectors") {
		for _, binding := range objects(collector, "channels") {
			if field(binding, "pointId") == id {
				return true
			}
		}
		for _, key := range []string{"rtu", "modbusTcp", "pt100"} {
			if extensionPointIDs(collector, key)[id] {
				return true
			}
		}
	}
	for _, device := range a.entities("devices") {
		for _, rule := range objects(device, "stopLimits") {
			if field(rule, "pointId") == id {
				return true
			}
		}
	}
	return false
}

func (a *app) initConfiguration() {
	if entity(a.store, "organizations", rootID) == nil {
		put(a.store, "organizations", rootID, document{"id": rootID, "name": "根组织", "parentId": ""})
	}
	for _, organization := range a.entities("organizations") {
		id := field(organization, "id")
		if id != rootID && field(organization, "parentId") == "" {
			organization["parentId"] = rootID
			put(a.store, "organizations", id, organization)
		}
	}
	a.migrateAxes()
}

func (a *app) save(kind, id string, data document) document {
	checkKind(kind)
	require(kind != "organizations" || id != rootID, "The root organization cannot be changed or deleted")
	var previous document
	if id == "" {
		id = randomID()
	} else {
		previous = a.record(kind, id)
	}
	data["id"], data["updatedAt"] = id, a.now()
	data["name"] = requiredText(data, "name", 150, "Enter a name of up to 150 characters")
	switch kind {
	case "organizations":
		a.validateOrganization(data, previous)
	case "devices":
		a.validateDevice(data, previous)
	case "points":
		if field(data, "type") == threeAxisType {
			require(previous == nil, "Create the three axis points together as a new measurement group")
			a.referenced("devices", field(data, "deviceId"), "Select an existing device")
			for _, alarm := range objects(data, "alarms") {
				dir := integer(alarm, "direction", 0)
				require(dir >= 0 && dir <= 3, "Invalid direction for the alarm threshold")
			}
			points := splitAxes(data, false)
			a.transaction(func(tx *sql.Tx) {
				for _, point := range points {
					validateAlarms(point)
					put(tx, "points", field(point, "id"), point)
				}
			})
			return points[0]
		}
		a.validatePoint(data, previous)
	case "collectors":
		a.validateCollector(data, previous)
		edition := a.buildEdition(data, a.latestEdition(id))
		a.transaction(func(tx *sql.Tx) {
			put(tx, kind, id, data)
			execute(tx, insertEdition, id, field(edition, "version"), jsonText(edition))
		})
		return data
	}
	put(a.store, kind, id, data)
	return data
}

func (a *app) remove(kind, id string) {
	current := a.record(kind, id)
	require(kind != "organizations" || id != rootID, "The root organization cannot be changed or deleted")
	switch kind {
	case "organizations":
		require(!a.used("organizations", "parentId", id) && !a.used("devices", "organizationId", id), "Remove the child organizations and devices first")
	case "devices":
		require(!a.used("points", "deviceId", id), "Remove the measurement points for this device first")
	case "points":
		if group := field(current, "axisGroupId"); group != "" {
			for _, point := range a.pointGroup(group) {
				require(!a.pointUsed(field(point, "id")), "Remove the channel bindings and start/stop rules for these axes first")
			}
			execute(a.store, "DELETE FROM entity WHERE kind='points' AND json_extract(body,'$.axisGroupId')=?", group)
			return
		}
		require(!a.pointUsed(id), "This point is assigned to a channel or start/stop rule")
	}
	execute(a.store, "DELETE FROM entity WHERE kind=? AND id=?", kind, id)
}

func requiredText(data document, key string, limit int, message string) string {
	result := strings.TrimSpace(field(data, key))
	require(result != "" && len(utf16.Encode([]rune(result))) <= limit, message)
	return result
}

func (a *app) validateOrganization(data, previous document) {
	parent := field(data, "parentId")
	if parent == "" {
		parent = rootID
	}
	data["parentId"] = parent
	require(parent == field(previous, "parentId") || !a.used("devices", "organizationId", parent), "Move the devices before adding a child organization")
	seen := map[string]bool{field(data, "id"): true}
	for parent != "" {
		require(!seen[parent], "An organization cannot be its own ancestor")
		seen[parent] = true
		parent = field(a.referenced("organizations", parent, "Parent organization not found"), "parentId")
	}
}

func (a *app) validateDevice(data, previous document) {
	organization := field(data, "organizationId")
	a.referenced("organizations", organization, "Select an existing organization")
	require(organization != rootID, "Only organizations can be added to the root organization")
	require(organization == field(previous, "organizationId") || !a.used("organizations", "parentId", organization), "Add devices to an organization without child organizations")
	seen := make(map[string]bool)
	for _, rule := range objects(data, "stopLimits") {
		pointID := field(rule, "pointId")
		point := a.referenced("points", pointID, "Start/stop point not found")
		code, direction := integer(rule, "code", 107), stopDirection(point, rule)
		threshold := number(rule, "limit", math.NaN())
		key := fmt.Sprintf("%s/%d/%d", pointID, code, direction)
		require(field(point, "deviceId") == field(data, "id") && !seen[key] && code >= 0 && finite(threshold) && direction >= 0 && direction <= 3,
			"Select a valid indicator and point on this device for the start/stop rule")
		seen[key] = true
		if !supportsStop(point, rule) {
			unchanged := false
			for _, before := range objects(previous, "stopLimits") {
				if field(before, "pointId") == pointID && code == integer(before, "code", 107) &&
					direction == integer(before, "direction", integer(point, "direction", 0)) && threshold == number(before, "limit", math.NaN()) {
					unchanged = true
					break
				}
			}
			require(unchanged, "Start/stop detection requires a single-axis vibration point or the Z axis of a three-axis point")
		}
		rule["direction"] = direction
	}
}

func (a *app) validatePoint(data, previous document) {
	a.referenced("devices", field(data, "deviceId"), "Select an existing device")
	kind := field(data, "type")
	switch kind {
	case "VIBRATION", "TEMPERATURE", "HUMIDITY", "PROCESS", "RTU", "MODBUS_TCP", "PT100":
	default:
		reject(400, "Select a valid point type")
	}
	group := field(previous, "axisGroupId")
	if previous != nil && (kind != field(previous, "type") || field(data, "deviceId") != field(previous, "deviceId")) {
		require(group == "", "The device and point type are fixed for each member of a three-axis group")
		require(!a.pointUsed(field(previous, "id")), "Remove channel bindings and start/stop rules before changing the device or point type")
	}
	delete(data, "axisNames")
	delete(data, "axisGroupId")
	delete(data, "axisGroupName")
	data["direction"] = 0
	if group != "" {
		data["axisGroupId"], data["axisGroupName"], data["direction"] = group, field(previous, "axisGroupName"), integer(previous, "direction", 0)
	}
	validateAlarms(data)
}

func validateAlarms(data document) {
	direction := integer(data, "direction", 0)
	byCode := make(map[int]map[int]float64)
	for _, alarm := range objects(data, "alarms") {
		code, level, axis := integer(alarm, "code", -1), integer(alarm, "level", -1), integer(alarm, "direction", 0)
		limit := number(alarm, "limit", math.NaN())
		require(code >= 0 && level >= 1 && level <= 4 && integer(alarm, "type", 0) == 0 && finite(limit) && (axis == 0 || axis == direction),
			"Enter a valid metric, axis, level and alarm threshold")
		levels := byCode[code]
		if levels == nil {
			levels = make(map[int]float64)
			byCode[code] = levels
		}
		_, exists := levels[level]
		require(!exists, "Use each alarm level once per point direction and metric")
		levels[level] = limit
		alarm["type"], alarm["direction"] = 0, direction
	}
	for _, levels := range byCode {
		before := math.Inf(-1)
		for level := 1; level <= 4; level++ {
			if limit, ok := levels[level]; ok {
				require(limit > before, "Alarm thresholds must increase strictly by level for each point direction and metric")
				before = limit
			}
		}
	}
}

func axisID(group string, direction int) string {
	if direction != 2 && direction != 3 {
		return group
	}
	return nameID("sg500z-axis/" + group + "/" + strconv.Itoa(direction))
}

func splitAxes(group document, legacy bool) []document {
	id, name := field(group, "id"), field(group, "name")
	suffixes := []string{"", "-Z", "-X", "-Y"}
	points := make([]document, 0, 3)
	for direction := 1; direction <= 3; direction++ {
		axisName := strings.TrimSpace(field(object(group, "axisNames"), strconv.Itoa(direction)))
		require(legacy || len(utf16.Encode([]rune(axisName))) <= 80, "Axis names can have up to 80 characters")
		if axisName == "" {
			axisName = name + suffixes[direction]
		}
		point := clone(group)
		delete(point, "axisNames")
		point["id"], point["name"], point["type"] = axisID(id, direction), axisName, "VIBRATION"
		point["axisGroupId"], point["axisGroupName"], point["direction"] = id, name, direction
		alarms := []document{}
		positions := make(map[string]int)
		for _, alarm := range objects(group, "alarms") {
			configured := integer(alarm, "direction", 0)
			if configured != 0 && configured != direction || legacy && integer(alarm, "type", 0) != 0 {
				continue
			}
			row := clone(alarm)
			row["direction"] = direction
			if legacy {
				key := field(alarm, "code") + "/" + field(alarm, "level")
				if index, exists := positions[key]; exists {
					if configured != 0 {
						alarms[index] = row
					}
					continue
				}
				positions[key] = len(alarms)
			}
			alarms = append(alarms, row)
		}
		point["alarms"] = alarms
		points = append(points, point)
	}
	return points
}

func (a *app) migrateAxes() {
	a.transaction(func(tx *sql.Tx) {
		groups := make(map[string]bool)
		for _, group := range query(tx, "SELECT body FROM entity WHERE kind='points' AND json_extract(body,'$.type')=?", threeAxisType) {
			id := field(group, "id")
			groups[id] = true
			for _, point := range splitAxes(group, true) {
				axis := field(point, "id")
				direction := integer(point, "direction", 0)
				put(tx, "points", axis, point)
				if axis != id {
					execute(tx, "UPDATE metric SET point=? WHERE point=? AND direction=?", axis, id, direction)
					execute(tx, "UPDATE wave SET point=?,body=json_set(body,'$.pointId',?) WHERE point=? AND json_extract(body,'$.direction')=?", axis, axis, id, direction)
				}
			}
			execute(tx, "UPDATE OR IGNORE metric SET direction=1 WHERE point=? AND direction=0", id)
			execute(tx, "UPDATE wave SET body=json_set(body,'$.direction',1) WHERE point=? AND json_extract(body,'$.direction')=0", id)
		}
		if len(groups) == 0 {
			return
		}
		for _, device := range query(tx, "SELECT body FROM entity WHERE kind='devices'") {
			changed := migrateBindings(objects(device, "stopLimits"), groups, false)
			if changed {
				put(tx, "devices", field(device, "id"), device)
			}
		}
		for _, row := range query(tx, "SELECT json_object('id',collector,'version',version,'edition',json(body)) FROM edition") {
			edition := object(row, "edition")
			if migrateBindings(objects(object(edition, "collector"), "channels"), groups, true) {
				execute(tx, "UPDATE edition SET body=? WHERE collector=? AND version=?", jsonText(edition), field(row, "id"), field(row, "version"))
			}
		}
	})
}

func migrateBindings(bindings []document, groups map[string]bool, collector bool) bool {
	changed := false
	for _, binding := range bindings {
		id := field(binding, "pointId")
		if !groups[id] {
			continue
		}
		direction := max(1, integer(binding, "direction", 0))
		binding["pointId"], binding["direction"] = axisID(id, direction), direction
		if collector {
			binding["threeAxis"] = true
		}
		changed = true
	}
	return changed
}

func (a *app) validateCollector(data, previous document) {
	code := requiredText(data, "code", 150, "Enter a collector ID of up to 150 characters")
	require(collectorCodePattern.MatchString(code), "The collector ID contains invalid characters")
	id := field(data, "id")
	other := a.collectorByCode(code)
	require(other == nil || field(other, "id") == id, "This collector ID already exists")
	require(previous == nil || field(previous, "code") == code || a.latestEdition(id) == nil, "A collector ID is locked once settings have been published")
	data["code"], data["type"] = code, "SG500Z"
	system := object(data, "system")
	if system == nil {
		system = document{}
	}
	for _, key := range []string{"indexPeriod", "wavePeriod", "longWavePeriod", "uploadPeriod", "rtuPeriod", "techPeriod", "rpmPeriod", "longWaveOffset"} {
		require(integer(system, key, 0) >= 0, "Collection and upload intervals must be zero or more seconds")
	}
	compression := integer(system, "waveCompress", 1)
	require(compression >= 0 && compression <= 1, "Invalid waveform compression mode")
	system["collectType"], data["system"] = 1, system
	a.validateChannels(data)
	validateExtensions(data)
	for key, kind := range map[string]string{"rtu": "RTU", "modbusTcp": "MODBUS_TCP", "pt100": "PT100"} {
		for id := range extensionPointIDs(data, key) {
			point := a.referenced("points", id, "The point bound to the extension channel could not be found")
			require(field(point, "type") == kind, "Select a point that matches the extension channel type")
		}
	}
}

func (a *app) buildEdition(collector, previous document) document {
	snapshot := a.collectorSnapshot(collector)
	version := field(collector, "updatedAt")
	require(a.sampleTime(version) != "", "Invalid collector save time")
	if old := field(previous, "version"); version <= old {
		parsed, err := time.Parse(sampleLayout, old)
		must(err)
		version = parsed.Add(time.Millisecond).Format(sampleLayout)
	}
	return document{"version": version, "collector": snapshot, "wire": a.collectParameters(snapshot)}
}

func (a *app) savedParameters(id string) document {
	collector, previous := a.record("collectors", id), a.latestEdition(id)
	saved := field(collector, "updatedAt")
	if saved == "" || saved == field(object(previous, "collector"), "updatedAt") {
		return previous
	}
	a.validateCollector(collector, collector)
	edition := a.buildEdition(collector, previous)
	execute(a.store, insertEdition, id, field(edition, "version"), jsonText(edition))
	return edition
}

func (a *app) publishEdition(id string) document {
	collector := a.record("collectors", id)
	saved := a.savedParameters(id)
	if saved != nil && equalWire(a.collectParameters(a.collectorSnapshot(collector)), saved["wire"]) {
		return saved
	}
	a.save("collectors", id, collector)
	return a.latestEdition(id)
}

func equalWire(current, saved any) bool {
	// Java can store the same numeric setting as either 1 or 1.0.
	var currentWire, savedWire any
	must(json.Unmarshal([]byte(jsonText(current)), &currentWire))
	must(json.Unmarshal([]byte(jsonText(saved)), &savedWire))
	return reflect.DeepEqual(currentWire, savedWire)
}

func stopDirection(point, rule document) int {
	axis := integer(point, "direction", 0)
	direction := integer(rule, "direction", axis)
	if direction == 0 && axis == 1 && field(point, "axisGroupId") != "" && field(point, "type") == "VIBRATION" {
		return 1
	}
	return direction
}

func supportsStop(point, rule document) bool {
	if field(point, "type") != "VIBRATION" {
		return false
	}
	direction := stopDirection(point, rule)
	if field(point, "axisGroupId") == "" {
		return direction == 0
	}
	return integer(point, "direction", 0) == 1 && direction == 1
}

func (a *app) operationStatus(id string) document {
	device := a.record("devices", id)
	enabled := flag(device, "stopEnabled")
	rules := []document{}
	var observed string
	eligible, available := 0, 0
	running := false
	if enabled {
		for _, rule := range objects(device, "stopLimits") {
			pointID := field(rule, "pointId")
			point := entity(a.store, "points", pointID)
			code, direction := integer(rule, "code", 107), stopDirection(point, rule)
			limit := number(rule, "limit", math.NaN())
			row := document{"pointId": pointID, "pointName": field(point, "name"), "code": code, "direction": direction,
				"limit": nil, "value": nil, "time": nil, "collectorId": nil, "state": "UNSUPPORTED"}
			if finite(limit) {
				row["limit"] = limit
			}
			if supportsStop(point, rule) {
				eligible++
				row["state"] = "NO_DATA"
				var axis any
				if direction != 0 {
					axis = direction
				}
				metric := first(query(a.store, `SELECT json_object('collectorId',collector,'time',time,'value',value)
					FROM metric WHERE point=? AND code=? AND (? IS NULL OR direction=?) ORDER BY time DESC,rowid DESC LIMIT 1`, pointID, code, axis, axis))
				if value := number(metric, "value", math.NaN()); finite(value) && finite(limit) {
					available++
					row["value"], row["time"], row["collectorId"] = value, metric["time"], metric["collectorId"]
					observed = max(observed, field(metric, "time"))
					row["state"] = "BELOW"
					if value >= limit {
						row["state"], running = "AT_OR_ABOVE", true
					}
				}
			}
			rules = append(rules, row)
		}
	}
	state := "UNKNOWN"
	switch {
	case !enabled:
		state = "DISABLED"
	case running:
		state = "RUNNING"
	case eligible > 0 && available == eligible:
		state = "STOPPED"
	}
	return document{"deviceId": id, "enabled": enabled, "state": state, "observedAt": nullable(observed), "rules": rules}
}
