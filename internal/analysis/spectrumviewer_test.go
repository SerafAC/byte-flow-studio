package analysis

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpectrumViewerBufferAndSnapshot(t *testing.T) {
	factory, ok := processing.Registry["spectrum-viewer"]
	require.True(t, ok, "spectrum-viewer should be registered")

	block := factory("sv-1")
	assert.Equal(t, "spectrum-viewer", block.Type())
	assert.Equal(t, pipeline.CategoryAnalysis, block.Category())
	require.Len(t, block.InputPorts(), 1)
	assert.Equal(t, pipeline.DataTypeNum, block.InputPorts()[0].DataType)
	assert.Empty(t, block.OutputPorts())

	ab, ok := block.(pipeline.AnalysisBlock)
	require.True(t, ok, "should implement AnalysisBlock")

	cfg := ab.BufferConfig()
	assert.Equal(t, pipeline.BufferModeSamples, cfg.Mode)

	err := ab.SetBufferConfig(pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 5})
	require.NoError(t, err)

	in := make(chan pipeline.DataChunk, 20)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{}, errCh)

	// Send 8 chunks — buffer should cap at 5
	for i := 0; i < 8; i++ {
		in <- pipeline.DataChunk{Timestamp: int64(i), SourceID: "fft", Values: []float64{float64(i), float64(i + 1)}}
	}
	time.Sleep(50 * time.Millisecond)

	snap := ab.Snapshot()
	assert.LessOrEqual(t, len(snap), 5, "buffer should be capped at 5")
	assert.Greater(t, len(snap), 0)
}

func TestSpectrumViewerConfigure(t *testing.T) {
	factory := processing.Registry["spectrum-viewer"]
	block := factory("sv-cfg")

	err := block.Configure(map[string]any{
		"colorMap":     "magma",
		"minFreqHz":    10.0,
		"maxFreqHz":    500.0,
		"minAmplitude": 0.0,
		"maxAmplitude": 100.0,
	})
	require.NoError(t, err)

	svBlock := block.(*spectrumViewerBlock)
	assert.Equal(t, "magma", svBlock.colorMap)
	assert.InDelta(t, 10.0, svBlock.minFreqHz, 1e-10)
	assert.InDelta(t, 500.0, svBlock.maxFreqHz, 1e-10)

	// Invalid colorMap
	err = block.Configure(map[string]any{"colorMap": "invalid"})
	assert.Error(t, err)
}
