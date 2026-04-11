package analysis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"byteflow-studio/internal/pipeline"
)

func init() {
	registerAnalysis("spectrum-viewer", func(id string) pipeline.Block {
		return &spectrumViewerBlock{
			id:       id,
			colorMap: "viridis",
			cfg:      pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 100},
		}
	})
}

var validColorMaps = map[string]bool{
	"viridis":   true,
	"magma":     true,
	"inferno":   true,
	"plasma":    true,
	"grayscale": true,
}

type spectrumViewerBlock struct {
	mu           sync.Mutex
	id           string
	cfg          pipeline.BufferConfig
	buf          []pipeline.DataChunk
	colorMap     string
	minFreqHz    float64
	maxFreqHz    float64
	minAmplitude float64
	maxAmplitude float64
}

func (s *spectrumViewerBlock) ID() string                       { return s.id }
func (s *spectrumViewerBlock) Type() string                     { return "spectrum-viewer" }
func (s *spectrumViewerBlock) Category() pipeline.BlockCategory { return pipeline.CategoryAnalysis }

func (s *spectrumViewerBlock) Configure(params map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if v, ok := params["colorMap"].(string); ok {
		if !validColorMaps[v] {
			return fmt.Errorf("invalid colorMap: %s (valid: viridis, magma, inferno, plasma, grayscale)", v)
		}
		s.colorMap = v
	}
	if v, ok := params["minFreqHz"]; ok {
		s.minFreqHz = toFloatA(v, 0)
	}
	if v, ok := params["maxFreqHz"]; ok {
		s.maxFreqHz = toFloatA(v, 0)
	}
	if v, ok := params["minAmplitude"]; ok {
		s.minAmplitude = toFloatA(v, 0)
	}
	if v, ok := params["maxAmplitude"]; ok {
		s.maxAmplitude = toFloatA(v, 0)
	}
	if mode, ok := params["bufferMode"].(string); ok {
		switch pipeline.BufferMode(mode) {
		case pipeline.BufferModeSamples:
			s.cfg.Mode = pipeline.BufferModeSamples
			if n, ok := params["bufferSamples"]; ok {
				s.cfg.MaxSamples = toIntA(n, 100)
			}
		case pipeline.BufferModeDuration:
			s.cfg.Mode = pipeline.BufferModeDuration
			if sec, ok := params["bufferDurationSec"]; ok {
				s.cfg.MaxDurationSec = toFloatA(sec, 10.0)
			}
		}
	}
	return nil
}

func (s *spectrumViewerBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (s *spectrumViewerBlock) OutputPorts() []pipeline.Port { return nil }

func (s *spectrumViewerBlock) BufferConfig() pipeline.BufferConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *spectrumViewerBlock) SetBufferConfig(cfg pipeline.BufferConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	return nil
}

func (s *spectrumViewerBlock) Snapshot() []pipeline.DataChunk {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]pipeline.DataChunk, len(s.buf))
	copy(result, s.buf)
	return result
}

func (s *spectrumViewerBlock) addToBuffer(chunk pipeline.DataChunk) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf = append(s.buf, chunk)
	switch s.cfg.Mode {
	case pipeline.BufferModeSamples:
		if s.cfg.MaxSamples > 0 {
			for len(s.buf) > s.cfg.MaxSamples {
				s.buf = s.buf[1:]
			}
		}
	case pipeline.BufferModeDuration:
		if s.cfg.MaxDurationSec > 0 && len(s.buf) > 0 {
			cutoff := time.Now().UnixMilli() - int64(s.cfg.MaxDurationSec*1000)
			for len(s.buf) > 0 && s.buf[0].Timestamp < cutoff {
				s.buf = s.buf[1:]
			}
		}
	}
}

func (s *spectrumViewerBlock) Run(
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
			s.addToBuffer(chunk)
		}
	}
}
