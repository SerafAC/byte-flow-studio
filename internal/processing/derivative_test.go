package processing

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDerivativeConstantInput(t *testing.T) {
	factory, ok := Registry["derivative"]
	require.True(t, ok, "derivative should be registered")

	block := factory("deriv-const")
	assert.Equal(t, "derivative", block.Type())
	assert.Equal(t, pipeline.CategoryProcessing, block.Category())

	in := make(chan pipeline.DataChunk, 20)
	out := make(chan pipeline.DataChunk, 20)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	// Constant input at 1ms intervals: derivative should be 0 after first sample
	for i := 0; i < 5; i++ {
		in <- pipeline.DataChunk{Timestamp: int64(i * 1), SourceID: "sim", Values: []float64{5.0}}
	}

	var results []float64
	timeout := time.After(500 * time.Millisecond)
	for len(results) < 5 {
		select {
		case c := <-out:
			results = append(results, c.Values[0])
		case <-timeout:
			t.Fatalf("timeout after %d results", len(results))
		}
	}

	// First sample should be 0
	assert.InDelta(t, 0.0, results[0], 1e-10, "first sample derivative should be 0")
	// Subsequent samples: constant input → 0 derivative
	for i := 1; i < len(results); i++ {
		assert.InDelta(t, 0.0, results[i], 1e-10, "constant input derivative should be 0")
	}
}

func TestDerivativeLinearRamp(t *testing.T) {
	factory := Registry["derivative"]
	block := factory("deriv-ramp")

	in := make(chan pipeline.DataChunk, 20)
	out := make(chan pipeline.DataChunk, 20)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	// Linear ramp: value = timestamp, at 1ms intervals
	// Derivative = (v[n]-v[n-1])/(t[n]-t[n-1]) * 1000 = 1000 value/sec (since dt=1ms, dv=1)
	for i := 0; i < 10; i++ {
		in <- pipeline.DataChunk{Timestamp: int64(i * 10), SourceID: "sim", Values: []float64{float64(i)}}
	}

	var results []float64
	timeout := time.After(500 * time.Millisecond)
	for len(results) < 10 {
		select {
		case c := <-out:
			results = append(results, c.Values[0])
		case <-timeout:
			t.Fatalf("timeout after %d results", len(results))
		}
	}

	// First sample: 0
	assert.InDelta(t, 0.0, results[0], 1e-10)
	// Subsequent: dv/dt = (1)/(10ms) * 1000 = 100 value/sec
	for i := 1; i < len(results); i++ {
		assert.InDelta(t, 100.0, results[i], 1e-10, "linear ramp derivative should be constant")
	}
}

func TestDerivativeFirstSampleZero(t *testing.T) {
	factory := Registry["derivative"]
	block := factory("deriv-first")

	in := make(chan pipeline.DataChunk, 5)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	in <- pipeline.DataChunk{Timestamp: 100, SourceID: "sim", Values: []float64{42.0}}

	select {
	case c := <-out:
		assert.InDelta(t, 0.0, c.Values[0], 1e-10, "first sample should output 0")
		assert.Equal(t, int64(100), c.Timestamp, "timestamp should be preserved")
		assert.Equal(t, "sim", c.SourceID, "sourceID should be preserved")
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout")
	}
}

func TestDerivativeVariableTimeIntervals(t *testing.T) {
	factory := Registry["derivative"]
	block := factory("deriv-var-dt")

	in := make(chan pipeline.DataChunk, 10)
	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	// Variable intervals: timestamps at 0, 10, 30, 60 ms; values at 0, 1, 3, 6
	samples := []struct {
		ts  int64
		val float64
	}{
		{0, 0},
		{10, 1},   // dv=1, dt=10ms → 100/s
		{30, 3},   // dv=2, dt=20ms → 100/s
		{60, 6},   // dv=3, dt=30ms → 100/s
	}

	for _, s := range samples {
		in <- pipeline.DataChunk{Timestamp: s.ts, SourceID: "sim", Values: []float64{s.val}}
	}

	var results []float64
	timeout := time.After(500 * time.Millisecond)
	for len(results) < len(samples) {
		select {
		case c := <-out:
			results = append(results, c.Values[0])
		case <-timeout:
			t.Fatalf("timeout after %d results", len(results))
		}
	}

	assert.InDelta(t, 0.0, results[0], 1e-10)
	assert.InDelta(t, 100.0, results[1], 1e-10)
	assert.InDelta(t, 100.0, results[2], 1e-10)
	assert.InDelta(t, 100.0, results[3], 1e-10)
}
