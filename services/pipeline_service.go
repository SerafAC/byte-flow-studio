package services

import (
	"fmt"

	"byteflow-studio/internal/logging"
	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"
	"byteflow-studio/internal/session"
	"byteflow-studio/internal/workflow"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// PipelineService controls the pipeline runtime lifecycle.
type PipelineService struct {
	engine     *pipeline.Engine
	sessionMgr *session.Manager
	wfService  *WorkflowService
	sessionSvc *SessionService
	app        *application.App
	log        *logging.Logger
}

// NewPipelineService creates a PipelineService.
func NewPipelineService(engine *pipeline.Engine, sessionMgr *session.Manager, wfService *WorkflowService, log *logging.Logger) *PipelineService {
	return &PipelineService{
		engine:     engine,
		sessionMgr: sessionMgr,
		wfService:  wfService,
		log:        log,
	}
}

// SetApp stores the Wails app reference for event emission.
func (s *PipelineService) SetApp(app *application.App) {
	s.app = app
	s.engine.SetApp(app)
}

// SetSessionService wires the SessionService for storage estimate checks.
func (s *PipelineService) SetSessionService(ss *SessionService) {
	s.sessionSvc = ss
}

// GetState returns the current flow state.
func (s *PipelineService) GetState() pipeline.FlowState {
	return s.engine.GetState()
}

// Start validates the workflow DAG and starts the pipeline.
func (s *PipelineService) Start() error {
	wf, err := s.wfService.GetWorkflow()
	if err != nil {
		s.log.Error("failed to get workflow for pipeline start", logging.KeyError, err)
		return err
	}

	s.log.Debug("starting pipeline", "workflow_id", wf.ID, "blocks", len(wf.Blocks), "connections", len(wf.Connections))

	// Validate processing blocks are on a valid Input→Analysis path
	if err := s.validateConnectivity(wf); err != nil {
		s.log.Error("pipeline connectivity validation failed", logging.KeyError, err, "workflow_id", wf.ID)
		return err
	}

	// Build port type map from registered blocks
	portTypes := s.wfService.buildPortTypeMap()

	// Pre-validate and pre-construct blocks
	blocks, err := s.buildBlocks(wf)
	if err != nil {
		s.log.Error("failed to build blocks", logging.KeyError, err, "workflow_id", wf.ID)
		return err
	}

	sess, err := s.sessionMgr.CreateSession(wf.ID, sessionConfigFrom(wf.SessionConfig))
	if err != nil {
		s.log.Error("failed to create session for pipeline", logging.KeyError, err, "workflow_id", wf.ID)
		return fmt.Errorf("create session: %w", err)
	}

	// T096: auto-switch frontend back to live mode when starting a new run
	if s.app != nil {
		s.app.Event.Emit("session:view-changed", map[string]any{
			"sessionId": sess.ID,
			"mode":      "live",
		})
	}

	if err := s.engine.Start(wf, sess.ID, portTypes, blocks, s.sessionMgr); err != nil {
		s.log.Error("engine start failed", logging.KeyError, err, logging.KeySessionID, sess.ID)
		return err
	}
	return nil
}

// Pause suspends data flow.
func (s *PipelineService) Pause() error {
	return s.engine.Pause()
}

// Resume resumes data flow after a pause.
func (s *PipelineService) Resume() error {
	return s.engine.Resume()
}

// Stop stops the pipeline and completes the active session.
func (s *PipelineService) Stop() error {
	sessionID := s.sessionMgr.ActiveSessionID()
	s.log.Debug("stopping pipeline", logging.KeySessionID, sessionID)
	if err := s.engine.Stop(); err != nil {
		s.log.Error("engine stop failed", logging.KeyError, err, logging.KeySessionID, sessionID)
		return err
	}
	if sessionID != "" {
		if err := s.sessionMgr.CompleteSession(sessionID, "user-stop"); err != nil {
			s.log.Error("failed to complete session on stop", logging.KeyError, err, logging.KeySessionID, sessionID)
		}
	}
	// T086: emit storage warning if threshold exceeded
	if s.sessionSvc != nil && s.app != nil {
		est := s.sessionSvc.GetStorageEstimate()
		if est.ExceedsThreshold {
			s.log.Warn("storage threshold exceeded", "projected_bytes", est.ProjectedMaxBytes, "threshold_bytes", est.WarningThresholdBytes)
			s.app.Event.Emit("session:storage-warning", est)
		}
	}
	return nil
}

// GetBlockErrors returns a map of blockID → errorMessage.
func (s *PipelineService) GetBlockErrors() map[string]string {
	return s.engine.GetBlockErrors()
}

// buildBlocks constructs and configures block instances from the workflow definition.
func (s *PipelineService) buildBlocks(wf workflow.Workflow) (map[string]pipeline.Block, error) {
	blocks := make(map[string]pipeline.Block, len(wf.Blocks))
	for _, b := range wf.Blocks {
		factory, ok := processing.Registry[b.Type]
		if !ok {
			s.log.Error("unknown block type in workflow", logging.KeyBlockID, b.ID, "type", b.Type)
			return nil, fmt.Errorf("block %s (%s): unknown type", b.ID, b.Type)
		}
		block := factory(b.ID)
		if err := block.Configure(b.Params); err != nil {
			s.log.Error("block configuration failed", logging.KeyBlockID, b.ID, "type", b.Type, logging.KeyError, err)
			return nil, fmt.Errorf("block %s (%s): invalid configuration: %s", b.ID, b.Type, err.Error())
		}
		blocks[b.ID] = block
	}
	return blocks, nil
}

// validateConnectivity checks that the workflow has at least one Input and one Analysis block,
// and that every Processing block lies on a valid Input→Analysis path.
func (s *PipelineService) validateConnectivity(wf workflow.Workflow) error {
	var hasInput, hasAnalysis bool
	for _, b := range wf.Blocks {
		if b.Category == "input" {
			hasInput = true
		}
		if b.Category == "analysis" {
			hasAnalysis = true
		}
	}
	if !hasInput || !hasAnalysis {
		return fmt.Errorf("at least one Input and one Analysis block must be connected")
	}

	// Build adjacency: forward (from→to) and backward (to→from)
	forward := make(map[string][]string)  // blockID → downstream block IDs
	backward := make(map[string][]string) // blockID → upstream block IDs
	for _, c := range wf.Connections {
		forward[c.FromBlockID] = append(forward[c.FromBlockID], c.ToBlockID)
		backward[c.ToBlockID] = append(backward[c.ToBlockID], c.FromBlockID)
	}

	// Categorize blocks
	categories := make(map[string]string)
	for _, b := range wf.Blocks {
		categories[b.ID] = b.Category
	}

	// BFS from all Input blocks forward
	reachableFromInput := make(map[string]bool)
	queue := []string{}
	for _, b := range wf.Blocks {
		if b.Category == "input" {
			queue = append(queue, b.ID)
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if reachableFromInput[cur] {
			continue
		}
		reachableFromInput[cur] = true
		queue = append(queue, forward[cur]...)
	}

	// BFS from all Analysis blocks backward
	canReachAnalysis := make(map[string]bool)
	for _, b := range wf.Blocks {
		if b.Category == "analysis" {
			queue = append(queue, b.ID)
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if canReachAnalysis[cur] {
			continue
		}
		canReachAnalysis[cur] = true
		queue = append(queue, backward[cur]...)
	}

	// Every Processing block must be in both sets
	for _, b := range wf.Blocks {
		if b.Category != "processing" {
			continue
		}
		if !reachableFromInput[b.ID] || !canReachAnalysis[b.ID] {
			if s.app != nil {
				s.app.Event.Emit("pipeline:block-status", map[string]any{
					"blockId": b.ID,
					"status":  "error",
					"message": fmt.Sprintf("block %s (%s): not reachable from any Input→Analysis path", b.ID, b.Type),
				})
			}
			return fmt.Errorf("block %s (%s): not reachable from any Input→Analysis path", b.ID, b.Type)
		}
	}
	return nil
}

// sessionConfigFrom converts a workflow.SessionConfig to a session.SessionConfig.
func sessionConfigFrom(wfCfg workflow.SessionConfig) session.SessionConfig {
	return session.SessionConfig{
		StoreRaw:       wfCfg.StoreRaw,
		StoreProcessed: wfCfg.StoreProcessed,
		MaxSessions:    wfCfg.MaxSessions,
	}
}
