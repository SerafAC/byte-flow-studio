# Research: Serial Data Processing Studio

**Phase**: 0 | **Date**: 2026-03-19 | **Feature**: 001-serial-data-studio

All NEEDS CLARIFICATION items from Technical Context are resolved below.

---

## Decision 1: Desktop Application Framework

**Decision**: Wails v3 alpha (`github.com/wailsapp/wails/v3`)

**Rationale**: Requested by the project owner. Wails v3 compiles Go code into a native desktop app using the platform WebView (Edge on Windows, WebKit on macOS/Linux), with the frontend served as a local web app. v3 introduces auto-generated TypeScript bindings from Go services (via static analyzer) and a typed event system for real-time Go-to-frontend pushes — both critical for this application.

**Key v3 Mechanics**:
- **Services**: Regular Go structs registered in `ApplicationOptions.Services`. The `wails3` CLI generates TypeScript bindings in `frontend/bindings/` automatically on each build/dev run. Methods are called from the frontend as `await WorkflowService.MethodName(args)`.
- **Events**: Typed events emitted from Go via `application.EmitEvent(name, data)` and received in Vue via `Events.On(name, callback)`. This is the mechanism for streaming real-time pipeline data to the frontend without polling.
- **Project Init**: `wails3 init -n "ByteFlowStudio" -t vue` scaffolds the project with a Vue 3 + Vite frontend template.
- **Alpha Caveats**: API is described as "reasonably stable" but subject to change. Pin to a specific commit or tag. Maintain a `wails-upgrade.md` note for tracking any API breaking changes.

**Alternatives Considered**:
- Wails v2: Mature but lacks typed event system and multi-window support; service binding model is weaker.
- Electron: Much larger binary; requires Node.js runtime; not Go-native.
- Tauri (Rust): Not Go; would require rewriting core logic.

---

## Decision 2: Frontend Framework & UI Library

**Decision**: Vue 3.5 (Composition API + TypeScript) + PrimeVue 4.x

**Rationale**: Vue 3 is requested by the project owner. PrimeVue provides a comprehensive component library (dialogs, menus, panels, forms, data tables) covering all standard UI needs without custom implementation. PrimeVue 4.x is Vue 3 native.

**Alternatives Considered**:
- React: Not requested; would conflict with Vue Flow.
- Vuetify: Heavier Material Design library; less appropriate for a technical desktop tool aesthetic.
- Element Plus: Good alternative but PrimeVue has better data table and chart component coverage.

---

## Decision 3: Workflow Canvas / DAG Editor

**Decision**: Vue Flow (`@vue-flow/core` 1.x) — https://vueflow.dev

**Rationale**: Vue Flow is the only mature, actively maintained node-graph/DAG editor library for Vue 3. It supports custom node components (our Analysis blocks render charts inside nodes), pan/zoom, custom edges, programmatic layout via dagre, and multi-select. It uses Vue reactivity for efficient re-rendering (only changed elements update).

**Integration Note**: Vue Flow is the canvas shell. Each block node is a custom `BlockNode.vue` component. Analysis blocks render their uPlot/PrimeVue chart inside the node bounds. For auto-layout, integrate `dagre` (`npm install dagre`) for initial connection layout.

**Limitations**: Not a charting library — embedded charts (uPlot) handle all data visualisation inside nodes.

**Alternatives Considered**:
- Custom SVG canvas: Too much bespoke code; violates YAGNI.
- JointJS / GoJS: Commercial licensing; overkill for our scale.

---

## Decision 4: Real-Time Chart Library

**Decision**: uPlot (`uplot` + `skalinichev/uplot-wrappers`) for all high-frequency streaming charts; PrimeVue Charts (Chart.js) for any low-frequency secondary analytical views.

**Rationale**: uPlot is specifically designed for high-frequency time-series streaming (benchmarked at 60fps with 166,650 points from cold start). At 1,000+ pts/sec it uses only ~10% CPU vs ~40% for Chart.js. The `uplot-wrappers` Vue 3 integration avoids chart recreation on prop changes, using the uPlot incremental update API — critical for smooth streaming.

**PrimeVue Charts (Chart.js) is not suitable** as the primary chart for live data at the frequencies this application targets. It is acceptable for session overview / metadata charts where data is static.

**Integration Note**: uPlot renders a `<canvas>` element. Inside Vue Flow's custom node components, uPlot is mounted imperatively in `onMounted` and updated via `chart.setData(newData, false)` in response to Wails events.

**Alternatives Considered**:
- ECharts: More chart types but ~7x heavier than uPlot at runtime (70% CPU at 60fps test).
- Lightweight Charts (TradingView): OHLC-optimised, not general time-series.
- Chart.js alone: 4x heavier than uPlot; adequate only for static views.

---

## Decision 5: Serial/UART Go Library

**Decision**: `go.bug.st/serial` (import: `go.bug.st/serial`, GitHub: `bugst/go-serial`)

**Rationale**: De-facto standard Go serial library. Single clean interface across Windows, Linux, and macOS. CGO required only for USB enumeration on macOS (IOKit); core serial I/O is pure Go, enabling straightforward cross-compilation. The bundled `enumerator` sub-package lists available serial ports across all platforms — needed for the UART block's port dropdown.

**Alternatives Considered**:
- `tarm/serial`: Largely abandoned; no active maintenance.
- `jacobsa/go-serial`: Maintained but narrower feature set; no port enumeration.

---

## Decision 6: FFT Go Library

**Decision**: `gonum.org/v1/gonum/dsp/fourier`

**Rationale**: Part of the Gonum numerical computing ecosystem (actively maintained, December 2025 release). Provides `FFT` type for real-to-complex and `CmplxFFT` for complex-to-complex transforms. High accuracy (ported from FFTPACK). Gonum will likely also be used for other DSP operations (windowing, signal statistics), so a single dependency covers multiple needs.

**Alternatives Considered**:
- `mjibson/go-dsp/fft`: Written ~2011; low maintenance; slower for single calls due to goroutine overhead.
- `argusdusty/gofft`: Niche; limited community.

---

## Decision 7: Workflow & Session File Format

**Decision**: Single `.byteflow` file = SQLite database (`modernc.org/sqlite`)

**Rationale**: SQLite is endorsed by its own authors as an application file format. For our use case it directly solves the three competing constraints: (1) workflow JSON stored as TEXT column — human-readable and easily queried; (2) raw byte session data stored as BLOBs — no encoding overhead; (3) processed float64 arrays stored as BLOBs (binary IEEE 754) — efficient and queryable; (4) lazy-load of raw layer possible via row-level queries without loading the entire file. `modernc.org/sqlite` is CGO-free and trivially cross-compilable on all three target platforms.

**SQLite Schema Summary** (full detail in data-model.md):
```
workflows    — one row per file (workflow JSON definition)
sessions     — one row per session (metadata: id, start, end, config)
raw_data     — timestamped raw bytes per session per input block
processed_data — timestamped float64 arrays per session per analysis block
```

**Alternatives Considered**:
- Plain JSON + base64: 33% size overhead for binary data; no streaming/random access.
- ZIP archive: Good portability but lacks random access (raw layer lazy-load requires decompressing entire archive).
- MessagePack: Excellent for serialisation but no random-access file format semantics.

---

## Decision 8: Go Logging

**Decision**: `log/slog` (Go stdlib, available since Go 1.21)

**Rationale**: Structured logging with zero external dependencies. Supports JSON output handler for production and text handler for development. Meets Constitution Principle IV. All services and the pipeline engine will use slog with consistent field keys: `block_id`, `session_id`, `flow_state`, `error`.

---

## Decision 9: WebSocket Input

**Decision**: Go stdlib `net/http` + `golang.org/x/net/websocket` or `gorilla/websocket`

**Rationale**: For the WebSocket Input block (client connecting to a remote WebSocket server), `gorilla/websocket` is the most widely used Go WebSocket client/server library (stable, widely adopted). A minor dependency is acceptable given its ubiquity. If minimal dependencies are preferred, `nhooyr.io/websocket` is a modern alternative.

**Final pick**: `gorilla/websocket` — broader documentation and community examples.

---

## Decision 10: Go State Management (Frontend)

**Decision**: Pinia 2.x

**Rationale**: The official Vue 3 state management solution, replacing Vuex. Lightweight, TypeScript-first, and directly supported by Vue DevTools. Three stores map to the three Wails services: `workflow`, `pipeline`, `session`.
