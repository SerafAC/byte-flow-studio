package processing

import (
	"context"

	"byteflow-studio/internal/pipeline"
)

func init() {
	Register("passthrough", func(id string) pipeline.Block {
		return &passthroughBlock{id: id}
	})
}

type passthroughBlock struct {
	id string
}

func (p *passthroughBlock) ID() string                  { return p.id }
func (p *passthroughBlock) Type() string                { return "passthrough" }
func (p *passthroughBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }

func (p *passthroughBlock) Configure(params map[string]any) error { return nil }

func (p *passthroughBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{
		{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"},
	}
}

func (p *passthroughBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{
		{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"},
	}
}

func (p *passthroughBlock) Run(
	ctx context.Context,
	inputs map[string]<-chan pipeline.DataChunk,
	outputs map[string]chan<- pipeline.DataChunk,
	errCh chan<- pipeline.BlockError,
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
			select {
			case out <- chunk:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
