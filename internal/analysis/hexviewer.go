package analysis

import (
	"bytes"
	"context"
	"sync"

	"byteflow-studio/internal/pipeline"
)

func init() {
	registerAnalysis("hex-viewer", func(id string) pipeline.Block {
		return &hexViewerBlock{
			id:          id,
			maxBytes:    65536,
			bytesPerRow: 16,
			cfg:         pipeline.BufferConfig{Mode: pipeline.BufferModeSamples, MaxSamples: 1000},
		}
	})
}

type hexViewerBlock struct {
	mu          sync.Mutex
	id          string
	cfg         pipeline.BufferConfig
	buf         []pipeline.DataChunk
	totalBytes  int
	maxBytes    int
	bytesPerRow int
}

func (h *hexViewerBlock) ID() string                       { return h.id }
func (h *hexViewerBlock) Type() string                     { return "hex-viewer" }
func (h *hexViewerBlock) Category() pipeline.BlockCategory { return pipeline.CategoryAnalysis }

func (h *hexViewerBlock) Configure(params map[string]any) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if v, ok := params["maxBytes"]; ok {
		n := toIntA(v, 65536)
		if n < 1024 {
			n = 1024
		}
		if n > 1048576 {
			n = 1048576
		}
		h.maxBytes = n
	}
	if v, ok := params["bytesPerRow"]; ok {
		n := toIntA(v, 16)
		switch n {
		case 8, 16, 32:
			h.bytesPerRow = n
		default:
			h.bytesPerRow = 16
		}
	}
	return nil
}

func (h *hexViewerBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeRaw, Label: "In"}}
}
func (h *hexViewerBlock) OutputPorts() []pipeline.Port { return nil }

func (h *hexViewerBlock) BufferConfig() pipeline.BufferConfig {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cfg
}

func (h *hexViewerBlock) SetBufferConfig(cfg pipeline.BufferConfig) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg = cfg
	return nil
}

func (h *hexViewerBlock) Snapshot() []pipeline.DataChunk {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]pipeline.DataChunk, len(h.buf))
	copy(result, h.buf)
	return result
}

func (h *hexViewerBlock) addToBuffer(chunk pipeline.DataChunk) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buf = append(h.buf, chunk)
	h.totalBytes += len(chunk.Raw)

	// Enforce maxBytes: drop oldest chunks
	for h.totalBytes > h.maxBytes && len(h.buf) > 0 {
		h.totalBytes -= len(h.buf[0].Raw)
		h.buf = h.buf[1:]
	}
}

// SearchPattern searches for a byte pattern in the buffered data.
// Returns a slice of byte offsets where the pattern was found.
func (h *hexViewerBlock) SearchPattern(pattern []byte) []int {
	if len(pattern) == 0 {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	// Concatenate all buffered raw bytes
	var allBytes []byte
	for _, chunk := range h.buf {
		allBytes = append(allBytes, chunk.Raw...)
	}

	var matches []int
	offset := 0
	for {
		idx := bytes.Index(allBytes[offset:], pattern)
		if idx < 0 {
			break
		}
		matches = append(matches, offset+idx)
		offset += idx + 1
	}
	return matches
}

func (h *hexViewerBlock) Run(
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
			if len(chunk.Raw) > 0 {
				h.addToBuffer(chunk)
			}
		}
	}
}
