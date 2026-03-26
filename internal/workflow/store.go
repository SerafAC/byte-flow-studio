package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages the SQLite database for workflow persistence.
type Store struct {
	db *sql.DB
}

// OpenStore opens (or creates) a .byteflow SQLite file and initialises the schema.
func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite requires a single connection: in-memory databases are per-connection
	// (multiple pool connections each get an empty DB), and file-based SQLite
	// supports only one writer at a time regardless.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

const schema = `
CREATE TABLE IF NOT EXISTS workflow (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    definition  TEXT NOT NULL,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id                     TEXT PRIMARY KEY,
    workflow_id            TEXT NOT NULL,
    start_time             INTEGER NOT NULL,
    end_time               INTEGER,
    end_reason             TEXT,
    raw_layer_enabled      INTEGER NOT NULL DEFAULT 1,
    processed_layer_enabled INTEGER NOT NULL DEFAULT 1,
    data_volume            INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS raw_data (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL,
    block_id    TEXT NOT NULL,
    timestamp   INTEGER NOT NULL,
    data        BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS processed_data (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL,
    block_id    TEXT NOT NULL,
    timestamp   INTEGER NOT NULL,
    data_values BLOB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_raw_data_session    ON raw_data (session_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_processed_session   ON processed_data (session_id, block_id, timestamp);
`

func (s *Store) initSchema() error {
	_, err := s.db.Exec(schema)
	return err
}

// SaveWorkflow upserts the workflow into the database.
func (s *Store) SaveWorkflow(wf *Workflow) error {
	defBytes, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("marshal workflow: %w", err)
	}

	now := time.Now().UnixMilli()
	wf.UpdatedAt = now

	_, err = s.db.Exec(`
		INSERT INTO workflow (id, name, definition, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			definition = excluded.definition,
			updated_at = excluded.updated_at
	`, wf.ID, wf.Name, string(defBytes), wf.CreatedAt, now)
	return err
}

// LoadWorkflow reads the workflow from the database (first row, or specific ID).
func (s *Store) LoadWorkflow(id string) (*Workflow, error) {
	var defStr string
	var wf Workflow

	err := s.db.QueryRow(`
		SELECT id, name, definition, created_at, updated_at FROM workflow WHERE id = ?
	`, id).Scan(&wf.ID, &wf.Name, &defStr, &wf.CreatedAt, &wf.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query workflow: %w", err)
	}

	if err := json.Unmarshal([]byte(defStr), &wf); err != nil {
		return nil, fmt.Errorf("not a valid byteflow file: %w", err)
	}
	return &wf, nil
}

// LoadAnyWorkflow loads the first workflow in the database (for single-workflow files).
func (s *Store) LoadAnyWorkflow() (*Workflow, error) {
	var id string
	err := s.db.QueryRow(`SELECT id FROM workflow LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query workflow: %w", err)
	}
	return s.LoadWorkflow(id)
}

// DB returns the underlying database for use by other stores.
func (s *Store) DB() *sql.DB {
	return s.db
}
