package processing

import (
	"context"
	"encoding/binary"
	"math"
	"sync"

	"byteflow-studio/internal/pipeline"
)

func init() {
	Register("byte-parser", func(id string) pipeline.Block {
		return &byteParserBlock{id: id, format: "float32-le", channels: 1, frameSize: 4}
	})
}

type byteParserBlock struct {
	mu        sync.RWMutex
	id        string
	format    string
	channels  int
	frameSize int
	buf       []byte
}

func (b *byteParserBlock) ID() string                       { return b.id }
func (b *byteParserBlock) Type() string                     { return "byte-parser" }
func (b *byteParserBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }

func (b *byteParserBlock) Configure(params map[string]any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if v, ok := params["format"].(string); ok {
		b.format = v
	}
	if v, ok := params["channels"]; ok {
		b.channels = toIntP(v, 1)
	}
	if v, ok := params["frameSize"]; ok {
		b.frameSize = toIntP(v, 4)
	}
	return nil
}

func (b *byteParserBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeRaw, Label: "In"}}
}
func (b *byteParserBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"}}
}

func (b *byteParserBlock) Run(
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
			if len(chunk.Raw) == 0 {
				continue
			}

			b.mu.Lock()
			b.buf = append(b.buf, chunk.Raw...)
			frameSize := b.frameSize
			channels := b.channels
			format := b.format

			for len(b.buf) >= frameSize {
				frame := b.buf[:frameSize]
				b.buf = b.buf[frameSize:]

				values := parseFrame(frame, format, channels)
				b.mu.Unlock()

				if values != nil {
					out <- pipeline.DataChunk{
						Timestamp: chunk.Timestamp,
						SourceID:  chunk.SourceID,
						Values:    values,
					}
				}
				b.mu.Lock()
			}
			b.mu.Unlock()
		}
	}
}

func parseFrame(frame []byte, format string, channels int) []float64 {
	values := make([]float64, channels)
	bytesPerSample := len(frame) / channels

	for ch := 0; ch < channels; ch++ {
		start := ch * bytesPerSample
		end := start + bytesPerSample
		if end > len(frame) {
			return nil
		}
		sample := frame[start:end]

		switch format {
		case "float32-le":
			if len(sample) < 4 {
				return nil
			}
			bits := binary.LittleEndian.Uint32(sample)
			values[ch] = float64(math.Float32frombits(bits))
		case "float32-be":
			if len(sample) < 4 {
				return nil
			}
			bits := binary.BigEndian.Uint32(sample)
			values[ch] = float64(math.Float32frombits(bits))
		case "int16-le":
			if len(sample) < 2 {
				return nil
			}
			values[ch] = float64(int16(binary.LittleEndian.Uint16(sample)))
		case "int16-be":
			if len(sample) < 2 {
				return nil
			}
			values[ch] = float64(int16(binary.BigEndian.Uint16(sample)))
		case "uint8":
			if len(sample) < 1 {
				return nil
			}
			values[ch] = float64(sample[0])
		default:
			return nil
		}
	}
	return values
}
