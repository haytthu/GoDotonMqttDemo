package main

import (
	"bytes"
	"compress/bzip2"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/bits"
	"math/cmplx"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gonum.org/v1/gonum/dsp/fourier"
)

const sampleRateFactor = 2.56
const maxDecodedWaveBytes = 2 * 1048576

var waveIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var waveFilePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}\.f32$`)

func rms(values []float32) float64 {
	var sum float64
	for _, value := range values {
		sum += float64(value) * float64(value)
	}
	return math.Sqrt(sum / float64(len(values)))
}

type velocitySampling struct {
	lower, upper                   float64
	decimation, start, end, length int
}

func prepareVelocity(freq int, low, high float64, velocityLength, lowAccLength, plus int) velocitySampling {
	require(freq > 0 && velocityLength > 0 && lowAccLength >= 2 && plus >= 0 && int64(lowAccLength)+int64(plus) <= math.MaxInt32,
		"Invalid low-frequency acceleration sample count or waveform parameters")
	require(finite(low) && low >= 0 && finite(high) && high > low, "Invalid integration band for the velocity waveform")
	if low < 1 {
		low = 2
	}
	decimation := max(1, int(float64(freq)/high))
	upper := min(high, float64(freq))
	if decimation > 1 && math.Mod(float64(freq), upper) != 0 {
		upper = math.Floor(float64(freq) / float64(decimation))
	}
	require(upper > low, "Invalid integration band for the actual acquisition frequency")
	start, end := 3*plus/4, lowAccLength+plus-plus/4
	length := (end-start-1)/decimation + 1
	require(length >= velocityLength, "Not enough low-frequency acceleration samples for the configured velocity waveform")
	return velocitySampling{low, upper, decimation, start, end, length}
}

func centered(values []float32) []complex128 {
	mean := waveMean(values)
	result := make([]complex128, len(values))
	for i, value := range values {
		result[i] = complex(float64(value)-mean, 0)
	}
	return result
}

func waveMean(values []float32) float64 {
	var total float64
	for _, value := range values {
		require(finite(float64(value)), "The waveform contains invalid samples")
		total += float64(value)
	}
	return total / float64(len(values))
}

func spectrum(values []float32, rate float64) ([]float32, float64) {
	n := len(values)
	require(n >= 2 && finite(rate) && rate > 0, "A waveform needs at least two valid samples and a positive sampling rate")
	input := centered(values)
	newWaveFFT(n)(input, false)
	result := make([]float32, int(math.RoundToEven(float64(n)/sampleRateFactor)))
	for i := range result {
		result[i] = float32(cmplx.Abs(input[i]) / float64(n/2))
	}
	return result, rate / float64(n)
}

func velocity(values []float32, freq int, sampling velocitySampling) ([]float32, float64) {
	n := len(values)
	rate := float64(freq) * sampleRateFactor
	step := rate / float64(n)
	transform := centered(values)
	fft := newWaveFFT(n)
	fft(transform, false)
	first, last := int(math.Floor(sampling.lower/step))+1, int(math.Ceil(sampling.upper/step))
	for i := range transform {
		bin := i
		if i > (n-1)/2 {
			bin = i - n
		}
		if absBin := max(bin, -bin); absBin >= first && absBin <= last {
			transform[i] *= complex(0, -1/(2*math.Pi*float64(bin)*step))
		} else {
			transform[i] = 0
		}
	}
	fft(transform, true)
	mid := float64(n-1) / 2
	var mean, cross, sumSquares float64
	for i, entry := range transform {
		value := real(entry) / float64(n)
		pos := float64(i) - mid
		mean += value
		cross += pos * value
		sumSquares += pos * pos
	}
	mean /= float64(n)
	slope := cross / sumSquares
	for i, entry := range transform {
		transform[i] = complex(real(entry)/float64(n)-mean-slope*(float64(i)-mid), 0)
	}
	transform[n-1] = transform[n-2]
	var trimmedMean float64
	for i := sampling.start; i < sampling.end; i += sampling.decimation {
		trimmedMean += real(transform[i])
	}
	trimmedMean /= float64(sampling.length)
	result := make([]float32, sampling.length)
	for i := range result {
		result[i] = float32((real(transform[sampling.start+i*sampling.decimation]) - trimmedMean) * 1000)
	}
	return result, rate / float64(sampling.decimation)
}

func newWaveFFT(length int) func([]complex128, bool) {
	remainder := length
	for factor := 2; factor <= 31; factor++ {
		for remainder%factor == 0 {
			remainder /= factor
		}
	}
	if remainder == 1 {
		fft := fourier.NewCmplxFFT(length)
		return func(values []complex128, inverse bool) {
			if inverse {
				fft.Sequence(values, values)
			} else {
				fft.Coefficients(values, values)
			}
		}
	}

	// Bluestein's algorithm keeps arbitrary sample counts, including prime lengths,
	// on a power-of-two FFT rather than a quadratic transform.
	size := 1 << bits.Len(uint(2*length-2))
	fft := fourier.NewCmplxFFT(size)
	chirp := make([]complex128, length)
	kernel := make([]complex128, size)
	kernel[0], chirp[0] = 1, 1
	for i := 1; i < length; i++ {
		phase := -math.Pi * float64(int64(i)*int64(i)%int64(2*length)) / float64(length)
		chirp[i] = cmplx.Rect(1, phase)
		kernel[i], kernel[size-i] = cmplx.Conj(chirp[i]), cmplx.Conj(chirp[i])
	}
	fft.Coefficients(kernel, kernel)
	work := make([]complex128, size)
	return func(values []complex128, inverse bool) {
		clear(work[length:])
		for i, value := range values {
			if inverse {
				value = cmplx.Conj(value)
			}
			work[i] = value * chirp[i]
		}
		fft.Coefficients(work, work)
		for i := range work {
			work[i] *= kernel[i]
		}
		fft.Sequence(work, work)
		for i := range values {
			value := work[i] * chirp[i] / complex(float64(size), 0)
			if inverse {
				value = cmplx.Conj(value)
			}
			values[i] = value
		}
	}
}

func (a *app) waveform(code, id string, frame document, raw []byte) (success bool) {
	defer func() {
		if problem := recover(); problem != nil {
			message := problemText(problem)
			log.Printf("Waveform processing failed for %s: %s", code, message)
			a.logMessage(code, "SYSTEM", "WaveData", "", 0, message)
		}
	}()
	var err error
	switch integer(frame, "ZipType", 0) {
	case 0:
	case 1:
		raw, err = io.ReadAll(io.LimitReader(bzip2.NewReader(bytes.NewReader(raw)), maxDecodedWaveBytes+1))
		must(err)
	default:
		reject(400, "Unsupported waveform compression type")
	}
	require(len(raw) > 0 && len(raw) <= maxDecodedWaveBytes && len(raw)%2 == 0,
		fmt.Sprintf("16-bit waveform data must contain a positive, even number of bytes: %d", len(raw)))
	waveCode := integer(frame, "Code", 0)
	coef := float32(number(frame, "Coef", 0))
	audio := waveCode == 0
	var order binary.ByteOrder = binary.BigEndian
	if audio {
		order = binary.LittleEndian
		if coef == 0 {
			coef = 1
		}
	}
	require(finite(float64(coef)) && coef != 0, "Invalid waveform conversion coefficient")
	values := make([]float32, len(raw)/2)
	for i := range values {
		values[i] = float32(int16(order.Uint16(raw[i*2:]))) * coef
	}
	mean := waveMean(values)
	for i, value := range values {
		values[i] = float32(float64(value) - mean)
		require(finite(float64(values[i])), "The waveform contains invalid samples")
	}
	if waveCode == 15 {
		return true
	}
	time := a.sampleTime(field(frame, "Time"))
	require(time != "", "Invalid waveform acquisition timestamp")
	edition := a.editionAt(id, time)
	collector := object(edition, "collector")
	channelType, channelNo, dir := integer(frame, "ChType", 0), integer(frame, "ChNo", 0), integer(frame, "Dir", 0)
	channel := findChannel(collector, channelType, channelNo, dir)
	direction := sampleDirection(channel, dir)
	freq := number(frame, "Freq", 0)
	rate := freq * sampleRateFactor
	waveKey := fmt.Sprintf("%s/%d/%d/%d/%d/%d/%s", id, integer(frame, "Id", 0), channelType, channelNo, dir, waveCode, time)
	waveID := nameID(waveKey)
	unit := "m/s²"
	if audio {
		unit = "PCM"
		path := filepath.Join(a.directory, "audio", waveID+".wav")
		if !regularFile(path) {
			must(writeAtomic(path, waveAudio(rate, raw)))
		}
	}
	metadata := document{"id": waveID, "pointId": nullable(field(channel, "pointId")), "collectorId": id, "channelType": channelType, "channelNo": channelNo,
		"direction": direction, "code": waveCode, "time": time, "sampleRate": rate, "unit": unit, "configVersion": field(edition, "version"),
		"plusLength": integer(frame, "PlusLength", 0), "sourceWaveId": nil}
	a.saveWave(metadata, values)
	if waveCode == 12 && integer(frame, "Src", 0) != 12 && channel != nil {
		a.saveVelocity(code, collector, channel, metadata, waveKey, freq, values)
	}
	return true
}

func (a *app) saveVelocity(code string, collector, channel, metadata document, waveKey string, freq float64, values []float32) {
	length := setting(collector, channel, "velocityLength", 4096)
	if length == 0 {
		return
	}
	plus := setting(collector, channel, "plusLength", 1024)
	expected := setting(collector, channel, "lowAccLength", 16384)
	reported := integer(metadata, "plusLength", 0)
	if reported != plus || len(values) != expected+plus {
		a.logMessage(code, "SYSTEM", "WaveData", "", nil,
			fmt.Sprintf("Velocity waveform was not generated. Extra samples: configured %d, reported %d. Total samples: expected %d, received %d", plus, reported, expected+plus, len(values)))
		return
	}
	defer func() {
		if issue := recover(); issue != nil {
			if invalid, ok := issue.(requestError); ok {
				a.logMessage(code, "SYSTEM", "WaveData", "", nil, "Velocity waveform was not generated: "+invalid.message)
				return
			}
			panic(issue)
		}
	}()
	require(freq == math.Trunc(freq) && freq > 0 && freq <= math.MaxInt32, "Invalid waveform analysis frequency")
	sampling := prepareVelocity(int(freq), float64(setting(collector, channel, "velocityLowFreq", 2)),
		float64(setting(collector, channel, "velocityHighFreq", 1000)), length, expected, plus)
	integrated, rate := velocity(values, int(freq), sampling)
	metadata = clone(metadata)
	metadata["sourceWaveId"], metadata["id"] = metadata["id"], nameID(waveKey+"/velocity")
	metadata["code"], metadata["unit"], metadata["sampleRate"] = 15, "mm/s", rate
	a.saveWave(metadata, integrated)
}

func (a *app) saveWave(metadata document, values []float32) {
	id := field(metadata, "id")
	if a.wave(id) != nil {
		return
	}
	data := make([]byte, 4*len(values))
	for i, value := range values {
		binary.BigEndian.PutUint32(data[i*4:], math.Float32bits(value))
	}
	file := id + ".f32"
	must(writeAtomic(filepath.Join(a.directory, "waves", file), data))
	metadata["file"], metadata["length"] = file, len(values)
	if value := rms(values); finite(value) {
		metadata["rms"] = value
	}
	execute(a.store, "INSERT INTO wave(id,collector,point,time,body) VALUES(?,?,?,?,?) ON CONFLICT(id) DO NOTHING",
		id, field(metadata, "collectorId"), metadata["pointId"], field(metadata, "time"), jsonText(metadata))
}

func (a *app) readWave(wave document) ([]float32, error) {
	file, length := field(wave, "file"), integer(wave, "length", 0)
	if !waveFilePattern.MatchString(file) || length <= 0 || length > maxDecodedWaveBytes/2 {
		return nil, errors.New("invalid waveform file name or sample count")
	}
	path := filepath.Join(a.directory, "waves", file)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() != int64(length)*4 {
		return nil, errors.New("the waveform file size does not match the sample count")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) != int(info.Size()) {
		return nil, errors.New("the waveform file size changed during reading")
	}
	values := make([]float32, length)
	for i := range values {
		values[i] = math.Float32frombits(binary.BigEndian.Uint32(data[i*4:]))
	}
	return values, nil
}

func (a *app) cacheWaveRMS(wave document, values []float32) {
	if value := rms(values); finite(value) {
		wave["rms"] = value
		execute(a.store, "UPDATE wave SET body=json_set(body,'$.rms',?) WHERE id=? AND json_extract(body,'$.rms') IS NULL", value, field(wave, "id"))
	}
}

func waveAudio(rate float64, pcm []byte) []byte {
	rounded := math.Round(rate)
	require(finite(rate) && rounded >= 1 && rounded <= math.MaxInt32/2, "Invalid audio analysis frequency")
	audio := make([]byte, 44+len(pcm))
	copy(audio, "RIFF")
	copy(audio[8:], "WAVEfmt ")
	copy(audio[36:], "data")
	order := binary.LittleEndian
	order.PutUint32(audio[4:], uint32(36+len(pcm)))
	order.PutUint32(audio[16:], 16)
	order.PutUint16(audio[20:], 1)
	order.PutUint16(audio[22:], 1)
	order.PutUint32(audio[24:], uint32(rounded))
	order.PutUint32(audio[28:], uint32(rounded)*2)
	order.PutUint16(audio[32:], 2)
	order.PutUint16(audio[34:], 16)
	order.PutUint32(audio[40:], uint32(len(pcm)))
	copy(audio[44:], pcm)
	return audio
}

func legacyAudio(rate float64, values []float32) []byte {
	var peak float64
	for _, v := range values {
		if finite(float64(v)) {
			peak = max(peak, math.Abs(float64(v)))
		}
	}
	pcm := make([]byte, len(values)*2)
	if peak > 0 {
		for i, v := range values {
			if finite(float64(v)) {
				binary.LittleEndian.PutUint16(pcm[i*2:], uint16(int16(math.Floor(float64(v)*32767/peak+0.5))))
			}
		}
	}
	return waveAudio(rate, pcm)
}

func regularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, "pending-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	_, writeErr := temp.Write(data)
	if err := errors.Join(writeErr, temp.Close()); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func compactBase64(s string) string {
	var clean strings.Builder
	clean.Grow(len(s))
	for _, char := range s {
		switch char {
		case ' ', '\t', '\r', '\n', '\v', '\f':
		default:
			clean.WriteRune(char)
		}
	}
	return clean.String()
}
