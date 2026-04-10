# Tasks: New Signal Blocks

**Input**: Design documents from `/specs/004-new-signal-blocks/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/block-contracts.md

**Tests**: Included per Constitution Principle II (Test-First, NON-NEGOTIABLE).

**Organization**: Tasks grouped by user story. Each story is independently implementable and testable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Register all 6 new block types in backend and frontend so the canvas recognizes them.

- [ ] T001 Add 6 block type descriptors (bluetooth, multiply, filter, derivative, spectrum-viewer, hex-viewer) in services/workflow_service.go → blockTypeDescriptors()
- [ ] T002 Prepare frontend/src/components/canvas/BlockNode.vue for new block types — add placeholder comments in configComponentMap for bluetooth, multiply, filter, derivative, spectrum-viewer, hex-viewer (actual imports added per story phase when components are created)
- [ ] T003 Add default dimensions for spectrum-viewer and hex-viewer analysis blocks in frontend/src/components/canvas/WorkflowCanvas.vue → analysisDefaultDimensions

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared utilities needed by multiple blocks before story implementation begins.

**⚠️ BLOCKS**: User Story 2 (Filter) only — other stories depend on Phase 1 only

> **NOTE: Write tests FIRST, ensure they FAIL before implementation (Constitution Principle II)**

- [ ] T004 Write unit tests for Butterworth coefficient computation verifying known coefficient values for each filter mode (LP/HP/BP/BS) at various orders in internal/processing/filter_test.go
- [ ] T005 Implement Butterworth filter coefficient computation (bilinear transform, lowpass/highpass/bandpass/bandstop prototypes) as internal helper functions in internal/processing/filter.go — these are pure math functions testable in isolation

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 2 — Apply a Frequency Filter to a Signal (Priority: P1) 🎯 MVP

**Goal**: Users can filter signals using low-pass, high-pass, band-pass, and band-stop (notch) modes with configurable cutoff and order.

**Independent Test**: Connect a Simulator block to Filter block, apply low-pass filter, verify high frequencies are attenuated in downstream Line Chart.

### Tests for User Story 2 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T006 [P] [US2] Write unit tests for Filter block Configure() validating mode, cutoff, order params and rejection of invalid values in internal/processing/filter_test.go
- [ ] T007 [P] [US2] Write unit tests for Filter block Run() verifying low-pass filtering attenuates high frequencies using known sine wave inputs in internal/processing/filter_test.go
- [ ] T008 [P] [US2] Write unit tests for Filter block Run() verifying high-pass, band-pass, and band-stop modes produce correct frequency responses in internal/processing/filter_test.go
- [ ] T009 [P] [US2] Write unit test for Filter block verifying real-time parameter changes (reconfigure while running) in internal/processing/filter_test.go

### Implementation for User Story 2

- [ ] T010 [US2] Implement Filter block struct, ID/Type/Category/Ports methods, and Configure() with validation in internal/processing/filter.go
- [ ] T011 [US2] Implement Filter block Run() applying Direct Form II Transposed IIR filter with coefficient recomputation on parameter change in internal/processing/filter.go
- [ ] T012 [US2] Register filter block factory in init() in internal/processing/filter.go
- [ ] T013 [P] [US2] Create FilterConfig.vue with mode dropdown (lowpass/highpass/bandpass/bandstop), cutoff frequency inputs, order slider, and sample rate input in frontend/src/components/blocks/processing/FilterConfig.vue
- [ ] T014 [US2] Add FilterConfig import and mapping to configComponentMap in frontend/src/components/canvas/BlockNode.vue
- [ ] T015 [US2] Add structured logging for filter lifecycle (configure, mode change, coefficient recomputation, error) in internal/processing/filter.go

**Checkpoint**: Filter block fully functional — test with Simulator → Filter → Line Chart workflow

---

## Phase 4: User Story 1 — Connect and Read Data from a Bluetooth Device (Priority: P1)

**Goal**: Users can select a paired Bluetooth SPP device, receive raw byte data, and see auto-reconnection on connection loss.

**Independent Test**: Pair a Bluetooth device, add Bluetooth block, select device, connect to Hex Viewer, verify raw bytes appear.

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T016 [P] [US1] Write unit tests for Bluetooth block Configure() validating deviceAddress and serialPort params in internal/input/bluetooth_test.go
- [ ] T017 [P] [US1] Write unit tests for Bluetooth block Run() verifying raw byte streaming using io.Pipe() mock reader in internal/input/bluetooth_test.go
- [ ] T018 [P] [US1] Write unit tests for Bluetooth block disconnect detection (io.EOF → BlockError) and auto-reconnect with backoff in internal/input/bluetooth_test.go
- [ ] T019 [P] [US1] Write unit tests for Bluetooth block context cancellation causing clean shutdown in internal/input/bluetooth_test.go

### Implementation for User Story 1

- [ ] T020 [US1] Implement platform-specific Bluetooth device discovery (list paired SPP devices) using os/exec calls to bluetoothctl (Linux), system_profiler (macOS), PowerShell (Windows) in internal/input/bluetooth.go
- [ ] T021 [US1] Implement Bluetooth block struct, ID/Type/Category/Ports methods, and Configure() in internal/input/bluetooth.go
- [ ] T022 [US1] Implement Bluetooth block Run() with serial port connection via go.bug.st/serial, raw byte streaming, and disconnect detection in internal/input/bluetooth.go
- [ ] T023 [US1] Implement auto-reconnect with exponential backoff (1s-30s) and visible retry status via pipeline:block-status events in internal/input/bluetooth.go
- [ ] T024 [US1] Register bluetooth block factory in init() in internal/input/bluetooth.go
- [ ] T025 [US1] Add listBluetoothDevices() Wails-bound method returning paired device list in services/workflow_service.go
- [ ] T026 [P] [US1] Create BluetoothBlockConfig.vue with device dropdown (refresh button calling listBluetoothDevices()), connection status indicator, optional serial port override in frontend/src/components/blocks/inputs/BluetoothBlockConfig.vue
- [ ] T027 [US1] Add BluetoothBlockConfig import and mapping to configComponentMap in frontend/src/components/canvas/BlockNode.vue
- [ ] T028 [US1] Add structured logging for Bluetooth lifecycle (discovery, connect, disconnect, reconnect attempts, errors) in internal/input/bluetooth.go

**Checkpoint**: Bluetooth block fully functional — test with paired SPP device → Hex Viewer workflow

---

## Phase 5: User Story 5 — Compute the Derivative of a Signal (Priority: P2)

**Goal**: Users can compute the rate of change of a signal over time using backward finite difference.

**Independent Test**: Connect Simulator (linear ramp) → Derivative → Line Chart, verify output is constant.

### Tests for User Story 5 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T029 [P] [US5] Write unit tests for Derivative block verifying constant input → zero output, linear ramp → constant output, first sample → zero output in internal/processing/derivative_test.go
- [ ] T030 [P] [US5] Write unit test for Derivative block verifying variable time intervals (different timestamps between samples) produce correct dt-based derivatives in internal/processing/derivative_test.go

### Implementation for User Story 5

- [ ] T031 [US5] Implement Derivative block struct, ID/Type/Category/Ports methods, Configure(), and Run() with backward finite difference (y[n]-y[n-1])/(t[n]-t[n-1]) in internal/processing/derivative.go
- [ ] T032 [US5] Register derivative block factory in init() in internal/processing/derivative.go
- [ ] T033 [P] [US5] Create DerivativeConfig.vue as minimal config component (no user-configurable params, display-only info) in frontend/src/components/blocks/processing/DerivativeConfig.vue
- [ ] T034 [US5] Add DerivativeConfig import and mapping to configComponentMap in frontend/src/components/canvas/BlockNode.vue
- [ ] T035 [US5] Add structured logging for derivative block lifecycle in internal/processing/derivative.go

**Checkpoint**: Derivative block fully functional — test with Simulator → Derivative → Line Chart

---

## Phase 6: User Story 4 — Multiply Multiple Signals Together (Priority: P2)

**Goal**: Users can multiply two or more signal streams sample-by-sample for modulation or gating.

**Independent Test**: Connect two Simulator blocks (constant + sine) → Multiply → Line Chart, verify output matches product.

### Tests for User Story 4 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T036 [P] [US4] Write unit tests for Multiply block Configure() validating inputCount (2-8) and dynamic port creation in internal/processing/multiply_test.go
- [ ] T037 [P] [US4] Write unit tests for Multiply block Run() verifying 2-input product, 3-input product, and single-input passthrough in internal/processing/multiply_test.go
- [ ] T038 [P] [US4] Write unit test for Multiply block verifying latest-value-wins sync behavior when inputs arrive at different rates in internal/processing/multiply_test.go

### Implementation for User Story 4

- [ ] T039 [US4] Implement Multiply block struct with dynamic input ports (in-0..in-N), pending value map, and Configure() in internal/processing/multiply.go
- [ ] T040 [US4] Implement Multiply block Run() with fan-in from multiple input channels, latest-value-wins sync, and product computation in internal/processing/multiply.go
- [ ] T041 [US4] Register multiply block factory in init() in internal/processing/multiply.go
- [ ] T042 [P] [US4] Create MultiplyConfig.vue with inputCount selector (2-8) in frontend/src/components/blocks/processing/MultiplyConfig.vue
- [ ] T043 [US4] Add MultiplyConfig import and mapping to configComponentMap in frontend/src/components/canvas/BlockNode.vue
- [ ] T044 [US4] Add structured logging for multiply block lifecycle in internal/processing/multiply.go

**Checkpoint**: Multiply block fully functional — test with Simulator + Simulator → Multiply → Line Chart

---

## Phase 7: User Story 3 — View Signal Frequency Spectrum as Waterfall Spectrogram (Priority: P2)

**Goal**: Users can visualize spectral content over time as a scrolling waterfall spectrogram with configurable color mapping and frequency range.

**Independent Test**: Connect Simulator → FFT → Spectrum Viewer, verify waterfall displays frequency bands matching simulator waveform.

### Tests for User Story 3 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T045 [P] [US3] Write unit tests for SpectrumViewer analysis block verifying BufferConfig, Snapshot(), and data buffering in internal/analysis/spectrumviewer_test.go
- [ ] T046 [P] [US3] Write unit tests for SpectrumViewer Configure() validating colorMap, frequency range, and amplitude range params in internal/analysis/spectrumviewer_test.go

### Implementation for User Story 3

- [ ] T047 [US3] Implement SpectrumViewer analysis block struct implementing AnalysisBlock interface with buffer management in internal/analysis/spectrumviewer.go
- [ ] T048 [US3] Implement SpectrumViewer Configure() with colorMap, frequency range, and amplitude range validation in internal/analysis/spectrumviewer.go
- [ ] T049 [US3] Register spectrum-viewer analysis block factory via registerAnalysis() in internal/analysis/spectrumviewer.go
- [ ] T050 [US3] Create SpectrumViewerBlock.vue with Canvas 2D waterfall rendering — self-copy scroll, amplitude-to-color mapping (viridis/magma/inferno/plasma/grayscale), frequency/time axes in frontend/src/components/blocks/analysis/SpectrumViewerBlock.vue
- [ ] T051 [US3] Implement color map lookup tables (viridis, magma, inferno, plasma, grayscale) as reusable utility in frontend/src/components/blocks/analysis/SpectrumViewerBlock.vue
- [ ] T052 [US3] Wire SpectrumViewerBlock.vue to useAnalysisBlock() composable for pipeline:data event subscription and ring buffer management in frontend/src/components/blocks/analysis/SpectrumViewerBlock.vue
- [ ] T053 [US3] Add SpectrumViewerBlock import and mapping to configComponentMap in frontend/src/components/canvas/BlockNode.vue
- [ ] T054 [US3] Add structured logging for spectrum viewer lifecycle in internal/analysis/spectrumviewer.go

**Checkpoint**: Spectrum Viewer fully functional — test with Simulator → FFT → Spectrum Viewer workflow

---

## Phase 8: User Story 6 — Inspect Raw Data as Hex/Text/Bytes (Priority: P3)

**Goal**: Users can view raw data streams in a hex dump format with offset, hex, and ASCII columns, with pause/resume and byte pattern search.

**Independent Test**: Connect UART (or Simulator → ByteParser reverse) → Hex Viewer, verify hex dump displays correct byte values and ASCII.

### Tests for User Story 6 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T055 [P] [US6] Write unit tests for HexViewer analysis block verifying BufferConfig, Snapshot(), ring buffer byte limit enforcement, and raw data buffering in internal/analysis/hexviewer_test.go
- [ ] T056 [P] [US6] Write unit tests for HexViewer byte pattern search matching hex patterns against buffered data in internal/analysis/hexviewer_test.go

### Implementation for User Story 6

- [ ] T057 [US6] Implement HexViewer analysis block struct implementing AnalysisBlock interface with bounded ring buffer (maxBytes) in internal/analysis/hexviewer.go
- [ ] T058 [US6] Implement HexViewer byte pattern search method operating on ring buffer contents in internal/analysis/hexviewer.go
- [ ] T059 [US6] Register hex-viewer analysis block factory via registerAnalysis() in internal/analysis/hexviewer.go
- [ ] T060 [US6] Create HexViewerBlock.vue with virtualized scrolling hex dump layout (offset | hex bytes | ASCII), auto-scroll, pause/resume toggle in frontend/src/components/blocks/analysis/HexViewerBlock.vue
- [ ] T061 [US6] Implement byte pattern search UI (hex or ASCII input, match highlighting) in frontend/src/components/blocks/analysis/HexViewerBlock.vue
- [ ] T062 [US6] Wire HexViewerBlock.vue to useAnalysisBlock() composable for pipeline:data event subscription with raw byte mode in frontend/src/components/blocks/analysis/HexViewerBlock.vue
- [ ] T063 [US6] Add HexViewerBlock import and mapping to configComponentMap in frontend/src/components/canvas/BlockNode.vue
- [ ] T064 [US6] Add structured logging for hex viewer lifecycle in internal/analysis/hexviewer.go

**Checkpoint**: Hex Viewer fully functional — test with UART → Hex Viewer workflow

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Integration validation and cross-story quality improvements.

- [ ] T065 Validate end-to-end workflow: Bluetooth → Hex Viewer (raw byte display from BT device)
- [ ] T066 [P] Validate end-to-end workflow: Simulator → Filter → Line Chart (all 4 filter modes)
- [ ] T067 [P] Validate end-to-end workflow: Simulator → FFT → Spectrum Viewer (waterfall display)
- [ ] T068 [P] Validate end-to-end workflow: Simulator + Simulator → Multiply → Line Chart (product)
- [ ] T069 [P] Validate end-to-end workflow: Simulator → Derivative → Line Chart (rate of change)
- [ ] T070 Validate compound workflow: Bluetooth → ByteParser → Filter → FFT → Spectrum Viewer
- [ ] T071 Verify all 6 blocks persist and restore correctly in .byteflow workflow file (save/load)
- [ ] T072 Run quickstart.md validation scenarios

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS User Story 2 (Filter) only
- **US2 Filter (Phase 3)**: Depends on Phase 2 (Butterworth coefficients)
- **US1 Bluetooth (Phase 4)**: Depends on Phase 1 only — no dependency on Phase 2
- **US5 Derivative (Phase 5)**: Depends on Phase 1 only
- **US4 Multiply (Phase 6)**: Depends on Phase 1 only
- **US3 Spectrum Viewer (Phase 7)**: Depends on Phase 1 only
- **US6 Hex Viewer (Phase 8)**: Depends on Phase 1 only
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (Bluetooth, P1)**: Independent — no dependency on other stories
- **US2 (Filter, P1)**: Depends on Phase 2 (coefficient math) — no dependency on other stories
- **US3 (Spectrum Viewer, P2)**: Independent — uses existing FFT block upstream
- **US4 (Multiply, P2)**: Independent
- **US5 (Derivative, P2)**: Independent
- **US6 (Hex Viewer, P3)**: Independent — pairs naturally with US1 for end-to-end testing

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Backend block before frontend component
- Registration (init/factory) before UI wiring
- Logging after core implementation

### Parallel Opportunities

- After Phase 1 completes: US1, US3, US4, US5, US6 can ALL start in parallel (independent blocks)
- After Phase 2 completes: US2 can start (and run parallel with any other in-progress story)
- Within each story: all test tasks marked [P] can run in parallel
- Within each story: frontend [P] tasks can run parallel with backend tasks (different files)

---

## Parallel Example: User Story 2 (Filter)

```bash
# After Phase 2 completes, launch all filter tests in parallel:
Task T006: "Write unit tests for Filter block Configure()"
Task T007: "Write unit tests for Filter block Run() low-pass"
Task T008: "Write unit tests for Filter block Run() HP/BP/BS modes"
Task T009: "Write unit test for real-time parameter changes"

# After tests written, implementation is sequential (same file):
Task T010: "Implement Filter block struct and Configure()"
Task T011: "Implement Filter block Run() with IIR filter"
Task T012: "Register filter block factory"

# Frontend can start as soon as block descriptor is registered (Phase 1):
Task T013: "Create FilterConfig.vue" (parallel with backend)
Task T014: "Add FilterConfig import to BlockNode.vue"
```

## Parallel Example: Multiple Stories

```bash
# After Phase 1 completes, all independent stories can start simultaneously:
# Developer A: US5 (Derivative — simplest, validates pattern)
Task T029-T035

# Developer B: US4 (Multiply)
Task T036-T044

# Developer C: US1 (Bluetooth)
Task T016-T028

# Developer D: US6 (Hex Viewer)
Task T055-T064
```

---

## Implementation Strategy

### MVP First (User Story 2 — Filter)

1. Complete Phase 1: Setup (register all block types)
2. Complete Phase 2: Foundational (Butterworth coefficients)
3. Complete Phase 3: US2 Filter
4. **STOP and VALIDATE**: Test with Simulator → Filter → Line Chart
5. Functional MVP — users can filter signals

### Incremental Delivery

1. Setup + Foundational → Framework ready
2. Add US2 Filter (P1) → Test independently → MVP
3. Add US1 Bluetooth (P1) → Test independently → Full P1 delivery
4. Add US5 Derivative + US4 Multiply + US3 Spectrum Viewer (P2, parallel) → Test each → Full P2 delivery
5. Add US6 Hex Viewer (P3) → Test independently → Feature complete
6. Polish phase → End-to-end validation → Ship

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Tests MUST fail before implementing (Constitution Principle II)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
