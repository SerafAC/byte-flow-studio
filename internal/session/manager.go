package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager manages active session lifecycle and enforces retention policy.
type Manager struct {
	mu          sync.Mutex
	store       *Store
	activeID    string
	workflowID  string
	cfg         SessionConfig
}

// NewManager creates a new Manager with the given store and configuration.
func NewManager(store *Store, workflowID string, cfg SessionConfig) *Manager {
	return &Manager{
		store:      store,
		workflowID: workflowID,
		cfg:        cfg,
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
		return nil, fmt.Errorf("insert session: %w", err)
	}
	m.activeID = sess.ID
	m.workflowID = workflowID
	m.cfg = cfg
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
		return fmt.Errorf("update session: %w", err)
	}
	if m.activeID == id {
		m.activeID = ""
	}

	// Enforce retention: delete oldest if over limit
	return m.trimSessions()
}

// trimSessions deletes the oldest session(s) if count > MaxSessions.
func (m *Manager) trimSessions() error {
	count, err := m.store.CountSessions(m.workflowID)
	if err != nil {
		return err
	}
	for count > m.cfg.MaxSessions {
		oldest, err := m.store.GetOldestSession(m.workflowID)
		if err != nil || oldest == nil {
			return err
		}
		if err := m.store.DeleteSession(oldest.ID); err != nil {
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
