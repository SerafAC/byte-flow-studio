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

// mockReadCloser wraps a pipe reader to satisfy the serial.Port interface requirements.
type mockReadCloser struct {
	io.ReadCloser
}

func TestUARTBlockEmitsChunks(t *testing.T) {
	factory, ok := processing.Registry["uart"]
	require.True(t, ok, "uart should be registered")

	block := factory("uart-1")
	assert.Equal(t, "uart-1", block.ID())
	assert.Equal(t, "uart", block.Type())
	assert.Equal(t, pipeline.CategoryInput, block.Category())
	assert.Empty(t, block.InputPorts())
	require.Len(t, block.OutputPorts(), 1)
	assert.Equal(t, pipeline.DataTypeRaw, block.OutputPorts()[0].DataType)

	// Use os.Pipe() to simulate a serial port
	pr, pw := io.Pipe()

	uartBlock := block.(*uartBlock)
	uartBlock.reader = pr

	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		block.Run(ctx,
			map[string]<-chan pipeline.DataChunk{},
			map[string]chan<- pipeline.DataChunk{"out": out},
			errCh,
		)
	}()

	// Write test bytes
	testBytes := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	before := time.Now().UnixMilli()
	_, err := pw.Write(testBytes)
	require.NoError(t, err)

	// Collect chunks (one per byte)
	collected := make([]byte, 0, len(testBytes))
	timeout := time.After(500 * time.Millisecond)
	for len(collected) < len(testBytes) {
		select {
		case chunk := <-out:
			after := time.Now().UnixMilli()
			assert.GreaterOrEqual(t, chunk.Timestamp, before)
			assert.LessOrEqual(t, chunk.Timestamp, after)
			assert.NotNil(t, chunk.Raw)
			collected = append(collected, chunk.Raw...)
		case <-timeout:
			t.Fatalf("timed out after collecting %d bytes", len(collected))
		}
	}
	assert.Equal(t, testBytes, collected)

	// Cancel context and verify clean exit
	cancel()
	pw.Close()
	select {
	case be := <-errCh:
		// EOF on close is acceptable — no unexpected BlockError
		t.Logf("block error (may be ok on pipe close): %v", be.Err)
	case <-time.After(300 * time.Millisecond):
		// Clean exit — no error
	}
}

// TestUARTDisconnect verifies that an io.EOF mid-read causes a BlockError to be sent
// to errCh and the block goroutine to exit cleanly (T103a).
func TestUARTDisconnect(t *testing.T) {
	factory, ok := processing.Registry["uart"]
	require.True(t, ok)

	block := factory("uart-disc")
	uartBlock := block.(*uartBlock)

	pr, pw := io.Pipe()
	uartBlock.reader = pr

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

	// Write one byte then close write end — causes io.EOF on next read
	_, err := pw.Write([]byte{0xAA})
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond) // let the byte be consumed
	pw.Close()                         // triggers io.EOF

	// Block should send BlockError to errCh
	select {
	case be := <-errCh:
		assert.Equal(t, "uart-disc", be.BlockID)
		assert.NotNil(t, be.Err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected BlockError on EOF but got none")
	}

	// Block goroutine should exit cleanly
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("block goroutine did not exit after EOF")
	}
}

func TestUARTBlockContextCancellation(t *testing.T) {
	factory, ok := processing.Registry["uart"]
	require.True(t, ok)

	block := factory("uart-cancel")
	uartBlock := block.(*uartBlock)
	pr, pw := io.Pipe()
	uartBlock.reader = pr

	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 1)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		block.Run(ctx,
			map[string]<-chan pipeline.DataChunk{},
			map[string]chan<- pipeline.DataChunk{"out": out},
			errCh,
		)
		close(done)
	}()

	// Cancel immediately
	cancel()
	pw.Close()

	select {
	case <-done:
		// Clean exit
	case <-time.After(500 * time.Millisecond):
		t.Fatal("block did not exit after context cancellation")
	}
}
