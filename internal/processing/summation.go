package processing

import (
	"context"
	"sync"

	"byteflow-studio/internal/pipeline"
)

func init() {
	Register("summation", func(id string) pipeline.Block {
		return &summationBlock{id: id}
	})
}

type summationBlock struct {
	mu  sync.Mutex
	id  string
	sum float64
}

func (s *summationBlock) ID() string                       { return s.id }
func (s *summationBlock) Type() string                     { return "summation" }
func (s *summationBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }

func (s *summationBlock) Configure(_ map[string]any) error { return nil }

func (s *summationBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (s *summationBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"}}
}

func (s *summationBlock) Run(
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
			s.mu.Lock()
			s.sum += chunk.Values[0]
			sum := s.sum
			s.mu.Unlock()

			out <- pipeline.DataChunk{
				Timestamp: chunk.Timestamp,
				SourceID:  chunk.SourceID,
				Values:    []float64{sum},
			}
		}
	}
}
