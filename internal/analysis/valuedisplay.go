package analysis

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"byteflow-studio/internal/pipeline"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func init() {
	registerAnalysis("value-display", func(id string) pipeline.Block {
		return &valueDisplayBlock{
			id:       id,
			decimals: 2,
			cfg:      pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 100},
		}
	})
}

type valueDisplayBlock struct {
	mu       sync.Mutex
	id       string
	cfg      pipeline.BufferConfig
	buf      []pipeline.DataChunk
	decimals int
	app      *application.App
}

func (v *valueDisplayBlock) ID() string                    { return v.id }
func (v *valueDisplayBlock) Type() string                  { return "value-display" }
func (v *valueDisplayBlock) Category() pipeline.BlockCategory { return pipeline.CategoryAnalysis }

func (v *valueDisplayBlock) Configure(params map[string]any) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if d, ok := params["decimals"]; ok {
		v.decimals = toIntA(d, 2)
	} else {
		v.decimals = 2
	}
	if mode, ok := params["bufferMode"].(string); ok {
		switch pipeline.BufferMode(mode) {
		case pipeline.BufferModeSamples:
			v.cfg.Mode = pipeline.BufferModeSamples
			if n, ok := params["bufferSamples"]; ok {
				v.cfg.MaxSamples = toIntA(n, 100)
			}
		case pipeline.BufferModeDuration:
			v.cfg.Mode = pipeline.BufferModeDuration
			if sec, ok := params["bufferDurationSec"]; ok {
				v.cfg.MaxDurationSec = toFloatA(sec, 10.0)
			}
		}
	}
	return nil
}

func (v *valueDisplayBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{
		{ID: "in-numeric", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "Numeric In"},
		{ID: "in-raw", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeRaw, Label: "Raw In"},
	}
}
func (v *valueDisplayBlock) OutputPorts() []pipeline.Port { return nil }

func (v *valueDisplayBlock) BufferConfig() pipeline.BufferConfig {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.cfg
}

func (v *valueDisplayBlock) SetBufferConfig(cfg pipeline.BufferConfig) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.cfg = cfg
	return nil
}

func (v *valueDisplayBlock) Snapshot() []pipeline.DataChunk {
	v.mu.Lock()
	defer v.mu.Unlock()
	result := make([]pipeline.DataChunk, len(v.buf))
	copy(result, v.buf)
	return result
}

func (v *valueDisplayBlock) addToBuffer(chunk pipeline.DataChunk) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.buf = append(v.buf, chunk)
	v.trimBuffer()
}

func (v *valueDisplayBlock) trimBuffer() {
	switch v.cfg.Mode {
	case pipeline.BufferModeSamples:
		if v.cfg.MaxSamples > 0 {
			for len(v.buf) > v.cfg.MaxSamples {
				v.buf = v.buf[1:]
			}
		}
	case pipeline.BufferModeDuration:
		if v.cfg.MaxDurationSec > 0 && len(v.buf) > 0 {
			cutoff := time.Now().UnixMilli() - int64(v.cfg.MaxDurationSec*1000)
			for len(v.buf) > 0 && v.buf[0].Timestamp < cutoff {
				v.buf = v.buf[1:]
			}
		}
	}
}

func (v *valueDisplayBlock) renderChunk(chunk pipeline.DataChunk) string {
	if len(chunk.Raw) > 0 {
		parts := make([]string, len(chunk.Raw))
		for i, b := range chunk.Raw {
			parts[i] = fmt.Sprintf("%02X", b)
		}
		return strings.Join(parts, " ")
	}
	if len(chunk.Values) > 0 {
		return fmt.Sprintf("%.*f", v.decimals, chunk.Values[0])
	}
	return ""
}

func (v *valueDisplayBlock) Run(
	ctx context.Context,
	inputs map[string]<-chan pipeline.DataChunk,
	outputs map[string]chan<- pipeline.DataChunk,
	errCh chan<- pipeline.BlockError,
) error {
	// Merge whichever input port is connected
	merged := make(chan pipeline.DataChunk, 64)
	var wg sync.WaitGroup

	for _, ch := range inputs {
		chCopy := ch
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case chunk, ok := <-chCopy:
					if !ok {
						return
					}
					select {
					case merged <- chunk:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case chunk, ok := <-merged:
			if !ok {
				return nil
			}
			v.addToBuffer(chunk)

			if v.app != nil {
				mode := "numeric"
				if len(chunk.Raw) > 0 {
					mode = "raw"
				}
				v.app.Event.Emit("pipeline:data", map[string]any{
					"blockId": v.id,
					"points": []map[string]any{
						{
							"timestamp": chunk.Timestamp,
							"values":    chunk.Values,
							"raw":       chunk.Raw,
							"mode":      mode,
						},
					},
				})
			}
		}
	}
}

func toIntA(v any, def int) int {
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

func toFloatA(v any, def float64) float64 {
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
