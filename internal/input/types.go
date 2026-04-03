// Package input provides input block implementations for the pipeline.
package input

import (
	"byteflow-studio/internal/logging"
	"byteflow-studio/internal/pipeline"
)

// pkgLog is the package-level logger set via SetLogger.
var pkgLog *logging.Logger

// SetLogger sets the package-level logger for all input blocks.
func SetLogger(l *logging.Logger) {
	pkgLog = l
}

// InputConfig is a placeholder for shared input configuration.
type InputConfig struct{}

// DataChunk is re-exported from the pipeline package for use by input blocks.
type DataChunk = pipeline.DataChunk
