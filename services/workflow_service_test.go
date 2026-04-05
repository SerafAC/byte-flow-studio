package services

import (
	"testing"

	_ "byteflow-studio/internal/analysis" // registers analysis block types via init()
	"byteflow-studio/internal/workflow"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWorkflowService(t *testing.T) *WorkflowService {
	t.Helper()
	store, err := workflow.OpenStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return NewWorkflowService(store, nil)
}

func TestUpdateBlockSize_UpdatesDimensions(t *testing.T) {
	svc := newTestWorkflowService(t)

	def, err := svc.AddBlock("line-chart", 100, 200)
	require.NoError(t, err)

	err = svc.UpdateBlockSize(def.ID, 320, 200)
	require.NoError(t, err)

	b := svc.findBlock(def.ID)
	require.NotNil(t, b)
	assert.Equal(t, float64(320), b.Width)
	assert.Equal(t, float64(200), b.Height)
}

func TestUpdateBlockSize_UnknownBlockReturnsError(t *testing.T) {
	svc := newTestWorkflowService(t)

	err := svc.UpdateBlockSize("nonexistent-id", 320, 200)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-id")
}

func TestUpdateBlockSize_InvalidDimensionsReturnsError(t *testing.T) {
	svc := newTestWorkflowService(t)

	def, err := svc.AddBlock("line-chart", 100, 200)
	require.NoError(t, err)

	err = svc.UpdateBlockSize(def.ID, 0, 200)
	require.Error(t, err)

	err = svc.UpdateBlockSize(def.ID, 320, -1)
	require.Error(t, err)
}
