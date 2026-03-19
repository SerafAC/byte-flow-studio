# Data Model: Serial Data Processing Studio

**Phase**: 1 | **Date**: 2026-03-19 | **Feature**: 001-serial-data-studio

---

## Core Domain Entities

### DataChunk

The atomic unit of data flowing through the pipeline. Every chunk produced by an Input block carries a high-resolution timestamp.

| Field | Type | Description |
|---|---|---|
| `timestamp` | `int64` | Unix timestamp in milliseconds (assigned at the moment the application receives the data from the OS driver) |
| `sourceId` | `string` | ID of the originating Input block |
| `raw` | `[]byte` | Raw bytes as received from the hardware interface |
| `value` | `float64` (optional) | Parsed numeric value (set by Byte Parser or Signal Simulator; nil for unparsed raw chunks) |
| `values` | `[]float64` (optional) | Multi-channel parsed values (for multi-channel byte frames) |

**Identity**: Chunks are not individually identified; they flow as a continuous stream. The `timestamp` + `sourceId` pair is sufficient for session recording.

---

### Block

A discrete functional unit on the canvas. Three categories: Input, Processing, Analysis.

| Field | Type | Description |
|---|---|---|
| `id` | `string` | UUID, unique within a workflow |
| `type` | `string` | Block type identifier (e.g., `"uart"`, `"moving-average"`, `"line-chart"`) |
| `category` | `enum` | `input` \| `processing` \| `analysis` |
| `label` | `string` | User-editable display name |
| `params` | `map[string]any` | Type-specific configuration parameters (see Block Types below) |
| `position` | `{x: float64, y: float64}` | Canvas coordinates |
| `status` | `enum` | `idle` \| `connected` \| `error` — runtime state (not persisted) |
| `errorMessage` | `string` | Human-readable error text when `status == error` (not persisted) |

**Validation**:
- `id` MUST be non-empty and unique within the workflow
- `type` MUST match a registered block type in the block registry
- `category` MUST match the registered category for the given `type`

---

### Port

A typed connection point on a block. Input ports consume streams; output ports emit them.

| Field | Type | Description |
|---|---|---|
| `id` | `string` | Port identifier, unique within the block (e.g., `"in"`, `"out"`, `"out-raw"`, `"out-parsed"`) |
| `blockId` | `string` | Owning block's ID |
| `direction` | `enum` | `input` \| `output` |
| `dataType` | `enum` | `raw-bytes` \| `numeric` — prevents incompatible connections |
| `label` | `string` | Display label on the canvas |

**Compatibility Rule**: A connection is valid only if the source port's `dataType` equals the target port's `dataType`.

---

### Connection

A directional link between two ports carrying a data stream.

| Field | Type | Description |
|---|---|---|
| `id` | `string` | UUID, unique within a workflow |
| `fromBlockId` | `string` | Source block ID |
| `fromPortId` | `string` | Source port ID on the source block |
| `toBlockId` | `string` | Target block ID |
| `toPortId` | `string` | Target port ID on the target block |

**Validation**:
- No cycles allowed (validated by DAG topological sort on Start)
- Both port `dataType` values MUST be compatible
- A port MAY have multiple outgoing connections (fan-out); an input port MUST have at most one incoming connection

---

### Workflow

Top-level container for the entire canvas definition. Persisted as a JSON TEXT column in SQLite.

| Field | Type | Description |
|---|---|---|
| `id` | `string` | UUID |
| `name` | `string` | User-defined workflow name |
| `createdAt` | `int64` | Unix timestamp ms |
| `updatedAt` | `int64` | Unix timestamp ms |
| `blocks` | `[]Block` | All blocks in the workflow |
| `connections` | `[]Connection` | All connections in the workflow |
| `sessionConfig` | `SessionConfig` | Per-workflow session recording preferences |

---

### SessionConfig

Per-workflow configuration controlling session recording scope and retention.

| Field | Type | Default | Description |
|---|---|---|---|
| `storeRaw` | `bool` | `true` | Record raw byte layer in each Session |
| `storeProcessed` | `bool` | `true` | Record processed value layer in each Session |
| `maxSessions` | `int` | `10` | Maximum number of Sessions to retain (user-configurable in app settings; overrideable per workflow) |

---

### Session

A bounded record of data captured during one continuous Run (Start → Stop/Error).

| Field | Type | Description |
|---|---|---|
| `id` | `string` | UUID |
| `workflowId` | `string` | Owning workflow ID |
| `startTime` | `int64` | Unix timestamp ms (flow Start) |
| `endTime` | `int64` \| `null` | Unix timestamp ms (flow Stop/Error); null if session is still active |
| `endReason` | `enum` | `user-stop` \| `user-pause-then-stop` \| `error` \| `active` |
| `rawLayerEnabled` | `bool` | Whether raw bytes were recorded in this session |
| `processedLayerEnabled` | `bool` | Whether processed values were recorded in this session |
| `dataVolume` | `int64` | Total bytes stored for this session (raw + processed, for display in session browser) |

**Retention Rule**: When a new session completes and `sessionCount > maxSessions`, the session with the oldest `startTime` is deleted.

---

### RawDataRecord

One record in the raw byte layer. Stored in `raw_data` SQLite table.

| Field | Type | Description |
|---|---|---|
| `sessionId` | `string` | FK → Session.id |
| `blockId` | `string` | Source Input block ID |
| `timestamp` | `int64` | Unix timestamp ms |
| `data` | `BLOB` | Raw bytes for this chunk |

---

### ProcessedDataRecord

One record in the processed value layer. Stored in `processed_data` SQLite table.

| Field | Type | Description |
|---|---|---|
| `sessionId` | `string` | FK → Session.id |
| `blockId` | `string` | Analysis block ID that received this data |
| `timestamp` | `int64` | Unix timestamp ms |
| `values` | `BLOB` | IEEE 754 float64 array (little-endian binary encoding) |

---

## Block Types & Parameters

### Input Blocks

#### UART/Serial (`type: "uart"`)

| Param | Type | Description |
|---|---|---|
| `port` | `string` | Serial port path (e.g., `COM3`, `/dev/ttyUSB0`) |
| `baudRate` | `int` | Baud rate (e.g., `9600`, `115200`) |
| `dataBits` | `int` | 7 or 8 (default: 8) |
| `stopBits` | `float64` | 1, 1.5, or 2 (default: 1) |
| `parity` | `enum` | `none` \| `odd` \| `even` (default: `none`) |

Ports: one output port (`out`, type: `raw-bytes`)

#### WebSocket (`type: "websocket"`)

| Param | Type | Description |
|---|---|---|
| `url` | `string` | WebSocket server URL (e.g., `ws://192.168.1.10:8080/data`) |
| `subprotocol` | `string` | Optional WebSocket subprotocol |
| `reconnectIntervalMs` | `int` | Auto-reconnect delay in ms (default: 2000) |

Ports: one output port (`out`, type: `raw-bytes`)

#### Signal Simulator (`type: "simulator"`)

| Param | Type | Description |
|---|---|---|
| `waveform` | `enum` | `sine` \| `square` \| `sawtooth` \| `noise` |
| `frequencyHz` | `float64` | Signal frequency in Hz (default: 1.0) |
| `amplitude` | `float64` | Signal amplitude (default: 1.0) |
| `offset` | `float64` | DC offset (default: 0.0) |
| `sampleRateHz` | `float64` | Output samples per second (default: 100) |

Ports: one output port (`out`, type: `numeric`)

---

### Processing Blocks

#### Byte Parser (`type: "byte-parser"`)

| Param | Type | Description |
|---|---|---|
| `format` | `string` | Frame format (e.g., `float32-le`, `int16-be`, `uint8`) |
| `frameSize` | `int` | Expected bytes per value (auto-calculated from format if omitted) |
| `channels` | `int` | Number of values per frame (default: 1) |

Ports: one input (`in`, type: `raw-bytes`), one output (`out`, type: `numeric`)

#### Moving Average (`type: "moving-average"`)

| Param | Type | Description |
|---|---|---|
| `windowSize` | `int` | Number of samples in the rolling window (default: 10) |

Ports: one input (`in`, type: `numeric`), one output (`out`, type: `numeric`)

#### Summation (`type: "summation"`)

No additional params. Emits the running sum.

Ports: one input (`in`, type: `numeric`), one output (`out`, type: `numeric`)

#### FFT (`type: "fft"`)

| Param | Type | Description |
|---|---|---|
| `windowSize` | `int` | FFT window size (power of 2; default: 512) |
| `windowFunction` | `enum` | `none` \| `hann` \| `hamming` (default: `hann`) |

Ports: one input (`in`, type: `numeric`), one output (`out`, type: `numeric`) — output values are magnitude spectrum

#### Value Scaling/Offset (`type: "scaling"`)

| Param | Type | Description |
|---|---|---|
| `scale` | `float64` | Multiply factor (default: 1.0) |
| `offset` | `float64` | Additive offset applied after scaling (default: 0.0) |

Ports: one input (`in`, type: `numeric`), one output (`out`, type: `numeric`)

#### Passthrough (`type: "passthrough"`)

No params. Passes data unchanged (useful for forking or labelling streams).

Ports: one input (`in`, type: `numeric`), one output (`out`, type: `numeric`)

---

### Analysis Blocks

All Analysis blocks share these base params in addition to their type-specific params:

| Param | Type | Description |
|---|---|---|
| `bufferMode` | `enum` | `samples` \| `duration` — buffer size unit |
| `bufferSamples` | `int` | Max retained samples (used when `bufferMode == "samples"`; default: 1000) |
| `bufferDurationSec` | `float64` | Max retained data duration in seconds (used when `bufferMode == "duration"`; default: 30.0) |

#### Line Chart (`type: "line-chart"`)

| Param | Type | Description |
|---|---|---|
| `title` | `string` | Chart title |
| `xLabel` | `string` | X-axis label (default: `"Time"`) |
| `yLabel` | `string` | Y-axis label |
| `unit` | `string` | Value unit display (appended to tooltip) |
| `decimals` | `int` | Decimal places shown (default: 2) |

#### Bar Chart (`type: "bar-chart"`)

| Param | Type | Description |
|---|---|---|
| `title` | `string` | Chart title |
| `yLabel` | `string` | Y-axis label |
| `unit` | `string` | Value unit |
| `decimals` | `int` | Decimal places (default: 2) |

#### Numeric Value Display (`type: "value-display"`)

| Param | Type | Description |
|---|---|---|
| `label` | `string` | Display label |
| `unit` | `string` | Unit suffix |
| `decimals` | `int` | Decimal places (default: 3) |
| `showHistory` | `bool` | Show mini sparkline of recent values (default: false) |

#### Data Table (`type: "data-table"`)

| Param | Type | Description |
|---|---|---|
| `columns` | `string[]` | Column header labels (for multi-channel data) |
| `maxRows` | `int` | Maximum visible rows (default: 100; scrollable) |
| `decimals` | `int` | Decimal places (default: 3) |

#### FFT Spectrum Viewer (`type: "fft-spectrum"`)

| Param | Type | Description |
|---|---|---|
| `title` | `string` | Chart title |
| `xLabel` | `string` | X-axis label (default: `"Frequency (Hz)"`) |
| `yLabel` | `string` | Y-axis label (default: `"Magnitude"`) |
| `logScale` | `bool` | Use logarithmic Y axis (default: false) |
| `decimals` | `int` | Decimal places (default: 2) |

---

## SQLite Schema

```sql
-- Workflow definition (one row per .byteflow file)
CREATE TABLE IF NOT EXISTS workflow (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    definition  TEXT NOT NULL,  -- JSON: blocks, connections, sessionConfig
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

-- Session metadata
CREATE TABLE IF NOT EXISTS sessions (
    id                      TEXT PRIMARY KEY,
    workflow_id             TEXT NOT NULL,
    start_time              INTEGER NOT NULL,   -- Unix ms
    end_time                INTEGER,            -- NULL if active
    end_reason              TEXT,
    raw_layer_enabled       INTEGER NOT NULL DEFAULT 1,   -- boolean
    processed_layer_enabled INTEGER NOT NULL DEFAULT 1,   -- boolean
    data_volume             INTEGER NOT NULL DEFAULT 0    -- total bytes stored
);

-- Raw byte layer (lazy-loaded)
CREATE TABLE IF NOT EXISTS raw_data (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL,
    block_id    TEXT NOT NULL,
    timestamp   INTEGER NOT NULL,   -- Unix ms
    data        BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_raw_data_session ON raw_data(session_id, timestamp);

-- Processed value layer (loaded on workflow open)
CREATE TABLE IF NOT EXISTS processed_data (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL,
    block_id    TEXT NOT NULL,      -- Analysis block ID
    timestamp   INTEGER NOT NULL,   -- Unix ms
    values      BLOB NOT NULL       -- little-endian IEEE 754 float64 array
);
CREATE INDEX IF NOT EXISTS idx_processed_data_session ON processed_data(session_id, block_id, timestamp);
```

---

## Flow State Machine

```
              ┌─────────────────────────────────┐
              │                                 │
              ▼                                 │
         [Idle/Stopped] ──Start()──► [Validating]
                                          │
                                    valid │  invalid
                                          │       │
                                          ▼       ▼
                                      [Running]  [Error]
                                          │
                                    ┌─────┴─────┐
                              Pause()│           │hardware disconnect
                                     ▼           │or processing fault
                                  [Paused]       ▼
                                     │        [Error]
                               Resume()│
                                     ▼
                                  [Running]
                                     │
                                  Stop()│
                                     ▼
                              [Idle/Stopped]
```

**State Transitions**:
- `Idle → Running`: Start() called; DAG validated (at least one Input→Analysis path, no cycles); session created
- `Running → Paused`: Pause() called; goroutines receive pause signal via context; display state frozen
- `Paused → Running`: Resume() called; goroutines restarted
- `Running/Paused → Error`: Hardware disconnect, invalid config, or unhandled processing fault; affected block highlighted
- `Error → Running`: User reconfigures the offending block and calls Start() again
- `Running/Paused/Error → Idle`: Stop() called; session completed and recorded
