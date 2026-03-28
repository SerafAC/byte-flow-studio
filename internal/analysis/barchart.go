package analysis

import (
	"context"
	"sync"
	"time"

	"byteflow-studio/internal/pipeline"
)

func init() {
	registerAnalysis("bar-chart", func(id string) pipeline.Block {
		return &barChartBlock{
			id:  id,
			cfg: pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 50},
		}
	})
}

type barChartBlock struct {
	mu  sync.Mutex
	id  string
	cfg pipeline.BufferConfig
	buf []pipeline.DataChunk
}

func (b *barChartBlock) ID() string                       { return b.id }
func (b *barChartBlock) Type() string                     { return "bar-chart" }
func (b *barChartBlock) Category() pipeline.BlockCategory { return pipeline.CategoryAnalysis }

func (b *barChartBlock) Configure(params map[string]any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if mode, ok := params["bufferMode"].(string); ok {
		switch pipeline.BufferMode(mode) {
		case pipeline.BufferModeSamples:
			b.cfg.Mode = pipeline.BufferModeSamples
			if n, ok := params["bufferSamples"]; ok {
				b.cfg.MaxSamples = toIntA(n, 50)
			}
		case pipeline.BufferModeDuration:
			b.cfg.Mode = pipeline.BufferModeDuration
			if sec, ok := params["bufferDurationSec"]; ok {
				b.cfg.MaxDurationSec = toFloatA(sec, 5.0)
			}
		}
	}
	return nil
}

func (b *barChartBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (b *barChartBlock) OutputPorts() []pipeline.Port { return nil }

func (b *barChartBlock) BufferConfig() pipeline.BufferConfig {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.cfg
}

func (b *barChartBlock) SetBufferConfig(cfg pipeline.BufferConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cfg = cfg
	return nil
}

func (b *barChartBlock) Snapshot() []pipeline.DataChunk {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]pipeline.DataChunk, len(b.buf))
	copy(result, b.buf)
	return result
}

func (b *barChartBlock) addToBuffer(chunk pipeline.DataChunk) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, chunk)
	switch b.cfg.Mode {
	case pipeline.BufferModeSamples:
		if b.cfg.MaxSamples > 0 {
			for len(b.buf) > b.cfg.MaxSamples {
				b.buf = b.buf[1:]
			}
		}
	case pipeline.BufferModeDuration:
		if b.cfg.MaxDurationSec > 0 && len(b.buf) > 0 {
			cutoff := time.Now().UnixMilli() - int64(b.cfg.MaxDurationSec*1000)
			for len(b.buf) > 0 && b.buf[0].Timestamp < cutoff {
				b.buf = b.buf[1:]
			}
		}
	}
}

func (b *barChartBlock) Run(
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
			b.addToBuffer(chunk)
		}
	}
}
