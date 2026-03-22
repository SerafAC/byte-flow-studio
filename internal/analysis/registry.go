package analysis

import (
	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"
)

// AnalysisFactory creates a new pipeline.Block (AnalysisBlock) with a given ID.
type AnalysisFactory func(id string) pipeline.Block

// analysisRegistry holds analysis block factories (internal to the analysis package).
var analysisRegistry = map[string]AnalysisFactory{}

// registerAnalysis adds an analysis block factory to both the local registry
// and the shared processing.Registry so the engine can find it.
func registerAnalysis(blockType string, factory AnalysisFactory) {
	analysisRegistry[blockType] = factory
	processing.Register(blockType, func(id string) pipeline.Block {
		return factory(id)
	})
}
