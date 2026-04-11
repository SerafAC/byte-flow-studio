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

// --- Phase 2: Butterworth coefficient tests (T004) ---

func TestButterworthLowpassCoefficients(t *testing.T) {
	// 2nd order lowpass at 100 Hz, sample rate 1000 Hz
	b, a := butterworthCoeffs("lowpass", 100, 0, 0, 2, 1000)
	require.Len(t, b, 3, "2nd order filter should have 3 numerator coefficients")
	require.Len(t, a, 3, "2nd order filter should have 3 denominator coefficients")

	// a[0] should be 1.0 (normalized)
	assert.InDelta(t, 1.0, a[0], 1e-10, "a[0] must be 1.0")

	// Sum of b coefficients should equal (1 + a[1] + a[2]) for DC gain = 1
	bSum := b[0] + b[1] + b[2]
	aSum := 1.0 + a[1] + a[2]
	assert.InDelta(t, aSum, bSum, 1e-10, "DC gain should be 1.0 for lowpass")
}

func TestButterworthHighpassCoefficients(t *testing.T) {
	b, a := butterworthCoeffs("highpass", 100, 0, 0, 2, 1000)
	require.Len(t, b, 3)
	require.Len(t, a, 3)
	assert.InDelta(t, 1.0, a[0], 1e-10)

	// At DC (z=1), highpass gain should be 0
	bDC := b[0] + b[1] + b[2]
	assert.InDelta(t, 0.0, bDC, 1e-10, "highpass DC gain should be 0")
}

func TestButterworthBandpassCoefficients(t *testing.T) {
	b, a := butterworthCoeffs("bandpass", 0, 50, 200, 2, 1000)
	// Bandpass 2nd order prototype → 4th order digital
	require.Len(t, b, 5, "2nd order bandpass should produce 5 coefficients")
	require.Len(t, a, 5)
	assert.InDelta(t, 1.0, a[0], 1e-10)

	// At DC (z=1), bandpass gain should be 0
	bDC := 0.0
	for _, v := range b {
		bDC += v
	}
	assert.InDelta(t, 0.0, bDC, 1e-10, "bandpass DC gain should be 0")
}

func TestButterworthBandstopCoefficients(t *testing.T) {
	b, a := butterworthCoeffs("bandstop", 0, 50, 200, 2, 1000)
	require.Len(t, b, 5, "2nd order bandstop should produce 5 coefficients")
	require.Len(t, a, 5)
	assert.InDelta(t, 1.0, a[0], 1e-10)

	// At DC (z=1), bandstop gain should be 1
	bDC := 0.0
	aDC := 0.0
	for _, v := range b {
		bDC += v
	}
	for _, v := range a {
		aDC += v
	}
	assert.InDelta(t, 1.0, bDC/aDC, 1e-6, "bandstop DC gain should be 1")
}

func TestButterworthVariousOrders(t *testing.T) {
	for _, order := range []int{1, 2, 3, 4, 6, 8} {
		b, a := butterworthCoeffs("lowpass", 100, 0, 0, order, 1000)
		require.Len(t, b, order+1, "lowpass order %d: b length should be %d", order, order+1)
		require.Len(t, a, order+1, "lowpass order %d: a length should be %d", order, order+1)
		assert.InDelta(t, 1.0, a[0], 1e-10)

		// Bandpass: digital order = 2*N
		bBP, aBP := butterworthCoeffs("bandpass", 0, 50, 200, order, 1000)
		require.Len(t, bBP, 2*order+1, "bandpass order %d: b length should be %d", order, 2*order+1)
		require.Len(t, aBP, 2*order+1, "bandpass order %d: a length should be %d", order, 2*order+1)
	}
}

// --- Phase 3: Filter block tests (T006-T009) ---

func TestFilterBlockConfigure(t *testing.T) {
	factory, ok := Registry["filter"]
	require.True(t, ok, "filter should be registered")

	block := factory("filter-1")
	assert.Equal(t, "filter-1", block.ID())
	assert.Equal(t, "filter", block.Type())
	assert.Equal(t, pipeline.CategoryProcessing, block.Category())
	require.Len(t, block.InputPorts(), 1)
	require.Len(t, block.OutputPorts(), 1)
	assert.Equal(t, pipeline.DataTypeNum, block.InputPorts()[0].DataType)
	assert.Equal(t, pipeline.DataTypeNum, block.OutputPorts()[0].DataType)

	// Valid config
	err := block.Configure(map[string]any{
		"mode": "lowpass", "cutoffHz": 100.0, "order": 2, "sampleRateHz": 1000.0,
	})
	require.NoError(t, err)

	// Invalid mode
	err = block.Configure(map[string]any{"mode": "invalid"})
	assert.Error(t, err)

	// Invalid cutoffHz (>= Nyquist)
	err = block.Configure(map[string]any{
		"mode": "lowpass", "cutoffHz": 600.0, "sampleRateHz": 1000.0,
	})
	assert.Error(t, err)

	// Invalid order
	err = block.Configure(map[string]any{"mode": "lowpass", "cutoffHz": 100.0, "order": 0})
	assert.Error(t, err)
	err = block.Configure(map[string]any{"mode": "lowpass", "cutoffHz": 100.0, "order": 9})
	assert.Error(t, err)

	// Bandpass requires cutoffLowHz < cutoffHighHz
	err = block.Configure(map[string]any{
		"mode": "bandpass", "cutoffLowHz": 200.0, "cutoffHighHz": 100.0, "sampleRateHz": 1000.0,
	})
	assert.Error(t, err)
}

func TestFilterBlockLowpass(t *testing.T) {
	factory := Registry["filter"]
	block := factory("filter-lp")
	err := block.Configure(map[string]any{
		"mode": "lowpass", "cutoffHz": 50.0, "order": 4, "sampleRateHz": 1000.0,
	})
	require.NoError(t, err)

	in := make(chan pipeline.DataChunk, 2000)
	out := make(chan pipeline.DataChunk, 2000)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	// Generate a mixed signal: 10 Hz (pass) + 200 Hz (reject)
	sampleRate := 1000.0
	nSamples := 1000
	for i := 0; i < nSamples; i++ {
		t_s := float64(i) / sampleRate
		val := math.Sin(2*math.Pi*10*t_s) + math.Sin(2*math.Pi*200*t_s)
		in <- pipeline.DataChunk{
			Timestamp: int64(i),
			SourceID:  "sim",
			Values:    []float64{val},
		}
	}

	// Collect outputs
	var results []float64
	timeout := time.After(2 * time.Second)
	for len(results) < nSamples {
		select {
		case c := <-out:
			results = append(results, c.Values[0])
		case <-timeout:
			t.Fatalf("timeout after %d results", len(results))
		}
	}
	cancel()

	// After the filter settles (skip first 100 samples), the 200 Hz component should be heavily attenuated.
	// Measure RMS of the last 500 samples — should be close to the RMS of a pure 10 Hz sine (~0.707)
	var sumSq float64
	settled := results[500:]
	for _, v := range settled {
		sumSq += v * v
	}
	rms := math.Sqrt(sumSq / float64(len(settled)))
	// Pure 10 Hz sine RMS ≈ 0.707. With some residual 200 Hz, allow generous tolerance.
	assert.InDelta(t, 0.707, rms, 0.15, "lowpass should pass 10 Hz (RMS near 0.707), got %f", rms)
}

func TestFilterBlockHighpassBandpassBandstop(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		params  map[string]any
		freqPass float64  // frequency expected to pass
		freqStop float64  // frequency expected to be attenuated
	}{
		{
			name: "highpass",
			mode: "highpass",
			params: map[string]any{"mode": "highpass", "cutoffHz": 100.0, "order": 4, "sampleRateHz": 1000.0},
			freqPass: 200, freqStop: 10,
		},
		{
			name: "bandpass",
			mode: "bandpass",
			params: map[string]any{"mode": "bandpass", "cutoffLowHz": 80.0, "cutoffHighHz": 120.0, "order": 2, "sampleRateHz": 1000.0},
			freqPass: 100, freqStop: 10,
		},
		{
			name: "bandstop",
			mode: "bandstop",
			params: map[string]any{"mode": "bandstop", "cutoffLowHz": 80.0, "cutoffHighHz": 120.0, "order": 2, "sampleRateHz": 1000.0},
			freqPass: 10, freqStop: 100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			factory := Registry["filter"]
			block := factory("filter-" + tc.name)
			err := block.Configure(tc.params)
			require.NoError(t, err)

			in := make(chan pipeline.DataChunk, 2000)
			out := make(chan pipeline.DataChunk, 2000)
			errCh := make(chan pipeline.BlockError, 1)
			ctx, cancel := context.WithCancel(context.Background())

			go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

			sampleRate := 1000.0
			nSamples := 2000
			for i := 0; i < nSamples; i++ {
				t_s := float64(i) / sampleRate
				val := math.Sin(2*math.Pi*tc.freqPass*t_s) + math.Sin(2*math.Pi*tc.freqStop*t_s)
				in <- pipeline.DataChunk{Timestamp: int64(i), SourceID: "sim", Values: []float64{val}}
			}

			var results []float64
			timeout := time.After(3 * time.Second)
			for len(results) < nSamples {
				select {
				case c := <-out:
					results = append(results, c.Values[0])
				case <-timeout:
					t.Fatalf("timeout after %d results", len(results))
				}
			}
			cancel()

			// After settling, the stopped frequency should be attenuated
			// Measure max absolute amplitude of settled output — should be less than 2.0 (full amplitude of both)
			settled := results[1000:]
			var maxAbs float64
			for _, v := range settled {
				if math.Abs(v) > maxAbs {
					maxAbs = math.Abs(v)
				}
			}
			// With the stop frequency attenuated, max should be much closer to 1.0 than 2.0
			assert.Less(t, maxAbs, 1.6, "%s: max amplitude should be < 1.6 (stop freq attenuated)", tc.name)
		})
	}
}

func TestFilterBlockRealtimeParameterChange(t *testing.T) {
	factory := Registry["filter"]
	block := factory("filter-reconfigure")
	err := block.Configure(map[string]any{
		"mode": "lowpass", "cutoffHz": 50.0, "order": 2, "sampleRateHz": 1000.0,
	})
	require.NoError(t, err)

	in := make(chan pipeline.DataChunk, 1000)
	out := make(chan pipeline.DataChunk, 1000)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	// Send some samples
	for i := 0; i < 100; i++ {
		in <- pipeline.DataChunk{Timestamp: int64(i), SourceID: "sim", Values: []float64{1.0}}
	}
	// Drain
	drained := 0
	timeout := time.After(1 * time.Second)
	for drained < 100 {
		select {
		case <-out:
			drained++
		case <-timeout:
			t.Fatalf("timeout draining first batch, got %d", drained)
		}
	}

	// Reconfigure while running
	err = block.Configure(map[string]any{
		"mode": "highpass", "cutoffHz": 200.0, "order": 2, "sampleRateHz": 1000.0,
	})
	require.NoError(t, err)

	// Send more samples — should not crash or panic
	for i := 100; i < 200; i++ {
		in <- pipeline.DataChunk{Timestamp: int64(i), SourceID: "sim", Values: []float64{1.0}}
	}
	drained = 0
	timeout = time.After(1 * time.Second)
	for drained < 100 {
		select {
		case <-out:
			drained++
		case <-timeout:
			t.Fatalf("timeout draining second batch after reconfigure, got %d", drained)
		}
	}
}
