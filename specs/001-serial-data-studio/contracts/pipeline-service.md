# Contract: PipelineService

**Type**: Wails v3 Go Service (exposed to Vue frontend via generated TypeScript bindings)
**File**: `services/pipeline_service.go`
**Generated bindings**: `frontend/bindings/PipelineService.*`

---

## Purpose

Controls the runtime lifecycle of the data pipeline. Validates the workflow DAG, starts/stops/pauses the goroutine-per-block execution engine, and emits real-time data events to the frontend.

---

## Methods

### `Start() error`

Validates the current workflow DAG and starts the pipeline.

**Preconditions**: Flow must be in `Idle`/`Stopped` or `Error` state.

**Validation (in order)**:
1. At least one Input block is present and connected (directly or via Processing blocks) to at least one Analysis block.
2. No cycles exist in the DAG (topological sort).
3. All connections have compatible port types.
4. Each Input block's configuration is valid (e.g., non-empty port for UART).

**On success**: Creates a new Session; transitions state to `Running`; starts goroutines; emits `pipeline:state-changed`.

**Returns**: `nil` | descriptive validation error (human-readable, no stack trace)
**Errors**:
- `"no valid input-to-analysis path found"`
- `"cycle detected in connection graph"`
- `"block <id> (<type>): invalid configuration: <reason>"`
- `"flow is already running"`

**Logging**: `INFO pipeline_start blocks=<n> connections=<n> session_id=<id>`

---

### `Pause() error`

Suspends data ingestion and processing while preserving current display state.

**Preconditions**: Flow must be `Running`.

**On success**: Sends pause signal to all goroutines via context cancellation with pause flag; transitions state to `Paused`; emits `pipeline:state-changed`. Display state (last values in Analysis blocks) is frozen.

**Errors**: `"flow is not running"`

---

### `Resume() error`

Resumes data flow after a pause.

**Preconditions**: Flow must be `Paused`.

**On success**: Restarts goroutines; transitions state to `Running`; emits `pipeline:state-changed`.

**Errors**: `"flow is not paused"`

---

### `Stop() error`

Stops the pipeline and completes the current Session.

**Preconditions**: Flow must be `Running`, `Paused`, or `Error`.

**On success**: Cancels all goroutines; waits for clean shutdown; marks Session as complete with `end_reason: user-stop`; trims sessions beyond `maxSessions` limit; transitions state to `Idle`; emits `pipeline:state-changed`.

**Errors**: `"flow is not active"`

**Logging**: `INFO pipeline_stop session_id=<id> duration_ms=<ms> raw_bytes=<n> processed_records=<n>`

---

### `GetState() FlowState`

Returns the current flow state.

**Returns**: `"idle"` | `"running"` | `"paused"` | `"error"`

---

### `GetBlockErrors() map[string]string`

Returns a map of `blockId → errorMessage` for all blocks currently in error state.

---

## Events Emitted

All events are emitted via the Wails v3 event system. The frontend subscribes using the generated TypeScript event names.

---

### `pipeline:state-changed`

Emitted whenever the flow state transitions.

```typescript
type PipelineStateChangedEvent = {
  state: "idle" | "running" | "paused" | "error";
  blockErrors: Record<string, string>;  // blockId → human-readable error
  sessionId: string | null;             // active session ID, null when idle
}
```

---

### `pipeline:data`

Emitted for each batch of processed data points as they exit Analysis blocks. Batched at ~60fps to avoid overwhelming the frontend.

```typescript
type PipelineDataEvent = {
  blockId: string;           // Analysis block ID
  points: Array<{
    timestamp: number;       // Unix ms
    values: number[];        // float64 values (multi-channel support)
  }>;
}
```

**Emission rate**: Backend batches DataChunks and emits at most every 16 ms (~60fps). If data arrives faster, multiple chunks are batched into a single event.

---

### `pipeline:block-status`

Emitted when an individual block's status changes (e.g., Input block connects or disconnects, a processing block encounters an error).

```typescript
type PipelineBlockStatusEvent = {
  blockId: string;
  status: "idle" | "connected" | "error";
  message: string | null;  // null when status is not "error"
}
```

---

## Pipeline Engine Internals (Non-Contract, Informational)

The pipeline engine (`internal/pipeline/engine.go`) launches one goroutine per block. Blocks communicate via typed Go channels (`chan DataChunk`). The engine:

1. Builds a channel graph mirroring the workflow connections.
2. Starts each block goroutine in topological order (inputs first).
3. Fan-out is handled by broadcasting copies of each DataChunk to all downstream channels.
4. Pause is implemented via a shared `context.Context` with a pause-aware receive wrapper.
5. The Session store (`internal/session/store.go`) intercepts DataChunks at Input block outputs (raw layer) and DataPoints at Analysis block inputs (processed layer).

---

## Logging Contract

| Field | Type | Description |
|---|---|---|
| `flow_state` | string | Current or target flow state |
| `session_id` | string | Active session ID |
| `block_id` | string | Block generating a log event |
| `error` | error | Error value on failure |
| `duration_ms` | int64 | Duration for timing log events |
