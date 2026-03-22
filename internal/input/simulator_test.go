package input

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimulatorEmitsSampleRate(t *testing.T) {
	factory, ok := processing.Registry["simulator"]
	require.True(t, ok, "simulator should be registered")

	block := factory("sim-1")
	err := block.Configure(map[string]any{
		"waveform":     "sine",
		"frequencyHz":  1.0,
		"amplitude":    1.0,
		"offset":       0.0,
		"sampleRateHz": 50.0,
	})
	require.NoError(t, err)

	out := make(chan pipeline.DataChunk, 200)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 1100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		block.Run(ctx, nil, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)
		close(done)
	}()

	<-done

	count := len(out)
	// 50 Hz for ~1 second → expect 40–60 samples (within 20% tolerance)
	assert.GreaterOrEqual(t, count, 40, "expected at least 40 samples at 50 Hz")
	assert.LessOrEqual(t, count, 60, "expected at most 60 samples at 50 Hz")
}

func TestSimulatorMonotonicallyIncreasingTimestamps(t *testing.T) {
	factory := processing.Registry["simulator"]
	block := factory("sim-2")
	_ = block.Configure(map[string]any{"sampleRateHz": 100.0, "waveform": "sine"})

	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	go block.Run(ctx, nil, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)
	<-ctx.Done()

	chunks := make([]pipeline.DataChunk, 0)
	for {
		select {
		case c := <-out:
			chunks = append(chunks, c)
		default:
			goto done
		}
	}
done:
	for i := 1; i < len(chunks); i++ {
		assert.LessOrEqual(t, chunks[i-1].Timestamp, chunks[i].Timestamp)
	}
}

func TestSimulatorEmitsNumericNotRaw(t *testing.T) {
	factory := processing.Registry["simulator"]
	block := factory("sim-3")
	_ = block.Configure(map[string]any{"sampleRateHz": 100.0, "waveform": "noise"})

	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go block.Run(ctx, nil, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)
	<-ctx.Done()

	select {
	case chunk := <-out:
		assert.Nil(t, chunk.Raw, "simulator should not set Raw field")
		assert.NotEmpty(t, chunk.Values, "simulator should set Values field")
	default:
		t.Skip("no chunks emitted in time window")
	}
}
