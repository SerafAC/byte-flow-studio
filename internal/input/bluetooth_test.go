package input

import (
	"context"
	"io"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBluetoothBlockConfigure(t *testing.T) {
	factory, ok := processing.Registry["bluetooth"]
	require.True(t, ok, "bluetooth should be registered")

	block := factory("bt-1")
	assert.Equal(t, "bt-1", block.ID())
	assert.Equal(t, "bluetooth", block.Type())
	assert.Equal(t, pipeline.CategoryInput, block.Category())
	assert.Empty(t, block.InputPorts())
	require.Len(t, block.OutputPorts(), 1)
	assert.Equal(t, pipeline.DataTypeRaw, block.OutputPorts()[0].DataType)

	err := block.Configure(map[string]any{
		"deviceAddress": "AA:BB:CC:DD:EE:FF",
		"serialPort":    "/dev/rfcomm0",
	})
	require.NoError(t, err)

	btBlock := block.(*bluetoothBlock)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", btBlock.deviceAddress)
	assert.Equal(t, "/dev/rfcomm0", btBlock.serialPort)
}

func TestBluetoothBlockRawByteStreaming(t *testing.T) {
	factory := processing.Registry["bluetooth"]
	block := factory("bt-stream")

	pr, pw := io.Pipe()
	btBlock := block.(*bluetoothBlock)
	btBlock.reader = pr

	out := make(chan pipeline.DataChunk, 20)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx,
		map[string]<-chan pipeline.DataChunk{},
		map[string]chan<- pipeline.DataChunk{"out": out},
		errCh,
	)

	testBytes := []byte{0xCA, 0xFE, 0xBA, 0xBE}
	_, err := pw.Write(testBytes)
	require.NoError(t, err)

	// Collect raw byte chunks
	var collected []byte
	timeout := time.After(500 * time.Millisecond)
	for len(collected) < len(testBytes) {
		select {
		case chunk := <-out:
			assert.NotNil(t, chunk.Raw)
			assert.Equal(t, "bt-stream", chunk.SourceID)
			collected = append(collected, chunk.Raw...)
		case <-timeout:
			t.Fatalf("timeout after collecting %d bytes", len(collected))
		}
	}
	assert.Equal(t, testBytes, collected)

	cancel()
	pw.Close()
}

func TestBluetoothBlockDisconnectDetection(t *testing.T) {
	factory := processing.Registry["bluetooth"]
	block := factory("bt-disc")

	pr, pw := io.Pipe()
	btBlock := block.(*bluetoothBlock)
	btBlock.reader = pr
	btBlock.maxRetries = 0 // disable reconnect for this test

	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		block.Run(ctx,
			map[string]<-chan pipeline.DataChunk{},
			map[string]chan<- pipeline.DataChunk{"out": out},
			errCh,
		)
	}()

	// Write one byte then close
	_, err := pw.Write([]byte{0xAA})
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	pw.Close() // triggers io.EOF

	select {
	case be := <-errCh:
		assert.Equal(t, "bt-disc", be.BlockID)
		assert.NotNil(t, be.Err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected BlockError on EOF")
	}

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("block goroutine did not exit after EOF")
	}
}

func TestBluetoothBlockContextCancellation(t *testing.T) {
	factory := processing.Registry["bluetooth"]
	block := factory("bt-cancel")

	pr, pw := io.Pipe()
	btBlock := block.(*bluetoothBlock)
	btBlock.reader = pr

	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 1)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		block.Run(ctx,
			map[string]<-chan pipeline.DataChunk{},
			map[string]chan<- pipeline.DataChunk{"out": out},
			errCh,
		)
	}()

	cancel()
	pw.Close()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("block did not exit after context cancellation")
	}
}
