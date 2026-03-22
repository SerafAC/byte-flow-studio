package processing

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMovingAverageWindowSizes(t *testing.T) {
	for _, windowSize := range []int{1, 5, 10} {
		factory, ok := Registry["moving-average"]
		require.True(t, ok)

		block := factory("ma-test")
		err := block.Configure(map[string]any{"windowSize": windowSize})
		require.NoError(t, err)

		in := make(chan pipeline.DataChunk, 20)
		out := make(chan pipeline.DataChunk, 20)
		errCh := make(chan pipeline.BlockError, 1)
		ctx, cancel := context.WithCancel(context.Background())

		go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

		// Feed 10 known values
		for i := 1; i <= 10; i++ {
			in <- pipeline.DataChunk{Timestamp: int64(i), SourceID: "s1", Values: []float64{float64(i)}}
		}

		// Collect outputs
		var results []float64
		timeout := time.After(500 * time.Millisecond)
		for len(results) < 10 {
			select {
			case c := <-out:
				results = append(results, c.Values[0])
			case <-timeout:
				t.Fatalf("timeout at window=%d, got %d results", windowSize, len(results))
			}
		}

		cancel()

		if windowSize == 1 {
			// Moving average of window 1 = identity
			for i, v := range results {
				assert.InDelta(t, float64(i+1), v, 0.001)
			}
		}
		// For windowSize > 1, just verify values are within plausible range
		for _, v := range results {
			assert.GreaterOrEqual(t, v, 1.0)
			assert.LessOrEqual(t, v, 10.0)
		}
	}
}

func TestMovingAveragePreservesTimestamp(t *testing.T) {
	factory := Registry["moving-average"]
	block := factory("ma-ts")
	_ = block.Configure(map[string]any{"windowSize": 3})

	in := make(chan pipeline.DataChunk, 5)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	in <- pipeline.DataChunk{Timestamp: 42, SourceID: "src1", Values: []float64{5.0}}

	select {
	case got := <-out:
		assert.Equal(t, int64(42), got.Timestamp)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for moving average output")
	}
}

func TestMovingAverageHotReload(t *testing.T) {
	factory := Registry["moving-average"]
	block := factory("ma-reload")
	_ = block.Configure(map[string]any{"windowSize": 5})

	// Hot-reload to windowSize=1 while not running
	err := block.Configure(map[string]any{"windowSize": 1})
	require.NoError(t, err)

	in := make(chan pipeline.DataChunk, 5)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	in <- pipeline.DataChunk{Timestamp: 1, Values: []float64{7.0}}
	select {
	case got := <-out:
		assert.InDelta(t, 7.0, got.Values[0], 0.001, "window=1 should produce the input value")
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout")
	}
}
