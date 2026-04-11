package processing

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiplyBlockConfigure(t *testing.T) {
	factory, ok := Registry["multiply"]
	require.True(t, ok, "multiply should be registered")

	block := factory("mul-1")
	assert.Equal(t, "multiply", block.Type())
	assert.Equal(t, pipeline.CategoryProcessing, block.Category())

	// Default: 2 inputs
	require.Len(t, block.InputPorts(), 2)
	assert.Equal(t, "in-0", block.InputPorts()[0].ID)
	assert.Equal(t, "in-1", block.InputPorts()[1].ID)
	require.Len(t, block.OutputPorts(), 1)

	// Configure with 4 inputs
	err := block.Configure(map[string]any{"inputCount": 4})
	require.NoError(t, err)
	require.Len(t, block.InputPorts(), 4)

	// Invalid: < 2
	err = block.Configure(map[string]any{"inputCount": 1})
	assert.Error(t, err)

	// Invalid: > 8
	err = block.Configure(map[string]any{"inputCount": 9})
	assert.Error(t, err)
}

func TestMultiplyTwoInputs(t *testing.T) {
	factory := Registry["multiply"]
	block := factory("mul-2in")
	err := block.Configure(map[string]any{"inputCount": 2})
	require.NoError(t, err)

	in0 := make(chan pipeline.DataChunk, 10)
	in1 := make(chan pipeline.DataChunk, 10)
	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx,
		map[string]<-chan pipeline.DataChunk{"in-0": in0, "in-1": in1},
		map[string]chan<- pipeline.DataChunk{"out": out},
		errCh,
	)

	// Send values to both inputs
	in0 <- pipeline.DataChunk{Timestamp: 1, SourceID: "s1", Values: []float64{3.0}}
	in1 <- pipeline.DataChunk{Timestamp: 2, SourceID: "s2", Values: []float64{4.0}}

	select {
	case c := <-out:
		assert.InDelta(t, 12.0, c.Values[0], 1e-10, "3 * 4 = 12")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for multiply output")
	}
}

func TestMultiplyThreeInputs(t *testing.T) {
	factory := Registry["multiply"]
	block := factory("mul-3in")
	err := block.Configure(map[string]any{"inputCount": 3})
	require.NoError(t, err)

	in0 := make(chan pipeline.DataChunk, 10)
	in1 := make(chan pipeline.DataChunk, 10)
	in2 := make(chan pipeline.DataChunk, 10)
	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx,
		map[string]<-chan pipeline.DataChunk{"in-0": in0, "in-1": in1, "in-2": in2},
		map[string]chan<- pipeline.DataChunk{"out": out},
		errCh,
	)

	in0 <- pipeline.DataChunk{Timestamp: 1, Values: []float64{2.0}}
	in1 <- pipeline.DataChunk{Timestamp: 2, Values: []float64{3.0}}
	in2 <- pipeline.DataChunk{Timestamp: 3, Values: []float64{5.0}}

	select {
	case c := <-out:
		assert.InDelta(t, 30.0, c.Values[0], 1e-10, "2 * 3 * 5 = 30")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for multiply output")
	}
}

func TestMultiplyLatestValueWins(t *testing.T) {
	factory := Registry["multiply"]
	block := factory("mul-latest")
	err := block.Configure(map[string]any{"inputCount": 2})
	require.NoError(t, err)

	in0 := make(chan pipeline.DataChunk, 10)
	in1 := make(chan pipeline.DataChunk, 10)
	out := make(chan pipeline.DataChunk, 20)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx,
		map[string]<-chan pipeline.DataChunk{"in-0": in0, "in-1": in1},
		map[string]chan<- pipeline.DataChunk{"out": out},
		errCh,
	)

	// First: establish both values
	in0 <- pipeline.DataChunk{Timestamp: 1, Values: []float64{2.0}}
	in1 <- pipeline.DataChunk{Timestamp: 2, Values: []float64{3.0}}

	select {
	case c := <-out:
		assert.InDelta(t, 6.0, c.Values[0], 1e-10)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout")
	}

	// Update in-0 with new value; in-1 retains latest value (3.0)
	in0 <- pipeline.DataChunk{Timestamp: 3, Values: []float64{5.0}}

	select {
	case c := <-out:
		assert.InDelta(t, 15.0, c.Values[0], 1e-10, "5 * 3 (latest) = 15")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout")
	}
}
