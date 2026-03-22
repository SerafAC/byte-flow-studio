// Package pipeline provides the core pipeline types, interfaces, and execution engine.
package pipeline

import "context"

// BlockCategory identifies the category of a pipeline block.
type BlockCategory string

const (
	CategoryInput      BlockCategory = "input"
	CategoryProcessing BlockCategory = "processing"
	CategoryAnalysis   BlockCategory = "analysis"
)

// PortDir identifies the direction of a port.
type PortDir string

const (
	PortDirInput  PortDir = "input"
	PortDirOutput PortDir = "output"
)

// DataType identifies the type of data flowing through a port.
type DataType string

const (
	DataTypeRaw DataType = "raw-bytes"
	DataTypeNum DataType = "numeric"
)

// Port is a typed connection point on a block.
type Port struct {
	ID        string   // e.g., "in", "out"
	Direction PortDir  // "input" | "output"
	DataType  DataType // "raw-bytes" | "numeric"
	Label     string   // Display label
}

// DataChunk is the atomic unit flowing through pipeline channels.
type DataChunk struct {
	Timestamp int64     // Unix ms — assigned by Input block, never modified downstream
	SourceID  string    // ID of the originating Input block
	Raw       []byte    // Non-nil for raw-bytes streams
	Values    []float64 // Non-nil for numeric streams (single value: len==1)
}

// BlockError carries a runtime error from a specific block.
type BlockError struct {
	BlockID string
	Err     error
}

// BufferMode controls how an analysis block manages its ring buffer.
type BufferMode string

const (
	BufferModeSamples  BufferMode = "samples"
	BufferModeDuration BufferMode = "duration"
)

// BufferConfig configures the ring buffer of an analysis block.
type BufferConfig struct {
	Mode           BufferMode // "samples" | "duration"
	MaxSamples     int        // Used when Mode == "samples"
	MaxDurationSec float64    // Used when Mode == "duration"
}

// Block is the common interface implemented by all pipeline blocks.
type Block interface {
	// ID returns the unique identifier of this block instance.
	ID() string

	// Type returns the registered type string (e.g., "uart", "moving-average").
	Type() string

	// Category returns the block's category.
	Category() BlockCategory

	// Configure applies a map of parameters to this block.
	Configure(params map[string]any) error

	// InputPorts returns the set of input ports this block exposes.
	InputPorts() []Port

	// OutputPorts returns the set of output ports this block exposes.
	OutputPorts() []Port

	// Run starts the block's processing loop.
	// The block MUST return when ctx is cancelled (clean shutdown).
	// The block MUST NOT close the output channels (the engine owns channels).
	// Errors encountered during execution are reported via errCh.
	Run(ctx context.Context, inputs map[string]<-chan DataChunk, outputs map[string]chan<- DataChunk, errCh chan<- BlockError) error
}

// AnalysisBlock extends Block with buffer management and snapshot capabilities.
type AnalysisBlock interface {
	Block

	// BufferConfig returns the current buffer configuration.
	BufferConfig() BufferConfig

	// SetBufferConfig updates the buffer configuration.
	SetBufferConfig(cfg BufferConfig) error

	// Snapshot returns all currently buffered data points.
	Snapshot() []DataChunk
}

// FlowState represents the current state of the pipeline.
type FlowState string

const (
	FlowStateIdle    FlowState = "idle"
	FlowStateRunning FlowState = "running"
	FlowStatePaused  FlowState = "paused"
	FlowStateError   FlowState = "error"
)
