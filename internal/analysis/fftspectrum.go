package analysis

import (
	"context"
	"sync"
	"time"

	"byteflow-studio/internal/pipeline"
)

func init() {
	registerAnalysis("fft-spectrum", func(id string) pipeline.Block {
		return &fftSpectrumBlock{
			id:  id,
			cfg: pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 10},
		}
	})
}

type fftSpectrumBlock struct {
	mu  sync.Mutex
	id  string
	cfg pipeline.BufferConfig
	buf []pipeline.DataChunk
}

func (f *fftSpectrumBlock) ID() string                       { return f.id }
func (f *fftSpectrumBlock) Type() string                     { return "fft-spectrum" }
func (f *fftSpectrumBlock) Category() pipeline.BlockCategory { return pipeline.CategoryAnalysis }

func (f *fftSpectrumBlock) Configure(params map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if mode, ok := params["bufferMode"].(string); ok {
		switch pipeline.BufferMode(mode) {
		case pipeline.BufferModeSamples:
			f.cfg.Mode = pipeline.BufferModeSamples
			if n, ok := params["bufferSamples"]; ok {
				f.cfg.MaxSamples = toIntA(n, 10)
			}
		case pipeline.BufferModeDuration:
			f.cfg.Mode = pipeline.BufferModeDuration
			if sec, ok := params["bufferDurationSec"]; ok {
				f.cfg.MaxDurationSec = toFloatA(sec, 5.0)
			}
		}
	}
	return nil
}

func (f *fftSpectrumBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "Spectrum In"}}
}
func (f *fftSpectrumBlock) OutputPorts() []pipeline.Port { return nil }

func (f *fftSpectrumBlock) BufferConfig() pipeline.BufferConfig {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cfg
}

func (f *fftSpectrumBlock) SetBufferConfig(cfg pipeline.BufferConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cfg = cfg
	return nil
}

func (f *fftSpectrumBlock) Snapshot() []pipeline.DataChunk {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]pipeline.DataChunk, len(f.buf))
	copy(result, f.buf)
	return result
}

func (f *fftSpectrumBlock) addToBuffer(chunk pipeline.DataChunk) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.buf = append(f.buf, chunk)
	switch f.cfg.Mode {
	case pipeline.BufferModeSamples:
		if f.cfg.MaxSamples > 0 {
			for len(f.buf) > f.cfg.MaxSamples {
				f.buf = f.buf[1:]
			}
		}
	case pipeline.BufferModeDuration:
		if f.cfg.MaxDurationSec > 0 && len(f.buf) > 0 {
			cutoff := time.Now().UnixMilli() - int64(f.cfg.MaxDurationSec*1000)
			for len(f.buf) > 0 && f.buf[0].Timestamp < cutoff {
				f.buf = f.buf[1:]
			}
		}
	}
}

func (f *fftSpectrumBlock) Run(
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
			f.addToBuffer(chunk)
		}
	}
}
