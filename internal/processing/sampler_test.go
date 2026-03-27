package processing

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runSampler(t *testing.T, params map[string]any) (chan pipeline.DataChunk, chan pipeline.DataChunk, context.CancelFunc) {
	t.Helper()
	factory, ok := Registry["sampler"]
	require.True(t, ok)
	block := factory("sampler-test")
	require.NoError(t, block.Configure(params))

	in := make(chan pipeline.DataChunk, 32)
	out := make(chan pipeline.DataChunk, 32)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)
	return in, out, cancel
}

func TestSamplerEveryNSamples(t *testing.T) {
	in, out, cancel := runSampler(t, map[string]any{"mode": "every-n-samples", "n": 3})
	defer cancel()

	for i := 1; i <= 9; i++ {
		in <- pipeline.DataChunk{Timestamp: int64(i), SourceID: "s", Values: []float64{float64(i)}}
	}

	var results []float64
	timeout := time.After(500 * time.Millisecond)
	for len(results) < 3 {
		select {
		case c := <-out:
			results = append(results, c.Values[0])
		case <-timeout:
			t.Fatalf("timeout: got %d results, want 3", len(results))
		}
	}

	// With n=3: samples 3, 6, 9 should pass
	assert.Equal(t, []float64{3, 6, 9}, results)

	// No more output should arrive
	select {
	case extra := <-out:
		t.Fatalf("unexpected extra output: %v", extra)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSamplerFirstMode(t *testing.T) {
	in, out, cancel := runSampler(t, map[string]any{"mode": "first"})
	defer cancel()

	for i := 1; i <= 5; i++ {
		in <- pipeline.DataChunk{Timestamp: int64(i), SourceID: "s", Values: []float64{float64(i)}}
	}

	select {
	case c := <-out:
		assert.Equal(t, 1.0, c.Values[0], "only the first value should pass")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for first value")
	}

	select {
	case extra := <-out:
		t.Fatalf("unexpected extra output: %v", extra)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSamplerFirstInWindow(t *testing.T) {
	in, out, cancel := runSampler(t, map[string]any{"mode": "first-in-window", "intervalMs": 100.0})
	defer cancel()

	// ts=0: should pass (first)
	// ts=50: should be blocked (within 100ms window)
	// ts=150: should pass (past the 100ms window)
	timestamps := []int64{0, 50, 150}
	for _, ts := range timestamps {
		in <- pipeline.DataChunk{Timestamp: ts, SourceID: "s", Values: []float64{float64(ts)}}
	}

	var results []float64
	timeout := time.After(500 * time.Millisecond)
	for len(results) < 2 {
		select {
		case c := <-out:
			results = append(results, c.Values[0])
		case <-timeout:
			t.Fatalf("timeout: got %d results, want 2", len(results))
		}
	}

	assert.Equal(t, []float64{0, 150}, results)

	select {
	case extra := <-out:
		t.Fatalf("unexpected extra output: %v", extra)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSamplerLastInWindow(t *testing.T) {
	in, out, cancel := runSampler(t, map[string]any{"mode": "last-in-window", "intervalMs": 50.0})
	defer cancel()

	// Send 3 values quickly; only the last should be emitted when the ticker fires
	for i := 1; i <= 3; i++ {
		in <- pipeline.DataChunk{Timestamp: int64(i), SourceID: "s", Values: []float64{float64(i)}}
	}

	select {
	case c := <-out:
		assert.Equal(t, 3.0, c.Values[0], "last value in window should be emitted")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for last-in-window output")
	}
}

func TestSamplerPreservesTimestamp(t *testing.T) {
	in, out, cancel := runSampler(t, map[string]any{"mode": "every-n-samples", "n": 1})
	defer cancel()

	in <- pipeline.DataChunk{Timestamp: 42, SourceID: "src", Values: []float64{7.0}}

	select {
	case c := <-out:
		assert.Equal(t, int64(42), c.Timestamp)
		assert.Equal(t, "src", c.SourceID)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout")
	}
}
