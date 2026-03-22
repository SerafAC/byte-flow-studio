package processing

import (
	"context"
	"sync"

	"byteflow-studio/internal/pipeline"
)

func init() {
	Register("moving-average", func(id string) pipeline.Block {
		return &movingAverageBlock{id: id, windowSize: 10}
	})
}

type movingAverageBlock struct {
	mu         sync.RWMutex
	id         string
	windowSize int
	buf        []float64
}

func (m *movingAverageBlock) ID() string                    { return m.id }
func (m *movingAverageBlock) Type() string                  { return "moving-average" }
func (m *movingAverageBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }

func (m *movingAverageBlock) Configure(params map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := params["windowSize"]; ok {
		n := toIntP(v, 10)
		if n < 1 {
			n = 1
		}
		m.windowSize = n
		// Trim existing buffer to new window size
		if len(m.buf) > m.windowSize {
			m.buf = m.buf[len(m.buf)-m.windowSize:]
		}
	}
	return nil
}

func (m *movingAverageBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (m *movingAverageBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"}}
}

func (m *movingAverageBlock) Run(
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
			v := chunk.Values[0]

			m.mu.Lock()
			m.buf = append(m.buf, v)
			ws := m.windowSize
			if len(m.buf) > ws {
				m.buf = m.buf[len(m.buf)-ws:]
			}
			sum := 0.0
			for _, x := range m.buf {
				sum += x
			}
			avg := sum / float64(len(m.buf))
			m.mu.Unlock()

			out <- pipeline.DataChunk{
				Timestamp: chunk.Timestamp,
				SourceID:  chunk.SourceID,
				Values:    []float64{avg},
			}
		}
	}
}

func toIntP(v any, def int) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	}
	return def
}
