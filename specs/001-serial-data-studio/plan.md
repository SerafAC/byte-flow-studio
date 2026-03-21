# Implementation Plan: Serial Data Processing Studio (ByteFlow Studio)

**Branch**: `001-serial-data-studio` | **Date**: 2026-03-21 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-serial-data-studio/spec.md`

**Note**: This plan is maintained by the `/speckit.plan` command. Research, data model, contracts, and quickstart are in the same directory.

## Summary

ByteFlow Studio is a multiplatform desktop application (Windows 10+, macOS 12+, Linux Ubuntu 22.04+/Arch) built with Wails v3 (Go 1.23+ backend + Vue 3 TypeScript frontend). Users compose visual data-processing workflows on a canvas, connecting Input blocks (UART/Serial, WebSocket, Signal Simulator), Processing blocks (Moving Average, FFT, Byte Parser, etc.), and Analysis blocks (Line Chart, FFT Viewer, Value Display, etc.) into live data pipelines. Workflows, block configurations, canvas layout, and session data (raw bytes + processed values) are persisted in a single SQLite `.byteflow` file per workflow.

## Technical Context

**Language/Version**: Go 1.23+ (backend), TypeScript 5.x / Vue 3.5 (frontend)
**Primary Dependencies**: Wails v3 alpha, Vue Flow 1.x, uPlot, PrimeVue 4.x, Pinia 2.x, go.bug.st/serial, gonum (FFT), gorilla/websocket
**Package Manager (Frontend)**: pnpm (replaces npm; wails.json `frontend.installCommand` / `devCommand` / `buildCommand` must use pnpm)
**Storage**: SQLite via `modernc.org/sqlite` (CGO-free). Single `.byteflow` file per workflow (SQLite DB).
**Testing**: `go test` + testify (backend); Vitest (frontend via `pnpm test`)
**Target Platform**: Desktop — Windows 10+, macOS 12+, Linux Ubuntu 22.04+ / Arch
**Project Type**: Desktop application (Wails v3 — Go backend + WebView frontend)
**Performance Goals**: ≤200 ms input→display latency; ≥30 fps canvas at 20 blocks/30 connections; ≤3 s cold launch
**Constraints**: Single installable binary per platform; no CGO on Linux/Windows (CGO on macOS for serial port); SQLite CGO-free on all platforms
**Scale/Scope**: Single-user local desktop; up to 20 blocks, 30 connections, 10 sessions × 10 MB processed data each

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Spec-Driven Development | ✅ PASS | spec.md complete; all clarifications resolved in two sessions |
| II. Test-First (TDD) | ✅ PASS | tasks.md enforces test-first ordering; all Phase 3–7 tasks begin with test tasks |
| III. Component Modularity | ✅ PASS | Go internal packages per domain; Wails services as explicit contracts; frontend stores isolated from business logic |
| IV. Observability | ✅ PASS | `log/slog` structured logging throughout; `BYTEFLOW_LOG_LEVEL` env var; block-level error events |
| V. Simplicity (YAGNI) | ✅ PASS | Wails v3 chosen for native bindings (justified); uPlot for streaming perf (justified); no speculative abstractions |

**Constitution Check PASSED** — no violations requiring Complexity Tracking entries.

## Project Structure

### Documentation (this feature)

```text
specs/001-serial-data-studio/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output — all technology decisions resolved
├── data-model.md        # Phase 1 output — SQLite schema + Go types
├── quickstart.md        # Phase 1 output — dev environment setup with pnpm
├── contracts/           # Phase 1 output — Go service interface contracts
│   ├── block-interface.md
│   ├── pipeline-service.md
│   ├── session-service.md
│   └── workflow-service.md
└── tasks.md             # Phase 2 output (/speckit.tasks command — NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Wails v3 desktop application: Go backend + TypeScript/Vue frontend

main.go                  # Wails app entry point; wires and registers all three services
app.go                   # Application-level Wails service (lifecycle hooks)
wails.json               # Wails project config (frontend.installCommand: pnpm install,
                         #   frontend.devCommand: pnpm dev, frontend.buildCommand: pnpm build)

internal/
├── input/               # Input block implementations (uart.go, websocket.go, simulator.go)
├── pipeline/            # DAG engine, graph validation, types (engine.go, graph.go, types.go)
├── processing/          # Processing block impls + shared registry (registry.go, *.go)
├── analysis/            # Analysis block impls (valuedisplay.go, linechart.go, …)
├── session/             # Session manager + SQLite store (manager.go, store.go, types.go)
├── workflow/            # Workflow persistence + SQLite schema (store.go, types.go)
└── logging/             # slog wrapper (logger.go)

services/                # Wails service layer (registered in main.go)
├── workflow_service.go  # WorkflowService — canvas CRUD, block library
├── pipeline_service.go  # PipelineService — Start/Pause/Resume/Stop, event emission
└── session_service.go   # SessionService — session listing, historical data

frontend/
├── src/
│   ├── components/
│   │   ├── canvas/      # WorkflowCanvas.vue, BlockNode.vue, ConnectionEdge.vue
│   │   ├── blocks/
│   │   │   ├── inputs/  # UartBlockConfig.vue, WebSocketBlockConfig.vue, …
│   │   │   ├── processing/ # MovingAverageConfig.vue, ByteParserConfig.vue, …
│   │   │   └── analysis/   # ValueDisplayBlock.vue, LineChartBlock.vue, …
│   │   └── panels/      # BlockLibraryPanel.vue, SessionPanel.vue
│   ├── views/           # MainView.vue (single-page layout)
│   ├── stores/          # Pinia stores: workflow.ts, pipeline.ts, session.ts
│   └── services/        # wails.ts (typed wrapper over generated bindings)
├── bindings/            # Auto-generated by wails3 — DO NOT EDIT manually
├── vite.config.ts       # Wails v3 dev server integration; server.port: 5173 (explicit)
├── tsconfig.json        # strict: true
└── package.json         # pnpm-managed; all deps pinned to exact versions (no ^ or ~)

tests/
├── (Go tests co-located with source as *_test.go files per Go convention)
└── frontend/            # Vitest unit tests for Vue components and stores
```

**Structure Decision**: Wails v3 "Option 2 variant" — Go backend in `internal/` + `services/`; TypeScript frontend in `frontend/`. Co-located Go tests (standard Go convention). Frontend tests under `frontend/` via Vitest. Three Wails Services (`WorkflowService`, `PipelineService`, `SessionService`) map directly to the three contract files in `contracts/`.

## Complexity Tracking

> No constitution violations — table left blank.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
