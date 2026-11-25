package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type apiResponse struct {
	status                  int
	data                    any
	path, name, contentType string
	content                 []byte
}

type apiAction func(*http.Request) apiResponse
type jsonAction func(*http.Request, document) apiResponse

func ok(value any) apiResponse      { return apiResponse{status: 200, data: value} }
func created(value any) apiResponse { return apiResponse{status: 201, data: value} }

func writeJSON(w http.ResponseWriter, status int, value any) {
	result := jsonText(value)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, result)
}

func writeAPIResponse(w http.ResponseWriter, r *http.Request, result apiResponse) {
	if result.status == 204 {
		w.WriteHeader(204)
		return
	}
	if result.contentType == "" {
		writeJSON(w, result.status, result.data)
		return
	}
	w.Header().Set("Content-Type", result.contentType)
	if result.path == "" {
		http.ServeContent(w, r, "audio.wav", time.Time{}, bytes.NewReader(result.content))
		return
	}
	f, err := os.Open(result.path)
	if os.IsNotExist(err) {
		reject(404, "File not found")
	}
	must(err)
	defer f.Close()
	info, err := f.Stat()
	must(err)
	if result.name != "" {
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": result.name}))
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}
	http.ServeContent(w, r, result.name, info.ModTime(), f)
}

func apiHandler(run func(http.ResponseWriter, *http.Request) apiResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if issue := recover(); issue != nil {
				if err, ok := issue.(requestError); ok {
					writeJSON(w, err.status, document{"message": err.message})
					return
				}
				log.Printf("%s %s failed: %s\n%s", r.Method, r.URL.Path, problemText(issue), debug.Stack())
				writeJSON(w, 500, document{"message": "Internal server error"})
			}
		}()
		writeAPIResponse(w, r, run(w, r))
	}
}

func (a *app) stateHandler(action apiAction) http.HandlerFunc {
	return apiHandler(func(_ http.ResponseWriter, r *http.Request) apiResponse {
		a.mu.Lock()
		defer a.mu.Unlock()
		return action(r)
	})
}

func (a *app) jsonHandler(action jsonAction) http.HandlerFunc {
	return apiHandler(func(w http.ResponseWriter, r *http.Request) apiResponse {
		media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if media != "application/json" && !(strings.HasPrefix(media, "application/") && strings.HasSuffix(media, "+json")) {
			reject(415, "Content-Type must be application/json")
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2*1024*1024))
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			reject(413, "The JSON request is too large")
		}
		require(err == nil, "Unable to read the request")
		data := asObject(decodeJSON(string(raw)))
		a.mu.Lock()
		defer a.mu.Unlock()
		return action(r, data)
	})
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	state := func(pattern string, action apiAction) { mux.HandleFunc(pattern, a.stateHandler(action)) }
	jsonRoute := func(pattern string, action jsonAction) { mux.HandleFunc(pattern, a.jsonHandler(action)) }
	mux.HandleFunc("GET /api/health", apiHandler(func(_ http.ResponseWriter, _ *http.Request) apiResponse {
		return ok(document{"running": true, "brokerConnected": a.connected()})
	}))
	state("GET /api/{kind}", func(r *http.Request) apiResponse {
		kind := r.PathValue("kind")
		checkKind(kind)
		return ok(a.entities(kind))
	})
	state("GET /api/{kind}/{id}", func(r *http.Request) apiResponse { return ok(a.record(r.PathValue("kind"), r.PathValue("id"))) })
	jsonRoute("POST /api/{kind}", func(r *http.Request, data document) apiResponse {
		return created(a.save(r.PathValue("kind"), "", data))
	})
	jsonRoute("PUT /api/{kind}/{id}", func(r *http.Request, data document) apiResponse {
		return ok(a.save(r.PathValue("kind"), r.PathValue("id"), data))
	})
	state("DELETE /api/{kind}/{id}", func(r *http.Request) apiResponse {
		a.remove(r.PathValue("kind"), r.PathValue("id"))
		return apiResponse{status: 204}
	})
	state("GET /api/devices/{id}/operation-status", func(r *http.Request) apiResponse { return ok(a.operationStatus(r.PathValue("id"))) })
	state("GET /api/collectors/{id}/status", func(r *http.Request) apiResponse { return ok(a.collectorStatus(r.PathValue("id"))) })
	state("POST /api/collectors/{id}/publish", func(r *http.Request) apiResponse {
		id := r.PathValue("id")
		edition := a.publishEdition(id)
		a.issueParameters(field(object(edition, "collector"), "code"), field(edition, "version"))
		return ok(a.collectorStatus(id))
	})
	state("POST /api/collectors/{id}/restart", func(r *http.Request) apiResponse {
		id := r.PathValue("id")
		code := field(a.record("collectors", id), "code")
		kind := "0"
		if value := r.URL.Query().Get("type"); value != "" {
			kind = strconv.Itoa(queryInteger(value, "type"))
		}
		if !a.isOnline(code) {
			reject(409, "The collector is offline")
		}
		a.issueCommand(code, "IssueRestart", "", []byte(kind))
		return ok(a.collectorStatus(id))
	})
	jsonRoute("POST /api/collectors/{id}/upgrade", func(r *http.Request, data document) apiResponse {
		id := r.PathValue("id")
		collector := a.record("collectors", id)
		task, files := a.assignFirmware(collector, data)
		a.issueFirmware(collector, task, files)
		return ok(a.collectorStatus(id))
	})
	state("GET /api/firmware", func(_ *http.Request) apiResponse { return ok(a.entities(firmwareUploads)) })
	state("GET /api/firmware/history", func(_ *http.Request) apiResponse { return ok(a.firmwareHistory()) })
	mux.HandleFunc("POST /api/firmware", apiHandler(a.readFirmwareUpload))

	state("GET /api/metrics", func(r *http.Request) apiResponse {
		p := r.URL.Query()
		return ok(a.metrics(nullable(p.Get("collectorId")), nullable(p.Get("pointId")), queryOptionalInt(p, "channelType"),
			queryIntegers(p, "channelTypes"), queryOptionalInt(p, "channelNo"), queryOptionalInt(p, "direction"), queryOptionalInt(p, "code"),
			nullable(a.sampleTime(p.Get("until"))), queryLimit(p, 100)))
	})
	state("GET /api/waves", func(r *http.Request) apiResponse {
		p := r.URL.Query()
		waves := a.waveRecords(nullable(p.Get("collectorId")), nullable(p.Get("pointId")), queryOptionalInt(p, "direction"),
			queryIntegers(p, "code"), queryBool(p, "audio"), queryLimit(p, 100))
		for _, wave := range waves {
			if wave["rms"] == nil {
				if values, err := a.readWave(wave); err == nil {
					a.cacheWaveRMS(wave, values)
				}
			}
		}
		return ok(waves)
	})
	state("GET /api/waves/{id}", func(r *http.Request) apiResponse { return ok(a.waveResponse(r.PathValue("id"), false)) })
	state("GET /api/waves/{id}/spectrum", func(r *http.Request) apiResponse { return ok(a.waveResponse(r.PathValue("id"), true)) })
	state("GET /api/waves/{id}/audio", func(r *http.Request) apiResponse { return a.audioResponse(r.PathValue("id")) })
	state("GET /api/sensor-info", func(r *http.Request) apiResponse {
		p := r.URL.Query()
		return ok(a.reports("SensorInfo", p.Get("collectorId"), queryLimit(p, 100)))
	})
	state("GET /api/battery-info", func(r *http.Request) apiResponse {
		p := r.URL.Query()
		return ok(a.reports("BatteryInfo", p.Get("collectorId"), queryLimit(p, 100)))
	})
	state("GET /api/logs", func(r *http.Request) apiResponse {
		p := r.URL.Query()
		id := nullable(p.Get("collectorId"))
		return ok(query(a.store, "SELECT body FROM mqtt_log WHERE (? IS NULL OR collector=?) ORDER BY id DESC LIMIT ?", id, id, queryLimit(p, 100)))
	})
	state("GET /api/log-files", func(r *http.Request) apiResponse {
		p := r.URL.Query()
		return ok(a.uploadedLogs(p.Get("collectorId"), queryLimit(p, 250)))
	})
	state("GET /api/log-files/{id}/download", func(r *http.Request) apiResponse {
		row := entity(a.store, collectorLogFiles, r.PathValue("id"))
		if row == nil {
			reject(404, "Log file not found")
		}
		path := a.logPath(row)
		if !regularFile(path) {
			reject(404, "Log file not found")
		}
		return apiResponse{status: 200, path: path, name: field(row, "name"), contentType: "application/octet-stream"}
	})
	return mux
}

func (a *app) readFirmwareUpload(w http.ResponseWriter, r *http.Request) apiResponse {
	r.Body = http.MaxBytesReader(w, r.Body, 65*1024*1024)
	err := r.ParseMultipartForm(1024 * 1024)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	var limit *http.MaxBytesError
	if errors.As(err, &limit) {
		reject(413, "The firmware upload exceeds 65 MB")
	}
	require(err == nil, "Expected a multipart firmware upload")
	file, info, err := r.FormFile("file")
	require(err == nil, "Select a firmware file")
	defer file.Close()
	require(info.Size > 0 && info.Size <= firmwareLimit, "Upload a nonempty firmware file of up to 64 MB")
	a.mu.Lock()
	defer a.mu.Unlock()
	return created(a.uploadFirmware(file, info.Filename, r.FormValue("version")))
}

func queryInteger(value, key string) int {
	number, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	require(err == nil, key+" must be an integer")
	return int(number)
}

func queryOptionalInt(p url.Values, key string) any {
	if value := p.Get(key); value != "" {
		return queryInteger(value, key)
	}
	return nil
}

func queryIntegers(p url.Values, key string) any {
	var items []int
	for _, group := range p[key] {
		for _, item := range strings.Split(group, ",") {
			if strings.TrimSpace(item) != "" {
				items = append(items, queryInteger(item, key))
			}
		}
	}
	if len(items) == 0 {
		return nil
	}
	return jsonText(items)
}

func queryBool(p url.Values, key string) any {
	switch strings.ToLower(strings.TrimSpace(p.Get(key))) {
	case "":
		return nil
	case "true", "on", "yes", "1":
		return 1
	case "false", "off", "no", "0":
		return 0
	default:
		reject(400, key+" must be a boolean")
		return nil
	}
}

func queryLimit(p url.Values, fallback int) int {
	if value := p.Get("limit"); value != "" {
		return min(1000, max(1, queryInteger(value, "limit")))
	}
	return fallback
}

func (a *app) waveResponse(id string, transform bool) document {
	wave := a.wave(id)
	if wave == nil {
		reject(404, "Waveform not found")
	}
	values, err := a.readWave(wave)
	must(err)
	if wave["rms"] == nil {
		a.cacheWaveRMS(wave, values)
	}
	if transform {
		frequency, step := spectrum(values, number(wave, "sampleRate", 0))
		wave["values"], wave["frequencyStep"] = frequency, step
	} else {
		wave["values"] = values
	}
	return wave
}

func (a *app) audioResponse(id string) apiResponse {
	wave := a.wave(id)
	if integer(wave, "code", -1) != 0 {
		reject(404, "Audio recording not found")
	}
	require(waveIDPattern.MatchString(id), "Invalid waveform ID")
	path := filepath.Join(a.directory, "audio", id+".wav")
	if regularFile(path) {
		return apiResponse{status: 200, path: path, contentType: "audio/wav"}
	}
	values, err := a.readWave(wave)
	must(err)
	return apiResponse{status: 200, contentType: "audio/wav", content: legacyAudio(number(wave, "sampleRate", 0), values)}
}
