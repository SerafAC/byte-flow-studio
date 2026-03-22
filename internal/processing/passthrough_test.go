package processing

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPassthroughBlock(t *testing.T) {
	factory, ok := Registry["passthrough"]
	require.True(t, ok, "passthrough should be registered")

	block := factory("pt-1")
	assert.Equal(t, "pt-1", block.ID())
	assert.Equal(t, "passthrough", block.Type())
	assert.Equal(t, pipeline.CategoryProcessing, block.Category())

	in := make(chan pipeline.DataChunk, 5)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)
	}()

	chunks := []pipeline.DataChunk{
		{Timestamp: 1, SourceID: "s1", Values: []float64{1.0}},
		{Timestamp: 2, SourceID: "s1", Values: []float64{2.0}},
		{Timestamp: 3, SourceID: "s1", Values: []float64{3.0}},
	}
	for _, c := range chunks {
		in <- c
	}

	for i, expected := range chunks {
		select {
		case got := <-out:
			assert.Equal(t, expected.Timestamp, got.Timestamp, "chunk %d timestamp mismatch", i)
			assert.Equal(t, expected.Values, got.Values, "chunk %d values mismatch", i)
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timed out waiting for chunk %d", i)
		}
	}

	cancel()
}
