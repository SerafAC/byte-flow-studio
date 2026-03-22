package services

import (
	"fmt"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/session"
	"byteflow-studio/internal/workflow"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const warningThresholdBytes = 1 * 1024 * 1024 * 1024 // 1 GB

// StorageEstimate holds session storage stats.
type StorageEstimate struct {
	CurrentTotalBytes     int64 `json:"currentTotalBytes"`
	AverageSessionBytes   int64 `json:"averageSessionBytes"`
	ProjectedMaxBytes     int64 `json:"projectedMaxBytes"`
	WarningThresholdBytes int64 `json:"warningThresholdBytes"`
	ExceedsThreshold      bool  `json:"exceedsThreshold"`
}

// SessionService manages session lifecycle from the frontend's perspective.
type SessionService struct {
	sessionMgr  *session.Manager
	sessionStore *session.Store
	wfService   *WorkflowService
	pipelineSvc *PipelineService
	app         *application.App
}

// NewSessionService creates a SessionService.
func NewSessionService(
	sessionMgr *session.Manager,
	sessionStore *session.Store,
	wfService *WorkflowService,
) *SessionService {
	return &SessionService{
		sessionMgr:   sessionMgr,
		sessionStore: sessionStore,
		wfService:    wfService,
	}
}

// SetApp stores the Wails app reference for event emission.
func (s *SessionService) SetApp(app *application.App) {
	s.app = app
}

// SetPipelineService allows cross-service state checks.
func (s *SessionService) SetPipelineService(ps *PipelineService) {
	s.pipelineSvc = ps
}

// ListSessions returns session metadata for the current workflow.
func (s *SessionService) ListSessions() ([]session.SessionMeta, error) {
	wf, err := s.wfService.GetWorkflow()
	if err != nil {
		return nil, err
	}
	return s.sessionStore.ListSessions(wf.ID)
}

// GetActiveSessionID returns the ID of the currently active session.
func (s *SessionService) GetActiveSessionID() string {
	return s.sessionMgr.ActiveSessionID()
}

// SetViewSession switches Analysis blocks to display data from a historical session.
func (s *SessionService) SetViewSession(sessionID string) error {
	if s.pipelineSvc != nil {
		state := s.pipelineSvc.GetState()
		if state == pipeline.FlowStateRunning {
			return fmt.Errorf("cannot switch session view while flow is Running")
		}
	}

	wf, err := s.wfService.GetWorkflow()
	if err != nil {
		return err
	}
	sessions, err := s.sessionStore.ListSessions(wf.ID)
	if err != nil {
		return err
	}
	found := false
	for _, m := range sessions {
		if m.ID == sessionID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Emit session:view-changed first
	if s.app != nil {
		s.app.Event.Emit("session:view-changed", map[string]any{
			"sessionId": sessionID,
			"mode":      "historical",
		})
	}

	// Replay processed data as pipeline:data events
	dataByBlock, err := s.sessionStore.GetAllProcessedDataByBlock(sessionID)
	if err != nil {
		return err
	}
	if s.app != nil {
		for blockID, records := range dataByBlock {
			points := make([]map[string]any, 0, len(records))
			for _, rec := range records {
				points = append(points, map[string]any{
					"timestamp": rec.Timestamp,
					"values":    rec.Values,
					"replay":    true,
				})
			}
			s.app.Event.Emit("pipeline:data", map[string]any{
				"blockId": blockID,
				"points":  points,
			})
		}
	}
	return nil
}

// GetSessionConfig returns the current session recording configuration.
func (s *SessionService) GetSessionConfig() session.SessionConfig {
	wf := s.wfService.GetCurrentWorkflow()
	if wf == nil {
		return session.SessionConfig{StoreRaw: true, StoreProcessed: true, MaxSessions: 10}
	}
	return session.SessionConfig{
		StoreRaw:       wf.SessionConfig.StoreRaw,
		StoreProcessed: wf.SessionConfig.StoreProcessed,
		MaxSessions:    wf.SessionConfig.MaxSessions,
	}
}

// UpdateSessionConfig updates the session recording configuration.
func (s *SessionService) UpdateSessionConfig(cfg session.SessionConfig) error {
	if cfg.MaxSessions < 1 {
		return fmt.Errorf("maxSessions must be at least 1")
	}
	if !cfg.StoreRaw && !cfg.StoreProcessed {
		return fmt.Errorf("at least one storage layer (raw or processed) must be enabled")
	}

	wf := s.wfService.GetCurrentWorkflow()
	if wf == nil {
		return fmt.Errorf("no workflow loaded")
	}
	wf.SessionConfig = workflow.SessionConfig{
		StoreRaw:       cfg.StoreRaw,
		StoreProcessed: cfg.StoreProcessed,
		MaxSessions:    cfg.MaxSessions,
	}
	return nil
}

// DeleteSession permanently deletes a session.
func (s *SessionService) DeleteSession(sessionID string) error {
	if s.sessionMgr.ActiveSessionID() == sessionID {
		return fmt.Errorf("cannot delete the currently active session")
	}
	wf, err := s.wfService.GetWorkflow()
	if err != nil {
		return err
	}
	sessions, err := s.sessionStore.ListSessions(wf.ID)
	if err != nil {
		return err
	}
	found := false
	for _, m := range sessions {
		if m.ID == sessionID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if err := s.sessionStore.DeleteSession(sessionID); err != nil {
		return err
	}
	if s.app != nil {
		sessions, _ = s.sessionStore.ListSessions(wf.ID)
		s.app.Event.Emit("session:list-updated", map[string]any{"sessions": sessions})
	}
	return nil
}

// GetStorageEstimate returns current and projected session storage.
func (s *SessionService) GetStorageEstimate() StorageEstimate {
	wf := s.wfService.GetCurrentWorkflow()
	if wf == nil {
		return StorageEstimate{WarningThresholdBytes: warningThresholdBytes}
	}
	total, _ := s.sessionStore.SumDataVolume(wf.ID)
	sessions, _ := s.sessionStore.ListSessions(wf.ID)
	var avg int64
	if len(sessions) > 0 {
		avg = total / int64(len(sessions))
	}
	projected := avg * int64(wf.SessionConfig.MaxSessions)
	return StorageEstimate{
		CurrentTotalBytes:     total,
		AverageSessionBytes:   avg,
		ProjectedMaxBytes:     projected,
		WarningThresholdBytes: warningThresholdBytes,
		ExceedsThreshold:      projected > warningThresholdBytes,
	}
}
