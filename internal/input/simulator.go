package input

import (
	"context"
	"math"
	"math/rand"
	"time"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"
)

func init() {
	processing.Register("simulator", func(id string) pipeline.Block {
		return &simulatorBlock{id: id, waveform: "sine", frequencyHz: 1.0, amplitude: 1.0, sampleRateHz: 100.0}
	})
}

type simulatorBlock struct {
	id           string
	waveform     string
	frequencyHz  float64
	amplitude    float64
	offset       float64
	sampleRateHz float64
}

func (s *simulatorBlock) ID() string                    { return s.id }
func (s *simulatorBlock) Type() string                  { return "simulator" }
func (s *simulatorBlock) Category() pipeline.BlockCategory { return pipeline.CategoryInput }

func (s *simulatorBlock) Configure(params map[string]any) error {
	if v, ok := params["waveform"].(string); ok {
		s.waveform = v
	}
	if v, ok := params["frequencyHz"]; ok {
		s.frequencyHz = toFloat(v, 1.0)
	}
	if v, ok := params["amplitude"]; ok {
		s.amplitude = toFloat(v, 1.0)
	}
	if v, ok := params["offset"]; ok {
		s.offset = toFloat(v, 0.0)
	}
	if v, ok := params["sampleRateHz"]; ok {
		s.sampleRateHz = toFloat(v, 100.0)
	}
	return nil
}

func (s *simulatorBlock) InputPorts() []pipeline.Port  { return nil }
func (s *simulatorBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{
		{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeNum, Label: "Out"},
	}
}

func (s *simulatorBlock) Run(
	ctx context.Context,
	_ map[string]<-chan pipeline.DataChunk,
	outputs map[string]chan<- pipeline.DataChunk,
	_ chan<- pipeline.BlockError,
) error {
	out, ok := outputs["out"]
	if !ok {
		return nil
	}

	interval := time.Duration(float64(time.Second) / s.sampleRateHz)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var phase float64
	phaseStep := 2 * math.Pi * s.frequencyHz / s.sampleRateHz

	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			v := s.sample(phase)
			phase += phaseStep
			chunk := pipeline.DataChunk{
				Timestamp: t.UnixMilli(),
				SourceID:  s.id,
				Values:    []float64{v},
			}
			select {
			case out <- chunk:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (s *simulatorBlock) sample(phase float64) float64 {
	var v float64
	switch s.waveform {
	case "square":
		if math.Sin(phase) >= 0 {
			v = 1.0
		} else {
			v = -1.0
		}
	case "sawtooth":
		v = (phase/(2*math.Pi) - math.Floor(phase/(2*math.Pi)))*2 - 1
	case "noise":
		v = rand.Float64()*2 - 1
	default: // sine
		v = math.Sin(phase)
	}
	return v*s.amplitude + s.offset
}
