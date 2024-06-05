package persistence

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log"
	"os"
	"time"
)

const (
	DbConnTpl              = "postgres://%s:%s@%s:%s/%s"
	DbParticipantTableName = "participant"
	DbTrialTableName       = "trial"

	DbParticipantTableDefinition = `CREATE TABLE "%s" (
		id SERIAL PRIMARY KEY,
		data JSONB
	);`
	DbTrialTableDefinition = `CREATE TABLE "%s" (
		%s_id INTEGER NOT NULL,
		id SERIAL,
		data JSONB,
		PRIMARY KEY (%s_id, id),
        FOREIGN KEY (%s_id) REFERENCES %s (id)
	);`
	DbHyperTableDefinition = `CREATE TABLE "%s_data" (
        time TIMESTAMPTZ NOT NULL,
        %s_id INTEGER,
        %s_id INTEGER,
        data JSONB,
        FOREIGN KEY (%s_id, %s_id) REFERENCES %s (%s_id, id)
	);`

	DBInsertStatement = `INSERT INTO "%s_data" VALUES ($1, $2, $3, $4);`
)

type JsonEvent struct {
	Id      string `json:"id"`
	Message string `json:"msg"`
}

func (a JsonEvent) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *JsonEvent) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

type TimescaleHandler struct {
	ctx        context.Context
	connection *pgx.Conn
	writeChan  chan JsonEvent

	prefix      string
	participant int
	trial       int
}

// NewTimescaleHandler creates a new PersistenceHandler and creates persistence in TimescaleDB
func NewTimescaleHandler() *TimescaleHandler {
	ph := &TimescaleHandler{}
	return ph
}

func (h *TimescaleHandler) Init() error {
	var err error
	if err = h.verifyDatabase(); err != nil {
		return err
	}
	if err = h.connect(false); err != nil {
		return err
	}
	if err = h.verifyTables(); err != nil {
		return err
	}
	// Set latest participant and trial
	if h.participant, err = h.LastParticipant(); err != nil {
		return err
	}
	if h.trial, err = h.LastTrial(h.participant); err != nil {
		return err
	}
	h.writeChan = make(chan JsonEvent)
	go h.persistRoutine()
	return nil
}

func (h *TimescaleHandler) connect(global bool) error {
	var err error
	if h.ctx == nil {
		h.ctx = context.Background()
	}
	params := []any{
		os.Getenv("PERSIST_TIMESCALE_USERNAME"),
		os.Getenv("PERSIST_TIMESCALE_PASSWORD"),
		os.Getenv("PERSIST_TIMESCALE_HOST"),
		os.Getenv("PERSIST_TIMESCALE_PORT"),
	}
	if global {
		params = append(params, "")
	} else {
		// Connect to specific DB
		params = append(params, os.Getenv("PERSIST_TIMESCALE_DATABASE"))
	}
	h.connection, err = pgx.Connect(h.ctx, fmt.Sprintf(
		DbConnTpl,
		params...,
	))
	return err
}

func (h *TimescaleHandler) Close() error {
	return h.connection.Close(h.ctx)
}

func (h *TimescaleHandler) SetPrefix(prefix string) error {
	h.prefix = prefix
	return nil
}

func (h *TimescaleHandler) SetParticipant(participant int) error {
	h.participant = participant
	_, err := h.connection.Exec(h.ctx,
		fmt.Sprintf(
			"INSERT INTO %s (id, data) VALUES ($1, $2);",
			h.tbl(DbParticipantTableName),
		),
		h.participant,
		make(map[string]any),
	)
	return err
}

func (h *TimescaleHandler) LastParticipant() (int, error) {
	lastParticipant := 0
	qry := fmt.Sprintf(`SELECT MAX(id) FROM %s GROUP BY id;`, h.tbl(DbParticipantTableName))
	row := h.connection.QueryRow(h.ctx, qry)
	err := row.Scan(&lastParticipant)
	return lastParticipant, err
}

func (h *TimescaleHandler) SetTrial(trial int) error {
	h.trial = trial
	_, err := h.connection.Exec(h.ctx,
		fmt.Sprintf(
			"INSERT INTO %s (%s_id, id, data) VALUES ($1, $2, $3);",
			h.tbl(DbTrialTableName),
			DbParticipantTableName,
		),
		h.participant,
		h.trial,
		make(map[string]any),
	)
	return err
}

func (h *TimescaleHandler) LastTrial(participant int) (int, error) {
	lastTrial := 0
	qry := fmt.Sprintf(`SELECT MAX(id) FROM %s WHERE %s_id = $1 GROUP BY id;`, h.tbl(DbTrialTableName), DbParticipantTableName)
	row := h.connection.QueryRow(h.ctx, qry, participant)
	err := row.Scan(&lastTrial)
	return lastTrial, err
}

// persistRoutine reads from the persistence channel and writes to the file
func (h *TimescaleHandler) persistRoutine() {
	_, err := h.connection.Prepare(
		h.ctx,
		"timescale_stm",
		fmt.Sprintf(DBInsertStatement, h.tbl(DbTrialTableName)),
	)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		buf := <-h.writeChan
		if _, err = h.connection.Exec(
			h.ctx,
			"timescale_stm",
			time.Now(),
			h.participant,
			h.trial,
			buf,
		); err != nil {
			fmt.Printf("Error on Part %d, Trial %d: %v\n", h.participant, h.trial, err)
		}
	}
}

func (h *TimescaleHandler) AddEntry(id string, msg []byte) error {
	h.writeChan <- JsonEvent{
		Id:      id,
		Message: string(msg),
	}
	return nil
}

func (h *TimescaleHandler) verifyDatabase() error {
	var err error
	if err = h.connect(true); err != nil {
		return err
	}
	defer h.connection.Close(h.ctx)
	dbName := os.Getenv("PERSIST_TIMESCALE_DATABASE")
	qry := "SELECT EXISTS (SELECT FROM pg_catalog.pg_database WHERE datname = $1);"
	var dbExists bool
	if err = h.connection.QueryRow(
		h.ctx,
		qry,
		dbName,
	).Scan(&dbExists); err != nil {
		return err
	}
	if !dbExists {
		// Create Database
		qry = fmt.Sprintf(
			`CREATE DATABASE "%s" WITH OWNER "%s" ENCODING 'UTF8' LC_COLLATE = 'C.UTF-8' LC_CTYPE = 'C.UTF-8';`,
			dbName,
			os.Getenv("PERSIST_TIMESCALE_USERNAME"),
		)
		if _, err = h.connection.Exec(h.ctx, qry); err != nil {
			return err
		}
	}
	return err
}

func (h *TimescaleHandler) verifyTables() error {
	tExist, err := h.tablesExists()
	if err != nil {
		return err
	}
	if !tExist {
		return h.createTables()
	}
	return nil
}

func (h *TimescaleHandler) tablesExists() (bool, error) {
	qry := `SELECT EXISTS (SELECT FROM pg_catalog.pg_tables WHERE schemaname = 'public' AND tablename = $1);`
	var tableExists bool
	if err := h.connection.QueryRow(
		h.ctx,
		qry,
		h.tbl(DbParticipantTableName),
	).Scan(&tableExists); err != nil {
		log.Println(err)
		return false, err
	}
	return tableExists, nil
}

func (h *TimescaleHandler) createTables() error {
	var err error
	// 1. Participant
	qry := fmt.Sprintf(
		DbParticipantTableDefinition,
		h.tbl(DbParticipantTableName),
	)
	if _, err = h.connection.Exec(
		h.ctx,
		qry,
	); err != nil {
		return err
	}
	// 2. Trial
	qry = fmt.Sprintf(
		DbTrialTableDefinition,
		h.tbl(DbTrialTableName),
		DbParticipantTableName,
		DbParticipantTableName,
		DbParticipantTableName,
		h.tbl(DbParticipantTableName),
	)
	if _, err = h.connection.Exec(
		h.ctx,
		qry,
	); err != nil {
		return err
	}
	// 2. Trial data hypertable
	qry = fmt.Sprintf(
		DbHyperTableDefinition,
		h.tbl(DbTrialTableName),
		DbParticipantTableName,
		DbTrialTableName,
		DbParticipantTableName,
		DbTrialTableName,
		h.tbl(DbTrialTableName),
		DbParticipantTableName,
	)
	if _, err = h.connection.Exec(
		h.ctx,
		qry,
	); err != nil {
		return err
	}
	return nil
}

func (h *TimescaleHandler) tbl(name string) string {
	return h.prefix + "_" + name + "s"
}
