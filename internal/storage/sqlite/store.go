package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			source TEXT NOT NULL,
			session_id TEXT,
			payload_json TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS actions (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			risk TEXT NOT NULL,
			status TEXT NOT NULL,
			source TEXT NOT NULL,
			session_id TEXT,
			payload_json TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS workflows (
			id TEXT PRIMARY KEY,
			workflow_json TEXT NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SaveEvent(ctx context.Context, event types.Event) error {
	if err := event.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT OR REPLACE INTO events (id, type, source, session_id, payload_json, created_at) VALUES (?, ?, ?, ?, ?, ?)`, event.ID, event.Type, event.Source, event.SessionID, string(payload), event.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *Store) ListEvents(ctx context.Context, limit int) ([]types.Event, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, type, source, session_id, payload_json, created_at FROM events ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []types.Event{}
	for rows.Next() {
		var event types.Event
		var payload string
		var createdAt string
		if err := rows.Scan(&event.ID, &event.Type, &event.Source, &event.SessionID, &payload, &createdAt); err != nil {
			return nil, err
		}
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &event.Payload); err != nil {
				return nil, err
			}
		}
		parsed, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		event.CreatedAt = parsed
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) SaveAction(ctx context.Context, action types.ActionRequest) error {
	if err := action.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(action.Payload)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT OR REPLACE INTO actions (id, type, risk, status, source, session_id, payload_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, action.ID, action.Type, action.Risk, action.Status, action.Source, action.SessionID, string(payload), action.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *Store) ListActions(ctx context.Context, limit int) ([]types.ActionRequest, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, type, risk, status, source, session_id, payload_json, created_at FROM actions ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := []types.ActionRequest{}
	for rows.Next() {
		var action types.ActionRequest
		var risk string
		var status string
		var payload string
		var createdAt string
		if err := rows.Scan(&action.ID, &action.Type, &risk, &status, &action.Source, &action.SessionID, &payload, &createdAt); err != nil {
			return nil, err
		}
		action.Risk = types.ActionRisk(risk)
		action.Status = types.ActionStatus(status)
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &action.Payload); err != nil {
				return nil, err
			}
		}
		parsed, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		action.CreatedAt = parsed
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

func (s *Store) SaveWorkflow(ctx context.Context, workflow types.Workflow) error {
	if err := workflow.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT OR REPLACE INTO workflows (id, workflow_json) VALUES (?, ?)`, workflow.ID, string(data))
	return err
}

func (s *Store) ListWorkflows(ctx context.Context) ([]types.Workflow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT workflow_json FROM workflows ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workflows := []types.Workflow{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var workflow types.Workflow
		if err := json.Unmarshal([]byte(data), &workflow); err != nil {
			return nil, err
		}
		workflows = append(workflows, workflow)
	}
	return workflows, rows.Err()
}

func Memory() (*Store, error) {
	return Open(fmt.Sprintf("file:auxitalk-%d?mode=memory&cache=shared", time.Now().UnixNano()))
}
