package analysis

import (
	"context"
	"sync"
	"time"

	"byteflow-studio/internal/pipeline"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func init() {
	registerAnalysis("line-chart", func(id string) pipeline.Block {
		return &lineChartBlock{
			id:  id,
			cfg: pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 500},
		}
	})
}

type lineChartBlock struct {
	mu  sync.Mutex
	id  string
	cfg pipeline.BufferConfig
	buf []pipeline.DataChunk
	app *application.App
}

func (l *lineChartBlock) ID() string                       { return l.id }
func (l *lineChartBlock) Type() string                     { return "line-chart" }
func (l *lineChartBlock) Category() pipeline.BlockCategory { return pipeline.CategoryAnalysis }

func (l *lineChartBlock) Configure(params map[string]any) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if mode, ok := params["bufferMode"].(string); ok {
		switch pipeline.BufferMode(mode) {
		case pipeline.BufferModeSamples:
			l.cfg.Mode = pipeline.BufferModeSamples
			if n, ok := params["bufferSamples"]; ok {
				l.cfg.MaxSamples = toIntA(n, 500)
			}
		case pipeline.BufferModeDuration:
			l.cfg.Mode = pipeline.BufferModeDuration
			if sec, ok := params["bufferDurationSec"]; ok {
				l.cfg.MaxDurationSec = toFloatA(sec, 10.0)
			}
		}
	}
	return nil
}

func (l *lineChartBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (l *lineChartBlock) OutputPorts() []pipeline.Port { return nil }

func (l *lineChartBlock) BufferConfig() pipeline.BufferConfig {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cfg
}

func (l *lineChartBlock) SetBufferConfig(cfg pipeline.BufferConfig) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cfg = cfg
	return nil
}

func (l *lineChartBlock) Snapshot() []pipeline.DataChunk {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]pipeline.DataChunk, len(l.buf))
	copy(result, l.buf)
	return result
}

func (l *lineChartBlock) addToBuffer(chunk pipeline.DataChunk) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf = append(l.buf, chunk)
	switch l.cfg.Mode {
	case pipeline.BufferModeSamples:
		if l.cfg.MaxSamples > 0 {
			for len(l.buf) > l.cfg.MaxSamples {
				l.buf = l.buf[1:]
			}
		}
	case pipeline.BufferModeDuration:
		if l.cfg.MaxDurationSec > 0 && len(l.buf) > 0 {
			cutoff := time.Now().UnixMilli() - int64(l.cfg.MaxDurationSec*1000)
			for len(l.buf) > 0 && l.buf[0].Timestamp < cutoff {
				l.buf = l.buf[1:]
			}
		}
	}
}

func (l *lineChartBlock) Run(
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
			l.addToBuffer(chunk)

			if l.app != nil {
				l.app.Event.Emit("pipeline:data", map[string]any{
					"blockId": l.id,
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
