package services

import (
	"strings"
	"testing"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/session"
	"byteflow-studio/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPipelineService(t *testing.T, wf workflow.Workflow) *PipelineService {
	t.Helper()

	store, err := workflow.OpenStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	wfSvc := NewWorkflowService(store)
	wfSvc.SetCurrentWorkflow(&wf)

	sessionStore := session.NewStore(store.DB())
	sessionMgr := session.NewManager(sessionStore, wf.ID, session.SessionConfig{
		StoreRaw: false, StoreProcessed: false, MaxSessions: 5,
	})

	eng := pipeline.NewEngine()
	return NewPipelineService(eng, sessionMgr, wfSvc)
}

// TestPipelineServiceFR004NoInputNoAnalysis verifies that Start() returns
// "at least one Input and one Analysis block must be connected" when the workflow
// contains only Processing blocks (T107a — written before T107 implementation).
func TestPipelineServiceFR004NoInputNoAnalysis(t *testing.T) {
	wf := workflow.Workflow{
		ID:   "wf-fr004",
		Name: "No Input/Analysis",
		Blocks: []workflow.BlockDef{
			{ID: "proc1", Type: "moving-average", Category: "processing", Params: map[string]any{"windowSize": float64(4)}},
		},
		Connections:   []workflow.ConnectionDef{},
		SessionConfig: workflow.DefaultSessionConfig(),
	}

	pipeSvc := newTestPipelineService(t, wf)

	err := pipeSvc.Start()
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "at least one Input and one Analysis block must be connected"),
		"expected FR-004 error message, got: %s", err.Error(),
	)
}

// TestPipelineServiceFR004ProcessingBlockNotOnPath verifies that Start() returns
// an error when a Processing block is not connected on any Input→Analysis path.
func TestPipelineServiceFR004ProcessingBlockNotOnPath(t *testing.T) {
	wf := workflow.Workflow{
		ID:   "wf-disconnected",
		Name: "Disconnected Processing",
		Blocks: []workflow.BlockDef{
			{ID: "in1", Type: "simulator", Category: "input", Params: map[string]any{
				"waveform": "sine", "frequency": float64(1), "amplitude": float64(1), "sampleRate": float64(100),
			}},
			{ID: "an1", Type: "line-chart", Category: "analysis", Params: map[string]any{}},
			// proc1 is not connected to in1 → an1 path
			{ID: "proc1", Type: "moving-average", Category: "processing", Params: map[string]any{"windowSize": float64(4)}},
		},
		Connections: []workflow.ConnectionDef{
			{ID: "c1", FromBlockID: "in1", FromPortID: "out", ToBlockID: "an1", ToPortID: "in"},
		},
		SessionConfig: workflow.DefaultSessionConfig(),
	}

	pipeSvc := newTestPipelineService(t, wf)

	err := pipeSvc.Start()
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "not reachable from any Input") ||
			strings.Contains(err.Error(), "at least one Input"),
		"expected connectivity error, got: %s", err.Error(),
	)
}
