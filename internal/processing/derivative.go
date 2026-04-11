package processing

import (
	"context"

	"byteflow-studio/internal/pipeline"
)

func init() {
	Register("derivative", func(id string) pipeline.Block {
		return &derivativeBlock{id: id}
	})
}

type derivativeBlock struct {
	id            string
	prevValue     float64
	prevTimestamp  int64
	hasPrev       bool
}

func (d *derivativeBlock) ID() string                       { return d.id }
func (d *derivativeBlock) Type() string                     { return "derivative" }
func (d *derivativeBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }
func (d *derivativeBlock) Configure(_ map[string]any) error { return nil }

func (d *derivativeBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (d *derivativeBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"}}
}

func (d *derivativeBlock) Run(
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

			val := chunk.Values[0]
			var deriv float64

			if d.hasPrev {
				dt := float64(chunk.Timestamp - d.prevTimestamp)
				if dt > 0 {
					deriv = (val - d.prevValue) / dt * 1000 // per second
				}
			}

			d.prevValue = val
			d.prevTimestamp = chunk.Timestamp
			d.hasPrev = true

			out <- pipeline.DataChunk{
				Timestamp: chunk.Timestamp,
				SourceID:  chunk.SourceID,
				Values:    []float64{deriv},
			}
		}
	}
}
