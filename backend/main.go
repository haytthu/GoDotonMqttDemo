package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const sampleLayout = "2006-01-02T15:04:05.000"
const mqttLayout = "2006-01-02T15:04:05"

type document map[string]any

type app struct {
	*store
	mu            sync.Mutex
	location      *time.Location
	client        mqtt.Client
	prefix        string
	upgradeList   bool
	online        map[string]bool
	pending       map[string]pendingCommand
	messageID     uint32
	waveTransfers map[transferKey]*waveTransfer
	quickWaves    map[transferKey]*quickWave
	logTransfers  map[string]*logTransfer
}

type requestError struct {
	status  int
	message string
}

func reject(status int, message string) { panic(requestError{status, message}) }

func require(condition bool, message string) {
	if !condition {
		reject(http.StatusBadRequest, message)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	location, err := time.LoadLocation(env("DEMO_ZONE", "Asia/Shanghai"))
	if err != nil {
		log.Fatal(err)
	}
	directory, err := filepath.Abs(env("DEMO_DATA_DIR", "./data"))
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "logs"), 0755); err != nil {
		log.Fatal(err)
	}
	logFile, err := os.OpenFile(filepath.Join(directory, "logs", "application.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(os.Stderr, logFile))

	db := openStore(directory)
	defer db.Close()
	a := &app{
		store: db, location: location, prefix: env("MQTT_PREFIX", "iesd"),
		upgradeList: strings.EqualFold(env("MQTT_UPGRADE_LIST_PAYLOAD", "true"), "true"),
		online:      make(map[string]bool), pending: make(map[string]pendingCommand),
		waveTransfers: make(map[transferKey]*waveTransfer), quickWaves: make(map[transferKey]*quickWave),
		logTransfers: make(map[string]*logTransfer),
	}
	a.initConfiguration()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	stopMQTT := a.startMQTT(ctx)
	defer stopMQTT()

	server := &http.Server{
		Addr:    net.JoinHostPort(env("DEMO_BIND_ADDRESS", "127.0.0.1"), env("DEMO_PORT", "8080")),
		Handler: a.routes(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 90 * time.Second,
		WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Print(err)
		}
	}()
	log.Printf("SG500Z API listening on %s", server.Addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func jsonText(value any) string {
	var out strings.Builder
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	must(encoder.Encode(value))
	return strings.TrimSuffix(out.String(), "\n")
}

func decodeJSON(raw string) any {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	require(decoder.Decode(&value) == nil, "Invalid JSON")
	require(decoder.Decode(new(any)) == io.EOF, "Invalid JSON")
	return value
}

func asObject(value any) document {
	switch row := value.(type) {
	case document:
		return row
	case map[string]any:
		return document(row)
	default:
		reject(400, "Expected a JSON object")
		return nil
	}
}

func object(o document, key string) document {
	if o[key] == nil {
		return nil
	}
	return asObject(o[key])
}

func asArray(value any, key string) []any {
	if value == nil {
		return []any{}
	}
	items, ok := value.([]any)
	require(ok, key+" must be an array")
	return items
}

func objects(o document, key string) []document {
	if rows, ok := o[key].([]document); ok {
		return rows
	}
	items := asArray(o[key], key)
	rows := make([]document, 0, len(items))
	for _, item := range items {
		rows = append(rows, asObject(item))
	}
	return rows
}

func field(o document, key string) string {
	switch value := o[key].(type) {
	case nil:
		return ""
	case string:
		return value
	case json.Number, bool, int, int64, float64:
		return fmt.Sprint(value)
	}
	reject(400, key+" must be a primitive value")
	return ""
}

func number(o document, key string, fallback float64) float64 {
	if o[key] == nil {
		return fallback
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(field(o, key)), 64)
	require(err == nil, key+" must be a number")
	return value
}

func integer(o document, key string, fallback int) int {
	if o[key] == nil {
		return fallback
	}
	value := number(o, key, 0)
	require(finite(value) && value >= math.MinInt32 && value <= math.MaxInt32, key+" must be an integer")
	if _, ok := o[key].(string); ok {
		require(value == math.Trunc(value), key+" must be an integer")
	}
	return int(value)
}

func flag(o document, key string) bool { return strings.EqualFold(field(o, key), "true") }
func finite(value float64) bool        { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func clone(o document) document        { return asObject(decodeJSON(jsonText(o))) }

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (a *app) now() string     { return time.Now().In(a.location).Format(sampleLayout) }
func (a *app) nowMQTT() string { return time.Now().In(a.location).Format(mqttLayout) }

func (a *app) sampleTime(source string) string {
	source = strings.ReplaceAll(strings.TrimSpace(source), " ", "T")
	if parsed, err := time.Parse(time.RFC3339Nano, source); err == nil {
		return parsed.In(a.location).Format(sampleLayout)
	}
	for _, layout := range []string{mqttLayout, "2006-01-02T15:04"} {
		if parsed, err := time.Parse(layout, source); err == nil {
			return parsed.Format(sampleLayout)
		}
	}
	return ""
}
