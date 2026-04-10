# Data Model: New Signal Blocks

**Feature**: 004-new-signal-blocks  
**Date**: 2026-04-10

## Existing Entities (no changes)

### DataChunk
The existing `DataChunk` struct in `internal/pipeline/types.go` is sufficient for all new blocks:

| Field | Type | Description |
|-------|------|-------------|
| Timestamp | int64 | Unix milliseconds, set by input blocks |
| SourceID | string | Originating input block ID |
| Raw | []byte | Raw byte payload (used by Bluetooth, Hex Viewer) |
| Values | []float64 | Numeric values (used by Filter, Multiply, Derivative, Spectrum Viewer) |

No new fields are needed. The `Raw` / `Values` duality already supports both raw-byte and numeric data flows.

### Port DataTypes
Existing `DataType` constants are sufficient:
- `DataTypeRaw` ("raw-bytes") — Bluetooth output, Hex Viewer input
- `DataTypeNumeric` ("numeric") — Filter, Multiply, Derivative I/O; Spectrum Viewer input

## New Block State

### BluetoothBlock

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique block ID |
| deviceAddress | string | Selected Bluetooth device MAC/address |
| deviceName | string | Human-readable device name (display only) |
| serialPort | string | Resolved virtual serial port path (e.g., `/dev/rfcomm0`) |
| reconnectState | reconnectState | Current reconnection state (idle/connecting/retrying/connected) |
| retryCount | int | Current retry attempt number |
| reader | io.ReadCloser | Injectable I/O for testing |

**Configuration params** (from user):
- `deviceAddress` (string) — selected paired device
- `serialPort` (string) — optional manual override of virtual serial port path

**Ports**:
- Output: `"out"` (DataTypeRaw)

### FilterBlock

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique block ID |
| mode | string | "lowpass" / "highpass" / "bandpass" / "bandstop" |
| cutoffHz | float64 | Cutoff frequency (low/high-pass) |
| cutoffLowHz | float64 | Lower cutoff (band-pass/band-stop) |
| cutoffHighHz | float64 | Upper cutoff (band-pass/band-stop) |
| order | int | Filter order (1-8) |
| sampleRateHz | float64 | Input sample rate (needed for coefficient computation) |
| coeffB | []float64 | Computed numerator coefficients |
| coeffA | []float64 | Computed denominator coefficients |
| state | []float64 | Filter delay line (Direct Form II Transposed) |
| mu | sync.RWMutex | Protects concurrent parameter updates |

**Configuration params** (from user):
- `mode` (string)
- `cutoffHz` (float64)
- `cutoffLowHz` (float64, band-pass/band-stop only)
- `cutoffHighHz` (float64, band-pass/band-stop only)
- `order` (int, default 2)
- `sampleRateHz` (float64, default 1000)

**Ports**:
- Input: `"in"` (DataTypeNumeric)
- Output: `"out"` (DataTypeNumeric)

### MultiplyBlock

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique block ID |
| pending | map[string]float64 | Latest value per input port |
| portCount | int | Number of configured input ports |
| mu | sync.Mutex | Protects pending map |

**Configuration params** (from user):
- `inputCount` (int, default 2, range 2-8) — determines number of input ports

**Ports**:
- Inputs: `"in-0"`, `"in-1"`, ... `"in-N"` (DataTypeNumeric)
- Output: `"out"` (DataTypeNumeric)

### DerivativeBlock

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique block ID |
| prevValue | float64 | Previous sample value |
| prevTimestamp | int64 | Previous sample timestamp (Unix ms) |
| hasPrev | bool | Whether a previous sample exists |

**Configuration params**: None (stateless configuration).

**Ports**:
- Input: `"in"` (DataTypeNumeric)
- Output: `"out"` (DataTypeNumeric)

### SpectrumViewerBlock (Analysis)

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique block ID |
| buffer | []DataChunk | Buffered FFT magnitude frames |
| bufferCfg | BufferConfig | Buffer limits (samples or duration) |
| mu | sync.RWMutex | Protects buffer |

**Configuration params** (from user):
- `colorMap` (string, default "viridis") — color mapping scheme
- `minFreqHz` (float64, default 0) — lower frequency bound
- `maxFreqHz` (float64, default auto) — upper frequency bound
- `minAmplitude` (float64, default 0) — amplitude range min for color mapping
- `maxAmplitude` (float64, default auto) — amplitude range max for color mapping

**Ports**:
- Input: `"in"` (DataTypeNumeric) — expects FFT magnitude output from upstream FFT block

### HexViewerBlock (Analysis)

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique block ID |
| buffer | []DataChunk | Buffered raw byte chunks |
| bufferCfg | BufferConfig | Buffer limits |
| mu | sync.RWMutex | Protects buffer |

**Configuration params** (from user):
- `maxBytes` (int, default 65536) — scrollback buffer limit
- `bytesPerRow` (int, default 16) — display width

**Ports**:
- Input: `"in"` (DataTypeRaw) — receives raw byte stream

## Block Type Descriptors

New entries for `blockTypeDescriptors()` in `services/workflow_service.go`:

| Type | Category | Label | Default Params |
|------|----------|-------|----------------|
| `bluetooth` | input | Bluetooth (SPP) | `{deviceAddress: "", serialPort: ""}` |
| `multiply` | processing | Signal Multiply | `{inputCount: 2}` |
| `filter` | processing | Frequency Filter | `{mode: "lowpass", cutoffHz: 100, order: 2, sampleRateHz: 1000}` |
| `derivative` | processing | Derivative | `{}` |
| `spectrum-viewer` | analysis | Spectrum Viewer | `{colorMap: "viridis", minFreqHz: 0, maxFreqHz: 0, minAmplitude: 0, maxAmplitude: 0}` |
| `hex-viewer` | analysis | Hex Viewer | `{maxBytes: 65536, bytesPerRow: 16}` |

## Entity Relationships

```
BluetoothDevice (OS-level)
    └──> BluetoothBlock (discovers, connects via SPP/RFCOMM)
            └──> DataChunk (Raw bytes)
                    ├──> HexViewerBlock (displays raw)
                    └──> ByteParser (existing) ──> DataChunk (Numeric)
                            ├──> FilterBlock ──> DataChunk (Numeric)
                            ├──> MultiplyBlock ──> DataChunk (Numeric)
                            ├──> DerivativeBlock ──> DataChunk (Numeric)
                            └──> FFT (existing) ──> SpectrumViewerBlock
```
