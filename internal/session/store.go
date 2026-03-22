// Package session provides session persistence using SQLite.
package session

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
)

// Store manages session data in a shared SQLite database.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing *sql.DB (shared with workflow store).
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// InsertSession inserts a new session record.
func (s *Store) InsertSession(sess *Session) error {
	var endTime *int64
	if sess.EndTime != 0 {
		endTime = &sess.EndTime
	}
	_, err := s.db.Exec(`
		INSERT INTO sessions (id, workflow_id, start_time, end_time, end_reason,
			raw_layer_enabled, processed_layer_enabled, data_volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.WorkflowID, sess.StartTime, endTime, sess.EndReason,
		boolToInt(sess.RawLayerEnabled), boolToInt(sess.ProcessedLayerEnabled), sess.DataVolume,
	)
	return err
}

// UpdateSession updates an existing session record.
func (s *Store) UpdateSession(sess *Session) error {
	var endTime *int64
	if sess.EndTime != 0 {
		endTime = &sess.EndTime
	}
	_, err := s.db.Exec(`
		UPDATE sessions SET end_time=?, end_reason=?, data_volume=?
		WHERE id=?`,
		endTime, sess.EndReason, sess.DataVolume, sess.ID,
	)
	return err
}

// DeleteSession removes a session and cascades to raw_data and processed_data.
func (s *Store) DeleteSession(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, tbl := range []string{"raw_data", "processed_data"} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE session_id=?", tbl), id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("DELETE FROM sessions WHERE id=?", id); err != nil {
		return err
	}
	return tx.Commit()
}

// InsertRawData inserts a raw data record.
func (s *Store) InsertRawData(rec *RawDataRecord) error {
	_, err := s.db.Exec(
		`INSERT INTO raw_data (session_id, block_id, timestamp, data) VALUES (?, ?, ?, ?)`,
		rec.SessionID, rec.BlockID, rec.Timestamp, rec.Data,
	)
	return err
}

// InsertProcessedData inserts a processed data record.
func (s *Store) InsertProcessedData(rec *ProcessedDataRecord) error {
	blob := encodeFloat64s(rec.Values)
	_, err := s.db.Exec(
		`INSERT INTO processed_data (session_id, block_id, timestamp, data_values) VALUES (?, ?, ?, ?)`,
		rec.SessionID, rec.BlockID, rec.Timestamp, blob,
	)
	return err
}

// ListSessions returns session metadata for a workflow, ordered newest first.
func (s *Store) ListSessions(workflowID string) ([]SessionMeta, error) {
	rows, err := s.db.Query(`
		SELECT id, start_time, end_time, end_reason, raw_layer_enabled, processed_layer_enabled, data_volume
		FROM sessions WHERE workflow_id=? ORDER BY start_time DESC`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SessionMeta
	for rows.Next() {
		var m SessionMeta
		var endTime sql.NullInt64
		var endReason sql.NullString
		var rawEnabled, procEnabled int
		if err := rows.Scan(&m.ID, &m.StartTime, &endTime, &endReason, &rawEnabled, &procEnabled, &m.DataVolume); err != nil {
			return nil, err
		}
		if endTime.Valid {
			m.EndTime = endTime.Int64
		}
		if endReason.Valid {
			m.EndReason = endReason.String
		}
		m.RawLayerEnabled = rawEnabled != 0
		m.ProcessedLayerEnabled = procEnabled != 0
		result = append(result, m)
	}
	return result, rows.Err()
}

// CountSessions returns the number of sessions for a workflow.
func (s *Store) CountSessions(workflowID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE workflow_id=?`, workflowID).Scan(&count)
	return count, err
}

// GetOldestSession returns the session with the oldest start_time for a workflow.
func (s *Store) GetOldestSession(workflowID string) (*Session, error) {
	var sess Session
	var endTime sql.NullInt64
	var endReason sql.NullString
	var rawEnabled, procEnabled int
	err := s.db.QueryRow(`
		SELECT id, workflow_id, start_time, end_time, end_reason,
			raw_layer_enabled, processed_layer_enabled, data_volume
		FROM sessions WHERE workflow_id=? ORDER BY start_time ASC LIMIT 1`, workflowID,
	).Scan(&sess.ID, &sess.WorkflowID, &sess.StartTime, &endTime, &endReason,
		&rawEnabled, &procEnabled, &sess.DataVolume)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if endTime.Valid {
		sess.EndTime = endTime.Int64
	}
	if endReason.Valid {
		sess.EndReason = endReason.String
	}
	sess.RawLayerEnabled = rawEnabled != 0
	sess.ProcessedLayerEnabled = procEnabled != 0
	return &sess, nil
}

// GetProcessedData returns all processed data records for a session+block.
func (s *Store) GetProcessedData(sessionID, blockID string) ([]ProcessedDataRecord, error) {
	rows, err := s.db.Query(`
		SELECT session_id, block_id, timestamp, data_values
		FROM processed_data WHERE session_id=? AND block_id=?
		ORDER BY timestamp ASC`, sessionID, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProcessedDataRecord
	for rows.Next() {
		var rec ProcessedDataRecord
		var blob []byte
		if err := rows.Scan(&rec.SessionID, &rec.BlockID, &rec.Timestamp, &blob); err != nil {
			return nil, err
		}
		rec.Values = decodeFloat64s(blob)
		result = append(result, rec)
	}
	return result, rows.Err()
}

// GetAllProcessedDataByBlock returns a map of blockID → []ProcessedDataRecord for a session.
func (s *Store) GetAllProcessedDataByBlock(sessionID string) (map[string][]ProcessedDataRecord, error) {
	rows, err := s.db.Query(`
		SELECT session_id, block_id, timestamp, data_values
		FROM processed_data WHERE session_id=?
		ORDER BY block_id, timestamp ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]ProcessedDataRecord{}
	for rows.Next() {
		var rec ProcessedDataRecord
		var blob []byte
		if err := rows.Scan(&rec.SessionID, &rec.BlockID, &rec.Timestamp, &blob); err != nil {
			return nil, err
		}
		rec.Values = decodeFloat64s(blob)
		result[rec.BlockID] = append(result[rec.BlockID], rec)
	}
	return result, rows.Err()
}

// SumDataVolume returns the total data_volume for a workflow.
func (s *Store) SumDataVolume(workflowID string) (int64, error) {
	var total sql.NullInt64
	err := s.db.QueryRow(`SELECT SUM(data_volume) FROM sessions WHERE workflow_id=?`, workflowID).Scan(&total)
	if !total.Valid {
		return 0, err
	}
	return total.Int64, err
}

// encodeFloat64s encodes a slice of float64 as little-endian IEEE 754 bytes.
func encodeFloat64s(vals []float64) []byte {
	buf := make([]byte, len(vals)*8)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

// decodeFloat64s decodes little-endian IEEE 754 bytes into float64 slice.
func decodeFloat64s(buf []byte) []float64 {
	n := len(buf) / 8
	vals := make([]float64, n)
	for i := range vals {
		bits := binary.LittleEndian.Uint64(buf[i*8:])
		vals[i] = math.Float64frombits(bits)
	}
	return vals
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
