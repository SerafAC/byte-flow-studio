package input

import (
	"context"
	"fmt"
	"io"
	"time"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"

	"go.bug.st/serial"
)

func init() {
	processing.Register("uart", func(id string) pipeline.Block {
		return &uartBlock{id: id}
	})
}

// uartBlock reads bytes from a UART/serial port and emits one DataChunk per byte.
type uartBlock struct {
	id       string
	port     string
	baudRate int
	dataBits int
	stopBits serial.StopBits
	parity   serial.Parity
	reader   io.ReadCloser // injectable for testing
}

func (b *uartBlock) ID() string                    { return b.id }
func (b *uartBlock) Type() string                  { return "uart" }
func (b *uartBlock) Category() pipeline.BlockCategory { return pipeline.CategoryInput }

func (b *uartBlock) Configure(params map[string]any) error {
	if v, ok := params["port"].(string); ok {
		b.port = v
	}
	if v, ok := params["baudRate"]; ok {
		b.baudRate = toInt(v, 115200)
	} else {
		b.baudRate = 115200
	}
	if v, ok := params["dataBits"]; ok {
		b.dataBits = toInt(v, 8)
	} else {
		b.dataBits = 8
	}
	if v, ok := params["stopBits"]; ok {
		switch toFloat(v, 1.0) {
		case 1.5:
			b.stopBits = serial.OnePointFiveStopBits
		case 2:
			b.stopBits = serial.TwoStopBits
		default:
			b.stopBits = serial.OneStopBit
		}
	} else {
		b.stopBits = serial.OneStopBit
	}
	if v, ok := params["parity"].(string); ok {
		switch v {
		case "odd":
			b.parity = serial.OddParity
		case "even":
			b.parity = serial.EvenParity
		default:
			b.parity = serial.NoParity
		}
	} else {
		b.parity = serial.NoParity
	}
	return nil
}

func (b *uartBlock) InputPorts() []pipeline.Port  { return nil }
func (b *uartBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{
		{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeRaw, Label: "Out"},
	}
}

func (b *uartBlock) Run(
	ctx context.Context,
	inputs map[string]<-chan pipeline.DataChunk,
	outputs map[string]chan<- pipeline.DataChunk,
	errCh chan<- pipeline.BlockError,
) error {
	out, ok := outputs["out"]
	if !ok {
		return nil
	}

	var reader io.ReadCloser

	if b.reader != nil {
		reader = b.reader
	} else {
		if b.port == "" {
			err := fmt.Errorf("uart port not configured")
			pkgLog.Error("uart port not configured", "block_id", b.id)
			select {
			case errCh <- pipeline.BlockError{BlockID: b.id, Err: err}:
			default:
			}
			return err
		}
		mode := &serial.Mode{
			BaudRate: b.baudRate,
			DataBits: b.dataBits,
			StopBits: b.stopBits,
			Parity:   b.parity,
		}
		pkgLog.Debug("opening serial port", "block_id", b.id, "port", b.port, "baud_rate", b.baudRate)
		port, err := serial.Open(b.port, mode)
		if err != nil {
			pkgLog.Error("failed to open serial port", "block_id", b.id, "port", b.port, "error", err)
			select {
			case errCh <- pipeline.BlockError{BlockID: b.id, Err: err}:
			default:
			}
			return err
		}
		pkgLog.Info("serial port opened", "block_id", b.id, "port", b.port, "baud_rate", b.baudRate)
		reader = port
		defer reader.Close()
	}

	// Read one byte at a time and emit a DataChunk per byte
	buf := make([]byte, 256)
	readDone := make(chan struct {
		n   int
		err error
		buf []byte
	}, 1)

	for {
		// Launch a goroutine for the blocking read so we can also watch ctx.Done()
		go func() {
			n, err := reader.Read(buf)
			cp := make([]byte, n)
			copy(cp, buf[:n])
			readDone <- struct {
				n   int
				err error
				buf []byte
			}{n, err, cp}
		}()

		select {
		case <-ctx.Done():
			return nil
		case res := <-readDone:
			if res.err != nil {
				if ctx.Err() != nil {
					return nil
				}
				// EOF signals hardware disconnect — report as BlockError
				pkgLog.Error("serial read error", "block_id", b.id, "port", b.port, "error", res.err)
				select {
				case errCh <- pipeline.BlockError{BlockID: b.id, Err: res.err}:
				default:
				}
				return res.err
			}
			now := time.Now().UnixMilli()
			for _, byteVal := range res.buf[:res.n] {
				chunk := pipeline.DataChunk{
					Timestamp: now,
					SourceID:  b.id,
					Raw:       []byte{byteVal},
				}
				select {
				case out <- chunk:
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}

func toInt(v any, def int) int {
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

func toFloat(v any, def float64) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	}
	return def
}
