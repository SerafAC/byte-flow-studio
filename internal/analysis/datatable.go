package analysis

import (
	"context"
	"sync"
	"time"

	"byteflow-studio/internal/pipeline"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func init() {
	registerAnalysis("data-table", func(id string) pipeline.Block {
		return &dataTableBlock{
			id:  id,
			cfg: pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 200},
		}
	})
}

type dataTableBlock struct {
	mu  sync.Mutex
	id  string
	cfg pipeline.BufferConfig
	buf []pipeline.DataChunk
	app *application.App
}

func (d *dataTableBlock) ID() string                       { return d.id }
func (d *dataTableBlock) Type() string                     { return "data-table" }
func (d *dataTableBlock) Category() pipeline.BlockCategory { return pipeline.CategoryAnalysis }

func (d *dataTableBlock) Configure(params map[string]any) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if mode, ok := params["bufferMode"].(string); ok {
		switch pipeline.BufferMode(mode) {
		case pipeline.BufferModeSamples:
			d.cfg.Mode = pipeline.BufferModeSamples
			if n, ok := params["bufferSamples"]; ok {
				d.cfg.MaxSamples = toIntA(n, 200)
			}
		case pipeline.BufferModeDuration:
			d.cfg.Mode = pipeline.BufferModeDuration
			if sec, ok := params["bufferDurationSec"]; ok {
				d.cfg.MaxDurationSec = toFloatA(sec, 10.0)
			}
		}
	}
	return nil
}

func (d *dataTableBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (d *dataTableBlock) OutputPorts() []pipeline.Port { return nil }

func (d *dataTableBlock) BufferConfig() pipeline.BufferConfig {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cfg
}

func (d *dataTableBlock) SetBufferConfig(cfg pipeline.BufferConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cfg = cfg
	return nil
}

func (d *dataTableBlock) Snapshot() []pipeline.DataChunk {
	d.mu.Lock()
	defer d.mu.Unlock()
	result := make([]pipeline.DataChunk, len(d.buf))
	copy(result, d.buf)
	return result
}

func (d *dataTableBlock) addToBuffer(chunk pipeline.DataChunk) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.buf = append(d.buf, chunk)
	switch d.cfg.Mode {
	case pipeline.BufferModeSamples:
		if d.cfg.MaxSamples > 0 {
			for len(d.buf) > d.cfg.MaxSamples {
				d.buf = d.buf[1:]
			}
		}
	case pipeline.BufferModeDuration:
		if d.cfg.MaxDurationSec > 0 && len(d.buf) > 0 {
			cutoff := time.Now().UnixMilli() - int64(d.cfg.MaxDurationSec*1000)
			for len(d.buf) > 0 && d.buf[0].Timestamp < cutoff {
				d.buf = d.buf[1:]
			}
		}
	}
}

func (d *dataTableBlock) Run(
	ctx context.Context,
	inputs map[string]<-chan pipeline.DataChunk,
	_ map[string]chan<- pipeline.DataChunk,
	_ chan<- pipeline.BlockError,
) error {
	in, ok := inputs["in"]
	if !ok {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case chunk, open := <-in:
			if !open {
				return nil
			}
			d.addToBuffer(chunk)

			if d.app != nil {
				d.app.Event.Emit("pipeline:data", map[string]any{
					"blockId": d.id,
					"points": []map[string]any{
						{
							"timestamp": chunk.Timestamp,
							"values":    chunk.Values,
						},
					},
				})
			}
		}
	}
}
