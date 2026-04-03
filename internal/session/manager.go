package session

import (
	"fmt"
	"sync"
	"time"

	"byteflow-studio/internal/logging"

	"github.com/google/uuid"
)

// Manager manages active session lifecycle and enforces retention policy.
type Manager struct {
	mu          sync.Mutex
	store       *Store
	activeID    string
	workflowID  string
	cfg         SessionConfig
	log         *logging.Logger
}

// NewManager creates a new Manager with the given store and configuration.
func NewManager(store *Store, workflowID string, cfg SessionConfig, log *logging.Logger) *Manager {
	return &Manager{
		store:      store,
		workflowID: workflowID,
		cfg:        cfg,
		log:        log,
	}
}

// CreateSession creates and persists a new active session.
func (m *Manager) CreateSession(workflowID string, cfg SessionConfig) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess := &Session{
		ID:                    uuid.New().String(),
		WorkflowID:            workflowID,
		StartTime:             time.Now().UnixMilli(),
		EndReason:             "active",
		RawLayerEnabled:       cfg.StoreRaw,
		ProcessedLayerEnabled: cfg.StoreProcessed,
	}
	if err := m.store.InsertSession(sess); err != nil {
		m.log.Error("failed to insert session", "session_id", sess.ID, "error", err)
		return nil, fmt.Errorf("insert session: %w", err)
	}
	m.activeID = sess.ID
	m.workflowID = workflowID
	m.cfg = cfg
	m.log.Info("session created", "session_id", sess.ID, "workflow_id", workflowID)
	return sess, nil
}

// ActiveSessionID returns the ID of the currently active session.
func (m *Manager) ActiveSessionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeID
}

// CompleteSession marks the active session as complete and enforces retention.
func (m *Manager) CompleteSession(id string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess := &Session{
		ID:        id,
		EndTime:   time.Now().UnixMilli(),
		EndReason: reason,
	}
	if err := m.store.UpdateSession(sess); err != nil {
		m.log.Error("failed to update session", "session_id", id, "error", err)
		return fmt.Errorf("update session: %w", err)
	}
	if m.activeID == id {
		m.activeID = ""
	}
	m.log.Info("session completed", "session_id", id, "reason", reason)

	// Enforce retention: delete oldest if over limit
	return m.trimSessions()
}

// trimSessions deletes the oldest session(s) if count > MaxSessions.
func (m *Manager) trimSessions() error {
	count, err := m.store.CountSessions(m.workflowID)
	if err != nil {
		m.log.Error("failed to count sessions", "workflow_id", m.workflowID, "error", err)
		return err
	}
	for count > m.cfg.MaxSessions {
		oldest, err := m.store.GetOldestSession(m.workflowID)
		if err != nil || oldest == nil {
			if err != nil {
				m.log.Error("failed to get oldest session", "workflow_id", m.workflowID, "error", err)
			}
			return err
		}
		m.log.Debug("trimming oldest session", "session_id", oldest.ID, "workflow_id", m.workflowID, "count", count, "max", m.cfg.MaxSessions)
		if err := m.store.DeleteSession(oldest.ID); err != nil {
			m.log.Error("failed to delete oldest session", "session_id", oldest.ID, "error", err)
			return err
		}
		count--
	}
	return nil
}

// AppendRaw records a raw data chunk for the given session.
func (m *Manager) AppendRaw(sessionID string, rec RawDataRecord) error {
	return m.store.InsertRawData(&rec)
}

// AppendProcessed records a processed data record for the given session.
func (m *Manager) AppendProcessed(sessionID string, rec ProcessedDataRecord) error {
	return m.store.InsertProcessedData(&rec)
}
