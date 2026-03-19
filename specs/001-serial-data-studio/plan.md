# Implementation Plan: Serial Data Processing Studio (ByteFlow Studio)

**Branch**: `001-serial-data-studio` | **Date**: 2026-03-19 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-serial-data-studio/spec.md`

---

## Summary

ByteFlow Studio is a multiplatform desktop application built on **Wails v3 (alpha) + Go + Vue 3**. The Go backend drives the real-time data pipeline — reading byte streams from hardware inputs (UART, WebSocket) or a simulator, processing them through a configurable DAG of processing blocks, and recording sessions to a local SQLite file. The Vue 3 frontend (with PrimeVue + Vue Flow + uPlot) provides a drag-and-drop canvas editor, embedded analysis visualisations, and a session browser. Wails bridges Go services to the frontend via auto-generated TypeScript bindings; real-time data flows from Go to Vue via the Wails typed event system.

---

## Technical Context

**Language/Version**: Go 1.23+ (core backend); TypeScript 5.x (frontend)
**Framework**: Wails v3 alpha (`github.com/wailsapp/wails/v3`)
**Frontend Stack**: Vue 3.5 (Composition API) + PrimeVue 4.x + Vue Flow 1.x + uPlot 1.x + Pinia 2.x
**Storage**: SQLite via `modernc.org/sqlite` (CGO-free, cross-compilable) — workflow definition + session data stored in a single `.byteflow` file (SQLite database)
**Testing**: Go stdlib `testing` + `testify` for unit and integration tests; Vitest for Vue component tests
**Target Platform**: Windows 10+, macOS 12+, Linux (Ubuntu 22.04+, Arch)
**Project Type**: Multiplatform desktop application (Wails v3)
**Performance Goals**: ≤200 ms input-to-display latency; ≥30 fps canvas render with 20 blocks; chart rendering ≤10% CPU at 1,000 pts/sec (uPlot target)
**Constraints**: Single portable `.byteflow` file per project; no CGO required at build time; no runtime dependencies for end user; raw session layer lazy-loaded (must not block UI open)
**Scale/Scope**: Up to 20 blocks, 30 connections; up to 10 sessions per workflow; each session up to 10 MB processed data

### Key Dependencies (pinned at implementation time)

| Package | Purpose |
|---|---|
| `github.com/wailsapp/wails/v3` | Desktop app framework (Go + WebView) |
| `go.bug.st/serial` | UART/Serial port I/O |
| `gonum.org/v1/gonum/dsp/fourier` | FFT implementation |
| `modernc.org/sqlite` | CGO-free SQLite driver |
| `@vue-flow/core` | Workflow canvas / DAG editor |
| `uplot` + `uplot-wrappers` | High-frequency real-time time-series charts |
| `pinia` | Vue state management |
| `primevue` | UI component library |

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

### Pre-Phase-0 Check

| Principle | Status | Evidence |
|---|---|---|
| I. Spec-Driven Development | ✅ PASS | `spec.md` complete; all `[NEEDS CLARIFICATION]` resolved in clarifications sessions |
| II. Test-First (TDD) | ✅ PASS | Enforced in task execution (tasks.md will require test-first for each block/service) |
| III. Component Modularity | ✅ PASS | Go: each block type is a separate package; Services expose clear contracts; frontend: stores separated from views |
| IV. Observability | ✅ PASS | Go `log/slog` (stdlib structured logging) used in all services and pipeline engine |
| V. Simplicity (YAGNI) | ⚠️ NOTE | Two justified complexities — see Complexity Tracking |

### Post-Phase-1 Check

| Principle | Status | Notes |
|---|---|---|
| I. Spec-Driven | ✅ PASS | No scope added beyond spec |
| II. Test-First | ✅ PASS | Contract definitions enable test-first for each service |
| III. Modularity | ✅ PASS | `internal/` packages enforce package-level boundaries; services are injectable |
| IV. Observability | ✅ PASS | Structured log calls specified in each service contract |
| V. Simplicity | ✅ PASS | Justified in Complexity Tracking |

---

## Project Structure

### Documentation (this feature)

```text
specs/001-serial-data-studio/
├── plan.md              # This file
├── research.md          # Phase 0: tech decisions & rationale
├── data-model.md        # Phase 1: entities, fields, state machines
├── quickstart.md        # Phase 1: dev environment setup
├── contracts/           # Phase 1: service & block interface contracts
│   ├── workflow-service.md
│   ├── pipeline-service.md
│   ├── session-service.md
│   └── block-interface.md
└── tasks.md             # Phase 2: /speckit.tasks output (NOT created here)
```

### Source Code (repository root)

```text
byteflow-studio/               # repo root (wails3 init output)
├── main.go                    # Wails v3 application entry point
├── app.go                     # ApplicationService (lifecycle hooks)
├── go.mod
├── go.sum
├── wails.json                 # Wails v3 project config
│
├── internal/
│   ├── input/                 # Input source implementations
│   │   ├── types.go           # DataChunk, InputConfig, InputBlock interface
│   │   ├── uart.go            # UART/Serial input (go.bug.st/serial)
│   │   ├── websocket.go       # WebSocket input (stdlib net/http)
│   │   └── simulator.go       # Signal simulator (sine/square/noise)
│   │
│   ├── pipeline/              # Core pipeline engine (DAG executor)
│   │   ├── types.go           # Block, Connection, Port, Stream, FlowState
│   │   ├── graph.go           # DAG: validation, cycle detection, topology sort
│   │   └── engine.go          # Goroutine-per-block runner, pause/resume via context
│   │
│   ├── processing/            # Processing block implementations
│   │   ├── registry.go        # Block factory registry (type string → constructor)
│   │   ├── movingaverage.go
│   │   ├── summation.go
│   │   ├── fft.go             # gonum fourier
│   │   ├── scaling.go
│   │   ├── byteparser.go      # Byte frame → typed float64 values
│   │   └── passthrough.go
│   │
│   ├── session/               # Session management
│   │   ├── types.go           # Session, SessionMeta, SessionConfig, DataLayer
│   │   ├── manager.go         # Create/complete/switch active session
│   │   └── store.go           # SQLite read/write for sessions
│   │
│   ├── workflow/              # Workflow definition persistence
│   │   ├── types.go           # Workflow, BlockDef, ConnectionDef
│   │   └── store.go           # SQLite read/write for workflow definition
│   │
│   └── logging/
│       └── logger.go          # slog wrapper with consistent field keys
│
├── services/                  # Wails v3 service layer (exposed to frontend)
│   ├── workflow_service.go    # WorkflowService
│   ├── pipeline_service.go    # PipelineService
│   └── session_service.go     # SessionService
│
└── frontend/
    ├── package.json
    ├── vite.config.ts
    ├── tsconfig.json
    ├── index.html
    │
    ├── bindings/              # Auto-generated by wails3 (do not edit)
    │
    └── src/
        ├── main.ts
        ├── App.vue
        │
        ├── components/
        │   ├── canvas/
        │   │   ├── WorkflowCanvas.vue      # Vue Flow wrapper + drag-drop
        │   │   ├── BlockNode.vue           # Custom Vue Flow node (all block types)
        │   │   └── ConnectionEdge.vue      # Custom edge renderer
        │   │
        │   ├── blocks/
        │   │   ├── inputs/
        │   │   │   ├── UartBlockConfig.vue
        │   │   │   ├── WebSocketBlockConfig.vue
        │   │   │   └── SimulatorBlockConfig.vue
        │   │   ├── processing/
        │   │   │   ├── MovingAverageConfig.vue
        │   │   │   ├── FFTConfig.vue
        │   │   │   ├── ScalingConfig.vue
        │   │   │   └── ByteParserConfig.vue
        │   │   └── analysis/
        │   │       ├── LineChartBlock.vue       # uPlot streaming chart
        │   │       ├── BarChartBlock.vue
        │   │       ├── ValueDisplayBlock.vue
        │   │       ├── DataTableBlock.vue
        │   │       └── FftSpectrumBlock.vue     # uPlot frequency-domain chart
        │   │
        │   └── panels/
        │       ├── BlockLibraryPanel.vue
        │       └── SessionPanel.vue
        │
        ├── stores/
        │   ├── workflow.ts     # Canvas blocks and connections state
        │   ├── pipeline.ts     # Flow state (Running/Paused/Error), block errors
        │   └── session.ts      # Sessions list, active session, session config
        │
        ├── services/
        │   └── wails.ts        # Thin typed wrappers over generated bindings
        │
        └── views/
            └── MainView.vue    # Root layout: library panel + canvas + status bar
```

**Structure Decision**: Single-project Wails v3 layout. The `internal/` packages enforce Go visibility rules so that only `services/` can import from `internal/` — no frontend logic reaches into pipeline or session code directly. Generated `bindings/` are the sole bridge. Wails v3 events carry real-time data; service methods handle control operations.

---

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| SQLite as file format (instead of plain JSON) | Session raw byte layer cannot be efficiently stored in JSON (base64 adds 33% overhead; no streaming/random access; lazy-load of raw layer requires byte-range access) | JSON + base64 would bloat session files unacceptably; ZIP archive lacks random access needed for lazy-loading the raw layer |
| Dual-layer session storage (raw bytes + processed values) | Explicitly requested: processed values enable immediate display on open; raw bytes enable future pipeline replay | Single-layer (processed only) was the original design; the raw layer adds ~2x storage but is user-toggleable per workflow |

---

## Implementation Phases

### Phase 1 — Core MVP (P1 User Stories)

**Goal**: UART Input → pipeline → Raw Display Analysis. Flow start/pause/stop. Live data visible.

Deliverables:
- Wails v3 project init with Vue 3 template
- `internal/input/uart.go` with DataChunk + timestamp
- `internal/pipeline/engine.go` (2-block DAG, goroutine channels)
- `services/pipeline_service.go` with Start/Pause/Stop
- `WorkflowCanvas.vue` with Vue Flow (two nodes, one edge)
- `ValueDisplayBlock.vue` rendering live data via Wails events
- SQLite schema initialised (workflow + session tables)

### Phase 2 — Processing & Multi-Output (P2 User Stories)

**Goal**: Full processing block library; multiple inputs/outputs; WebSocket input.

Deliverables:
- `internal/processing/` all 6 block types
- `internal/input/websocket.go` + `simulator.go`
- Multi-node DAG support in pipeline engine
- All `*Config.vue` panels
- `BlockLibraryPanel.vue` (categorised, drag-to-canvas)
- `LineChartBlock.vue` + `FftSpectrumBlock.vue` (uPlot)

### Phase 3 — Session Management & Persistence (P3 User Stories)

**Goal**: Save/load workflow; session recording (both layers); session browser; N-session retention; analysis buffers.

Deliverables:
- `internal/session/store.go` (raw + processed SQLite tables)
- `services/session_service.go`
- `SessionPanel.vue` + session list/switch
- FR-028 buffer config (samples or duration) in Analysis block params
- FR-033 per-workflow session layer toggle

### Phase 4 — Polish & Full-Screen (P3 User Stories)

**Goal**: Full-screen analysis view; session-aware workflow open; error UX.

Deliverables:
- Full-screen mode for each Analysis block
- Degraded-state open (missing hardware Input handling)
- Storage threshold warning (FR-030 edge case)
- Lazy-load raw layer on demand
