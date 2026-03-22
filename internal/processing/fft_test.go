package processing

import (
	"context"
	"math"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFFTDominantFrequency(t *testing.T) {
	factory, ok := Registry["fft"]
	require.True(t, ok)

	const windowSize = 512
	const sampleRateHz = 100.0
	const signalHz = 10.0

	block := factory("fft-1")
	err := block.Configure(map[string]any{"windowSize": windowSize, "windowFunction": "hann"})
	require.NoError(t, err)

	in := make(chan pipeline.DataChunk, windowSize+10)
	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	// Feed a pure 10 Hz sine wave sampled at 100 Hz
	for i := 0; i < windowSize; i++ {
		v := math.Sin(2 * math.Pi * signalHz * float64(i) / sampleRateHz)
		in <- pipeline.DataChunk{Timestamp: int64(i), Values: []float64{v}}
	}

	select {
	case got := <-out:
		require.NotEmpty(t, got.Values, "FFT output should have values")
		assert.Equal(t, windowSize/2, len(got.Values), "FFT output length should be windowSize/2")

		// Find dominant bin
		maxIdx := 0
		for i, v := range got.Values {
			if v > got.Values[maxIdx] {
				maxIdx = i
			}
		}
		// Expected bin for 10 Hz at 100 Hz sample rate with 512 window: bin 51
		// (freqBin = freq * windowSize / sampleRate = 10 * 512 / 100 = 51.2 ≈ 51)
		assert.Equal(t, 51, maxIdx, "dominant frequency bin should be 51")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for FFT output")
	}
}

func TestFFTOutputLength(t *testing.T) {
	factory := Registry["fft"]
	block := factory("fft-2")
	_ = block.Configure(map[string]any{"windowSize": 256, "windowFunction": "none"})

	in := make(chan pipeline.DataChunk, 300)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	for i := 0; i < 256; i++ {
		in <- pipeline.DataChunk{Timestamp: int64(i), Values: []float64{float64(i % 10)}}
	}

	select {
	case got := <-out:
		assert.Equal(t, 128, len(got.Values), "FFT output should be windowSize/2")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
