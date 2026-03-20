# Contract: WorkflowService

**Type**: Wails v3 Go Service (exposed to Vue frontend via generated TypeScript bindings)
**File**: `services/workflow_service.go`
**Generated bindings**: `frontend/bindings/WorkflowService.*`

---

## Purpose

Manages the workflow definition: the collection of blocks, connections, and canvas positions. Handles file-level save/load operations and provides the block type catalogue.

---

## Methods

### `GetWorkflow() (Workflow, error)`

Returns the currently loaded workflow definition (all blocks, connections, session config).

**Returns**: `Workflow` struct | error
**Errors**: None expected for in-memory state; logs `ERROR` if state is inconsistent.

---

### `SaveWorkflow(path string) error`

Persists the current workflow definition and all retained sessions to a `.byteflow` SQLite file.

**Parameters**:
- `path` — absolute file system path (must end in `.byteflow`)

**Returns**: `nil` on success | descriptive error
**Side effects**: Updates `workflow.updatedAt`; upserts the `workflow` table row in SQLite
**Errors**:
- `"path must end in .byteflow"`
- `"failed to write workflow file: <os error>"`

**Logging**: `INFO save_workflow path=<path> blocks=<n> connections=<n>`

---

### `LoadWorkflow(path string) (Workflow, error)`

Opens a `.byteflow` file and loads the workflow definition into memory. Sessions are indexed (metadata loaded); raw data layer is NOT loaded at this point (lazy).

**Parameters**:
- `path` — absolute file system path

**Returns**: `Workflow` struct | error
**Errors**:
- `"file not found: <path>"`
- `"not a valid byteflow file: <reason>"`
- `"workflow schema version unsupported: <version>"` (for future-version files)
- If any Input block references unavailable hardware, block `status` is set to `error` with message `"hardware not available: <port>"` — the rest of the workflow loads normally (degraded-state open, FR-023)

**Logging**: `INFO load_workflow path=<path>`; `WARN block_hardware_unavailable block_id=<id> port=<port>` for each degraded block

---

### `AddBlock(blockType string, x float64, y float64) (Block, error)`

Creates a new block of the given type and places it at the given canvas coordinates.

**Parameters**:
- `blockType` — registered block type string (e.g., `"uart"`, `"line-chart"`)
- `x`, `y` — canvas position

**Returns**: Fully initialised `Block` with generated ID and default params | error
**Errors**:
- `"unknown block type: <blockType>"`
- `"cannot add blocks while flow is Running"` (canvas editing locked during active run)

---

### `RemoveBlock(blockId string) error`

Removes a block and all connections attached to it.

**Parameters**: `blockId` — UUID of the block to remove

**Errors**: `"block not found: <blockId>"`

---

### `AddConnection(fromBlockId, fromPortId, toBlockId, toPortId string) (Connection, error)`

Creates a connection between two ports.

**Returns**: `Connection` with generated ID | error
**Errors**:
- `"source/target block/port not found"`
- `"incompatible port types: <type1> → <type2>"`
- `"connection would create a cycle"`
- `"input port already has a connection"` (each input port accepts only one incoming connection)

---

### `RemoveConnection(connectionId string) error`

Removes a connection by ID.

**Errors**: `"connection not found: <connectionId>"`

---

### `UpdateBlockParams(blockId string, params map[string]any) error`

Updates configuration parameters for a block. Validates the params against the block type's schema.

**Parameters**:
- `params` — partial map of params to update (unspecified keys remain unchanged)

**Errors**: `"block not found"` | `"invalid param <key>: <reason>"`

---

### `UpdateBlockPosition(blockId string, x float64, y float64) error`

Updates canvas position for a block (called on drag-end in Vue Flow).

---

### `GetAvailableBlockTypes() []BlockTypeDescriptor`

Returns the full catalogue of registered block types, grouped by category.

**Returns**:
```go
type BlockTypeDescriptor struct {
    Type        string         // e.g., "uart"
    Category    BlockCategory  // input | processing | analysis
    Label       string         // "UART/Serial"
    Description string         // One-line user-facing description
    DefaultParams map[string]any
}
```

---

### `ListSerialPorts() ([]SerialPortInfo, error)`

Enumerates available serial ports on the current machine (used to populate the UART block's port dropdown).

**Returns**:
```go
type SerialPortInfo struct {
    Name        string  // e.g., "COM3", "/dev/ttyUSB0"
    VendorID    string
    ProductID   string
    Description string
}
```

---

## Events Emitted

None. WorkflowService is control-plane only; data flows via PipelineService events.

---

## Logging Contract

All methods use `log/slog` with the following structured fields where applicable:

| Field | Type | Description |
|---|---|---|
| `block_id` | string | Block being operated on |
| `block_type` | string | Type string of the block |
| `connection_id` | string | Connection being operated on |
| `path` | string | File system path |
| `error` | error | Error value on failure |
