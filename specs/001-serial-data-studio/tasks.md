# Tasks: Serial Data Processing Studio (ByteFlow Studio)

**Input**: Design documents from `/specs/001-serial-data-studio/`
**Prerequisites**: plan.md ✓, spec.md ✓, data-model.md ✓, contracts/ ✓, quickstart.md ✓, research.md ✓

**Tests**: Included per project Constitution (TDD enforced — test-first for each block/service, per plan.md Phase 0/1 gates)

**Organization**: Tasks grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no shared incomplete dependencies)
- **[Story]**: User story label (US1–US6) — maps to spec.md stories
- File paths are relative to repository root

---

## Phase 1: Setup (Project Initialization)

**Purpose**: Initialize Wails v3 project, install all dependencies, create directory skeleton, and configure tooling.

- [ ] T001 Initialize Wails v3 project with Vue 3 template: run `wails3 init -n ByteFlowStudio -t vue` in repo root; verify `main.go`, `app.go`, `wails.json`, and `frontend/` scaffold are created
- [ ] T001b [P] Configure wails.json for pnpm: update the frontend section to set installCommand to "pnpm install", devCommand to "pnpm dev", buildCommand to "pnpm build"; this is required for wails3 dev and wails3 build to invoke pnpm instead of npm
- [ ] T002 Add Go module dependencies: `go get go.bug.st/serial gonum.org/v1/gonum/dsp/fourier modernc.org/sqlite github.com/gorilla/websocket github.com/stretchr/testify`; commit updated `go.mod` and `go.sum`
- [ ] T003 [P] Add frontend pnpm dependencies: cd frontend && pnpm add --save-exact @vue-flow/core @vue-flow/background @vue-flow/controls @vue-flow/minimap primevue @primevue/themes primeicons uplot pinia dagre @types/dagre; verify package.json contains exact version strings (no ^ or ~ prefixes) before committing — required by constitution Technical Standards (dependency pinning)
- [ ] T004 [P] Create Go internal directory skeleton: `internal/input/`, `internal/pipeline/`, `internal/processing/`, `internal/analysis/`, `internal/session/`, `internal/workflow/`, `internal/logging/`, `services/`; add `.gitkeep` where needed
- [ ] T005 [P] Configure `frontend/vite.config.ts` for Wails v3 dev server integration (proxy, HMR port): set `server.port: 5173` and `strictPort: true` so the Wails dev proxy always connects to the correct port; configure `frontend/tsconfig.json` for strict TypeScript (`"strict": true`)
- [ ] T006 [P] Configure `frontend/src/main.ts`: install Pinia, register PrimeVue with Aura preset theme, mount App component to `#app`
- [ ] T007 Implement structured logging in `internal/logging/logger.go`: wrap `log/slog` with a `Logger` type exposing consistent field keys (`block_id`, `session_id`, `flow_state`, `error`, `duration_ms`); read `BYTEFLOW_LOG_LEVEL` (default `INFO`) and `BYTEFLOW_LOG_FORMAT` (`text`/`json`) env vars at init
- [ ] T008 [P] Create `frontend/src/App.vue`: root component providing global PrimeVue Toast provider and `<RouterView>`-equivalent single-page layout; import `MainView.vue` directly (no router needed for single-view app)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types, interfaces, DAG engine scaffold, SQLite schema, block registry, and service skeleton — everything ALL user stories depend on.

**⚠️ CRITICAL**: No user story implementation can begin until this phase is complete.

### Tests for Phase 2

> **Write these tests FIRST — verify they FAIL before implementation**

- [ ] T014 Write unit tests for SQLite schema in `internal/workflow/store_test.go`: open in-memory DB (`":memory:"`), assert all four tables exist, assert both indexes exist, perform insert+select round-trip on the `workflow` table
- [ ] T017 Write unit tests for DAG validation in `internal/pipeline/graph_test.go`: test (a) valid linear graph, (b) graph with direct cycle, (c) graph with indirect cycle, (d) graph with incompatible port types, (e) graph with no valid Input→Analysis path, (f) fan-out graph (1 input → 2 analysis)

### Implementation for Phase 2

- [ ] T009 Define Block interface and all supporting types in `internal/pipeline/types.go`: `Block`, `AnalysisBlock`, `BlockCategory` (`input`/`processing`/`analysis`), `Port`, `PortDir`, `DataType` (`raw-bytes`/`numeric`), `DataChunk` (Timestamp int64, SourceID string, Raw []byte, Values []float64), `BlockError`, `BufferConfig`, `BufferMode` — match `contracts/block-interface.md` exactly
- [ ] T010 [P] Define input source types in `internal/input/types.go`: `InputConfig` placeholder; import and re-export `DataChunk` from `internal/pipeline` package
- [ ] T011 [P] Define workflow persistence types in `internal/workflow/types.go`: `Workflow` (ID, Name, CreatedAt, UpdatedAt, Blocks, Connections, SessionConfig), `BlockDef`, `ConnectionDef` — JSON-serializable structs matching `data-model.md`
- [ ] T012 [P] Define session types in `internal/session/types.go`: `Session`, `SessionMeta`, `SessionConfig` (StoreRaw, StoreProcessed, MaxSessions), `RawDataRecord`, `ProcessedDataRecord` — matching `data-model.md` fields
- [ ] T013 Implement SQLite schema initializer in `internal/workflow/store.go`: `OpenStore(path string) (*Store, error)` opens/creates a `.byteflow` SQLite file using `modernc.org/sqlite` and runs all `CREATE TABLE IF NOT EXISTS` statements for `workflow`, `sessions`, `raw_data` (with index), and `processed_data` (with index) as specified in `data-model.md`
- [ ] T015 Implement block registry in `internal/processing/registry.go`: define `BlockFactory func(id string) Block`, global `Registry map[string]BlockFactory`, and `Register(blockType string, factory BlockFactory)` function
- [ ] T016 Implement DAG graph validation in `internal/pipeline/graph.go`: `ValidateGraph(blocks []BlockDef, connections []ConnectionDef) error` — (1) DFS cycle detection returning `"cycle detected"` error, (2) Kahn's algorithm topological sort, (3) port type compatibility check (source.DataType == target.DataType), (4) at-least-one-Input→Analysis-path check; return `([]string, error)` where first return is topological order of block IDs
- [ ] T018 Implement passthrough block in `internal/processing/passthrough.go`: satisfies `Block` interface; `Run()` copies each received `DataChunk` to output unchanged; registers itself via `init()` calling `registry.Register("passthrough", ...)`; write test in `internal/processing/passthrough_test.go`
- [ ] T019 [P] Create `WorkflowService` struct in `services/workflow_service.go`: fields for `workflowStore *workflow.Store` and `engine *pipeline.Engine`; implement Wails v3 `ServiceName()` method; stub all contract methods returning `fmt.Errorf("not implemented")`
- [ ] T020 [P] Create `PipelineService` struct in `services/pipeline_service.go`: fields for `engine *pipeline.Engine`, `sessionMgr *session.Manager`, `wailsCtx context.Context`; stub all contract methods
- [ ] T021 [P] Create `SessionService` struct in `services/session_service.go`: fields for `sessionMgr *session.Manager`, `sessionStore *session.Store`; stub all contract methods
- [ ] T022 Wire all three services in `main.go`: construct `workflow.Store`, `session.Store`, `session.Manager`, `pipeline.Engine`; pass dependencies into services; register `WorkflowService`, `PipelineService`, `SessionService` via `wails3.NewApplication()` options; remove Wails v3 template boilerplate from `app.go`
- [ ] T023 [P] Create Pinia store scaffolding in `frontend/src/stores/`: `workflow.ts` (state: blocks[], connections[]; actions: addBlock, removeBlock, addConnection, removeConnection, updateBlockParams, updateBlockPosition, undo, redo), `pipeline.ts` (state: flowState, blockErrors, sessionId), `session.ts` (state: sessions[], activeSessionId, viewMode)
- [ ] T024 [P] Create `frontend/src/services/wails.ts`: typed wrapper functions over Wails-generated bindings in `frontend/bindings/`; export named functions `getWorkflow()`, `addBlock()`, `startFlow()`, `listSessions()`, etc. matching all three service contracts
- [ ] T025 Create `frontend/src/views/MainView.vue`: three-panel layout — `BlockLibraryPanel` (left sidebar), `WorkflowCanvas` (center flex), status bar (bottom); include Start/Pause/Resume/Stop toolbar; stub child components with `<!-- TODO -->` placeholders; layout must be responsive

**Checkpoint**: `wails3 dev` launches app window; Go compiles; SQLite schema is created on first run; status bar visible.

---

## Phase 3: User Story 1 — Connect a Data Source and View Live Data (Priority: P1) 🎯 MVP

**Goal**: UART Input block → pipeline engine → Value Display Analysis block; Start/Pause/Resume/Stop controls; live bytes visible in display within 200 ms of hardware receipt.

**Independent Test**: Connect a UART device or serial loopback; drag UART Input + Value Display onto canvas; connect output → input; click Start; confirm live bytes appear in display within 200 ms.

### Tests for User Story 1

> **Write these tests FIRST — verify they FAIL before implementation**

- [ ] T026 [P] [US1] Write unit test for UART input block in `internal/input/uart_test.go`: use a pair of `os.Pipe()` or mock serial port; write bytes to pipe; assert `DataChunk.Timestamp` is within 5 ms of `time.Now().UnixMilli()`, `DataChunk.Raw` matches written bytes, block exits cleanly on context cancellation
- [ ] T027 [P] [US1] Write unit test for pipeline engine 2-block run in `internal/pipeline/engine_test.go`: create minimal stub Input block that emits 5 DataChunks and stub Analysis block that records received chunks; start engine; assert all 5 chunks arrive within 500 ms; assert engine shuts down on context cancel
- [ ] T028 [P] [US1] Write unit test for Value Display analysis block in `internal/analysis/valuedisplay_test.go`: assert ring-buffer respects `MaxSamples`; assert oldest entry evicted when buffer full; assert `Snapshot()` returns all buffered chunks in order; assert that a DataChunk with `Raw []byte{0xDE, 0xAD}` is rendered as `"DE AD"` (hex mode); assert that a DataChunk with `Values []float64{3.14}` is rendered as `"3.14"` (numeric mode)

### Implementation for User Story 1

- [ ] T029 [US1] Implement UART input block in `internal/input/uart.go`: open port using `go.bug.st/serial` with params (port, baudRate, dataBits, stopBits, parity); emit one `DataChunk{Timestamp: now.UnixMilli(), SourceID: id, Raw: []byte{b}}` per received byte; expose output port `"out"` type `raw-bytes`; handle context cancellation; on I/O error send `BlockError` via `errCh` and return; register with `registry.Register("uart", ...)` in `init()`
- [ ] T030 [US1] Implement Value Display analysis block in `internal/analysis/valuedisplay.go`: satisfies `AnalysisBlock` interface; maintains ring buffer using `BufferConfig` (samples/duration mode); expose TWO input ports — `"in-numeric"` type `numeric` and `"in-raw"` type `raw-bytes` — accepting whichever is connected (only one may be connected at a time; validation enforced at graph level); on each received DataChunk emits Wails event `pipeline:data` with `blockId` and `{timestamp, values, raw, mode}` where `mode` is `"numeric"` or `"raw"`; implements `Snapshot() []DataChunk`; register with `registry.Register("value-display", ...)` in `init()` (imports shared registry from `internal/processing`)
- [ ] T031 [US1] Implement pipeline engine in `internal/pipeline/engine.go`: `Engine` struct with `Start(ctx context.Context, wf workflow.Workflow, sessionMgr *session.Manager) error`, `Pause() error`, `Resume() error`, `Stop() error`, `GetState() FlowState`, `GetBlockErrors() map[string]string` methods; on Start: validate DAG via `graph.ValidateGraph()`, build channel graph, launch one goroutine per block in topological order; implement fan-out (broadcast copies to all downstream channels); implement pause via shared `pauseCh chan struct{}`; intercept DataChunks at Input outputs for raw session recording; intercept DataPoints at Analysis inputs for processed recording; emit `pipeline:state-changed` and `pipeline:block-status` Wails events on state changes
- [ ] T032 [US1] Implement session manager in `internal/session/manager.go`: `Manager` struct with `CreateSession(workflowId string, cfg SessionConfig) (*Session, error)`, `CompleteSession(id string, reason string) error`, `AppendRaw(sessionId string, rec RawDataRecord) error`, `AppendProcessed(sessionId string, rec ProcessedDataRecord) error`; on `CompleteSession`, if session count > `MaxSessions`, delete the session with oldest `startTime`
- [ ] T033 [US1] Implement session SQLite store in `internal/session/store.go`: methods `InsertSession`, `UpdateSession`, `DeleteSession`, `InsertRawData`, `InsertProcessedData`, `ListSessions(workflowId string) ([]SessionMeta, error)`, `GetProcessedData(sessionId, blockId string) ([]ProcessedDataRecord, error)`, `CountSessions(workflowId string) (int, error)`, `GetOldestSession(workflowId string) (*Session, error)` — all using `modernc.org/sqlite`
- [ ] T034 [US1] Implement workflow persistence in `internal/workflow/store.go`: `SaveWorkflow(path string, wf Workflow) error` — serialize blocks+connections+sessionConfig as JSON into `definition` TEXT column; upsert `workflow` table; `LoadWorkflow(path string) (Workflow, error)` — open SQLite, read workflow row, deserialize JSON `definition`; check each Input block's hardware availability (list serial ports, check WebSocket URL reachability), set `status = "error"` + `errorMessage = "hardware not available: <port>"` on unreachable blocks
- [ ] T035 [US1] Implement all `WorkflowService` methods in `services/workflow_service.go`: `GetWorkflow()`, `SaveWorkflow(path)`, `LoadWorkflow(path)`, `AddBlock(blockType, x, y)`, `RemoveBlock(blockId)`, `AddConnection(fromBlockId, fromPortId, toBlockId, toPortId)`, `RemoveConnection(connectionId)`, `UpdateBlockParams(blockId, params)`, `UpdateBlockPosition(blockId, x, y)`, `GetAvailableBlockTypes()`, `ListSerialPorts()` — wire to `internal/workflow/store.go` and block registry; enforce "cannot edit while Running" guard
- [ ] T036 [US1] Implement all `PipelineService` methods in `services/pipeline_service.go`: `Start()`, `Pause()`, `Resume()`, `Stop()`, `GetState()`, `GetBlockErrors()` — wire to `internal/pipeline/engine.go` and `internal/session/manager.go`; store Wails context for event emission; log per contract (`INFO pipeline_start`, `INFO pipeline_stop`)
- [ ] T037 [P] [US1] Create `frontend/src/components/canvas/WorkflowCanvas.vue`: wrap `@vue-flow/core` `<VueFlow>` component; initialize nodes/edges from `workflow.ts` Pinia store on mount; call `AddBlock()` on node drop from library; call `AddConnection()` on edge draw-complete; call `UpdateBlockPosition()` on node drag-end; call `RemoveBlock()` / `RemoveConnection()` on delete key; show cycle-error styling on failed connection attempt
- [ ] T038 [P] [US1] Create `frontend/src/components/canvas/BlockNode.vue`: custom Vue Flow `NodeTemplate`; display block label, type badge (Input/Processing/Analysis category color), status indicator dot (idle=grey, connected=green, error=red); render port handles for each InputPort/OutputPort; on double-click show block config panel (slot or teleport)
- [ ] T039 [P] [US1] Create `frontend/src/components/canvas/ConnectionEdge.vue`: custom Vue Flow edge renderer; default bezier curve; on type-mismatch (edge draw rejected) show red dashed preview with tooltip `"Incompatible port types"`
- [ ] T040 [US1] Create `frontend/src/components/blocks/inputs/UartBlockConfig.vue`: port dropdown populated via `ListSerialPorts()` (refresh button included); baud rate `<select>` (common rates + custom input); data bits, stop bits, parity selectors; all changes call `UpdateBlockParams(blockId, params)` on blur/change
- [ ] T041 [US1] Create `frontend/src/components/blocks/analysis/ValueDisplayBlock.vue`: subscribes to `pipeline:data` Wails event filtered by `blockId`; when `mode === "numeric"` shows latest `values[0]` formatted to `decimals` decimal places with `unit` suffix and `label`; when `mode === "raw"` shows bytes as space-separated hex (e.g. `"DE AD BE EF"`); updates DOM within 200 ms; optionally shows mini sparkline when `showHistory = true` using last 20 buffered values (numeric mode only); buffer config UI: samples/duration toggle + count/seconds number input (FR-028)
- [ ] T042 [US1] Update `pipeline.ts` Pinia store: subscribe to `pipeline:state-changed` and `pipeline:block-status` Wails events on app mount; expose `startFlow()`, `pauseFlow()`, `resumeFlow()`, `stopFlow()` actions that call corresponding `pipelineService.*` wrapper; expose computed `canStart`, `canPause`, `canResume`, `canStop` based on flowState
- [ ] T043 [US1] Update `workflow.ts` Pinia store: call `getWorkflow()` on app mount and populate `blocks` and `connections`; implement `addBlock`, `removeBlock`, `addConnection`, `removeConnection`, `updateBlockParams`, `updateBlockPosition` actions (call service, then update local state on success); implement `undo()` / `redo()` with a history stack (FR-007)
- [ ] T044 [P] [US1] Create `frontend/src/components/panels/BlockLibraryPanel.vue`: vertical panel with category sections (Input, Processing, Analysis); each block type shown as draggable card with label + description from `GetAvailableBlockTypes()`; on drag-end over canvas calls `AddBlock(blockType, x, y)`; searchable by name; for Phase 3 MVP must show at minimum: UART Input and Value Display
- [ ] T045 [US1] Wire toolbar in `frontend/src/views/MainView.vue`: Start button calls `pipeline.startFlow()` (disabled unless `canStart`); Pause calls `pauseFlow()` (disabled unless `canPause`); Resume calls `resumeFlow()` (disabled unless `canResume`); Stop calls `stopFlow()` (disabled unless `canStop`); status badge shows current `flowState`; error banner shows `blockErrors` when flowState is `"error"`

**Checkpoint**: US1 fully functional — UART → Value Display pipeline starts, live data visible within 200 ms, Pause/Resume/Stop work correctly.

---

## Phase 4: User Story 2 — Build a Multi-Step Processing Pipeline (Priority: P2)

**Goal**: Signal Simulator → Moving Average → Line Chart + Data Table fan-out; pipeline validation prevents unconnected processing blocks.

**Independent Test**: Add Simulator + Moving Average + Line Chart + Data Table; connect Simulator → MovingAverage → fan-out to both analysis blocks; Start; confirm both update live. Change window size parameter; confirm chart updates within one render cycle.

### Tests for User Story 2

> **Write these tests FIRST — verify they FAIL before implementation**

- [ ] T046 [P] [US2] Write unit tests for Moving Average block in `internal/processing/movingaverage_test.go`: feed known values, assert rolling average is correct for window sizes 1, 5, 10; assert `Timestamp` preserved from input; assert `Configure()` with new `windowSize` takes effect immediately on next chunk
- [ ] T047 [P] [US2] Write unit tests for Signal Simulator in `internal/input/simulator_test.go`: assert sine waveform emits exactly `sampleRateHz` chunks per second (within 5% tolerance over 1 second); assert `Values[0]` set (not `Raw`); assert each chunk's `Timestamp` is monotonically increasing
- [ ] T048 [P] [US2] Write unit tests for Byte Parser in `internal/processing/byteparser_test.go`: test float32-le parsing (4-byte frames → float64 value), int16-be (2-byte), uint8 (1-byte); test multi-channel (`channels=2`): 8-byte frame → `Values` of length 2; test partial frame buffering (feed 3 bytes of a 4-byte frame, then 1 more → one chunk emitted)
- [ ] T049 [P] [US2] Write unit tests for FFT block in `internal/processing/fft_test.go`: feed a 512-sample pure sine wave at 10 Hz (sample rate 100 Hz); assert dominant frequency bin in output is bin 51 (10 Hz); assert Hann window applied when `windowFunction = "hann"`; assert output DataChunk `Values` length equals `windowSize/2`
- [ ] T050 [P] [US2] Write unit tests for Summation and Scaling in `internal/processing/summation_test.go` and `internal/processing/scaling_test.go`: Summation: assert cumulative sum after N inputs; Scaling: assert `output = input*scale + offset` for edge values (zero, negative, large)

### Implementation for User Story 2

- [ ] T051 [US2] Implement Signal Simulator input block in `internal/input/simulator.go`: generate sine/square/sawtooth/noise waveforms at `sampleRateHz` using a ticker; emit `DataChunk{Timestamp: now.UnixMilli(), SourceID: id, Values: []float64{sample}}` per tick; expose port `"out"` type `numeric`; handle context cancellation; register `"simulator"` in `init()`
- [ ] T052 [US2] Implement Moving Average block in `internal/processing/movingaverage.go`: maintain circular buffer of `windowSize` float64 samples; on each DataChunk compute average of buffer; emit `DataChunk` with same `Timestamp` and `SourceID` as input; support `Configure()` hot-reload of `windowSize` while paused; register `"moving-average"` in `init()`
- [ ] T053 [P] [US2] Implement Summation block in `internal/processing/summation.go`: maintain running sum; emit DataChunk with same Timestamp and running sum as `Values[0]`; register `"summation"` in `init()`
- [ ] T054 [P] [US2] Implement Value Scaling/Offset block in `internal/processing/scaling.go`: `output = input*scale + offset`; preserve Timestamp; register `"scaling"` in `init()`
- [ ] T055 [P] [US2] Implement Byte Parser block in `internal/processing/byteparser.go`: buffer incoming `Raw` bytes; when buffer contains `frameSize` bytes, parse `channels` values per frame using `format` (float32-le, int16-be, uint8, etc.); emit `DataChunk{Values: parsedValues, Timestamp: inputTimestamp}`; input port `"in"` type `raw-bytes`, output port `"out"` type `numeric`; register `"byte-parser"` in `init()`
- [ ] T056 [US2] Implement FFT processing block in `internal/processing/fft.go`: accumulate `windowSize` float64 samples in buffer; when full, apply window function (`none`/`hann`/`hamming`) using `gonum.org/v1/gonum/dsp/fourier`; compute real FFT; emit magnitude spectrum as `DataChunk{Values: magnitudes, Timestamp: midpointTimestamp}`; register `"fft"` in `init()`
- [ ] T057 [US2] Implement Line Chart analysis block in `internal/analysis/linechart.go`: satisfies `AnalysisBlock`; ring buffer with `BufferConfig` (samples or duration); on each DataChunk emits `pipeline:data` Wails event with `blockId` and `{timestamp, values}[]`; input port `"in"` type `numeric`; register `"line-chart"` in `init()` (imports shared registry from `internal/processing`)
- [ ] T058 [US2] Implement Data Table analysis block in `internal/analysis/datatable.go`: satisfies `AnalysisBlock`; ring buffer; emits `pipeline:data` events with all channel values per chunk; input port `"in"` type `numeric`; register `"data-table"` in `init()` (imports shared registry from `internal/processing`)
- [ ] T059 [US2] Add fan-out test to `internal/pipeline/engine_test.go`: create 1 stub Input → 1 stub Processing → 2 stub Analysis blocks; run engine; assert both Analysis blocks receive every DataChunk emitted by the Input; no DataChunk lost or duplicated
- [ ] T060 [US2] Implement pipeline start validation for unconnected blocks in `services/pipeline_service.go`: on `Start()`, check that every Processing block in the workflow has at least one connected input port and one connected output port that forms part of a valid Input→Analysis path; return `"block <id> (<type>): not reachable from any Input→Analysis path"` and emit `pipeline:block-status` with `status: "error"` for the offending block(s)
- [ ] T061 [P] [US2] Create `frontend/src/components/blocks/inputs/SimulatorBlockConfig.vue`: waveform `<select>` (sine/square/sawtooth/noise), frequency Hz, amplitude, DC offset, sample rate Hz number inputs; calls `UpdateBlockParams` on change
- [ ] T062 [P] [US2] Create `frontend/src/components/blocks/processing/MovingAverageConfig.vue`: `windowSize` number input (min 1); calls `UpdateBlockParams` on change
- [ ] T063 [P] [US2] Create `frontend/src/components/blocks/processing/FFTConfig.vue`: `windowSize` select (128/256/512/1024/2048), `windowFunction` select (none/hann/hamming); calls `UpdateBlockParams`
- [ ] T064 [P] [US2] Create `frontend/src/components/blocks/processing/ScalingConfig.vue`: `scale` and `offset` number inputs with default 1.0 and 0.0; calls `UpdateBlockParams`
- [ ] T065 [P] [US2] Create `frontend/src/components/blocks/processing/ByteParserConfig.vue`: `format` select (float32-le/float32-be/int16-le/int16-be/uint8), `channels` number (min 1), `frameSize` number (auto-calculated from format with override option); calls `UpdateBlockParams`
- [ ] T066 [US2] Create `frontend/src/components/blocks/analysis/LineChartBlock.vue`: uPlot streaming time-series chart; subscribes to `pipeline:data` Wails event filtered by `blockId`; maintains client-side ring buffer of `bufferSamples` or `bufferDurationSec` seconds of data; re-renders incrementally (append only, no full redraw); configurable title, xLabel, yLabel, unit, decimals; targets ≤10% CPU at 1,000 pts/sec
- [ ] T067 [P] [US2] Create `frontend/src/components/blocks/analysis/DataTableBlock.vue`: PrimeVue `DataTable` with virtual scroll; shows `timestamp` (formatted) and one column per channel; respects `maxRows`; updates from `pipeline:data` events; appends rows, evicts oldest when full
- [ ] T068 [US2] Expand `frontend/src/components/panels/BlockLibraryPanel.vue`: add all Processing block types (Moving Average, Summation, FFT, Scaling, Byte Parser, Passthrough) and additional Analysis types (Line Chart, Data Table) to the categorized library with drag-to-canvas support

**Checkpoint**: US2 fully functional — Simulator → Moving Average → Line Chart + Data Table fan-out runs live; window size change reflects in one render cycle; unconnected blocks blocked at Start.

---

## Phase 5: User Story 5 — Multiple Inputs and Analysis Blocks (Priority: P2)

**Goal**: WebSocket Input block live alongside UART; both inputs independently route to their own analysis blocks and merge into a shared FFT Spectrum block; Bar Chart and FFT Spectrum block types added.

**Independent Test**: Two Simulator inputs wired independently to their own Line Charts, plus both merged into an FFT Spectrum block; all three analysis blocks update concurrently during a live run.

### Tests for User Story 5

- [ ] T069 [P] [US5] Write unit tests for WebSocket input block in `internal/input/websocket_test.go`: spin up `httptest.NewServer` with WebSocket handler; assert DataChunks emitted with correct timestamp and Raw bytes per message; write a test for auto-reconnect: close server, restart, assert block re-establishes connection within `reconnectIntervalMs + 100ms`
- [ ] T070 [P] [US5] Write multi-input engine test in `internal/pipeline/engine_test.go`: two stub Input blocks each emit 3 DataChunks → both connected to one stub Processing block (fan-in); assert Processing block receives 6 total DataChunks (3 from each input, interleaved); no chunks lost

### Implementation for User Story 5

- [ ] T071 [US5] Implement WebSocket input block in `internal/input/websocket.go`: connect using `github.com/gorilla/websocket`; emit one `DataChunk{Timestamp: now.UnixMilli(), SourceID: id, Raw: msg}` per received message; on disconnect, sleep `reconnectIntervalMs` and retry; after 5 consecutive failed reconnects emit `BlockError` and transition to error state; expose port `"out"` type `raw-bytes`; register `"websocket"` in `init()`
- [ ] T072 [US5] Implement Bar Chart analysis block in `internal/analysis/barchart.go`: satisfies `AnalysisBlock`; ring buffer; aggregates latest N values (most recent `bufferSamples`) per channel; emits `pipeline:data` events; input port `"in"` type `numeric`; register `"bar-chart"` in `init()` (imports shared registry from `internal/processing`)
- [ ] T073 [US5] Implement FFT Spectrum analysis block in `internal/analysis/fftspectrum.go`: satisfies `AnalysisBlock`; expects frequency-domain input (from FFT processing block); ring buffer for latest spectrum; emits `pipeline:data` events with magnitude array; input port `"in"` type `numeric`; register `"fft-spectrum"` in `init()` (imports shared registry from `internal/processing`)
- [ ] T074 [US5] Verify multi-input (fan-in) support in `internal/pipeline/engine.go`: when multiple connections terminate at the same input port of a Processing block, merge all upstream channels onto a single receive channel using a fan-in goroutine; extend engine_test.go if not already covered by T070 test
- [ ] T075 [P] [US5] Create `frontend/src/components/blocks/inputs/WebSocketBlockConfig.vue`: URL text input (validated as `ws://` or `wss://`), subprotocol text, reconnect interval ms number input; calls `UpdateBlockParams`
- [ ] T076 [P] [US5] Create `frontend/src/components/blocks/analysis/BarChartBlock.vue`: uPlot or PrimeVue Chart bar chart; subscribes to `pipeline:data` for its `blockId`; shows latest value per channel as bars; configurable title, yLabel, unit, decimals; buffer config UI: samples/duration toggle + count/seconds number input (FR-028)
- [ ] T077 [US5] Create `frontend/src/components/blocks/analysis/FftSpectrumBlock.vue`: uPlot frequency-domain chart (X = frequency Hz bins, Y = magnitude); subscribes to `pipeline:data` for its `blockId`; togglable log-scale Y axis; configurable xLabel, yLabel, decimals
- [ ] T078 [US5] Update `frontend/src/components/panels/BlockLibraryPanel.vue`: add WebSocket Input, Bar Chart, FFT Spectrum Viewer to the library drag palette

**Checkpoint**: US5 functional — two simultaneous inputs route independently; fan-in merge works; all five analysis block types render correctly.

---

## Phase 6: User Story 3 — Manage and Persist Workflows (Priority: P3)

**Goal**: Save workflow + all session data to a single `.byteflow` file; reopen on same or different machine; degraded-state open for unavailable hardware.

**Independent Test**: Build a workflow, run it to collect data, save it, close app, reopen — canvas layout, block configs, connections, and session data all restored.

### Tests for User Story 3

- [ ] T079 [P] [US3] Write integration test for SaveWorkflow/LoadWorkflow round-trip in `internal/workflow/store_test.go`: save a Workflow with 3 blocks (UART, MovingAverage, LineChart), 2 connections, and non-default SessionConfig; reload from file; assert all block params, connection IDs, and SessionConfig fields are identical
- [ ] T080 [P] [US3] Write degraded-state test in `internal/workflow/store_test.go`: save workflow with UART block using port `"/dev/nonexistent999"`; reload; assert UART block `status == "error"` and `errorMessage` contains `"hardware not available"`; assert other blocks load normally

### Implementation for User Story 3

- [ ] T081 [US3] Complete `WorkflowService.SaveWorkflow(path string)` in `services/workflow_service.go`: validate path ends in `.byteflow`; serialize full workflow (blocks, connections, sessionConfig) as JSON into `workflow.definition` column; upsert `workflow` table row; set `updatedAt = now()`; log `INFO save_workflow path=<path> blocks=<n> connections=<n>`
- [ ] T082 [US3] Complete `WorkflowService.LoadWorkflow(path string)` in `services/workflow_service.go`: open SQLite; read workflow row; deserialize JSON `definition`; index session metadata from `sessions` table (`processed_data` loaded on-demand; `raw_data` NOT loaded — lazy per SC-005); call hardware availability check for each Input block; emit `workflow:loaded` internal event; log warnings for unavailable hardware via `WARN block_hardware_unavailable`
- [ ] T083 [US3] Implement session data recording in `internal/pipeline/engine.go` (T032 session manager and T033 session store from Phase 3 are now complete — this task depends on both): batch `AppendRaw()` and `AppendProcessed()` calls every 100 ms using a ticker-driven flush goroutine; reduce SQLite write contention; accumulate DataChunks in-memory between flushes
- [ ] T084 [US3] Implement `SessionService.UpdateSessionConfig(config SessionConfig)` in `services/session_service.go`: validate `MaxSessions >= 1`; validate at least one layer enabled; persist updated config back into `workflow.definition` JSON column; if MaxSessions decreased, trim oldest sessions immediately; log `INFO session_config_updated`
- [ ] T085 [US3] Implement `SessionService.GetStorageEstimate()` in `services/session_service.go`: sum `data_volume` from `sessions` table; compute `AverageSessionBytes`; compute `ProjectedMaxBytes = AverageSessionBytes * MaxSessions`; compare to `WarningThresholdBytes` (default 1 GB); set `ExceedsThreshold`
- [ ] T086 [US3] Implement storage threshold warning: in `PipelineService.Stop()`, after session completed call `SessionService.GetStorageEstimate()`; if `ExceedsThreshold`, emit Wails event `session:storage-warning` with estimate data; display PrimeVue Toast warning in frontend `pipeline.ts` store subscriber
- [ ] T087 [US3] Add file open/save dialogs in `frontend/src/views/MainView.vue`: menu bar with items "New Workflow", "Open… (Ctrl+O)", "Save (Ctrl+S)", "Save As…"; use Wails v3 `wails.OpenFileDialog()` and `wails.SaveFileDialog()` filtered to `*.byteflow`; on open call `loadWorkflow(path)` from `wails.ts` and refresh `workflow.ts` store; on save call `saveWorkflow(path)`; show PrimeVue ConfirmDialog if unsaved changes on New/Open
- [ ] T088 [P] [US3] Update `frontend/src/components/canvas/WorkflowCanvas.vue`: apply orange border styling to nodes with `status == "error"`; show PrimeVue Tooltip with `errorMessage` on hover; for degraded Input blocks show "Reconfigure" button in block config panel that re-opens UartBlockConfig / WebSocketBlockConfig
- [ ] T089 [P] [US3] Create `frontend/src/components/panels/SessionPanel.vue` (Phase 6 scope): session recording config toggle (raw layer / processed layer / both); MaxSessions number input; storage estimate bar (current bytes / projected max bytes); "Exceeds threshold" warning badge; all changes call `UpdateSessionConfig()`; subscribe to `session:storage-warning` to show alert

**Checkpoint**: US3 functional — save/load round-trip restores all data; degraded-state open shows error blocks; storage threshold warns user.

---

## Phase 7: User Story 6 — Review a Past Data Session (Priority: P3)

**Goal**: Session list visible with metadata; switch to any retained historical session; Analysis blocks display historical data statically; starting a new live run auto-switches to live mode.

**Independent Test**: Run flow → stop (session 1); run again → stop (session 2); switch to session 1 in session panel; confirm all Analysis blocks show session 1 data.

### Tests for User Story 6

- [ ] T090 [P] [US6] Write unit test for `SessionService.SetViewSession()` in `services/session_service_test.go`: mock session store with 3 ProcessedDataRecords; call SetViewSession; assert `session:view-changed` event emitted with correct sessionId and `mode: "historical"`; assert `pipeline:data` events replayed for all stored records
- [ ] T091 [P] [US6] Write session retention test in `internal/session/manager_test.go`: call `CompleteSession()` 11 times with `MaxSessions = 10`; after 11th call assert only 10 sessions remain in store; assert the session with the oldest `startTime` was deleted

### Implementation for User Story 6

- [ ] T092 [US6] Implement `SessionService.ListSessions()` in `services/session_service.go`: query `sessions` table WHERE `workflow_id = currentWorkflowId` ORDER BY `start_time DESC`; map rows to `[]SessionMeta`; log `INFO session_list` with count
- [ ] T093 [US6] Implement `SessionService.SetViewSession(sessionId string)` in `services/session_service.go`: validate flow is `Idle` or `Paused`; load all `processed_data` rows for the session from SQLite (not lazy — full load for historical view); emit `session:view-changed` with `{sessionId, mode: "historical"}` **first** (frontend components must receive this before any data events so they can set their local mode flag); then replay stored DataPoints as `pipeline:data` Wails events with payload `{blockId, timestamp, values, replay: true}` (batched, respecting 60fps throttle) to all Analysis blocks; the `replay: true` field distinguishes replayed events from live events at the component level — live events always omit this field (or carry `replay: false`); log `INFO session_view_switch`
- [ ] T094 [US6] Implement `SessionService.GetActiveSessionID()` and `SessionService.DeleteSession(sessionId)` in `services/session_service.go`: GetActiveSessionID returns in-memory active session ID from manager; DeleteSession validates not the active session, deletes from SQLite (cascade to `raw_data` and `processed_data`), emits `session:list-updated`; log `INFO session_deleted`
- [ ] T095 [US6] Add historical mode support to all Analysis block Vue components (`ValueDisplayBlock.vue`, `LineChartBlock.vue`, `DataTableBlock.vue`, `BarChartBlock.vue`, `FftSpectrumBlock.vue`): each component subscribes to the `session:view-changed` Wails event and sets a local `isHistorical` boolean flag (`true` when `mode === "historical"`, `false` when `mode === "live"`); on the first `pipeline:data` event carrying `replay: true` while `isHistorical` is set, clear the current ring buffer; accumulate all subsequent replayed chunks into the buffer and render statically (do not update on further live events until `isHistorical` is cleared); when a `session:view-changed` event arrives with `mode: "live"`, clear `isHistorical` and resume normal live-update behaviour; note: Go backend Analysis block structs (`internal/analysis/*.go`) need no changes — historical replay is handled entirely in the frontend components via the `replay` field on `pipeline:data` events
- [ ] T096 [US6] In `PipelineService.Start()`: if `session.ts` viewMode is `"historical"`, auto-switch viewMode back to `"live"` by emitting `session:view-changed` with `{sessionId: newSessionId, mode: "live"}` before starting the engine
- [ ] T097 [US6] Update `session.ts` Pinia store: subscribe to `session:view-changed` and `session:list-updated` Wails events on mount; expose `switchSession(id: string)` action (calls `setViewSession(id)` from wails.ts); expose `deleteSession(id: string)` action; track `viewMode: "live" | "historical"` and update `activeSessionId` on events
- [ ] T098 [US6] Expand `frontend/src/components/panels/SessionPanel.vue` (full US6 scope): session list section with rows showing start time, end time (or "Active"), data volume (formatted as KB/MB), end reason; "View" button per row (calls `switchSession()`); "Delete" button per row (disabled for active session, shows confirm dialog); active session highlighted; list auto-refreshes on `session:list-updated` event

**Checkpoint**: US6 functional — session list shows all retained sessions; switching to historical session loads and displays data statically; new Start auto-returns to live mode.

---

## Phase 8: User Story 4 — Full-Screen Analysis View (Priority: P3)

**Goal**: Any Analysis block can expand to fill the entire application window; live data continues updating in full-screen; Escape or close button returns to canvas.

**Independent Test**: Start flow with Line Chart; right-click → "Full Screen"; chart fills window; data continues updating; Escape returns to canvas view with flow still running.

### Tests for User Story 4

- [ ] T099 [US4] Write Vitest component test in `frontend/src/components/blocks/analysis/LineChartBlock.test.ts`: mount component with minimal props; assert that calling `enterFullscreen()` method emits `"fullscreen:enter"` event with correct `blockId`; assert that `Escape` keydown on the document fires `"fullscreen:exit"` event

### Implementation for User Story 4

- [ ] T100 [US4] Add full-screen support to `frontend/src/components/blocks/analysis/LineChartBlock.vue`: add right-click context menu (PrimeVue ContextMenu) with "Full Screen" item; on select emit `fullscreen:enter` with `{blockId}`; add an `isFullscreen` prop/state; when fullscreen, component continues subscribing to `pipeline:data` events; emit `fullscreen:exit` when Escape pressed (document keydown listener, removed on component unmount)
- [ ] T101 [P] [US4] Add identical full-screen context menu and event emission to `frontend/src/components/blocks/analysis/ValueDisplayBlock.vue`, `BarChartBlock.vue`, `DataTableBlock.vue`, and `FftSpectrumBlock.vue`
- [ ] T102 [US4] Implement full-screen overlay manager in `frontend/src/views/MainView.vue`: listen for `fullscreen:enter` events bubbled from WorkflowCanvas; when received, teleport (Vue `<Teleport to="body">`) the triggering Analysis block component into a `position: fixed; inset: 0; z-index: 9999` overlay with a close button and Escape key handler; when `fullscreen:exit` received, destroy overlay and return focus to canvas; flow engine continues running uninterrupted

**Checkpoint**: US4 functional — all five Analysis block types support full-screen; live data updates in full-screen; Escape returns to canvas.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Edge-case handling, error UX hardening, performance validation, and final integration verification.

> **TDD gate**: Write each test task (T103a, T104a, etc.) first — verify it FAILS before the corresponding implementation task.

### Tests for Phase 9

- [ ] T103a [P] Write unit test for UART disconnect → BlockError in `internal/input/uart_test.go`: simulate an `io.EOF` mid-read on the mock pipe; assert `BlockError` is sent to `errCh` with the correct `BlockID`; assert the block goroutine exits cleanly; assert no BlockError is sent to other independent block channels
- [ ] T104a [P] Write unit test for backpressure / frame-drop in `internal/pipeline/engine_test.go`: create a stub Input that emits chunks faster than a slow stub downstream can consume (block the downstream goroutine); assert that after 100 consecutive drops a `pipeline:block-status` event is captured with a message matching `"frames dropped: 100"`; assert the engine does not deadlock or panic
- [ ] T106a [P] Write unit test for corrupted `.byteflow` recovery in `internal/workflow/store_test.go`: write a file with invalid SQLite header bytes to a temp path; call `LoadWorkflow(path)`; assert it returns an error whose message contains `"not a valid byteflow file"`; call `LoadWorkflow` again with `recoverySessions=false` flag; assert it succeeds (empty session list) without panicking
- [ ] T107a [P] Write unit test for FR-004 enforcement in `services/pipeline_service_test.go`: call `Start()` on a workflow with only a Processing block and no Input or Analysis block; assert the returned error message is `"at least one Input and one Analysis block must be connected"`; assert `pipeline:block-status` is emitted with `status: "error"` for the offending block(s)
- [ ] T108a Write Vitest component test in `frontend/src/components/canvas/WorkflowCanvas.test.ts`: mount component with 3 nodes; simulate shift+click selecting 2 nodes; simulate Delete keydown; assert `RemoveBlock` was called twice (once per selected node) and a single combined undo entry was pushed to the history stack

### Implementation for Phase 9

- [ ] T103 [P] Handle UART hardware disconnect during active run in `internal/input/uart.go`: detect `io.EOF` or non-nil read error; send `BlockError{BlockID: id, Err: err}` to `errCh`; transition block status to `"error"` via `pipeline:block-status` event; do NOT stop other independent input blocks
- [ ] T104 [P] Implement graceful backpressure / frame-drop in `internal/pipeline/engine.go`: use non-blocking channel sends with `select { case ch <- chunk: default: dropped++ }`; after 100 consecutive drops emit `pipeline:block-status` with `message: fmt.Sprintf("frames dropped: %d", dropped)` for the source block; reset counter after emission
- [ ] T105 [P] Cycle prevention in `frontend/src/components/canvas/WorkflowCanvas.vue`: on Vue Flow `connect` event, call `addConnection()` service; if returns `"connection would create a cycle"` error, reject the edge (do not add to Vue Flow state) and show PrimeVue Toast `"Circular connection is not allowed"`
- [ ] T106 [P] Corrupted/incompatible `.byteflow` file recovery in `internal/workflow/store.go` and `frontend/src/views/MainView.vue`: if `LoadWorkflow()` returns `"not a valid byteflow file"` or schema version error, show PrimeVue Dialog with two options: "Recovery Mode" (retry LoadWorkflow with sessions discarded) and "Cancel"; log `ERROR load_workflow_failed path=<path> reason=<reason>`
- [ ] T107 [P] Frontend FR-004 error surface in `frontend/src/views/MainView.vue`: when `startFlow()` returns `"at least one Input and one Analysis block must be connected"` (already enforced server-side by T060's DAG validation), display a persistent error banner above the canvas (not just a toast); banner must name the offending block(s) from the `blockErrors` map; dismisses on successful start. *(Note: server-side validation logic lives in T060; this task is display only.)*
- [ ] T108 Implement FR-006 multi-block selection delete in `frontend/src/components/canvas/WorkflowCanvas.vue`: enable Vue Flow multi-select (shift+click, drag-select); bind Delete / Backspace key to batch `RemoveBlock()` + `RemoveConnection()` calls for all selected elements; push combined undo entry to history stack
- [ ] T109 [P] Verify Analysis block `bufferMode` / `bufferSamples` / `bufferDurationSec` round-trips through `SaveWorkflow` / `LoadWorkflow` (FR-028): add assertions to `internal/workflow/store_test.go` for (a) LineChart block with `bufferMode="duration"`, `bufferDurationSec=45.0`; (b) ValueDisplay block with `bufferMode="samples"`, `bufferSamples=500`; (c) BarChart block with `bufferMode="duration"`, `bufferDurationSec=10.0` — all must persist and restore identically
- [ ] T111 Performance validation: run `wails3 dev`; connect two Simulator inputs at 500 pts/sec each (total 1,000 pts/sec); verify with browser DevTools that `LineChartBlock.vue` CPU stays ≤10% (SC-004); verify canvas renders at ≥30 fps with 20 blocks (SC-004); if CPU/fps fails, optimize `pipeline:data` event batching threshold in engine (currently 16 ms / 60fps — try 32 ms)
- [ ] T112 [P] Audit all user-visible error messages: ensure no raw Go error stack traces appear in `pipeline:state-changed` or `pipeline:block-status` events; strip stack traces using `errors.Unwrap()` / `err.Error()` only; verify SC-007 compliance for all `blockErrors` map values
- [ ] T113 Run full quickstart.md validation: execute `wails3 doctor`, `go test -race ./...`, `cd frontend && pnpm test`; fix all failing tests; confirm app builds via `wails3 build`; measure cold launch time by running the built binary with `time` and assert canvas appears within 3 000 ms (SC-003); if launch exceeds 3 s, profile startup (pprof or Wails lifecycle timestamps) and defer non-critical init work
- [ ] T114 [P] Implement the '+' quick-add button in `frontend/src/components/canvas/BlockNode.vue` (FR-003): render a `+` icon button adjacent to each output port handle; show the button only while the parent block is hovered (CSS `group-hover` or Vue `mouseenter`/`mouseleave`); button MUST NOT interfere with port-handle drag events (use `@mousedown.stop` on button only, not on port handle); on click, open a `QuickAddMenu` popover anchored to the output port; write Vitest test asserting button visibility toggles on block hover and that `mousedown` on the port handle is not consumed by the button
- [ ] T115 [P] Implement `QuickAddMenu.vue` in `frontend/src/components/canvas/`: PrimeVue Popover containing a searchable list of block types filtered to those with at least one input port whose `DataType` matches the source output port's `DataType` (query `GetAvailableBlockTypes()` then filter client-side); on block-type selection: call `AddBlock(blockType, sourceX + 250, sourceY)` then call `AddConnection(sourceBlockId, sourcePortId, newBlockId, compatibleInputPortId)`; dismiss popover on Escape or outside click; write Vitest test asserting the compatibility filter (raw-bytes source hides numeric-only blocks, and vice versa)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Requires Phase 1 complete — **BLOCKS all user story phases**
- **US1 (Phase 3)**: Requires Phase 2 complete — first MVP deliverable
- **US2 (Phase 4)**: Requires Phase 2 complete; can run in parallel with US1 after Phase 2
- **US5 (Phase 5)**: Requires Phase 2 complete; mostly independent of US1/US2
- **US3 (Phase 6)**: Requires US1 (T031 engine, T032–T033 session infra) before T083
- **US6 (Phase 7)**: Requires US3 complete (needs persisted sessions)
- **US4 (Phase 8)**: Requires US1 Analysis block components (T037–T041); otherwise independent
- **Polish (Phase 9)**: Requires all user story phases complete

### User Story Dependencies (Summary)

| Story | Depends On | Can Start After |
|---|---|---|
| US1 (P1) | Phase 2 | T025 |
| US2 (P2) | Phase 2 | T025 (independent of US1) |
| US5 (P2) | Phase 2 | T025 (independent of US1/US2) |
| US3 (P3) | US1 (T031–T033) | T033 |
| US6 (P3) | US3 complete | T089 |
| US4 (P3) | US1 (T041) | T041 |

### Within Each Phase

1. Write tests first (failing red before implementation)
2. Types/interfaces before implementations
3. Internal packages before service layer
4. Service layer before frontend components
5. Config components can run in parallel once the block is registered

### Parallel Opportunities

- **Phase 1**: T003, T004, T005, T006, T008 all in parallel after T001 and T002
- **Phase 2**: T010–T012 in parallel; T019–T021 in parallel after T009; T023–T024 in parallel after T022
- **Phase 3**: T026–T028 tests in parallel; T029–T030 implementations in parallel; T037–T039 canvas components in parallel; T040–T041 config+analysis components in parallel
- **Phase 4**: T046–T050 tests all in parallel; T053–T055 implementations in parallel; T061–T065 config components in parallel
- **Phase 5**: T069–T070 tests in parallel; T075–T076 components in parallel
- **Phase 9**: T103–T109, T111–T115 all in parallel (different files, independent concerns)

---

## Parallel Example: User Story 1

```bash
# Step 1 — Write all US1 tests together (parallel):
T026: internal/input/uart_test.go
T027: internal/pipeline/engine_test.go
T028: internal/analysis/valuedisplay_test.go

# Step 2 — Implement backend (after tests fail):
T029: internal/input/uart.go         ──┐ parallel
T030: internal/processing/valuedisplay.go ──┘

# Step 3 — Service layer (after T029–T033):
T035: services/workflow_service.go
T036: services/pipeline_service.go   (depends on T031)

# Step 4 — Frontend canvas (parallel, can start once T009 types available):
T037: frontend/src/components/canvas/WorkflowCanvas.vue  ──┐
T038: frontend/src/components/canvas/BlockNode.vue        ──┤ parallel
T039: frontend/src/components/canvas/ConnectionEdge.vue   ──┘

# Step 5 — Block config panels (parallel, can start once service stubs exist):
T040: UartBlockConfig.vue   ──┐ parallel
T041: ValueDisplayBlock.vue ──┘
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational — **CRITICAL, blocks everything**
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: UART → Value Display pipeline; Start/Pause/Resume/Stop; 200 ms latency (SC-002); basic usability check (SC-001, SC-006)
5. Demo or ship MVP if validated

### Incremental Delivery

1. Setup + Foundational → project compiles, SQLite schema ready
2. US1 (P1) → live UART data visible **(MVP!)**
3. US2 (P2) → full processing pipeline with Simulator (all Processing blocks, Line Chart, Data Table)
4. US5 (P2) → WebSocket input, Bar Chart, FFT Spectrum, multi-input topologies
5. US3 (P3) → save/load workflows with session data
6. US6 (P3) → session history browser and historical playback
7. US4 (P3) → full-screen analysis view
8. Polish → edge cases, error UX, performance (SC-004: ≥30 fps / ≤10% CPU at 1,000 pts/sec)

### Parallel Team Strategy

After Phase 2 (Foundational) is complete:
- **Developer A**: US1 (T026–T045) — UART pipeline MVP
- **Developer B**: US2 processing blocks (T046–T068) — all 6 block types + charts
- **Developer C**: US5 WebSocket + chart types (T069–T078)
- Merge into US3 once all three complete (session recording requires full pipeline)

---

## Notes

- `[P]` tasks operate on distinct files with no shared incomplete dependencies — safe to run concurrently
- `[USn]` labels provide traceability from tasks back to spec.md user stories and acceptance scenarios
- Every Go block package MUST have unit tests (project constitution: TDD enforced)
- Wails bindings in `frontend/bindings/` are auto-generated — **never edit manually**; run `wails3 generate bindings` after any Go service method change
- Use `go test -race ./...` for all pipeline engine tests (goroutine-heavy code; race detector required)
- `modernc.org/sqlite` is CGO-free — cross-compilation works from any host; macOS serial builds must be compiled on macOS runners (`go.bug.st/serial` requires CGO on macOS)
- Commit after each task or logical group; each Phase 3+ checkpoint should be a tagged release candidate
- Avoid raw error stack traces in any user-facing message (SC-007 compliance)
