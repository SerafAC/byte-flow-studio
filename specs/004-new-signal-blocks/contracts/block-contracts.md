# Block Contracts: New Signal Blocks

**Feature**: 004-new-signal-blocks  
**Date**: 2026-04-10

This document defines the interface contracts for the six new blocks. All blocks implement the existing `pipeline.Block` interface. Analysis blocks additionally implement `pipeline.AnalysisBlock`.

## Port Contracts

### Bluetooth Block (`bluetooth`)
```
Category: input
Inputs:  (none)
Outputs: "out" → DataTypeRaw
```
- Emits one DataChunk per read cycle containing raw bytes from SPP/RFCOMM connection
- Sets `Timestamp` (Unix ms) and `SourceID` on each chunk
- Never emits chunks with empty `Raw` field

### Signal Multiply Block (`multiply`)
```
Category: processing
Inputs:  "in-0" → DataTypeNumeric, "in-1" → DataTypeNumeric, ... "in-N" → DataTypeNumeric
Outputs: "out" → DataTypeNumeric
```
- Number of input ports determined by `inputCount` configuration parameter (2-8)
- Output chunk contains `Values` with single element: product of latest values from all connected inputs
- With single input connected: passes through unchanged
- Preserves `Timestamp` from the most recent input chunk that triggered output

### Filter Block (`filter`)
```
Category: processing
Inputs:  "in" → DataTypeNumeric
Outputs: "out" → DataTypeNumeric
```
- One-to-one: each input chunk produces one output chunk
- Output `Values` contains filtered version of input `Values`
- Preserves `Timestamp` and `SourceID` from input chunk
- Filter state resets when `mode`, `cutoffHz`, `cutoffLowHz`, `cutoffHighHz`, or `order` parameters change

### Derivative Block (`derivative`)
```
Category: processing
Inputs:  "in" → DataTypeNumeric
Outputs: "out" → DataTypeNumeric
```
- One-to-one: each input chunk produces one output chunk
- First chunk: output `Values` = `[0.0]`
- Subsequent chunks: output `Values` = `[(currentValue - previousValue) / (currentTimestamp - previousTimestamp) * 1000]` (value per second)
- Preserves `Timestamp` and `SourceID` from input chunk

### Spectrum Viewer Block (`spectrum-viewer`)
```
Category: analysis
Inputs:  "in" → DataTypeNumeric
Outputs: (none — analysis/display only)
```
- Expects input from an FFT processing block (magnitude spectrum as `Values`)
- Each received chunk represents one FFT frame
- Buffers chunks according to `BufferConfig`
- Emits `pipeline:data` events to frontend with FFT magnitude data

### Hex Viewer Block (`hex-viewer`)
```
Category: analysis
Inputs:  "in" → DataTypeRaw
Outputs: (none — analysis/display only)
```
- Accepts raw byte data
- Buffers chunks according to `BufferConfig`, enforces `maxBytes` limit
- Emits `pipeline:data` events to frontend with raw byte arrays

## Configuration Parameter Contracts

### Bluetooth Block
| Parameter | Type | Required | Default | Validation |
|-----------|------|----------|---------|------------|
| deviceAddress | string | yes | "" | Must be non-empty when workflow starts |
| serialPort | string | no | "" | Auto-resolved from deviceAddress if empty |

### Filter Block
| Parameter | Type | Required | Default | Validation |
|-----------|------|----------|---------|------------|
| mode | string | yes | "lowpass" | One of: "lowpass", "highpass", "bandpass", "bandstop" |
| cutoffHz | float64 | yes* | 100.0 | > 0, < sampleRateHz/2. *Required for lowpass/highpass. |
| cutoffLowHz | float64 | yes* | 50.0 | > 0, < cutoffHighHz. *Required for bandpass/bandstop. |
| cutoffHighHz | float64 | yes* | 200.0 | > cutoffLowHz, < sampleRateHz/2. *Required for bandpass/bandstop. |
| order | int | no | 2 | Range: 1-8 |
| sampleRateHz | float64 | yes | 1000.0 | > 0 |

### Multiply Block
| Parameter | Type | Required | Default | Validation |
|-----------|------|----------|---------|------------|
| inputCount | int | no | 2 | Range: 2-8 |

### Derivative Block
No configurable parameters.

### Spectrum Viewer Block
| Parameter | Type | Required | Default | Validation |
|-----------|------|----------|---------|------------|
| colorMap | string | no | "viridis" | One of: "viridis", "magma", "inferno", "plasma", "grayscale" |
| minFreqHz | float64 | no | 0 | >= 0 |
| maxFreqHz | float64 | no | 0 (auto) | > minFreqHz or 0 for auto |
| minAmplitude | float64 | no | 0 | >= 0 |
| maxAmplitude | float64 | no | 0 (auto) | > minAmplitude or 0 for auto |

### Hex Viewer Block
| Parameter | Type | Required | Default | Validation |
|-----------|------|----------|---------|------------|
| maxBytes | int | no | 65536 | Range: 1024-1048576 |
| bytesPerRow | int | no | 16 | One of: 8, 16, 32 |

## Frontend Event Contracts

### pipeline:data (Spectrum Viewer)
```typescript
{
  blockId: string,
  points: Array<{
    timestamp: number,
    values: number[]  // FFT magnitudes for one frame
  }>
}
```

### pipeline:data (Hex Viewer)
```typescript
{
  blockId: string,
  points: Array<{
    timestamp: number,
    raw: number[],   // byte values 0-255
    mode: "raw"
  }>
}
```

## Wails-Bound Service Methods

### New methods on WorkflowService

```
listBluetoothDevices() → Array<{address: string, name: string, port: string}>
```
Returns list of paired Bluetooth devices from the host OS. Each entry includes the device address, display name, and resolved virtual serial port path (if available).
