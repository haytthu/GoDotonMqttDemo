package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"
)

const firmwareUploads = "firmware_upload"
const upgrades = "upgrade"
const upgradeHistory = "upgrade_history"
const firmwareLimit = 64 * 1024 * 1024

var md5Pattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
var controlChars = regexp.MustCompile(`[\x00-\x1f\x7f]`)

func baseName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	return name[strings.LastIndexByte(name, '/')+1:]
}

func (a *app) firmwarePath(hash string) string {
	require(md5Pattern.MatchString(hash), "Invalid firmware MD5")
	return filepath.Join(a.directory, "firmware", hash+".bin")
}

func (a *app) uploadFirmware(input io.Reader, original, version string) document {
	name := controlChars.ReplaceAllString(baseName(original), "_")
	require(name != "" && len(utf16.Encode([]rune(name))) <= 180, "Invalid firmware file name")
	dir := filepath.Join(a.directory, "firmware")
	must(os.MkdirAll(dir, 0755))
	file, err := os.CreateTemp(dir, "upload-*.tmp")
	must(err)
	defer os.Remove(file.Name())
	defer file.Close()
	digest, crc := md5.New(), crc32.NewIEEE()
	size, err := io.Copy(io.MultiWriter(file, digest, crc), io.LimitReader(input, firmwareLimit+1))
	must(err)
	must(file.Close())
	require(size > 0 && size <= firmwareLimit, "Upload a nonempty firmware file of up to 64 MB")
	hash := fmt.Sprintf("%X", digest.Sum(nil))
	path := a.firmwarePath(hash)
	if !regularFile(path) {
		must(os.Rename(file.Name(), path))
	}
	id := randomID()
	entry := document{"id": id, "name": name, "size": size, "md5": hash, "crc": crc.Sum32(), "version": strings.TrimSpace(version), "time": a.now()}
	put(a.store, firmwareUploads, id, entry)
	return entry
}

func taskFiles(task document) []document {
	if task == nil {
		return []document{}
	}
	if _, exists := task["files"]; exists {
		return objects(task, "files")
	}
	files := []document{}
	if file := object(task, "file"); file != nil {
		progress := integer(task, "upgradeProgress", 0)
		file["progress"], file["confirmed"], task["targetVersion"] = progress, progress == 100, field(file, "version")
		files = append(files, file)
	}
	task["files"] = files
	return files
}

func (a *app) firmwareHistory() []document {
	history := query(a.store, `SELECT CASE WHEN json_extract(body,'$.id') IS NULL
		THEN json_set(body,'$.id','legacy-' || id,'$.collectorId',id) ELSE body END FROM entity WHERE kind IN (?,?)
		ORDER BY COALESCE(json_extract(body,'$.startedAt'),json_extract(body,'$.file.time')) DESC,rowid DESC`, upgrades, upgradeHistory)
	for _, task := range history {
		taskFiles(task)
	}
	return history
}

func firmwareDescriptor(file document) document {
	return document{"Name": file["name"], "Size": file["size"], "Md5Code": file["md5"], "CRC": file["crc"]}
}

func selectedFirmware(value any) string {
	id, ok := value.(string)
	require(ok && strings.TrimSpace(id) != "", "Select a firmware file")
	return id
}

func (a *app) assignFirmware(collector, request document) (document, []document) {
	ids := []string{}
	if raw, present := request["firmwareIds"]; present {
		items, ok := raw.([]any)
		require(ok && len(items) >= 1 && len(items) <= 10, "Select between 1 and 10 firmware files")
		for _, item := range items {
			ids = append(ids, selectedFirmware(item))
		}
	} else {
		ids = append(ids, selectedFirmware(request["firmwareId"]))
	}
	version, ok := request["targetVersion"].(string)
	require(request["targetVersion"] == nil || ok, "The target version must be text")
	descriptors, files := []document{}, []document{}
	seen := make(map[string]bool)
	for _, id := range ids {
		file := entity(a.store, firmwareUploads, id)
		if file == nil {
			reject(404, "Upload a firmware file for this upgrade")
		}
		hash := field(file, "md5")
		require(!seen[hash], "The upgrade contains duplicate firmware files")
		seen[hash] = true
		file["progress"], file["confirmed"] = 0, false
		descriptors, files = append(descriptors, firmwareDescriptor(file)), append(files, file)
	}
	if request["targetVersion"] == nil && len(files) == 1 {
		version = field(files[0], "version")
	}
	id := field(collector, "id")
	task := document{"id": randomID(), "collectorId": id, "collectorCode": field(collector, "code"), "collectorName": field(collector, "name"),
		"startedAt": a.now(), "targetVersion": strings.TrimSpace(version), "files": files, "upgradeState": "PENDING", "upgradeProgress": 0,
		"upgradeMessage": "等待设备请求升级文件"}
	a.transaction(func(tx *sql.Tx) {
		for _, upload := range ids {
			affected, err := execute(tx, "DELETE FROM entity WHERE kind=? AND id=?", firmwareUploads, upload).RowsAffected()
			must(err)
			if affected != 1 {
				reject(404, "Upload a firmware file for this upgrade")
			}
		}
		if previous := entity(tx, upgrades, id); previous != nil {
			previousID := field(previous, "id")
			if previousID == "" {
				previousID = "legacy-" + id
				previous["id"], previous["collectorId"] = previousID, id
			}
			put(tx, upgradeHistory, previousID, previous)
		}
		put(tx, upgrades, id, task)
	})
	return task, descriptors
}

func canTransfer(task document) bool {
	if task == nil {
		return false
	}
	switch field(task, "upgradeState") {
	case "DOWNLOADED", "RE_REGISTERED", "VERSION_MISMATCH", "FAILED":
		return false
	default:
		return true
	}
}

func (a *app) availableFirmware(id string) []document {
	result := []document{}
	task := entity(a.store, upgrades, id)
	if canTransfer(task) {
		for _, file := range taskFiles(task) {
			if !flag(file, "confirmed") {
				result = append(result, firmwareDescriptor(file))
			}
		}
	}
	return result
}

func (a *app) updateUpgrade(id string, task document, state string, progress int, message string) {
	task["upgradeState"], task["upgradeProgress"], task["upgradeMessage"] = state, progress, message
	put(a.store, upgrades, id, task)
}

func (a *app) upgradeNotification(id, attempt, status, message string) bool {
	task := entity(a.store, upgrades, id)
	before := field(task, "upgradeState")
	if attempt != field(task, "id") || before != "PENDING" && before != "NOTIFIED" {
		return false
	}
	a.updateUpgrade(id, task, status, integer(task, "upgradeProgress", 0), message)
	return true
}

func totalProgress(files []document) int {
	var total, weighted int64
	for _, file := range files {
		size := int64(integer(file, "size", 0))
		total += size
		weighted += size * int64(integer(file, "progress", 0))
	}
	if total == 0 {
		return 0
	}
	return int(weighted / total)
}

func (a *app) firmwarePacket(id string, request document) document {
	task := entity(a.store, upgrades, id)
	files := taskFiles(task)
	hash := field(request, "Md5Code")
	var file document
	for _, candidate := range files {
		if strings.EqualFold(hash, field(candidate, "md5")) {
			file = candidate
			break
		}
	}
	require(file != nil, "No matching firmware update task for this collector")
	if integer(request, "IsConfirm", 0) == 1 {
		if !canTransfer(task) || flag(file, "confirmed") {
			return nil
		}
		file["progress"], file["confirmed"] = 100, true
		count := 0
		for _, row := range files {
			if flag(row, "confirmed") {
				count++
			}
		}
		if count == len(files) {
			a.updateUpgrade(id, task, "DOWNLOADED", 100, "设备已确认下载，等待重启上报版本")
		} else {
			a.updateUpgrade(id, task, "TRANSFERRING", totalProgress(files), fmt.Sprintf("Confirmed %d of %d firmware downloads", count, len(files)))
		}
		return nil
	}
	require(canTransfer(task) && !flag(file, "confirmed"), "No active firmware transfer for this collector")
	size := integer(file, "size", 0)
	maxLength, index := integer(request, "PackMaxLength", 0), integer(request, "PackIndex", 0)
	offset := int64(index) * int64(maxLength)
	require(maxLength >= 1 && maxLength <= 1024*1024 && index >= 0 && offset < int64(size), "Invalid firmware packet index or packet size")
	input, err := os.Open(a.firmwarePath(field(file, "md5")))
	must(err)
	defer input.Close()
	data := make([]byte, min(size-int(offset), maxLength))
	_, err = input.ReadAt(data, offset)
	must(err)
	count := (size + maxLength - 1) / maxLength
	progress := min(99, int((offset+int64(len(data)))*100/int64(size)))
	file["progress"] = max(progress, integer(file, "progress", 0))
	a.updateUpgrade(id, task, "TRANSFERRING", totalProgress(files), fmt.Sprintf("已响应设备请求，第 %d / %d 包", index+1, count))
	return document{"Md5Code": hash, "PackIndex": index, "PackCount": count, "Content": base64.StdEncoding.EncodeToString(data)}
}

func (a *app) firmwareRegistered(id, version string) {
	task := entity(a.store, upgrades, id)
	if field(task, "upgradeState") != "DOWNLOADED" {
		return
	}
	taskFiles(task)
	expected := field(task, "targetVersion")
	state := "RE_REGISTERED"
	if expected != "" && expected != version {
		state = "VERSION_MISMATCH"
	}
	reported := version
	if reported == "" {
		reported = "未知"
	}
	message := "重启后上报版本 " + reported
	if expected != "" {
		message += "，期望版本 " + expected
	}
	a.updateUpgrade(id, task, state, 100, message)
}
