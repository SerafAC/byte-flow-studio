# Quickstart: New Signal Blocks

**Feature**: 004-new-signal-blocks  
**Date**: 2026-04-10

## Prerequisites

- Go 1.23+ installed
- Node.js / npm for frontend
- Wails v3 CLI installed
- A Bluetooth-capable machine with at least one paired SPP device (for Bluetooth block testing)
- Existing ByteFlow Studio project checked out on `004-new-signal-blocks` branch

## Development Order

Implement blocks in dependency order to enable incremental testing:

1. **Derivative** — simplest processing block, no dependencies, validates the processing block pattern
2. **Signal Multiply** — multi-input processing, tests dynamic port creation
3. **Filter** — most complex processing block, requires coefficient computation
4. **Hex Viewer** — raw-byte analysis block, can be tested with existing UART/WebSocket inputs
5. **Spectrum Viewer** — numeric analysis block, requires upstream FFT block; most complex frontend component
6. **Bluetooth** — input block, requires platform-specific discovery; can be tested last with all other blocks ready

## Adding a New Processing Block (Example: Derivative)

### Backend

1. Create `internal/processing/derivative.go`:
   - Define struct implementing `pipeline.Block`
   - Register in `init()` via `processing.Register("derivative", factory)`
   - Implement `Run()`: read from `inputs["in"]`, compute derivative, write to `outputs["out"]`

2. Create `internal/processing/derivative_test.go`:
   - Use channel-based test pattern (see `movingaverage_test.go`)
   - Test: constant input → zero output
   - Test: linear ramp → constant output
   - Test: first sample → zero output
   - Test: variable dt handling

3. Add block descriptor in `services/workflow_service.go` → `blockTypeDescriptors()`

### Frontend

4. Create `frontend/src/components/blocks/processing/DerivativeConfig.vue`:
   - Minimal config (no user-configurable params)
   - Follow existing config component pattern (`useBlockConfig()` composable)

5. Register in `BlockNode.vue` → `configComponentMap`

## Adding a New Analysis Block (Example: Hex Viewer)

### Backend

1. Create `internal/analysis/hexviewer.go`:
   - Implement `pipeline.AnalysisBlock` interface (includes `BufferConfig()`, `Snapshot()`)
   - Register via `registerAnalysis("hex-viewer", factory)`
   - Buffer incoming raw DataChunks, enforce max byte limit

2. Create `internal/analysis/hexviewer_test.go`:
   - Test buffering, snapshot, byte search

3. Add block descriptor in `services/workflow_service.go`

### Frontend

4. Create `frontend/src/components/blocks/analysis/HexViewerBlock.vue`:
   - Use `useAnalysisBlock()` composable for data subscription
   - Render hex dump layout (offset | hex bytes | ASCII)
   - Implement virtual scrolling, pause/resume, search

5. Register in `BlockNode.vue` → `configComponentMap`
6. Add default dimensions in `WorkflowCanvas.vue` → `analysisDefaultDimensions`

## Adding the Bluetooth Input Block

### Backend

1. Create `internal/input/bluetooth.go`:
   - Follow `uart.go` pattern closely
   - Add platform-specific device discovery (list paired BT devices)
   - Connect via virtual serial port path using `go.bug.st/serial`
   - Implement auto-reconnect with exponential backoff

2. Create `internal/input/bluetooth_test.go`:
   - Use `io.Pipe()` mock pattern (same as `uart_test.go`)
   - Test: data streaming, disconnect detection, reconnect behavior

3. Add Wails-bound method for `listBluetoothDevices()` in workflow service

### Frontend

4. Create `frontend/src/components/blocks/inputs/BluetoothBlockConfig.vue`:
   - Device list dropdown with refresh button (calls `listBluetoothDevices()`)
   - Connection status indicator
   - Follow `UartBlockConfig.vue` pattern

## Running Tests

```bash
# Backend unit tests for new blocks
go test ./internal/processing/... -run "TestDerivative|TestMultiply|TestFilter"
go test ./internal/analysis/... -run "TestSpectrumViewer|TestHexViewer"
go test ./internal/input/... -run "TestBluetooth"

# All backend tests
go test ./...

# Frontend
cd frontend && npm test
```

## Verifying End-to-End

1. Start the app: `wails3 dev`
2. Create a workflow: `Simulator → Derivative → Line Chart` — verify derivative output
3. Create a workflow: `Simulator (sine) + Simulator (square) → Multiply → Line Chart` — verify product
4. Create a workflow: `Simulator → Filter (lowpass) → Line Chart` — verify filtering
5. Create a workflow: `Simulator → FFT → Spectrum Viewer` — verify waterfall display
6. Create a workflow: `UART → Hex Viewer` — verify hex dump (or use Simulator with byte-parser in reverse)
7. If Bluetooth hardware available: `Bluetooth → Hex Viewer` — verify BT data reception
