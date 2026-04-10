# Feature Specification: New Signal Blocks

**Feature Branch**: `004-new-signal-blocks`  
**Created**: 2026-04-10  
**Status**: Draft  
**Input**: User description: "New blocks: bluetooth input, signal multiply, filter, derivative processing, spectrum viewer, hex viewer output"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Connect and Read Data from a Bluetooth Device (Priority: P1)

A user wants to receive data from a paired Bluetooth device (e.g., a sensor or microcontroller) into their workflow. They add a Bluetooth block to the canvas, select a paired device from a list of available devices, and start receiving a live raw byte stream. The raw bytes can be routed directly to output blocks (e.g., Hex Viewer) or through parser/deserializer blocks before reaching processing blocks (e.g., Filter, Multiply, Derivative).

**Why this priority**: Input is the foundation of any workflow. Without a data source, no processing or visualization blocks can function. Bluetooth expands the application beyond wired serial connections.

**Independent Test**: Can be fully tested by pairing a Bluetooth device, adding the Bluetooth block, selecting the device, and verifying that raw bytes appear on the block's output port (e.g., by connecting to a Hex Viewer).

**Acceptance Scenarios**:

1. **Given** the user has a Bluetooth device paired with their system, **When** they add a Bluetooth block and open its configuration, **Then** the paired device appears in the device list.
2. **Given** a Bluetooth block is configured with a paired device, **When** the user starts the workflow, **Then** the block begins receiving data and outputs raw bytes to connected blocks.
3. **Given** a Bluetooth device disconnects mid-session, **When** the connection is lost, **Then** the block displays a disconnection indicator, stops outputting data, and automatically begins reconnection attempts with visible retry status.
4. **Given** a Bluetooth block is retrying connection, **When** the device becomes available again, **Then** the block reconnects and resumes outputting data automatically.
5. **Given** no Bluetooth devices are paired, **When** the user opens the Bluetooth block configuration, **Then** a message indicates no devices are available and suggests pairing a device through the operating system.

---

### User Story 2 - Apply a Frequency Filter to a Signal (Priority: P1)

A user wants to filter noise or isolate a frequency band from a signal. They add a Filter block, connect it to a signal source, and configure the filter type (low-pass, high-pass, band-pass, or band-stop/notch) along with cutoff frequency parameters. The block outputs the filtered signal in real time.

**Why this priority**: Filtering is one of the most fundamental and frequently used signal processing operations. It enables meaningful analysis by removing unwanted frequency components.

**Independent Test**: Can be tested by connecting a known signal source to the Filter block, applying a low-pass filter, and verifying that high-frequency components are attenuated in the output.

**Acceptance Scenarios**:

1. **Given** a signal source is connected to a Filter block, **When** the user selects "Low-Pass" and sets a cutoff frequency, **Then** the output signal contains only frequencies below the cutoff.
2. **Given** a signal source is connected to a Filter block, **When** the user selects "High-Pass" and sets a cutoff frequency, **Then** the output signal contains only frequencies above the cutoff.
3. **Given** a signal source is connected to a Filter block, **When** the user selects "Band-Pass" and sets lower and upper cutoff frequencies, **Then** the output signal contains only frequencies within the specified band.
4. **Given** a signal source is connected to a Filter block, **When** the user selects "Band-Stop" (notch) and sets lower and upper cutoff frequencies, **Then** the output signal excludes frequencies within the specified band (e.g., removing 50/60 Hz mains hum).
5. **Given** a Filter block is configured, **When** the user changes filter parameters while the workflow is running, **Then** the filter updates in real time without interrupting data flow.

---

### User Story 3 - View a Signal's Frequency Spectrum as a Color Gradient (Priority: P2)

A user wants to visualize the frequency content of a signal over time as a waterfall spectrogram. They add a Spectrum Viewer block, connect it to a signal source, and see a live, scrolling display where the vertical axis is frequency, the horizontal axis is time, and color intensity represents signal amplitude — allowing them to observe how the spectral content evolves.

**Why this priority**: Spectral visualization is a key analysis tool for understanding signal characteristics, identifying patterns, and diagnosing issues. It adds a powerful new output modality.

**Independent Test**: Can be tested by connecting a signal source with known frequency components to the Spectrum Viewer and verifying that the correct frequencies appear as highlighted bands in the color gradient.

**Acceptance Scenarios**:

1. **Given** a signal source is connected to a Spectrum Viewer block, **When** the workflow is running, **Then** the block displays a live waterfall spectrogram with frequency on the vertical axis, time on the horizontal axis, and amplitude mapped to color.
2. **Given** a Spectrum Viewer is displaying data, **When** the input signal's dominant frequency changes, **Then** the displayed spectrum updates to reflect the change in real time.
3. **Given** a Spectrum Viewer block, **When** the user configures the color scale or frequency range, **Then** the display adjusts accordingly.

---

### User Story 4 - Multiply Multiple Signals Together (Priority: P2)

A user wants to combine multiple signals by multiplying their sample values together (e.g., for amplitude modulation or gating). They add a Signal Multiply block, connect two or more signal inputs, and the block outputs a sample-by-sample product of all inputs.

**Why this priority**: Signal multiplication enables modulation, windowing, and gating operations that are common in signal processing workflows.

**Independent Test**: Can be tested by connecting two known signal sources (e.g., a constant value and a sine wave) and verifying that the output equals the expected product at each sample.

**Acceptance Scenarios**:

1. **Given** two signal sources are connected to a Signal Multiply block, **When** the workflow is running, **Then** the output is the sample-by-sample product of both inputs.
2. **Given** three or more signal sources are connected, **When** the workflow is running, **Then** the output is the product of all connected input samples.
3. **Given** only one input is connected to a Signal Multiply block, **When** the workflow is running, **Then** the output passes through the single input unchanged.

---

### User Story 5 - Compute the Derivative of a Signal (Priority: P2)

A user wants to compute the rate of change of a signal over time. They add a Derivative block, connect it to a signal source, and the block outputs the time derivative (difference between consecutive samples divided by the time interval).

**Why this priority**: Derivative computation is essential for velocity/acceleration analysis, edge detection, and trend analysis in time-series data.

**Independent Test**: Can be tested by connecting a known linear ramp signal and verifying that the output is a constant value equal to the slope of the ramp.

**Acceptance Scenarios**:

1. **Given** a signal source is connected to a Derivative block, **When** the workflow is running, **Then** the output represents the rate of change of the input signal over time.
2. **Given** a constant input signal, **When** the workflow is running, **Then** the Derivative block outputs zero (or near-zero values).
3. **Given** a step change in the input signal, **When** the step occurs, **Then** the Derivative block outputs a spike corresponding to the instantaneous rate of change.

---

### User Story 6 - Inspect Raw Data as Hex/Text/Bytes (Priority: P3)

A user wants to examine the raw bytes of a data stream for debugging or protocol analysis. They add a Hex Viewer block, connect it to a data source, and see the incoming data displayed in hex, ASCII text, and raw byte representations simultaneously.

**Why this priority**: Raw data inspection is a critical debugging tool, especially when working with binary protocols or diagnosing encoding issues. Lower priority because it serves a narrower debugging use case.

**Independent Test**: Can be tested by connecting a known data source and verifying that the Hex Viewer correctly displays byte values in hexadecimal, the corresponding ASCII characters, and a byte offset column.

**Acceptance Scenarios**:

1. **Given** a data source is connected to a Hex Viewer block, **When** the workflow is running, **Then** incoming data is displayed in a hex dump format with offset, hex bytes, and ASCII columns.
2. **Given** data is streaming into the Hex Viewer, **When** new data arrives, **Then** the display scrolls to show the latest data while keeping history accessible.
3. **Given** the Hex Viewer is displaying data, **When** the user pauses the display, **Then** incoming data continues to buffer but the view freezes for inspection.
4. **Given** the Hex Viewer is displaying data, **When** the user searches for a byte pattern, **Then** matching occurrences are highlighted in the display.

---

### Edge Cases

- What happens when a Bluetooth device goes out of range during active data capture?
- How does the Filter block behave when the cutoff frequency is set to zero or above the Nyquist frequency?
- How does the Signal Multiply block handle mismatched sample rates between inputs?
- What happens when the Derivative block receives its first sample (no previous sample to diff against)?
- How does the Hex Viewer handle extremely high data throughput without lagging the UI?
- What happens when the Spectrum Viewer receives a signal with no meaningful frequency content (e.g., DC-only)?

## Requirements *(mandatory)*

### Functional Requirements

**Bluetooth Input Block**

- **FR-001**: System MUST discover and list Bluetooth devices that are paired with the host operating system.
- **FR-002**: System MUST allow the user to select a paired Bluetooth device and establish a data connection.
- **FR-003**: System MUST stream received Bluetooth data as raw bytes at the rate determined by the device. The block does not interpret or parse the data structure; downstream blocks are responsible for deserialization.
- **FR-004**: System MUST detect connection loss, display a visual disconnection indicator on the block, and automatically attempt reconnection with visible retry status. The workflow MUST NOT crash during disconnection or reconnection attempts.
- **FR-005**: System MUST support Classic Bluetooth (SPP/RFCOMM) serial profiles for data transfer. BLE (GATT) is explicitly out of scope.

**Signal Multiply Block**

- **FR-006**: System MUST accept two or more input connections.
- **FR-007**: System MUST output the sample-by-sample product of all connected inputs.
- **FR-008**: System MUST pass through the input unchanged when only one input is connected.
- **FR-009**: System MUST handle the case where inputs have different sample rates by synchronizing to the slowest input rate.

**Filter Block**

- **FR-010**: System MUST support low-pass, high-pass, band-pass, and band-stop (notch) filter modes.
- **FR-011**: System MUST allow the user to configure cutoff frequency (single for low/high-pass, two for band-pass and band-stop).
- **FR-012**: System MUST allow the user to configure filter order/strength.
- **FR-013**: System MUST apply the filter in real time without introducing perceptible latency for the user.
- **FR-014**: System MUST allow filter parameter changes while the workflow is running.

**Derivative Block**

- **FR-015**: System MUST compute the discrete time derivative of the input signal (difference between consecutive samples divided by the time interval).
- **FR-016**: System MUST output zero (or no output) for the first sample when no previous sample exists.
- **FR-017**: System MUST handle variable sample rates by using actual time intervals between samples.

**Spectrum Viewer Block**

- **FR-018**: System MUST display a live waterfall spectrogram where the vertical axis represents frequency, the horizontal axis represents time, and color intensity represents amplitude.
- **FR-019**: System MUST allow the user to configure the frequency range displayed.
- **FR-020**: System MUST allow the user to configure the color scale/mapping.
- **FR-021**: System MUST update the display in real time as new data arrives.
- **FR-022**: System MUST show a scrolling time axis so the user can observe spectral changes over time.

**Hex Viewer Block**

- **FR-023**: System MUST display incoming data in a hex dump format with byte offset, hexadecimal values, and ASCII representation columns.
- **FR-024**: System MUST auto-scroll to show the latest data while maintaining scrollback history.
- **FR-025**: System MUST allow the user to pause the display for inspection without losing incoming data.
- **FR-026**: System MUST allow the user to search for specific byte patterns in the received data.

### Key Entities

- **Block**: A processing unit on the workflow canvas with typed input/output ports, configuration parameters, and a visual representation.
- **Sample**: A single data point with a value and timestamp, the fundamental unit passed between blocks.
- **Connection**: A directional link between an output port of one block and an input port of another, carrying a stream of samples.
- **Filter Configuration**: The set of parameters defining a filter's behavior — mode (low/high/band-pass), cutoff frequencies, and order.
- **Spectrum Data**: A frequency-domain representation of a signal computed via FFT, rendered as a waterfall spectrogram (time x frequency x amplitude-as-color).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can establish a Bluetooth data connection and see live data within 30 seconds of adding the block.
- **SC-002**: The Filter block produces correctly filtered output that matches expected frequency characteristics for all four filter modes (verified by spectral analysis of output).
- **SC-003**: The Spectrum Viewer updates at a minimum of 10 frames per second during live data capture.
- **SC-004**: The Hex Viewer can display data streams without visible lag or dropped bytes at typical serial/Bluetooth data rates.
- **SC-005**: All six new block types can be added to the canvas, configured, and connected to other blocks using the same interaction patterns as existing blocks.
- **SC-006**: The Signal Multiply block produces mathematically correct output (product of inputs) with less than 0.1% numerical error.
- **SC-007**: Users can build a complete workflow using the new blocks (e.g., Bluetooth input -> Filter -> Spectrum Viewer) without encountering errors or data loss.

## Clarifications

### Session 2026-04-10

- Q: Should the Bluetooth block support Classic Bluetooth (SPP/RFCOMM), BLE (GATT), or both? → A: Classic Bluetooth (SPP/RFCOMM) only.
- Q: Should the Bluetooth block output parsed numeric samples or raw bytes? → A: Raw bytes only. The data structure from the device is unknown; users must use other blocks to deserialize/parse the data before routing to processing blocks.
- Q: Should the Filter block include a band-stop (notch) filter mode? → A: Yes, include band-stop as a fourth filter mode.
- Q: Should the Bluetooth block auto-reconnect on connection loss or require manual reconnection? → A: Auto-reconnect with retries and visible retry status indicator.
- Q: Should the Spectrum Viewer be a waterfall spectrogram or a live snapshot spectrum plot? → A: Waterfall spectrogram (time x frequency, color = amplitude).

## Assumptions

- Bluetooth device pairing is handled by the host operating system; the application only needs to discover and connect to already-paired devices.
- Bluetooth communication uses SPP (Serial Port Profile) / RFCOMM, consistent with the application's serial data focus.
- The Filter block uses standard IIR or FIR filter implementations — the specific algorithm is an implementation detail.
- The Spectrum Viewer uses FFT to compute frequency data.
- The Derivative block uses a simple finite difference method; more advanced differentiation methods are out of scope.
- Sample rate synchronization for the Signal Multiply block uses a simple "wait for all inputs" approach — more advanced interpolation-based sync is out of scope.
- The Hex Viewer maintains a bounded scrollback buffer to prevent unbounded memory growth — the specific buffer size is an implementation detail.
