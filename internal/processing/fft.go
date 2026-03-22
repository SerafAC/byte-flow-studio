package processing

import (
	"context"
	"math"
	"math/cmplx"
	"sync"

	"byteflow-studio/internal/pipeline"
	"gonum.org/v1/gonum/dsp/fourier"
)

func init() {
	Register("fft", func(id string) pipeline.Block {
		return &fftBlock{id: id, windowSize: 512, windowFunc: "hann"}
	})
}

type fftBlock struct {
	mu         sync.Mutex
	id         string
	windowSize int
	windowFunc string
	buf        []float64
}

func (f *fftBlock) ID() string                       { return f.id }
func (f *fftBlock) Type() string                     { return "fft" }
func (f *fftBlock) Category() pipeline.BlockCategory { return pipeline.CategoryProcessing }

func (f *fftBlock) Configure(params map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := params["windowSize"]; ok {
		n := toIntP(v, 512)
		if n < 2 {
			n = 2
		}
		f.windowSize = n
	}
	if v, ok := params["windowFunction"].(string); ok {
		f.windowFunc = v
	}
	return nil
}

func (f *fftBlock) InputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "in", Direction: pipeline.PortDirInput, DataType: pipeline.DataTypeNum, Label: "In"}}
}
func (f *fftBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Magnitudes"}}
}

func (f *fftBlock) Run(
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

			f.mu.Lock()
			f.buf = append(f.buf, chunk.Values[0])
			ws := f.windowSize
			wf := f.windowFunc

			if len(f.buf) < ws {
				f.mu.Unlock()
				continue
			}

			frame := make([]float64, ws)
			copy(frame, f.buf[len(f.buf)-ws:])
			f.mu.Unlock()

			applyWindow(frame, wf)

			fft := fourier.NewFFT(ws)
			coeff := fft.Coefficients(nil, frame)

			// Output magnitudes for positive frequencies only (N/2 bins)
			halfN := ws / 2
			mags := make([]float64, halfN)
			for i := 0; i < halfN; i++ {
				mags[i] = cmplx.Abs(coeff[i])
			}

			out <- pipeline.DataChunk{
				Timestamp: chunk.Timestamp,
				SourceID:  chunk.SourceID,
				Values:    mags,
			}
		}
	}
}

func applyWindow(frame []float64, name string) {
	n := len(frame)
	switch name {
	case "hann":
		for i := range frame {
			w := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
			frame[i] *= w
		}
	case "hamming":
		for i := range frame {
			w := 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(n-1))
			frame[i] *= w
		}
	// "none" or unknown: no windowing
	}
}
