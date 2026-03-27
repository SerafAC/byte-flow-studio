package processing

import (
	"context"
	"sync"
	"time"

	"byteflow-studio/internal/pipeline"
)

func init() {
	Register("sampler", func(id string) pipeline.Block {
		return &samplerBlock{id: id, mode: "every-n-samples", n: 10, intervalMs: 100}
	})
}

type samplerBlock struct {
	mu         sync.Mutex
	id         string
	mode       string
	n          int
	intervalMs float64

	// state
	count    int
	lastEmit int64
	fired    bool
}

func (s *samplerBlock) ID() string                       { return s.id }
func (s *samplerBlock) Type() string                     { return "sampler" }
func (s *samplerBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }

func (s *samplerBlock) Configure(params map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := params["mode"]; ok {
		if m, ok := v.(string); ok {
			s.mode = m
		}
	}
	if v, ok := params["n"]; ok {
		n := toIntP(v, 10)
		if n < 1 {
			n = 1
		}
		s.n = n
	}
	if v, ok := params["intervalMs"]; ok {
		ms := toFloat(v, 100)
		if ms < 1 {
			ms = 1
		}
		s.intervalMs = ms
	}
	return nil
}

func (s *samplerBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}

func (s *samplerBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"}}
}

func (s *samplerBlock) Run(
	ctx context.Context,
	inputs map[string]<-chan pipeline.DataChunk,
	outputs map[string]chan<- pipeline.DataChunk,
	_ chan<- pipeline.BlockError,
) error {
	in, ok := inputs["in"]
	if !ok {
		return nil
	}
	out, ok := outputs["out"]
	if !ok {
		return nil
	}

	s.mu.Lock()
	mode := s.mode
	s.mu.Unlock()

	if mode == "last-in-window" {
		return s.runLastInWindow(ctx, in, out)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case chunk, open := <-in:
			if !open {
				return nil
			}
			if len(chunk.Values) == 0 {
				continue
			}
			if s.shouldEmit(chunk.Timestamp) {
				out <- pipeline.DataChunk{
					Timestamp: chunk.Timestamp,
					SourceID:  chunk.SourceID,
					Values:    chunk.Values,
				}
			}
		}
	}
}

func (s *samplerBlock) shouldEmit(ts int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.mode {
	case "every-n-samples":
		s.count++
		if s.count >= s.n {
			s.count = 0
			return true
		}
		return false
	case "first-in-window":
		if ts-s.lastEmit >= int64(s.intervalMs) {
			s.lastEmit = ts
			return true
		}
		return false
	case "first":
		if !s.fired {
			s.fired = true
			return true
		}
		return false
	}
	return false
}

func (s *samplerBlock) runLastInWindow(ctx context.Context, in <-chan pipeline.DataChunk, out chan<- pipeline.DataChunk) error {
	s.mu.Lock()
	interval := time.Duration(s.intervalMs) * time.Millisecond
	s.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var last *pipeline.DataChunk

	for {
		select {
		case <-ctx.Done():
			return nil
		case chunk, open := <-in:
			if !open {
				return nil
			}
			if len(chunk.Values) == 0 {
				continue
			}
			c := chunk
			s.mu.Lock()
			last = &c
			s.mu.Unlock()
		case <-ticker.C:
			s.mu.Lock()
			toSend := last
			last = nil
			s.mu.Unlock()
			if toSend != nil {
				select {
				case out <- *toSend:
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}
