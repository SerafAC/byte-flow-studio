package analysis

import (
	"testing"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValueDisplayRingBufferRespectMaxSamples(t *testing.T) {
	factory, ok := analysisRegistry["value-display"]
	require.True(t, ok, "value-display should be registered")

	block := factory("vd-1").(pipeline.AnalysisBlock)
	err := block.SetBufferConfig(pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 3})
	require.NoError(t, err)

	// Feed 5 chunks — only last 3 should be retained
	chunks := []pipeline.DataChunk{
		{Timestamp: 1, Values: []float64{1.0}},
		{Timestamp: 2, Values: []float64{2.0}},
		{Timestamp: 3, Values: []float64{3.0}},
		{Timestamp: 4, Values: []float64{4.0}},
		{Timestamp: 5, Values: []float64{5.0}},
	}
	vd := block.(*valueDisplayBlock)
	for _, c := range chunks {
		vd.addToBuffer(c)
	}

	snap := block.Snapshot()
	require.Len(t, snap, 3)
	// Oldest evicted — remaining are chunks 3, 4, 5
	assert.Equal(t, int64(3), snap[0].Timestamp)
	assert.Equal(t, int64(5), snap[2].Timestamp)
}

func TestValueDisplayOldestEvicted(t *testing.T) {
	factory := analysisRegistry["value-display"]
	block := factory("vd-2").(pipeline.AnalysisBlock)
	_ = block.SetBufferConfig(pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 2})

	vd := block.(*valueDisplayBlock)
	vd.addToBuffer(pipeline.DataChunk{Timestamp: 10, Values: []float64{10}})
	vd.addToBuffer(pipeline.DataChunk{Timestamp: 20, Values: []float64{20}})
	// This should evict timestamp=10
	vd.addToBuffer(pipeline.DataChunk{Timestamp: 30, Values: []float64{30}})

	snap := block.Snapshot()
	require.Len(t, snap, 2)
	assert.Equal(t, int64(20), snap[0].Timestamp)
	assert.Equal(t, int64(30), snap[1].Timestamp)
}

func TestValueDisplaySnapshotOrdered(t *testing.T) {
	factory := analysisRegistry["value-display"]
	block := factory("vd-3").(pipeline.AnalysisBlock)
	_ = block.SetBufferConfig(pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 10})

	vd := block.(*valueDisplayBlock)
	for i := int64(0); i < 5; i++ {
		vd.addToBuffer(pipeline.DataChunk{Timestamp: i, Values: []float64{float64(i)}})
	}

	snap := block.Snapshot()
	for i := 1; i < len(snap); i++ {
		assert.Less(t, snap[i-1].Timestamp, snap[i].Timestamp, "snapshot should be in order")
	}
}

func TestValueDisplayRawHexRendering(t *testing.T) {
	factory := analysisRegistry["value-display"]
	block := factory("vd-4").(*valueDisplayBlock)

	chunk := pipeline.DataChunk{Timestamp: 1, Raw: []byte{0xDE, 0xAD}}
	rendered := block.renderChunk(chunk)
	assert.Equal(t, "DE AD", rendered)
}

func TestValueDisplayNumericRendering(t *testing.T) {
	factory := analysisRegistry["value-display"]
	block := factory("vd-5").(*valueDisplayBlock)

	chunk := pipeline.DataChunk{Timestamp: 1, Values: []float64{3.14}}
	rendered := block.renderChunk(chunk)
	assert.Contains(t, rendered, "3.14")
}
