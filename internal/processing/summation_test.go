package processing

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummationCumulativeSum(t *testing.T) {
	factory, ok := Registry["summation"]
	require.True(t, ok)

	block := factory("sum-1")
	_ = block.Configure(map[string]any{})

	in := make(chan pipeline.DataChunk, 10)
	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	inputs := []float64{1, 2, 3, 4, 5}
	expected := []float64{1, 3, 6, 10, 15}

	for _, v := range inputs {
		in <- pipeline.DataChunk{Timestamp: 1, Values: []float64{v}}
	}

	for i, exp := range expected {
		select {
		case got := <-out:
			require.Len(t, got.Values, 1)
			assert.InDelta(t, exp, got.Values[0], 0.001, "cumulative sum at step %d", i)
		case <-time.After(300 * time.Millisecond):
			t.Fatalf("timeout at step %d", i)
		}
	}
}
