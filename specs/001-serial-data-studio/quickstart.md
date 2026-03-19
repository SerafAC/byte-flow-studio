# Quickstart: ByteFlow Studio Development Environment

**Feature**: 001-serial-data-studio | **Date**: 2026-03-19

---

## Prerequisites

### All Platforms

| Tool | Version | Install |
|---|---|---|
| Go | 1.23+ | https://go.dev/dl/ |
| Node.js | 20+ LTS | https://nodejs.org |
| wails3 CLI | latest alpha | `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` |

### Platform-Specific

**Windows**
- WebView2 Runtime (pre-installed on Windows 10/11; verify in Programs)
- No extra steps required

**macOS**
- Xcode Command Line Tools: `xcode-select --install`
- Note: `go.bug.st/serial` requires CGO for USB enumeration on macOS. Ensure `clang` is available.

**Linux (Ubuntu 22.04+)**
```bash
sudo apt install gcc libgtk-3-dev libwebkit2gtk-4.1-dev
```

**Linux (Arch)**
```bash
sudo pacman -S gcc gtk3 webkit2gtk-4.1
```

---

## Project Initialisation

> If the repo is already initialised (this project exists), skip to **Running in Development**.

```bash
# Verify Wails v3 installation
wails3 doctor

# Init the project (run once, in repo root)
wails3 init -n "ByteFlowStudio" -t vue

# Install frontend dependencies
cd frontend && npm install
```

The `wails3 init` command creates:
- `main.go` — application entry point
- `app.go` — application service
- `wails.json` — project config
- `frontend/` — Vite + Vue 3 scaffold

---

## Add Go Dependencies

```bash
# Serial port I/O
go get go.bug.st/serial

# FFT
go get gonum.org/v1/gonum/dsp/fourier

# SQLite (CGO-free)
go get modernc.org/sqlite

# WebSocket client
go get github.com/gorilla/websocket

# Test assertions
go get github.com/stretchr/testify
```

---

## Add Frontend Dependencies

```bash
cd frontend

# DAG canvas editor
npm install @vue-flow/core @vue-flow/background @vue-flow/controls @vue-flow/minimap

# PrimeVue UI components
npm install primevue @primevue/themes primeicons

# uPlot for high-frequency real-time charts
npm install uplot uplot-wrappers

# State management
npm install pinia

# Auto-layout for canvas (optional, used for "auto-arrange" feature)
npm install dagre @types/dagre
```

---

## Running in Development

```bash
# From repo root — starts Go backend + Vite HMR frontend
wails3 dev
```

The dev server:
- Watches Go files and recompiles on change
- Reloads the frontend via Vite HMR
- Regenerates TypeScript bindings in `frontend/bindings/` on each Go compilation
- Opens the app window automatically

**Important**: After adding a new Go method to a Service, the bindings are regenerated on the next `wails3 dev` restart. The TypeScript types in `frontend/bindings/` are authoritative — do not edit them manually.

---

## Running Tests

### Go Backend

```bash
# All tests
go test ./...

# A specific package
go test ./internal/pipeline/...

# With race detector (recommended for goroutine-heavy pipeline code)
go test -race ./...

# With verbose output
go test -v ./internal/input/...
```

### Frontend (Vitest)

```bash
cd frontend

# Run once
npm run test

# Watch mode
npm run test:watch
```

---

## Building for Production

```bash
# Current platform
wails3 build

# Output: build/bin/ByteFlowStudio (Linux/macOS) or build/bin/ByteFlowStudio.exe (Windows)
```

Cross-compilation targets (from CI):
```bash
# Linux
GOOS=linux GOARCH=amd64 wails3 build

# Windows
GOOS=windows GOARCH=amd64 wails3 build

# macOS (must be run on macOS for code signing)
GOOS=darwin GOARCH=amd64 wails3 build
GOOS=darwin GOARCH=arm64 wails3 build  # Apple Silicon
```

> **Note**: `modernc.org/sqlite` is CGO-free, so all cross-compilation targets work from any host OS. The only exception is macOS where `go.bug.st/serial` requires CGO for USB port enumeration — macOS builds should be compiled on macOS runners.

---

## Project Structure Reference

See [plan.md](plan.md) → **Project Structure** section for the full annotated directory tree.

---

## Key Development Workflows

### Adding a New Processing Block

1. Create `internal/processing/<blockname>.go` — implement the `Block` interface.
2. Write tests in `internal/processing/<blockname>_test.go` (test-first).
3. Register in `internal/processing/registry.go` via `Register("type-name", NewBlockName)`.
4. Add a `BlockTypeDescriptor` in `WorkflowService.GetAvailableBlockTypes()`.
5. Create `frontend/src/components/blocks/processing/<BlockName>Config.vue`.

### Adding a New Analysis Block

1. Create `internal/pipeline/<blockname>.go` — implement `AnalysisBlock` interface.
2. Write tests first.
3. Register in the block registry.
4. Create `frontend/src/components/blocks/analysis/<BlockName>Block.vue` with uPlot or PrimeVue chart.
5. Add the block's event subscription in `pipeline.ts` Pinia store.

### Wails Bindings Refresh

If TypeScript bindings are stale (after Go service method changes), restart `wails3 dev` or run:
```bash
wails3 generate bindings
```

---

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `BYTEFLOW_LOG_LEVEL` | Log level: `DEBUG`, `INFO`, `WARN`, `ERROR` | `INFO` |
| `BYTEFLOW_LOG_FORMAT` | Log format: `text` (dev) or `json` (prod) | `text` |
| `BYTEFLOW_DATA_DIR` | Override default data directory for `.byteflow` files | OS-specific user data dir |

These are read at startup in `internal/logging/logger.go` and `main.go`. They MUST NOT be committed to version control; use a `.env.local` file (gitignored) for local overrides.
