package main

import (
	"bytes"
	"hash/crc32"
	"math"
	"time"
)

const transferTTL = 10 * time.Minute
const maxWaveBytes = 32 * 1024 * 1024
const maxWaveBuffer = 128 * 1024 * 1024

type transferKey struct {
	collector                      string
	id                             int
	channel, kind, code, direction int
	time                           string
}

type waveTransfer struct {
	frame       document
	metadata    string
	packets     map[int][]byte
	count, size int
	complete    bool
	lastSeen    time.Time
}

type quickWave struct {
	packets     map[int]legacyWavePacket
	count, size int
	lastSeen    time.Time
}

type legacyWavePacket struct {
	frame   document
	content []byte
}

func (a *app) expireTransfers(now time.Time) {
	for key, transfer := range a.waveTransfers {
		if now.Sub(transfer.lastSeen) >= transferTTL {
			delete(a.waveTransfers, key)
		}
	}
	for key, transfer := range a.quickWaves {
		if now.Sub(transfer.lastSeen) >= transferTTL {
			delete(a.quickWaves, key)
		}
	}
	for key, transfer := range a.logTransfers {
		if now.Sub(transfer.lastSeen) >= transferTTL {
			delete(a.logTransfers, key)
		}
	}
}

func (a *app) bufferedWaves() int {
	var size int
	for _, transfer := range a.waveTransfers {
		size += transfer.size
	}
	for _, transfer := range a.quickWaves {
		size += transfer.size
	}
	return size
}

func waveCRC(frame document) uint32 {
	crc := number(frame, "CRC", 0)
	require(finite(crc) && crc == math.Trunc(crc) && crc >= -1<<63 && crc < 1<<63, "Invalid waveform CRC")
	return uint32(int64(crc))
}

func (a *app) packetWave(collector, id, command, messageID string, frame document) {
	key := transferKey{collector: collector, id: inRange(frame["Id"], "Id", math.MinInt32, math.MaxInt32)}
	transfer := a.waveTransfers[key]
	now := time.Now()
	switch command {
	case "WaveDataIndex":
		if inRange(frame["Code"], "Code", math.MinInt32, math.MaxInt32) == 2 {
			delete(a.waveTransfers, key)
			a.reply(collector, command, messageID, 0, nil)
			return
		}
		count := inRange(frame["PackCount"], "PackCount", 1, 65536)
		delete(frame, "Content")
		frame["PackIndex"] = -1
		if frame["CRC"] != nil {
			frame["CRC"] = int64(waveCRC(frame))
		}
		metadata := jsonText(frame)
		if transfer != nil && transfer.metadata == metadata {
			transfer.lastSeen = now
		} else {
			if transfer == nil && len(a.waveTransfers) >= 128 {
				for old, item := range a.waveTransfers {
					if item.complete {
						delete(a.waveTransfers, old)
						break
					}
				}
				if len(a.waveTransfers) >= 128 {
					a.reply(collector, command, messageID, -6, nil)
					return
				}
			}
			a.waveTransfers[key] = &waveTransfer{frame: frame, metadata: metadata, packets: make(map[int][]byte), count: count, lastSeen: now}
		}
		a.reply(collector, command, messageID, 1, nil)
	case "WaveDataPacket":
		if transfer == nil || transfer.complete {
			return
		}
		position := inRange(frame["PackIndex"], "PackIndex", 1, transfer.count)
		data := decodeBytes(frame["Content"], "Content")
		require(len(data) > 0, "Waveform packet lacks data")
		delta := len(data) - len(transfer.packets[position])
		if transfer.size+delta > maxWaveBytes || a.bufferedWaves()+delta > maxWaveBuffer {
			clear(transfer.packets)
			transfer.size = 0
			reject(400, "Waveform packets exceed the receive buffer capacity")
		}
		transfer.lastSeen, transfer.size, transfer.packets[position] = now, transfer.size+delta, data
		if len(transfer.packets) == transfer.count {
			raw := make([]byte, 0, transfer.size)
			for i := 1; i <= transfer.count; i++ {
				raw = append(raw, transfer.packets[i]...)
			}
			clear(transfer.packets)
			transfer.size = 0
			if transfer.frame["CRC"] == nil || crc32.ChecksumIEEE(raw) == waveCRC(transfer.frame) {
				transfer.complete = a.waveform(collector, id, transfer.frame, raw)
			} else {
				a.logMessage(collector, "SYSTEM", command, "", 0, "Waveform packet CRC32 mismatch")
			}
		}
	case "WaveDataConfirm":
		missing := []int{}
		if transfer != nil {
			if _, exists := frame["PackCount"]; exists {
				require(inRange(frame["PackCount"], "PackCount", 1, 65536) == transfer.count, "Waveform confirmation packet count differs from the index message")
			}
			transfer.lastSeen = now
			if !transfer.complete {
				for i := 1; i <= transfer.count; i++ {
					if _, received := transfer.packets[i]; !received {
						missing = append(missing, i)
					}
				}
			}
		}
		a.reply(collector, command, messageID, 1, document{"Content": missing})
	}
}

func missingPackets[T any](packets map[int]T, start, count int) []int {
	missing := []int{}
	for i := start; i < start+count; i++ {
		if _, exists := packets[i]; !exists {
			missing = append(missing, i)
		}
	}
	return missing
}

func wavePacketContent(frame document) []byte {
	if frame["Content"] == nil {
		return nil
	}
	if data, ok := frame["Content"].(string); ok {
		result, err := decodeBase64(compactBase64(data))
		require(err == nil, "Invalid waveform Base64 content")
		return result
	}
	return decodeBytes(frame["Content"], "Content")
}

func (a *app) legacyWave(collector, id, messageID string, frame document) {
	const command = "WaveData"
	code := inRange(frame["Code"], "Code", math.MinInt32, math.MaxInt32)
	if code == 2 {
		a.reply(collector, command, messageID, 0, nil)
		return
	}
	key := transferKey{collector, integer(frame, "Id", 0), integer(frame, "ChNo", 0),
		integer(frame, "ChType", 0), code, integer(frame, "Dir", 0), field(frame, "Time")}
	count, position := integer(frame, "PackCount", 0), integer(frame, "PackIndex", 0)
	require(count >= 1 && count <= 4096 && position >= 0 && position <= count, "Invalid waveform packet index or count")
	status, processed, complete := 1, true, false
	quick := integer(frame, "SendModel", 0) == 1
	payload := document{"Id": key.id, "PackIndex": position}
	transfer := a.quickWaves[key]
	if transfer == nil {
		require(len(a.quickWaves) < 128, "Waveform packet buffer is full")
		transfer = &quickWave{packets: make(map[int]legacyWavePacket), count: count}
		a.quickWaves[key] = transfer
	}
	require(transfer.count == count, "Waveform packets disagree on the packet count")
	content, valid := legacyContent(frame)
	if valid {
		delta := len(content) - len(transfer.packets[position].content)
		if transfer.size+delta > maxWaveBytes || a.bufferedWaves()+delta > maxWaveBuffer {
			delete(a.quickWaves, key)
			reject(400, "Waveform packets exceed the receive buffer capacity")
		}
		transfer.lastSeen, transfer.size = time.Now(), transfer.size+delta
		delete(frame, "Content")
		transfer.packets[position] = legacyWavePacket{frame, content}
		start := -1
		if len(transfer.packets) >= count {
			for _, base := range []int{0, 1} {
				if len(missingPackets(transfer.packets, base, count)) == 0 {
					start = base
					break
				}
			}
		}
		if start >= 0 {
			complete = true
			delete(a.quickWaves, key)
			var raw bytes.Buffer
			raw.Grow(transfer.size)
			for i := start; i < start+count; i++ {
				raw.Write(transfer.packets[i].content)
			}
			expected := waveCRC(transfer.packets[start+count-1].frame)
			if expected == 0 || crc32.ChecksumIEEE(raw.Bytes()) == expected {
				processed = a.waveform(collector, id, transfer.packets[start].frame, raw.Bytes())
			} else {
				processed = false
				a.logMessage(collector, "SYSTEM", command, "", 0, "Waveform CRC32 mismatch")
			}
		}
	} else {
		delete(a.quickWaves, key)
		complete, processed = true, false
	}
	if !processed {
		status = 0
	}
	if quick {
		if complete {
			a.reply(collector, command, messageID, status, document{"UnRceivePack": []int{}})
			return
		}
		start := -1
		if _, exists := transfer.packets[0]; exists {
			start = 0
		} else if _, exists := transfer.packets[count]; exists {
			start = 1
		}
		if start >= 0 && position == start+count-1 {
			a.reply(collector, command, messageID, status, document{"UnRceivePack": missingPackets(transfer.packets, start, count)})
		}
		return
	}
	a.reply(collector, command, messageID, status, payload)
}

func legacyContent(frame document) (data []byte, valid bool) {
	defer func() {
		if recover() != nil {
			valid = false
		}
	}()
	return wavePacketContent(frame), true
}
