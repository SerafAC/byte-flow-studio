# Research: New Signal Blocks

**Feature**: 004-new-signal-blocks  
**Date**: 2026-04-10

## R1: Bluetooth Classic (SPP/RFCOMM) in Go

### Decision
Use platform-level virtual serial ports for data transfer. SPP/RFCOMM devices appear as virtual serial ports on all target platforms:
- **Linux**: `/dev/rfcomm*` (via BlueZ `rfcomm bind`)
- **macOS**: `/dev/tty.*-SPPDev` or `/dev/tty.*-SerialPort`
- **Windows**: `COM*` ports (paired BT devices appear in Device Manager as serial ports)

For **device discovery** (listing paired Bluetooth devices), use platform-specific commands invoked via `os/exec`:
- **Linux**: `bluetoothctl devices Paired` (BlueZ CLI)
- **macOS**: `system_profiler SPBluetoothDataType -json`
- **Windows**: PowerShell `Get-PnpDevice -Class Bluetooth`

For **data transfer**, reuse the existing `go.bug.st/serial` library since SPP devices present as serial ports.

### Rationale
- Avoids adding a heavy Bluetooth library dependency (most Go BT libraries target BLE, not Classic)
- `go.bug.st/serial` is already a project dependency and handles cross-platform serial I/O
- SPP is explicitly a serial port emulation protocol — using the serial library is semantically correct
- Device discovery via OS commands is simple, reliable, and avoids CGO dependencies
- Keeps the codebase CGO-free (consistent with `modernc.org/sqlite` choice)

### Alternatives Considered
- **`tinygo.org/x/bluetooth`**: Primarily BLE-focused; Classic Bluetooth support is limited and platform coverage is incomplete. Would add CGO dependency on some platforms.
- **Direct BlueZ D-Bus API (Linux only)**: More programmatic but Linux-only; not cross-platform. Adds complexity for one platform.
- **Dedicated Go SPP library**: No mature, well-maintained option exists for Classic Bluetooth SPP across all three platforms.

### Auto-Reconnection Strategy
Use exponential backoff with jitter (1s, 2s, 4s, 8s, max 30s). Pattern mirrors the existing WebSocket block's reconnect logic in `internal/input/websocket.go`. On reconnect success, resume data streaming. On context cancellation, stop retrying.

---

## R2: IIR Filter Implementation (Butterworth)

### Decision
Implement Butterworth IIR filter with configurable order (1-8). Use the bilinear transform method to compute filter coefficients from analog prototype. Apply as a Direct Form II Transposed structure for numerical stability.

### Rationale
- Butterworth has maximally flat passband response — the most intuitive "standard" filter for general-purpose use
- IIR is computationally cheaper than FIR for equivalent frequency selectivity — important for real-time processing
- Bilinear transform is well-documented and straightforward to implement
- Direct Form II Transposed has better numerical properties than Direct Form I for floating-point arithmetic
- Filter order 1-8 covers all practical use cases without excessive complexity

### Alternatives Considered
- **FIR filters**: More computationally expensive for sharp cutoffs; latency proportional to filter length. Better phase characteristics but not needed for this use case.
- **Chebyshev/Elliptic**: Sharper rolloff but with ripple — more complex to configure, less intuitive for users.
- **External Go DSP library**: No mature library with ready-made filter design. Implementing coefficients directly is cleaner than pulling in an unmaintained dependency.

### Filter Mode Implementation
- **Low-pass**: Standard Butterworth lowpass prototype, transformed via bilinear transform
- **High-pass**: Frequency inversion of lowpass prototype (s → 1/s in analog domain)
- **Band-pass**: Lowpass-to-bandpass transform (s → (s² + ω₀²)/(s·BW))
- **Band-stop**: Lowpass-to-bandstop transform (s → (s·BW)/(s² + ω₀²))

### Real-Time Parameter Changes
When filter parameters change mid-stream, recompute coefficients and reset filter state (delay line). This causes a brief transient but avoids the complexity of cross-fading between filter states. The transient settles within a few samples for typical filter orders.

---

## R3: Waterfall Spectrogram Rendering

### Decision
Use HTML5 Canvas 2D API for the waterfall spectrogram display. Render each new FFT frame as a single-pixel-height horizontal line, shifting existing content up (or using `drawImage` self-copy for efficient scrolling).

### Rationale
- Canvas 2D is sufficient for spectrogram rendering (essentially pixel-level color mapping)
- No additional dependencies needed
- The existing FFT processing block (`internal/processing/fft.go`) already computes magnitude spectra — the Spectrum Viewer analysis block receives FFT output and renders it
- Self-copy scroll (`ctx.drawImage(canvas, 0, -1)`) is GPU-accelerated in modern browsers and handles 10+ FPS easily
- Color mapping (amplitude → color) uses a lookup table (e.g., viridis, magma, or custom gradient)

### Alternatives Considered
- **WebGL**: More powerful but unnecessary complexity for 2D spectrogram; Canvas 2D performance is sufficient at target frame rates.
- **uPlot heatmap plugin**: uPlot doesn't natively support heatmap/spectrogram mode. Would require significant custom work.
- **Third-party spectrogram library**: Adds dependency; most are designed for standalone use, not embedded in Vue Flow nodes.

### FFT Data Flow
The Spectrum Viewer block does NOT perform its own FFT. The user must connect an FFT processing block upstream:
```
Signal Source → FFT Block → Spectrum Viewer
```
The Spectrum Viewer receives numeric `DataChunk` values (FFT magnitudes) and maps them to colors. This follows the modular block architecture and avoids duplicating FFT logic.

---

## R4: Hex Viewer Rendering

### Decision
Implement as a virtualized scrolling view using a fixed-width monospace layout with three columns: offset (hex), hex bytes (grouped by 16 per row), and ASCII representation. Use a ring buffer in the backend analysis block to limit memory. Frontend renders only visible rows.

### Rationale
- Classic hex dump layout is universally understood for binary data inspection
- Virtual scrolling prevents DOM bloat at high data rates
- Ring buffer (bounded at configurable max bytes, default 64KB) prevents unbounded memory growth
- 16 bytes per row is the standard hex dump format

### Search Implementation
Byte pattern search operates on the ring buffer content. User enters hex pattern (e.g., `FF 00 A5`) or ASCII string. Matches are highlighted with a distinct background color. Search runs on the backend to avoid shipping entire buffer to frontend.

### Pause/Resume
Pause freezes the frontend display position but the backend continues buffering. On resume, the view jumps to the latest data. Buffered-while-paused data is retained in the ring buffer (subject to buffer limits).

---

## R5: Signal Multiply — Sample Synchronization

### Decision
Use a "wait for all inputs" strategy. The block maintains one pending sample per input port. When all ports have a pending sample, compute the product and emit. If inputs arrive at different rates, faster inputs overwrite their pending slot (latest-value-wins).

### Rationale
- Simple and predictable behavior
- Avoids the complexity of interpolation or resampling
- "Latest value" semantics are intuitive for real-time signal processing
- Consistent with the spec's stated assumption about sync strategy

### Alternatives Considered
- **Interpolation-based sync**: Accurate but significantly more complex; explicitly out of scope per spec.
- **Buffer-and-align by timestamp**: Requires timestamp matching with tolerance; adds latency and complexity.
- **Output at fastest rate**: Would require sample-and-hold for slower inputs; more complex than latest-value.

---

## R6: Derivative — Finite Difference Method

### Decision
Use backward finite difference: `dy/dt = (y[n] - y[n-1]) / (t[n] - t[n-1])`. First sample outputs zero (no previous sample). Use actual timestamps from DataChunk for `dt` calculation to handle variable sample rates.

### Rationale
- Simplest correct implementation
- Handles variable sample rates naturally via timestamp-based dt
- No buffering needed (only stores previous sample)
- Zero output for first sample is explicitly specified in FR-016

### Alternatives Considered
- **Central difference**: Better accuracy but requires lookahead (one sample delay); adds complexity.
- **Savitzky-Golay derivative**: Smoothed derivative but requires window buffer; out of scope.
