# Contract: Block Interface (Go Internal)

**Type**: Go interface — internal to `internal/pipeline/`
**Not exposed to frontend directly** (used only by the pipeline engine)

---

## Purpose

Defines the contract that every block implementation (Input, Processing, Analysis) must satisfy to be runnable by the pipeline engine. This enables the engine to treat all block types uniformly.

---

## Go Interface Definition

```go
// Block is the common interface implemented by all pipeline blocks.
// Implementations live in internal/input/, internal/processing/, and are
// referenced by the pipeline engine via the block registry.
type Block interface {
    // ID returns the unique identifier of this block instance.
    ID() string

    // Type returns the registered type string (e.g., "uart", "moving-average").
    Type() string

    // Category returns the block's category.
    Category() BlockCategory // BlockCategory: "input" | "processing" | "analysis"

    // Configure applies a map of parameters to this block.
    // Called once before Start and may be called again while paused.
    // Returns an error if any parameter is invalid.
    Configure(params map[string]any) error

    // InputPorts returns the set of input ports this block exposes.
    // Input blocks return an empty slice.
    InputPorts() []Port

    // OutputPorts returns the set of output ports this block exposes.
    // Analysis blocks return an empty slice.
    OutputPorts() []Port

    // Run starts the block's processing loop.
    // Inputs is a map of portId → receive channel.
    // Outputs is a map of portId → send channel.
    // The block MUST return when ctx is cancelled (clean shutdown).
    // The block MUST NOT close the output channels (the engine owns channels).
    // Errors encountered during execution are reported via errCh.
    Run(ctx context.Context, inputs map[string]<-chan DataChunk, outputs map[string]chan<- DataChunk, errCh chan<- BlockError) error
}
```

---

## Supporting Types

```go
type BlockCategory string

const (
    CategoryInput      BlockCategory = "input"
    CategoryProcessing BlockCategory = "processing"
    CategoryAnalysis   BlockCategory = "analysis"
)

type Port struct {
    ID        string    // e.g., "in", "out", "out-raw", "out-parsed"
    Direction PortDir   // "input" | "output"
    DataType  DataType  // "raw-bytes" | "numeric"
    Label     string    // Display label
}

type PortDir  string
type DataType string

const (
    PortDirInput  PortDir  = "input"
    PortDirOutput PortDir  = "output"
    DataTypeRaw   DataType = "raw-bytes"
    DataTypeNum   DataType = "numeric"
)

// DataChunk is the atomic unit flowing through channels.
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
```

---

## Block Registry

The block registry maps type strings to factory functions, allowing the pipeline engine and the workflow loader to construct blocks by type name.

```go
// BlockFactory creates a new Block instance with a given ID.
type BlockFactory func(id string) Block

// Registry holds all registered block factories.
// Registration happens in init() functions in each block's package.
var Registry = map[string]BlockFactory{}

// Register adds a factory to the registry. Called from package init().
func Register(blockType string, factory BlockFactory) {
    Registry[blockType] = factory
}
```

---

## Implementation Responsibilities

Each block implementation MUST:

1. **Return promptly** when `ctx.Done()` is closed — no lingering goroutines.
2. **Never panic** — all errors MUST be sent to `errCh` and cause the method to return.
3. **Propagate timestamps unchanged** — a Processing block receiving a DataChunk MUST preserve the original `Timestamp` and `SourceID` unless it explicitly transforms the time axis (e.g., FFT outputs a new frequency-domain value with a synthetic timestamp representing the window's midpoint).
4. **Be idempotent on Configure** — calling `Configure` with the same params twice MUST produce the same result without side effects.
5. **Be independently testable** — each block package MUST have unit tests that create a block directly, call `Run` with synthetic input channels, and assert output values, without needing the pipeline engine.

---

## Analysis Block Extension

Analysis blocks additionally implement the `AnalysisBlock` interface (extends `Block`):

```go
type AnalysisBlock interface {
    Block

    // BufferConfig returns the current buffer configuration.
    BufferConfig() BufferConfig

    // SetBufferConfig updates the buffer configuration.
    SetBufferConfig(cfg BufferConfig) error

    // Snapshot returns all currently buffered data points.
    // Used when switching to historical session view.
    Snapshot() []DataChunk
}

type BufferConfig struct {
    Mode           BufferMode // "samples" | "duration"
    MaxSamples     int        // Used when Mode == "samples"
    MaxDurationSec float64    // Used when Mode == "duration"
}

type BufferMode string
const (
    BufferModeSamples  BufferMode = "samples"
    BufferModeDuration BufferMode = "duration"
)
```
