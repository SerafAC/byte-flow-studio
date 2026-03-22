package processing

import (
	"context"
	"sync"

	"byteflow-studio/internal/pipeline"
)

func init() {
	Register("scaling", func(id string) pipeline.Block {
		return &scalingBlock{id: id, scale: 1.0}
	})
}

type scalingBlock struct {
	mu     sync.RWMutex
	id     string
	scale  float64
	offset float64
}

func (s *scalingBlock) ID() string                       { return s.id }
func (s *scalingBlock) Type() string                     { return "scaling" }
func (s *scalingBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }

func (s *scalingBlock) Configure(params map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := params["scale"]; ok {
		s.scale = toFloat(v, 1.0)
	}
	if v, ok := params["offset"]; ok {
		s.offset = toFloat(v, 0.0)
	}
	return nil
}

func (s *scalingBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (s *scalingBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"}}
}

func (s *scalingBlock) Run(
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
			s.mu.RLock()
			scale := s.scale
			offset := s.offset
			s.mu.RUnlock()

			out <- pipeline.DataChunk{
				Timestamp: chunk.Timestamp,
				SourceID:  chunk.SourceID,
				Values:    []float64{chunk.Values[0]*scale + offset},
			}
		}
	}
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
