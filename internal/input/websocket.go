package input

import (
	"context"
	"sync"
	"time"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"

	"github.com/gorilla/websocket"
)

func init() {
	processing.Register("websocket", func(id string) pipeline.Block {
		return &wsBlock{
			id:                 id,
			url:                "",
			reconnectIntervalMs: 2000,
		}
	})
}

type wsBlock struct {
	mu                  sync.RWMutex
	id                  string
	url                 string
	subprotocol         string
	reconnectIntervalMs int
}

func (w *wsBlock) ID() string                       { return w.id }
func (w *wsBlock) Type() string                     { return "websocket" }
func (w *wsBlock) Category() pipeline.BlockCategory { return pipeline.CategoryInput }

func (w *wsBlock) Configure(params map[string]any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if v, ok := params["url"].(string); ok {
		w.url = v
	}
	if v, ok := params["subprotocol"].(string); ok {
		w.subprotocol = v
	}
	if v, ok := params["reconnectIntervalMs"]; ok {
		w.reconnectIntervalMs = toIntP(v, 2000)
	}
	return nil
}

func (w *wsBlock) InputPorts() []pipeline.Port  { return nil }
func (w *wsBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeRaw, Label: "Out"}}
}

func (w *wsBlock) Run(
	ctx context.Context,
	_ map[string]<-chan pipeline.DataChunk,
	outputs map[string]chan<- pipeline.DataChunk,
	errCh chan<- pipeline.BlockError,
) error {
	out, ok := outputs["out"]
	if !ok {
		return nil
	}

	const maxFailedReconnects = 5
	failedReconnects := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		w.mu.RLock()
		url := w.url
		subproto := w.subprotocol
		reconnectMs := w.reconnectIntervalMs
		w.mu.RUnlock()

		if url == "" {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		dialer := websocket.Dialer{}
		var headers map[string][]string
		if subproto != "" {
			headers = map[string][]string{"Sec-WebSocket-Protocol": {subproto}}
		}

		conn, _, err := dialer.DialContext(ctx, url, headers)
		if err != nil {
			failedReconnects++
			pkgLog.Warn("websocket connection failed", "block_id", w.id, "url", url, "attempt", failedReconnects, "error", err)
			if failedReconnects >= maxFailedReconnects {
				pkgLog.Error("websocket max reconnect attempts reached", "block_id", w.id, "url", url, "max_attempts", maxFailedReconnects)
				select {
				case errCh <- pipeline.BlockError{BlockID: w.id, Err: err}:
				default:
				}
				return nil
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Duration(reconnectMs) * time.Millisecond):
			}
			continue
		}

		pkgLog.Info("websocket connected", "block_id", w.id, "url", url)
		failedReconnects = 0
		err = w.readLoop(ctx, conn, out)
		conn.Close()

		if err != nil && ctx.Err() != nil {
			return nil
		}
		if err != nil {
			pkgLog.Warn("websocket disconnected", "block_id", w.id, "url", url, "error", err)
		}

		// Disconnected — retry after interval
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(reconnectMs) * time.Millisecond):
		}
	}
}

func (w *wsBlock) readLoop(ctx context.Context, conn *websocket.Conn, out chan<- pipeline.DataChunk) error {
	// Close the connection when context is cancelled
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		chunk := pipeline.DataChunk{
			Timestamp: time.Now().UnixMilli(),
			SourceID:  w.id,
			Raw:       msg,
		}
		select {
		case out <- chunk:
		case <-ctx.Done():
			return nil
		}
	}
}

func toIntP(v any, def int) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	}
	return def
}
