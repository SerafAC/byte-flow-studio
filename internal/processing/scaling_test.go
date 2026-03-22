package processing

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScalingFormula(t *testing.T) {
	factory, ok := Registry["scaling"]
	require.True(t, ok)

	for _, tc := range []struct {
		input, scale, offset, expected float64
	}{
		{0, 2.0, 1.0, 1.0},
		{-5.0, 3.0, 0.0, -15.0},
		{1e6, 0.001, 0.0, 1000.0},
	} {
		block := factory("scale-test")
		_ = block.Configure(map[string]any{"scale": tc.scale, "offset": tc.offset})

		in := make(chan pipeline.DataChunk, 5)
		out := make(chan pipeline.DataChunk, 5)
		errCh := make(chan pipeline.BlockError, 1)
		ctx, cancel := context.WithCancel(context.Background())

		go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

		in <- pipeline.DataChunk{Timestamp: 1, Values: []float64{tc.input}}

		select {
		case got := <-out:
			require.Len(t, got.Values, 1)
			assert.InDelta(t, tc.expected, got.Values[0], 0.001)
		case <-time.After(300 * time.Millisecond):
			t.Fatalf("timeout for input=%v", tc.input)
		}
		cancel()
	}
}
