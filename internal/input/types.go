// Package input provides input block implementations for the pipeline.
package input

import "byteflow-studio/internal/pipeline"

// InputConfig is a placeholder for shared input configuration.
type InputConfig struct{}

// DataChunk is re-exported from the pipeline package for use by input blocks.
type DataChunk = pipeline.DataChunk
