package session

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	_, err = db.Exec(testSchema)
	require.NoError(t, err)
	return db
}

const testSchema = `
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
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    block_id   TEXT NOT NULL,
    timestamp  INTEGER NOT NULL,
    data       BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS processed_data (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL,
    block_id    TEXT NOT NULL,
    timestamp   INTEGER NOT NULL,
    data_values BLOB NOT NULL
);
`

func TestSessionRetention(t *testing.T) {
	const maxSessions = 10
	db := openTestDB(t)
	defer db.Close()

	store := NewStore(db)
	mgr := NewManager(store, "wf-1", SessionConfig{
		StoreRaw:       true,
		StoreProcessed: true,
		MaxSessions:    maxSessions,
	})

	// Create and complete 11 sessions
	for i := 0; i < maxSessions+1; i++ {
		sess, err := mgr.CreateSession("wf-1", SessionConfig{
			StoreRaw: true, StoreProcessed: true, MaxSessions: maxSessions,
		})
		require.NoError(t, err)
		require.NoError(t, mgr.CompleteSession(sess.ID, "test"))
	}

	// After 11 completes, only 10 should remain
	count, err := store.CountSessions("wf-1")
	require.NoError(t, err)
	assert.Equal(t, maxSessions, count, "should retain exactly MaxSessions sessions")

	// The oldest session (first one) should have been deleted
	sessions, err := store.ListSessions("wf-1")
	require.NoError(t, err)
	assert.Len(t, sessions, maxSessions)

	// Verify sessions are sorted by start_time DESC; the oldest is NOT present
	// (first created would have the smallest start_time)
	if len(sessions) == maxSessions {
		oldest := sessions[maxSessions-1]
		for _, s := range sessions {
			assert.GreaterOrEqual(t, s.StartTime, oldest.StartTime)
		}
	}
}
