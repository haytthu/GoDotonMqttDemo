package main

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const collectorLogFiles = "collector-log-files"
const maxLogContent = 96 * 1024 * 1024

var logNameChars = regexp.MustCompile(`[:*?"<>|\x00-\x1f\x7f]`)
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type logTransfer struct {
	parts       map[int]string
	count, size int
	name        string
	lastSeen    time.Time
}

func cleanLogName(name string) string {
	result := strings.TrimSpace(logNameChars.ReplaceAllString(baseName(name), "_"))
	if result == "." || result == ".." {
		return ""
	}
	return result
}

func logFileID(code, name, hash string) string { return nameID(code + "\x00" + name + "\x00" + hash) }

func (a *app) logPath(row document) string {
	id := field(row, "id")
	if strings.HasPrefix(id, "legacy-") {
		name := field(row, "fileRef")
		require(name != "" && name != "." && name != ".." && name == baseName(name) && filepath.VolumeName(name) == "",
			"Invalid collector log file storage location")
		return filepath.Join(a.directory, "collector-logs", name)
	}
	require(uuidPattern.MatchString(id), "Invalid collector log file ID")
	return filepath.Join(a.directory, "collector-logs", "objects", id+".bin")
}

func (a *app) acceptLog(code string, frame document) {
	hash := strings.ToLower(strings.TrimSpace(field(frame, "Md5Code")))
	require(hash == "" || md5Pattern.MatchString(hash), "Invalid collector log file MD5")
	name := cleanLogName(field(frame, "FileName"))
	count, index := integer(frame, "PackCount", 0), integer(frame, "PackIndex", 0)
	require(count >= 1 && count <= 65536 && index >= 0 && index <= count && frame["Content"] != nil, "Invalid collector log file packet")
	if hash != "" && name != "" {
		previous := entity(a.store, collectorLogFiles, logFileID(code, name, hash))
		if field(previous, "status") == "VERIFIED" && regularFile(a.logPath(previous)) {
			return
		}
	}
	key := code + "\x00" + hash
	if hash == "" {
		key = code + "\x00" + name
	}
	now := time.Now()
	transfer := a.logTransfers[key]
	if transfer == nil {
		require(len(a.logTransfers) < 256, "Collector log packet buffer is full")
		transfer = &logTransfer{parts: make(map[int]string), count: count, name: name, lastSeen: now}
		a.logTransfers[key] = transfer
	}
	require(transfer.count == count && (name == "" || transfer.name == "" || name == transfer.name), "Collector log packets disagree on the file name or packet count")
	if transfer.name == "" {
		transfer.name = name
	}
	content := compactBase64(field(frame, "Content"))
	change := len(content) - len(transfer.parts[index])
	active := 0
	for _, item := range a.logTransfers {
		active += item.size
	}
	if transfer.size+change > maxLogContent || active+change > 2*maxLogContent {
		delete(a.logTransfers, key)
		reject(400, "Collector log file exceeds the receive buffer capacity")
	}
	transfer.parts[index], transfer.size, transfer.lastSeen = content, transfer.size+change, now
	_, zero := transfer.parts[0]
	_, last := transfer.parts[count]
	if len(transfer.parts) != count || zero == last {
		return
	}
	defer delete(a.logTransfers, key)
	name = transfer.name
	if name == "" {
		name = "collector.log"
		if hash != "" {
			name = hash + ".log"
		}
	}
	start := 0
	if last {
		start = 1
	}
	parts := make([]string, count)
	for i := range parts {
		parts[i] = transfer.parts[start+i]
	}
	a.saveLog(code, name, hash, parts)
}

func (a *app) saveLog(code, name, expected string, parts []string) {
	dir := filepath.Join(a.directory, "collector-logs", "objects")
	must(os.MkdirAll(dir, 0755))
	file, err := os.CreateTemp(dir, "pending-*.tmp")
	must(err)
	defer os.Remove(file.Name())
	defer file.Close()
	digest := md5.New()
	output := io.MultiWriter(file, digest)
	valid := true
	for _, part := range parts {
		data, err := decodeBase64(part)
		if err != nil {
			valid = false
			break
		}
		_, err = output.Write(data)
		must(err)
	}
	actual := fmt.Sprintf("%x", digest.Sum(nil))
	if expected == "" {
		require(valid, "Collector log file has no valid Base64 content")
	} else if !valid || expected != actual {
		must(file.Truncate(0))
		_, err := file.Seek(0, io.SeekStart)
		must(err)
		digest.Reset()
		inputs := make([]io.Reader, 0, len(parts)+1)
		length := 0
		for _, part := range parts {
			inputs = append(inputs, strings.NewReader(part))
			length += len(part)
		}
		if padding := (4 - length%4) % 4; padding != 0 {
			inputs = append(inputs, strings.NewReader(strings.Repeat("=", padding)))
		}
		decoder := base64.NewDecoder(base64.StdEncoding, io.MultiReader(inputs...))
		if _, err := io.Copy(output, decoder); err != nil {
			var corrupt base64.CorruptInputError
			if errors.As(err, &corrupt) || errors.Is(err, io.ErrUnexpectedEOF) {
				reject(400, "Collector log file MD5 mismatch")
			}
			must(err)
		}
		actual = fmt.Sprintf("%x", digest.Sum(nil))
		require(expected == actual, "Collector log file MD5 mismatch")
	}
	info, err := file.Stat()
	must(err)
	must(file.Close())
	id := logFileID(code, name, actual)
	target := a.logPath(document{"id": id})
	if previous := entity(a.store, collectorLogFiles, id); previous != nil && regularFile(target) {
		if expected != "" && field(previous, "status") == "UNVERIFIED" {
			previous["status"] = "VERIFIED"
			put(a.store, collectorLogFiles, id, previous)
		}
		return
	}
	must(os.Rename(file.Name(), target))
	status := "UNVERIFIED"
	if expected != "" {
		status = "VERIFIED"
	}
	row := document{"id": id, "name": name, "size": info.Size(), "time": a.now(), "status": status, "code": code, "md5": actual,
		"collectorId": nullable(field(a.collectorByCode(code), "id"))}
	put(a.store, collectorLogFiles, id, row)
}

func (a *app) importLegacyLogs() {
	dir := filepath.Join(a.directory, "collector-logs")
	children, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	must(err)
	for _, entry := range children {
		if !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()
		id := "legacy-" + nameID(name)
		if entity(a.store, collectorLogFiles, id) != nil {
			continue
		}
		info, err := entry.Info()
		must(err)
		row := document{"id": id, "name": cleanLogName(name), "size": info.Size(), "time": info.ModTime().In(a.location).Format(sampleLayout), "status": "LEGACY", "fileRef": name}
		put(a.store, collectorLogFiles, id, row)
	}
}

func (a *app) uploadedLogs(id string, limit int) []document {
	a.importLegacyLogs()
	rows := []document{}
	for _, row := range a.entities(collectorLogFiles) {
		if (id == "" || id == field(row, "collectorId")) && regularFile(a.logPath(row)) {
			delete(row, "fileRef")
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		left, right := field(rows[i], "time"), field(rows[j], "time")
		if left != right {
			return left > right
		}
		return field(rows[i], "id") < field(rows[j], "id")
	})
	return rows[:min(limit, len(rows))]
}
