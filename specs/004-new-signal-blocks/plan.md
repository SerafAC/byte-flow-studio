# Implementation Plan: New Signal Blocks

**Branch**: `004-new-signal-blocks` | **Date**: 2026-04-10 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-new-signal-blocks/spec.md`

## Summary

Add six new workflow blocks to ByteFlow Studio: one input block (Bluetooth/SPP), three processing blocks (Signal Multiply, Filter, Derivative), and two output/analysis blocks (Spectrum Viewer as waterfall spectrogram, Hex Viewer). Each block follows the existing `Block` interface pattern with Go backend processing and Vue 3 frontend configuration/visualization. The Bluetooth block outputs raw bytes (like the existing UART block); processing blocks operate on numeric streams; the Spectrum Viewer uses HTML5 Canvas for waterfall rendering; the Hex Viewer displays a scrollable hex dump.

## Technical Context

**Language/Version**: Go 1.23+ (backend), TypeScript 5.x / Vue 3.5 (frontend)  
**Primary Dependencies**: Wails v3, Vue Flow 1.48, uPlot 1.6, PrimeVue 4.5, Pinia 3, `go.bug.st/serial`, `gonum.org/v1/gonum/dsp/fourier`, SCSS (sass)  
**New Dependencies**: Go Bluetooth discovery library (see research.md); no new frontend dependencies (Canvas 2D API for spectrogram)  
**Storage**: SQLite via `modernc.org/sqlite` — workflow definitions as JSON blob; session data for raw/processed streams  
**Testing**: Go `testing` package with mock I/O injection (io.Pipe pattern); Vue component tests  
**Target Platform**: Linux, macOS, Windows (desktop via Wails)  
**Project Type**: Desktop application (Wails v3)  
**Performance Goals**: Spectrum Viewer at 10+ FPS; Hex Viewer lag-free at typical serial/BT rates (up to 115200 baud); Filter/Multiply/Derivative processing with no perceptible latency  
**Constraints**: CGO-free where possible (consistent with `modernc.org/sqlite` choice); Bluetooth discovery may require platform-specific code  
**Scale/Scope**: 6 new block types; ~12 new Go files, ~8 new Vue components

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| **I. Spec-Driven Development** | PASS | spec.md complete with 6 user stories, 26 FRs, 7 success criteria, 5 clarifications resolved. No unresolved `[NEEDS CLARIFICATION]` markers. |
| **II. Test-First (NON-NEGOTIABLE)** | PASS (planned) | All blocks will follow Red → Green → Refactor. Tests use mock I/O injection pattern (io.Pipe for Bluetooth, channels for processing). Task ordering: tests before implementation. |
| **III. Component Modularity** | PASS | Each block is a self-contained module implementing the `Block` interface. Independent testability via mock inputs/outputs. No cross-block dependencies. UI separated from logic (Go backend vs Vue frontend). |
| **IV. Observability** | PASS (planned) | All blocks will use structured logging for lifecycle events (connect, disconnect, reconnect, error). BlockError channel for error propagation. `pipeline:block-status` events for frontend status display. |
| **V. Simplicity (YAGNI)** | PASS | No speculative abstractions. Bluetooth uses existing serial port pattern. Filter uses standard Butterworth coefficients. Derivative uses simple finite difference. No interpolation-based sync for Multiply. |

**Gate result: PASS** — proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/004-new-signal-blocks/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
# Backend — new block implementations
internal/
├── input/
│   ├── bluetooth.go              # Bluetooth SPP input block
│   └── bluetooth_test.go         # Tests with mock RFCOMM reader
├── processing/
│   ├── multiply.go               # Signal multiply block
│   ├── multiply_test.go
│   ├── filter.go                 # Frequency filter block (LP/HP/BP/BS)
│   ├── filter_test.go
│   ├── derivative.go             # Derivative block
│   └── derivative_test.go
├── analysis/
│   ├── spectrumviewer.go         # Spectrum viewer (waterfall) analysis block
│   ├── spectrumviewer_test.go
│   ├── hexviewer.go              # Hex viewer analysis block
│   └── hexviewer_test.go
└── pipeline/
    └── types.go                  # No changes needed — existing DataChunk/Block interfaces suffice

# Backend — block registration & descriptors
services/
└── workflow_service.go           # Add block type descriptors for all 6 blocks

# Frontend — block configuration components
frontend/src/components/blocks/
├── inputs/
│   └── BluetoothBlockConfig.vue  # Bluetooth device selection UI
├── processing/
│   ├── MultiplyConfig.vue        # Multiply block config (minimal — no params beyond connections)
│   ├── FilterConfig.vue          # Filter mode, cutoff, order controls
│   └── DerivativeConfig.vue      # Derivative config (minimal)
└── analysis/
    ├── SpectrumViewerBlock.vue   # Waterfall spectrogram (Canvas 2D)
    └── HexViewerBlock.vue        # Hex dump display with search/pause

# Frontend — registration
frontend/src/components/canvas/
├── BlockNode.vue                 # Add config component mappings for 6 new types
└── WorkflowCanvas.vue            # Add default dimensions for 2 new analysis blocks
```

**Structure Decision**: Follows existing project layout. New blocks are placed alongside existing ones in `internal/input/`, `internal/processing/`, `internal/analysis/`, and `frontend/src/components/blocks/`. No new directories needed beyond what exists.

## Complexity Tracking

> No constitution violations to justify.
