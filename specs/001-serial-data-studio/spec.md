# Feature Specification: Serial Data Processing Studio

**Feature Branch**: `001-serial-data-studio`
**Created**: 2026-03-19
**Status**: Draft
**Input**: User description: "Multiplatform desktop application for working with serial data sources (byte streams), processing them, visualising and analysing — an N8N-style visual workflow builder for electronics and live data processing."

---

## Overview

ByteFlow Studio is a multiplatform desktop application that enables engineers, makers, and researchers to connect to live byte-stream data sources, build visual processing pipelines, and observe or analyse the results in real time. Users compose workflows by dragging and connecting blocks from three categories — Inputs, Processing, and Analysis — onto a shared canvas. Workflows can run, be paused, or signal error states. The application targets professionals and hobbyists working with embedded systems, sensors, instruments, and any device that exposes a serial byte stream.

---

## Clarifications

### Session 2026-03-19

- Q: Should input stream data chunks be timestamped? → A: Yes — every data chunk produced by an Input block MUST carry a millisecond-resolution timestamp assigned at the moment of receipt (e.g., every byte from a UART input carries its own timestamp).
- Q: Should timestamp metadata be available to downstream blocks? → A: Yes — timestamps MUST be propagated through all connections and remain accessible to Processing and Analysis blocks for further use.
- Q: Is USB input required for the first version? → A: No — USB input is deferred to a future release. v1 includes UART/Serial, WebSocket, and Signal Simulator only.
- Q: Should Analysis blocks have configurable data buffers? → A: Yes — each Analysis block MUST expose a configurable buffer that controls how much data is retained for display.
- Q: Should the current data session be included in the workflow save file? → A: Yes — saving a workflow MUST also persist the current session's recorded data.
- Q: Should multiple past data sessions be retained? → A: Yes — the system MUST persist the last N complete data sessions per workflow.
- Q: Is N (session retention count) fixed or user-configurable? → A: User-configurable in application settings, with a default of 10 sessions.
- Q: What unit defines an Analysis block's buffer size? → A: Either — each Analysis block lets the user choose between number of samples or a time duration, configured per block.
- Q: What data does a Session store — raw bytes, processed values, or both? → A: Both — raw timestamped bytes from Input blocks (enabling future pipeline replay) AND processed values as they enter each Analysis block (for immediate display on open). Each storage layer is independently toggleable by the user to manage file size.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Connect a Data Source and View Live Data (Priority: P1)

An engineer connects a microcontroller over UART to their laptop, opens ByteFlow Studio, drags a UART Input block onto the canvas, configures the COM port and baud rate, then drags a Raw Value Display Analysis block and connects the two. They start the flow and immediately see live bytes rendered in the display.

**Why this priority**: Delivering raw data from a source to a display represents the minimal viable product — all other features build on this core path.

**Independent Test**: Fully testable by connecting a UART device (or serial loopback), adding one Input block and one Analysis block, running the flow, and confirming live data appears in the display.

**Acceptance Scenarios**:

1. **Given** a blank canvas, **When** the user drags a UART Input block from the library and a Raw Display Analysis block and connects them, **Then** a valid two-block flow is formed and the Start button becomes active.
2. **Given** a running flow with a UART Input and Raw Display, **When** the connected device transmits bytes, **Then** those bytes appear in the display within 200 ms of receipt.
3. **Given** a running flow, **When** the user clicks Pause, **Then** data ingestion halts, the display freezes, and the flow state changes to "Paused".
4. **Given** a paused flow, **When** the user clicks Resume, **Then** data ingestion restarts and the display updates again.

---

### User Story 2 — Build a Multi-Step Processing Pipeline (Priority: P2)

A researcher captures accelerometer data from a UART/serial device, applies a Moving Average processing block to smooth the signal, and routes the output to both a Line Chart Analysis block (visible on the canvas) and a Table Analysis block. They want to compare raw and smoothed signals side by side.

**Why this priority**: Processing pipelines are the core differentiator; without them the application is a simple terminal. Multi-output routing demonstrates the N8N-parallel capability.

**Independent Test**: Testable with a Signal Simulator Input (built-in simulated source), a Moving Average block, and two Analysis blocks wired from the same processing output, confirming both Analysis blocks update in real time.

**Acceptance Scenarios**:

1. **Given** an Input block and a Processing block, **When** the user connects them and then connects the Processing block to two different Analysis blocks, **Then** both Analysis blocks receive the same processed stream.
2. **Given** a running pipeline with a Moving Average block, **When** the window size parameter is changed, **Then** the chart updates to reflect the new smoothing within one render cycle.
3. **Given** a pipeline with an unconnected Processing block, **When** the user attempts to start the flow, **Then** the system highlights the unconnected block and refuses to start, displaying an actionable error message.

---

### User Story 3 — Manage and Persist Workflows (Priority: P3)

A technician builds a diagnostic workflow for testing a product on the factory floor, saves it, and later reopens it on a different workstation running the same OS. The entire canvas layout, block configurations, connections, and the recorded data from the current session are restored exactly.

**Why this priority**: Reusability and persistence are essential for professional workflows; without save/load the tool cannot be used in repeatable processes.

**Independent Test**: Testable by creating a workflow, running it to collect some data, saving it, closing and reopening the application, and verifying the canvas and session data are restored identically.

**Acceptance Scenarios**:

1. **Given** a configured workflow with a running or completed session, **When** the user saves it, **Then** a single portable file is produced containing the complete workflow definition and the current session's recorded data.
2. **Given** a saved workflow file, **When** the user opens it on the same or a different machine, **Then** all blocks, connections, parameters, layout, and session data are restored without data loss.
3. **Given** a workflow referencing a hardware Input (e.g., COM3) that is unavailable on the current machine, **When** the workflow is opened, **Then** the affected Input block is shown in an error state with a clear message prompting the user to reconfigure the connection; previously recorded session data remains accessible.

---

### User Story 4 — Full-Screen Analysis View (Priority: P3)

During a live demonstration, an engineer wants to project a single oscilloscope-style chart for the audience without showing the canvas clutter.

**Why this priority**: The fullscreen mode is a professional presentation and monitoring feature; it adds significant value for deployment and demonstration scenarios.

**Independent Test**: Testable by right-clicking any Analysis block in a running flow and selecting "Full Screen"; the block should fill the display and exit gracefully on Escape.

**Acceptance Scenarios**:

1. **Given** a running flow with at least one Analysis block, **When** the user activates full-screen on that block, **Then** the block's visualisation expands to fill the entire application window with no canvas chrome visible.
2. **Given** a full-screen Analysis view, **When** new data arrives, **Then** the visualisation continues to update in real time without interruption.
3. **Given** a full-screen Analysis view, **When** the user presses Escape or clicks a close/exit control, **Then** the application returns to the normal canvas view with the flow still running.

---

### User Story 5 — Multiple Inputs and Analysis Blocks (Priority: P2)

A lab engineer monitors two separate instruments simultaneously — one over WebSocket and one over UART — and routes each to its own chart, while also feeding both into a combined FFT Analysis block.

**Why this priority**: Multi-source, multi-output topologies are a primary use case; the feature description explicitly lists multiple Inputs and multiple Analysis blocks as requirements.

**Independent Test**: Testable with two simulated Input blocks wired independently and then both merged into a single Processing block, verifying independent and combined outputs are rendered correctly.

**Acceptance Scenarios**:

1. **Given** a canvas with two Input blocks each connected to independent Analysis blocks, **When** the flow runs, **Then** both Analysis blocks update independently based on their respective sources.
2. **Given** two Input blocks wired into a single Processing block, **When** the flow runs, **Then** the Processing block merges or selects from the streams as configured, and its output reaches the downstream Analysis blocks.

---

### User Story 6 — Review a Past Data Session (Priority: P3)

An engineer ran a diagnostic test yesterday, and today wants to review the recorded data without reconnecting to the device. They open the workflow, switch to a previous session, and navigate through the historical data in the Analysis blocks.

**Why this priority**: Session history turns the tool from a live monitor into a replayable data archive, enabling post-hoc analysis and comparison across test runs. Storing raw bytes alongside processed values also unlocks future pipeline-replay capability (re-running a different processing configuration against previously captured raw data).

**Independent Test**: Testable by running a flow, stopping it, running it again to generate a second session, then switching the active view back to the first session and confirming Analysis blocks display that session's data.

**Acceptance Scenarios**:

1. **Given** a workflow with multiple persisted sessions, **When** the user opens the workflow, **Then** a list of available sessions is visible with timestamps and metadata.
2. **Given** the session list, **When** the user selects a past session, **Then** all Analysis blocks switch to display that session's data (static, non-live).
3. **Given** a past session view, **When** the user starts a new live run, **Then** a new session begins and the view switches to live mode automatically.

---

### Edge Cases

- What happens when a hardware port (UART/Serial) is disconnected while the flow is running? — The affected Input block must transition to an error state and display a reconnection prompt; other blocks with independent inputs should continue running.
- What happens when incoming data arrives faster than the processing or rendering pipeline can handle? — The system must apply backpressure or drop frames gracefully with a visible warning rather than crashing or silently corrupting data.
- What happens when a circular connection is attempted? — The system must detect and prevent cycles, highlighting the offending connection with an error.
- What happens when a user tries to start a flow with only one block (no input-to-analysis path)? — The system must block the start and explain the minimum requirement (at least one Input and one Analysis block must be connected).
- What happens when a saved workflow file is corrupted or from an incompatible version? — The system must display a clear error and offer to open the file in a recovery mode or ignore unknown blocks.
- What happens when an Analysis block's buffer is full? — The oldest data in the buffer is discarded to make room for incoming data (ring-buffer behaviour); the user is not interrupted.
- What happens when the total size of N persisted sessions grows very large? — The system should warn the user if total session storage exceeds a configurable threshold, and must always drop the oldest session when adding a new one beyond N.

---

## Requirements *(mandatory)*

### Functional Requirements

#### Canvas & Workflow Editor

- **FR-001**: The application MUST provide a pannable and zoomable canvas on which blocks can be placed, moved, and connected.
- **FR-002**: Users MUST be able to drag blocks from a categorised block library panel onto the canvas.
- **FR-003**: The system MUST support automatic connection of compatible block outputs to inputs when blocks are placed in proximity, with the ability for users to manually draw connections by dragging from one port to another.
- **FR-004**: The system MUST enforce that a flow requires at least one Input block and at least one Analysis block with a valid connection path before the flow can be started.
- **FR-005**: The system MUST detect and prevent circular connections, providing a visual indication of the invalid connection attempt.
- **FR-006**: Users MUST be able to delete individual blocks and connections, as well as delete a multi-block selection.
- **FR-007**: The system MUST support undo and redo for all canvas editing operations.

#### Block Library — Inputs

- **FR-008**: The system MUST provide the following Input block types in v1: UART/Serial and WebSocket, plus a built-in Signal Simulator (for development and testing without physical hardware). USB input is deferred to a future release.
- **FR-009**: Each Input block MUST expose configurable parameters relevant to its connection type (e.g., port and baud rate for UART; URL and subprotocol for WebSocket).
- **FR-010**: Input blocks MUST display their connection status (Connected, Disconnected, Error) directly on the block on the canvas.
- **FR-026**: Every data chunk produced by an Input block MUST carry a millisecond-resolution timestamp recording the moment the data was received by the application (e.g., each byte from a UART input carries its own timestamp; each message from a WebSocket input carries a single timestamp).
- **FR-027**: Timestamp metadata MUST be propagated through all connections and remain accessible to downstream Processing and Analysis blocks. Processing blocks MUST pass timestamps through to their output streams unless they explicitly transform the time axis (e.g., an FFT block may produce a new time coordinate).

#### Block Library — Processing

- **FR-011**: The system MUST provide at minimum the following Processing block types: Moving Average, Summation, FFT (Fast Fourier Transform), Value Scaling/Offset, Byte Parser (extract typed values from raw byte frames), and Passthrough.
- **FR-012**: Processing blocks MUST accept one or more input streams and produce one or more output streams.
- **FR-013**: Each Processing block MUST expose configurable parameters relevant to its operation (e.g., window size for Moving Average; frame format specification for Byte Parser).

#### Block Library — Analysis

- **FR-014**: The system MUST provide at minimum the following Analysis block types: Line Chart, Bar Chart, Numeric Value Display, Data Table, and FFT Spectrum Viewer.
- **FR-015**: Analysis blocks MUST update their visualisation in real time as data flows through the connected pipeline.
- **FR-016**: Each Analysis block MUST support a full-screen mode that expands the visualisation to fill the entire application window.
- **FR-017**: Analysis blocks MUST allow configuration of display parameters (e.g., time window, axis labels, value units, number of decimal places shown).
- **FR-028**: Each Analysis block MUST expose a configurable data buffer that determines how much data is retained for that block's display. The buffer size MUST be expressible in either number of samples (e.g., "keep last 1 000 data points") or a time duration (e.g., "keep last 30 seconds"), with the unit selectable per block. When the buffer is full, the oldest data is discarded (ring-buffer behaviour). The buffer configuration MUST persist as part of the block's saved parameters.

#### Flow Lifecycle

- **FR-018**: A flow MUST be in one of three states at any time: Running, Paused, or Error.
- **FR-019**: Users MUST be able to Start, Pause, and Stop a flow using clearly labelled controls visible at all times.
- **FR-020**: When a flow enters an Error state (invalid configuration, lost connection, or processing fault), the system MUST highlight the offending block(s) and display a human-readable description of the error.
- **FR-021**: Pausing a flow MUST suspend data ingestion and processing while retaining the current display state; resuming MUST restart data flow without requiring reconfiguration.

#### Session Management

- **FR-029**: Each continuous Run of a flow (from Start to Stop or error) constitutes a data Session. By default, a Session records TWO layers of data, both tagged with millisecond-resolution timestamps:
  - **Raw layer**: the raw bytes as received from each Input block (enables future pipeline replay with a different processing configuration).
  - **Processed layer**: the values as they enter each Analysis block, after all Processing blocks have run (enables immediate display when opening the workflow without re-running the pipeline).
- **FR-033**: The user MUST be able to configure which session data layers are stored, independently per workflow. Options are: raw layer only, processed layer only, or both (default). A clear indication of the estimated storage impact of each option MUST be shown when the setting is changed.
- **FR-030**: The system MUST persist the last N complete Sessions per workflow, where N is user-configurable in application settings with a default of 10. When a new Session completes and the count exceeds N, the oldest Session is automatically discarded.
- **FR-031**: Users MUST be able to browse the list of available Sessions for a workflow, showing at minimum the start time, end time, and data volume of each Session.
- **FR-032**: Users MUST be able to switch the Analysis blocks' display to show data from any persisted Session (historical playback mode) without starting a new live run.

#### Persistence

- **FR-022**: Users MUST be able to save a workflow to a single file. The save file MUST include the complete canvas state (block positions, connections, parameter values including buffer sizes), the current active Session's recorded data, and all previously persisted Sessions for that workflow.
- **FR-023**: When a saved workflow references a hardware Input unavailable on the current machine, the system MUST open the workflow in a degraded state, notify the user of the missing resource, and allow them to reconfigure the affected block without discarding the rest of the workflow or any recorded Session data.

#### Multiplatform Support

- **FR-024**: The application MUST run on Windows (10+), macOS (12+), and Linux (Ubuntu 22.04+ and Arch-based distributions) without requiring source compilation by the end user.
- **FR-025**: The application MUST be distributed as a single installable package or portable binary for each supported platform with no additional runtime dependencies required from the user.

---

### Key Entities

- **Workflow**: The top-level container holding all blocks, connections, canvas layout, and the collection of persisted Sessions. Represents a complete data-processing graph that can be saved, loaded, started, and stopped.
- **Block**: A discrete functional unit placed on the canvas. Has a type (Input | Processing | Analysis), a unique ID, configurable parameters, and zero or more typed input/output ports.
- **Connection**: A directional link between an output port of one block and an input port of another block. Carries a typed data stream at runtime.
- **Port**: A typed connection point on a block. Output ports emit streams; input ports consume them. Ports are typed to prevent incompatible connections.
- **Stream**: A continuous sequence of typed data values flowing along a connection at runtime. Each value in a stream carries a millisecond-resolution timestamp indicating when the originating data was received by its Input block. Timestamps are preserved (and accessible) at every hop through the pipeline.
- **Data Chunk**: The atomic unit of data produced by an Input block. For UART/Serial, a data chunk is a single received byte. For WebSocket, a data chunk is a single received message. Each chunk carries its own timestamp.
- **Session**: A bounded record of data captured during a single continuous Run of a workflow (from Start to Stop or error). Each Session has a start time, end time, and may contain up to two data layers (both timestamped at millisecond resolution): a **raw layer** (bytes as received from Input blocks, for future replay) and a **processed layer** (values as they entered each Analysis block, for immediate display on open). Which layers are stored is user-configurable per workflow. Sessions are stored per workflow; the most recent N (default 10, user-configurable) are retained.
- **Flow State**: The runtime lifecycle state of a workflow: Running, Paused, or Error.
- **Block Library**: The catalogue of available block types, organised by category (Input, Processing, Analysis), browsable and searchable by the user.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user with no prior knowledge of the application can connect a hardware Input, add a display Analysis block, and view live data within 5 minutes of first launch without consulting documentation.
- **SC-002**: Live data from an Input block appears in a connected Analysis block within 200 ms of the data being received at the hardware interface under normal operating conditions.
- **SC-003**: The application launches and presents an empty canvas within 3 seconds on a mid-range laptop released in 2020 or later.
- **SC-004**: A workflow with up to 20 blocks and 30 connections renders and updates at no less than 30 visualisation frames per second on a mid-range laptop.
- **SC-005**: A saved workflow file opens and fully restores canvas state and the processed-value layer of all sessions within 5 seconds for workflows with up to 20 blocks and up to 10 sessions of up to 10 MB of processed data each. Loading of the raw byte layer MAY be deferred (lazy-loaded) and must not block the UI.
- **SC-006**: 90% of users in usability testing can successfully build and run a two-block flow (Input → Analysis) without consulting documentation.
- **SC-007**: All error states are communicated to the user with a human-readable message and a clear indication of which block caused the error; no raw error stack traces are shown to the end user.
- **SC-008**: The application is available as a single installable package or portable binary for each of the three supported platforms with no additional runtime dependencies required from the user.
- **SC-009**: Timestamp resolution of data chunks is accurate to within 5 ms of the true receive time as measured against the system clock.

---

## Assumptions

- The initial release targets individual users (single-user desktop application); multi-user collaboration and cloud sync are out of scope.
- Workflow files (including all session data) are stored locally on the user's file system; no cloud storage or synchronisation is required in the initial version.
- USB input support is deferred to a future release; v1 provides UART/Serial, WebSocket, and Signal Simulator input types only.
- The built-in Signal Simulator block generates configurable synthetic waveforms (e.g., sine, square, random noise) sufficient for development and demonstration without physical hardware. Simulated data chunks also carry timestamps.
- "Byte stream" data sources produce raw binary data; higher-level protocol decoding (e.g., Modbus framing, custom packet formats) is handled by Processing blocks (Byte Parser), not by Input blocks directly.
- The application does not need to support writing data back to connected devices in the initial release (read-only data acquisition).
- Block-to-block connections carry timestamped numeric values after initial parsing by a Byte Parser; raw byte streams (also timestamped) exist only between an Input block and a Byte Parser or compatible Processing block.
- Timestamps are assigned using the host system clock at the point the application receives the data from the OS driver; hardware-level timestamping (e.g., via FPGA or USB timestamp) is out of scope for v1.
- Report generation (export of analysis results to PDF/CSV) is a desirable future feature but is not required for the initial release.
- The system will use the host operating system's serial driver stack; custom driver installation is out of scope.
- The block library is extensible in future versions via a plugin or custom block mechanism, but the initial release ships only the built-in block types listed in the requirements.
- By default, Sessions store both a raw byte layer (from Input blocks) and a processed value layer (as data enters each Analysis block). Either layer can be disabled per workflow by the user to reduce file size. When only processed values are stored, the workflow definition plus a new live run is sufficient to regenerate raw data.
