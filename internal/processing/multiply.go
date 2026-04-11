package processing

import (
	"context"
	"fmt"
	"sync"

	"byteflow-studio/internal/pipeline"
)

func init() {
	Register("multiply", func(id string) pipeline.Block {
		return &multiplyBlock{id: id, portCount: 2}
	})
}

type multiplyBlock struct {
	mu        sync.Mutex
	id        string
	portCount int
	pending   map[string]float64 // latest value per input port
	ready     map[string]bool    // tracks which ports have received at least one value
}

func (m *multiplyBlock) ID() string                       { return m.id }
func (m *multiplyBlock) Type() string                     { return "multiply" }
func (m *multiplyBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }

func (m *multiplyBlock) Configure(params map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if v, ok := params["inputCount"]; ok {
		n := toIntP(v, 2)
		if n < 2 || n > 8 {
			return fmt.Errorf("inputCount must be 2-8, got %d", n)
		}
		m.portCount = n
	}
	// Reset pending state
	m.pending = nil
	m.ready = nil
	return nil
}

func (m *multiplyBlock) InputPorts() []pipeline.Port {
	m.mu.Lock()
	defer m.mu.Unlock()
	ports := make([]pipeline.Port, m.portCount)
	for i := 0; i < m.portCount; i++ {
		ports[i] = pipeline.Port{
			ID:        fmt.Sprintf("in-%d", i),
			Direction: pipeline.PortDirInput,
			DataType:  pipeline.DataTypeNum,
			Label:     fmt.Sprintf("In %d", i),
		}
	}
	return ports
}

func (m *multiplyBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"}}
}

func (m *multiplyBlock) Run(
	ctx context.Context,
	inputs map[string]<-chan pipeline.DataChunk,
	outputs map[string]chan<- pipeline.DataChunk,
	_ chan<- pipeline.BlockError,
) error {
	out, ok := outputs["out"]
	if !ok {
		return nil
	}

	m.mu.Lock()
	m.pending = make(map[string]float64)
	m.ready = make(map[string]bool)
	pc := m.portCount
	m.mu.Unlock()

	// Fan-in: merge all input channels into one tagged channel
	type taggedChunk struct {
		portID string
		chunk  pipeline.DataChunk
	}
	merged := make(chan taggedChunk, 64)
	var wg sync.WaitGroup

	for i := 0; i < pc; i++ {
		portID := fmt.Sprintf("in-%d", i)
		ch, exists := inputs[portID]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(pid string, c <-chan pipeline.DataChunk) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case chunk, open := <-c:
					if !open {
						return
					}
					select {
					case merged <- taggedChunk{portID: pid, chunk: chunk}:
					case <-ctx.Done():
						return
					}
				}
			}
		}(portID, ch)
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case tc, ok := <-merged:
			if !ok {
				return nil
			}
			if len(tc.chunk.Values) == 0 {
				continue
			}

			m.mu.Lock()
			m.pending[tc.portID] = tc.chunk.Values[0]
			m.ready[tc.portID] = true

			// Check if all ports have received at least one value
			allReady := len(m.ready) >= pc
			var product float64
			if allReady {
				product = 1.0
				for i := 0; i < pc; i++ {
					pid := fmt.Sprintf("in-%d", i)
					product *= m.pending[pid]
				}
			}
			m.mu.Unlock()

			if allReady {
				out <- pipeline.DataChunk{
					Timestamp: tc.chunk.Timestamp,
					SourceID:  tc.chunk.SourceID,
					Values:    []float64{product},
				}
			}
		}
	}
}
