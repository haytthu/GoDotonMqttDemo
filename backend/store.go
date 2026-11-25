package main

import (
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const upsertEntity = "INSERT INTO entity(kind,id,body) VALUES(?,?,?) ON CONFLICT(kind,id) DO UPDATE SET body=excluded.body"
const insertEdition = "INSERT INTO edition(collector,version,body) VALUES(?,?,?)"
const collectorPointFilter = "(? IS NULL OR collector=?) AND (? IS NULL OR point=?)"

type store struct {
	*sql.DB
	directory string
}

type database interface {
	Query(string, ...any) (*sql.Rows, error)
	Exec(string, ...any) (sql.Result, error)
}

func openStore(directory string) *store {
	db, err := sql.Open("sqlite", filepath.Join(directory, "sg500z.sqlite"))
	must(err)
	db.SetMaxOpenConns(1)
	s := &store{db, directory}
	execute(s, "PRAGMA busy_timeout=5000")
	execute(s, "PRAGMA journal_mode=WAL")
	execute(s, `
CREATE TABLE IF NOT EXISTS entity (kind TEXT NOT NULL, id TEXT NOT NULL, body TEXT NOT NULL, PRIMARY KEY(kind,id));
CREATE TABLE IF NOT EXISTS edition (collector TEXT NOT NULL, version TEXT NOT NULL, activated TEXT, body TEXT NOT NULL, PRIMARY KEY(collector,version));
CREATE INDEX IF NOT EXISTS edition_active ON edition(collector,activated);
CREATE TABLE IF NOT EXISTS metric (collector TEXT NOT NULL, point TEXT NOT NULL, channel_type INTEGER NOT NULL, channel_no INTEGER NOT NULL, direction INTEGER NOT NULL, sub_channel INTEGER NOT NULL, code INTEGER NOT NULL, time TEXT NOT NULL, value REAL NOT NULL, PRIMARY KEY(collector,point,channel_type,channel_no,direction,sub_channel,code,time));
CREATE INDEX IF NOT EXISTS metric_point_time ON metric(point,time DESC);
CREATE INDEX IF NOT EXISTS metric_collector_time ON metric(collector,time DESC);
CREATE TABLE IF NOT EXISTS wave (id TEXT PRIMARY KEY NOT NULL, collector TEXT NOT NULL, point TEXT, time TEXT NOT NULL, body TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS wave_point_time ON wave(point,time DESC);
CREATE TABLE IF NOT EXISTS mqtt_log (id INTEGER PRIMARY KEY, collector TEXT, time TEXT NOT NULL, body TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS mqtt_log_collector ON mqtt_log(collector,id DESC);
CREATE TABLE IF NOT EXISTS collector_report (id TEXT PRIMARY KEY NOT NULL, kind TEXT NOT NULL, collector_code TEXT NOT NULL, time TEXT NOT NULL, body TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS collector_report_time ON collector_report(kind,time DESC);`)
	return s
}

func (s *store) transaction(action func(*sql.Tx)) {
	tx, err := s.Begin()
	must(err)
	defer tx.Rollback()
	action(tx)
	must(tx.Commit())
}

func execute(db database, statement string, args ...any) sql.Result {
	result, err := db.Exec(statement, args...)
	must(err)
	return result
}

func query(db database, statement string, args ...any) []document {
	cursor, err := db.Query(statement, args...)
	must(err)
	defer cursor.Close()
	result := []document{}
	for cursor.Next() {
		var raw string
		must(cursor.Scan(&raw))
		result = append(result, asObject(decodeJSON(raw)))
	}
	must(cursor.Err())
	return result
}

func first(rows []document) document {
	if len(rows) == 0 {
		return nil
	}
	return rows[0]
}

func (s *store) entities(kind string) []document {
	return query(s, "SELECT body FROM entity WHERE kind=? ORDER BY rowid", kind)
}

func entity(db database, kind, id string) document {
	return first(query(db, "SELECT body FROM entity WHERE kind=? AND id=?", kind, id))
}

func put(db database, kind, id string, body document) {
	execute(db, upsertEntity, kind, id, jsonText(body))
}

func (s *store) collectorByCode(code string) document {
	return first(query(s, "SELECT body FROM entity WHERE kind='collectors' AND json_extract(body,'$.code')=?", code))
}

func (s *store) pointGroup(group string) []document {
	return query(s, "SELECT body FROM entity WHERE kind='points' AND json_extract(body,'$.axisGroupId')=? ORDER BY json_extract(body,'$.direction')", group)
}

func (s *store) latestEdition(id string) document {
	return first(query(s, "SELECT body FROM edition WHERE collector=? ORDER BY version DESC LIMIT 1", id))
}

func (s *store) editionAt(id, time string) document {
	return first(query(s, "SELECT body FROM edition WHERE collector=? AND activated<=? ORDER BY activated DESC,version DESC LIMIT 1", id, time))
}

type sample struct {
	point                                               string
	channelType, channelNo, direction, subChannel, code int
	value                                               float64
	time                                                string
}

func (s *store) addMetrics(id string, entries []sample) {
	if len(entries) == 0 {
		return
	}
	s.transaction(func(tx *sql.Tx) {
		statement, err := tx.Prepare(`INSERT INTO metric(collector,point,channel_type,channel_no,direction,sub_channel,code,time,value) VALUES(?,?,?,?,?,?,?,?,?)
			ON CONFLICT(collector,point,channel_type,channel_no,direction,sub_channel,code,time) DO UPDATE SET value=excluded.value`)
		must(err)
		defer statement.Close()
		for _, m := range entries {
			_, err := statement.Exec(id, m.point, m.channelType, m.channelNo, m.direction, m.subChannel, m.code, m.time, m.value)
			must(err)
		}
	})
}

func (s *store) metrics(collector, point, channelType, channelTypes, channelNo, direction, code, until any, limit int) []document {
	return query(s, `SELECT json_object('collectorId',collector,'pointId',point,'channelType',channel_type,'channelNo',channel_no,
		'direction',direction,'subChannel',sub_channel,'code',code,'time',time,'value',value) FROM metric WHERE `+collectorPointFilter+
		` AND (? IS NULL OR channel_type=?) AND (? IS NULL OR channel_no=?) AND (? IS NULL OR channel_type IN (SELECT value FROM json_each(?)))
		AND (? IS NULL OR direction=?) AND (? IS NULL OR code=?) AND (? IS NULL OR time<=?) ORDER BY time DESC,rowid DESC LIMIT ?`,
		collector, collector, point, point, channelType, channelType, channelNo, channelNo, channelTypes, channelTypes, direction, direction, code, code, until, until, limit)
}

func (s *store) wave(id string) document {
	return first(query(s, "SELECT body FROM wave WHERE id=?", id))
}

func (s *store) saveReports(kind, code string, reports []document) {
	s.transaction(func(tx *sql.Tx) {
		for _, report := range reports {
			execute(tx, "INSERT INTO collector_report(id,kind,collector_code,time,body) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body",
				field(report, "id"), kind, code, field(report, "time"), jsonText(report))
		}
	})
}

func (s *store) waveRecords(collector, point, direction, codes, audio any, limit int) []document {
	return query(s, "SELECT body FROM wave WHERE "+collectorPointFilter+
		` AND (? IS NULL OR json_extract(body,'$.direction')=?) AND (? IS NULL OR (COALESCE(json_extract(body,'$.code'),-1)=0)=?)
		AND (? IS NULL OR json_extract(body,'$.code') IN (SELECT value FROM json_each(?))) ORDER BY time DESC,rowid DESC LIMIT ?`,
		collector, collector, point, point, direction, direction, audio, audio, codes, codes, limit)
}

func uuid(data [16]byte, version byte) string {
	data[6] = data[6]&0x0f | version<<4
	data[8] = data[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", data[:4], data[4:6], data[6:8], data[8:10], data[10:])
}

func nameID(key string) string { return uuid(md5.Sum([]byte(key)), 3) }

func randomID() string {
	var data [16]byte
	_, err := rand.Read(data[:])
	must(err)
	return uuid(data, 4)
}
