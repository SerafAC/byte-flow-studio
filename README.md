# ByteFlow Studio

A multiplatform desktop application for connecting to live byte-stream data sources, building visual processing pipelines, and observing or analysing results in real time. Think N8N for electronics and serial data.

## Overview

ByteFlow Studio lets engineers, makers, and researchers compose workflows by dragging and connecting blocks onto a shared canvas. Workflows consist of three block categories:

- **Inputs** — UART/Serial, WebSocket, Signal Simulator
- **Processing** — Byte Parser, Moving Average, FFT, Sampling, Scaling, Summation, Passthrough
- **Analysis** — Line Chart, Bar Chart, FFT Spectrum, Value Display, Data Table

Workflows are saved as `.byteflow` files — a single SQLite database that stores the workflow definition, block configuration, and recorded data sessions.

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.23+, Wails v3 alpha |
| Frontend | TypeScript 5.x, Vue 3.5, Vue Flow 1.x, PrimeVue 4.x, Pinia 2.x, uPlot |
| Storage | SQLite via `modernc.org/sqlite` (CGO-free, cross-compilable) |
| Serial I/O | `go.bug.st/serial` |
| Signal Processing | `gonum` (FFT) |
| Real-time transport | `gorilla/websocket` |

## Project Structure

```
.
├── main.go                  # Application entry point
├── services/                # Wails-exposed services (workflow, pipeline, session)
├── internal/
│   ├── input/               # Input block implementations (UART, WebSocket, Simulator)
│   ├── processing/          # Processing block implementations (FFT, filters, parsers)
│   ├── analysis/            # Analysis block implementations (charts, displays)
│   ├── pipeline/            # Pipeline engine (block execution, data routing)
│   ├── workflow/            # Workflow definition + SQLite store
│   ├── session/             # Data session management and persistence
│   └── logging/             # Structured logging
├── frontend/
│   └── src/
│       ├── components/      # Canvas, block, and panel components
│       ├── stores/          # Pinia stores
│       ├── services/        # Frontend API bindings
│       └── views/           # Top-level views
└── specs/                   # Feature specifications and plans
```

## Getting Started

### Prerequisites

- Go 1.23+
- Node.js 20+
- [Wails v3](https://v3.wails.io/) CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- [Task](https://taskfile.dev/) task runner: `go install github.com/go-task/task/v3/cmd/task@latest`

### Development

```sh
task dev
```

Starts the application with hot-reload for both frontend and backend changes.

### Build

```sh
task build
```

Produces a native executable for the current platform under `bin/`.

### Test

```sh
task test          # run all tests (frontend + Go)
task test:frontend # Vitest (frontend only)
task test:internal # go test ./internal/... (backend only)
```

### Server Mode (headless)

ByteFlow Studio can run without a GUI as an HTTP server — useful for CI, Docker, or remote deployments:

```sh
task build:server
task run:server
```

A Docker image is also available:

```sh
task build:docker
task run:docker
```

## Data Sessions

Every workflow retains the last N complete data sessions (default: 10, user-configurable). Each session stores:

- **Raw timestamped bytes** from Input blocks (millisecond resolution, enabling pipeline replay)
- **Processed values** as they enter each Analysis block (for instant display on re-open)

Both storage layers are independently toggleable to manage file size.

## License

See [LICENSE](LICENSE).
