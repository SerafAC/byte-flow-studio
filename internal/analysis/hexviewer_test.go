package analysis

import (
	"context"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHexViewerBufferAndSnapshot(t *testing.T) {
	factory, ok := processing.Registry["hex-viewer"]
	require.True(t, ok, "hex-viewer should be registered")

	block := factory("hv-1")
	assert.Equal(t, "hex-viewer", block.Type())
	assert.Equal(t, pipeline.CategoryAnalysis, block.Category())
	require.Len(t, block.InputPorts(), 1)
	assert.Equal(t, pipeline.DataTypeRaw, block.InputPorts()[0].DataType)
	assert.Empty(t, block.OutputPorts())

	ab, ok := block.(pipeline.AnalysisBlock)
	require.True(t, ok, "should implement AnalysisBlock")

	// Configure small buffer
	hvBlock := block.(*hexViewerBlock)
	hvBlock.maxBytes = 32

	in := make(chan pipeline.DataChunk, 20)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{}, errCh)

	// Send 64 bytes in chunks — buffer should cap at 32
	for i := 0; i < 8; i++ {
		raw := make([]byte, 8)
		for j := range raw {
			raw[j] = byte(i*8 + j)
		}
		in <- pipeline.DataChunk{Timestamp: int64(i), SourceID: "uart", Raw: raw}
	}
	time.Sleep(50 * time.Millisecond)

	snap := ab.Snapshot()
	totalBytes := 0
	for _, chunk := range snap {
		totalBytes += len(chunk.Raw)
	}
	assert.LessOrEqual(t, totalBytes, 32, "total bytes in buffer should be <= maxBytes")
	assert.Greater(t, totalBytes, 0)
}

func TestHexViewerBytePatternSearch(t *testing.T) {
	factory := processing.Registry["hex-viewer"]
	block := factory("hv-search")
	hvBlock := block.(*hexViewerBlock)
	hvBlock.maxBytes = 1024

	in := make(chan pipeline.DataChunk, 20)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{}, errCh)

	// Send data with a known pattern
	data := []byte{0x00, 0x01, 0xCA, 0xFE, 0x02, 0x03, 0xCA, 0xFE, 0x04}
	in <- pipeline.DataChunk{Timestamp: 1, SourceID: "test", Raw: data}
	time.Sleep(50 * time.Millisecond)

	// Search for pattern 0xCA 0xFE
	matches := hvBlock.SearchPattern([]byte{0xCA, 0xFE})
	assert.Len(t, matches, 2, "should find pattern at 2 positions")
	assert.Equal(t, 2, matches[0])
	assert.Equal(t, 6, matches[1])

	// Search for non-existent pattern
	matches = hvBlock.SearchPattern([]byte{0xFF, 0xFF})
	assert.Len(t, matches, 0)
}
