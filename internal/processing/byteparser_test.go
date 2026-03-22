package processing

import (
	"context"
	"encoding/binary"
	"math"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chunkWithRaw(ts int64, data []byte) pipeline.DataChunk {
	return pipeline.DataChunk{Timestamp: ts, SourceID: "src", Raw: data}
}

func float32leBytes(v float32) []byte {
	bits := math.Float32bits(v)
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, bits)
	return b
}

func int16beBytes(v int16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func TestByteParserFloat32LE(t *testing.T) {
	factory, ok := Registry["byte-parser"]
	require.True(t, ok)

	block := factory("bp-1")
	err := block.Configure(map[string]any{"format": "float32-le", "channels": 1, "frameSize": 4})
	require.NoError(t, err)

	in := make(chan pipeline.DataChunk, 5)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	expected := float32(3.14)
	in <- chunkWithRaw(100, float32leBytes(expected))

	select {
	case got := <-out:
		require.Len(t, got.Values, 1)
		assert.InDelta(t, float64(expected), got.Values[0], 0.001)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout")
	}
}

func TestByteParserInt16BE(t *testing.T) {
	factory := Registry["byte-parser"]
	block := factory("bp-2")
	_ = block.Configure(map[string]any{"format": "int16-be", "channels": 1, "frameSize": 2})

	in := make(chan pipeline.DataChunk, 5)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	in <- chunkWithRaw(1, int16beBytes(1000))

	select {
	case got := <-out:
		require.Len(t, got.Values, 1)
		assert.InDelta(t, 1000.0, got.Values[0], 0.001)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout")
	}
}

func TestByteParserUint8(t *testing.T) {
	factory := Registry["byte-parser"]
	block := factory("bp-3")
	_ = block.Configure(map[string]any{"format": "uint8", "channels": 1, "frameSize": 1})

	in := make(chan pipeline.DataChunk, 5)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	in <- chunkWithRaw(1, []byte{255})

	select {
	case got := <-out:
		require.Len(t, got.Values, 1)
		assert.InDelta(t, 255.0, got.Values[0], 0.001)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout")
	}
}

func TestByteParserMultiChannel(t *testing.T) {
	factory := Registry["byte-parser"]
	block := factory("bp-4")
	_ = block.Configure(map[string]any{"format": "float32-le", "channels": 2, "frameSize": 8})

	in := make(chan pipeline.DataChunk, 5)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	frame := append(float32leBytes(1.5), float32leBytes(2.5)...)
	in <- chunkWithRaw(1, frame)

	select {
	case got := <-out:
		require.Len(t, got.Values, 2)
		assert.InDelta(t, 1.5, got.Values[0], 0.001)
		assert.InDelta(t, 2.5, got.Values[1], 0.001)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout")
	}
}

func TestByteParserPartialFrameBuffering(t *testing.T) {
	factory := Registry["byte-parser"]
	block := factory("bp-5")
	_ = block.Configure(map[string]any{"format": "float32-le", "channels": 1, "frameSize": 4})

	in := make(chan pipeline.DataChunk, 5)
	out := make(chan pipeline.DataChunk, 5)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, map[string]<-chan pipeline.DataChunk{"in": in}, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	// Send 3 bytes first — not enough for a frame
	expected := float32(6.28)
	all := float32leBytes(expected)
	in <- chunkWithRaw(1, all[:3])

	// No output expected yet
	select {
	case <-out:
		t.Fatal("should not emit output with only 3 bytes")
	case <-time.After(100 * time.Millisecond):
	}

	// Send the remaining 1 byte to complete the frame
	in <- chunkWithRaw(2, all[3:])

	select {
	case got := <-out:
		require.Len(t, got.Values, 1)
		assert.InDelta(t, float64(expected), got.Values[0], 0.001)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for completed frame")
	}
}
