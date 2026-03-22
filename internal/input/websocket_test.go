package input

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func wsHandler(messages [][]byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for _, msg := range messages {
			_ = conn.WriteMessage(websocket.BinaryMessage, msg)
		}
		// Hold connection open briefly so the block can read all messages
		time.Sleep(200 * time.Millisecond)
	}
}

func TestWebSocketReceivesMessages(t *testing.T) {
	messages := [][]byte{
		{0xDE, 0xAD},
		{0xBE, 0xEF},
		{0xCA, 0xFE},
	}

	srv := httptest.NewServer(wsHandler(messages))
	defer srv.Close()

	factory, ok := processing.Registry["websocket"]
	require.True(t, ok, "websocket block must be registered")

	block := factory("ws-1")
	wsURL := "ws" + srv.URL[4:] + "/"
	err := block.Configure(map[string]any{
		"url":                wsURL,
		"reconnectIntervalMs": 5000,
	})
	require.NoError(t, err)

	out := make(chan pipeline.DataChunk, 10)
	errCh := make(chan pipeline.BlockError, 5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, nil, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	received := make([]pipeline.DataChunk, 0, len(messages))
	deadline := time.After(2 * time.Second)
	for len(received) < len(messages) {
		select {
		case chunk := <-out:
			received = append(received, chunk)
		case <-deadline:
			t.Fatalf("timeout: only received %d/%d messages", len(received), len(messages))
		}
	}

	require.Len(t, received, len(messages))
	for i, chunk := range received {
		assert.Equal(t, messages[i], chunk.Raw, "chunk %d Raw mismatch", i)
		assert.NotZero(t, chunk.Timestamp, "chunk %d must have timestamp", i)
	}
}

func TestWebSocketReconnect(t *testing.T) {
	const reconnectMs = 100

	// First server — sends one message then closes
	msg1 := [][]byte{{0x01}}
	srv1 := httptest.NewServer(wsHandler(msg1))

	factory := processing.Registry["websocket"]
	block := factory("ws-reconnect")
	wsURL := "ws" + srv1.URL[4:] + "/"
	_ = block.Configure(map[string]any{
		"url":                wsURL,
		"reconnectIntervalMs": reconnectMs,
	})

	out := make(chan pipeline.DataChunk, 20)
	errCh := make(chan pipeline.BlockError, 5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go block.Run(ctx, nil, map[string]chan<- pipeline.DataChunk{"out": out}, errCh)

	// Receive the first message
	select {
	case <-out:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first message")
	}

	// Close first server — block will detect disconnect and attempt reconnect
	srv1.Close()

	// Start second server at same relative path — block will reconnect
	msg2 := [][]byte{{0x02}}
	// We can't bind the same port, so we update the URL via Configure
	srv2 := httptest.NewServer(wsHandler(msg2))
	defer srv2.Close()
	newURL := "ws" + srv2.URL[4:] + "/"
	_ = block.Configure(map[string]any{
		"url":                newURL,
		"reconnectIntervalMs": reconnectMs,
	})

	// Should receive message from new server within reconnectMs + generous buffer
	select {
	case chunk := <-out:
		assert.Equal(t, msg2[0], chunk.Raw)
	case <-time.After(time.Duration(reconnectMs)*time.Millisecond + 500*time.Millisecond):
		t.Fatal("timeout: block did not reconnect to new server")
	}
}
