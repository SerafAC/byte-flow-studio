# Contract: SessionService

**Type**: Wails v3 Go Service (exposed to Vue frontend via generated TypeScript bindings)
**File**: `services/session_service.go`
**Generated bindings**: `frontend/bindings/SessionService.*`

---

## Purpose

Manages session lifecycle from the frontend's perspective: listing available sessions, switching the active view between live and historical sessions, configuring session recording preferences, and deleting sessions.

---

## Methods

### `ListSessions() ([]SessionMeta, error)`

Returns metadata for all persisted sessions for the current workflow, ordered by `startTime` descending (most recent first).

**Returns**:
```go
type SessionMeta struct {
    ID                     string
    StartTime              int64    // Unix ms
    EndTime                int64    // Unix ms; 0 if active
    EndReason              string   // "user-stop" | "error" | "active"
    RawLayerEnabled        bool
    ProcessedLayerEnabled  bool
    DataVolume             int64    // Total bytes stored
}
```

---

### `GetActiveSessionID() string`

Returns the ID of the session currently displayed in Analysis blocks. This may be the live session (while Running) or a historical session (during historical playback mode).

---

### `SetViewSession(sessionId string) error`

Switches the Analysis blocks' display to show data from the specified historical session.

**Preconditions**: Flow MUST be in `Idle` or `Paused` state. Cannot switch sessions during a live run (use the live session for that).

**On success**: Emits `session:view-changed`; triggers `pipeline:data` events replaying the session's processed layer from the SQLite store to all subscribed Analysis blocks.

**Errors**:
- `"session not found: <sessionId>"`
- `"cannot switch session view while flow is Running"`

**Logging**: `INFO session_view_switch session_id=<id>`

---

### `GetSessionConfig() SessionConfig`

Returns the current per-workflow session recording configuration.

```go
type SessionConfig struct {
    StoreRaw        bool
    StoreProcessed  bool
    MaxSessions     int
}
```

---

### `UpdateSessionConfig(config SessionConfig) error`

Updates the session recording configuration for the current workflow.

**Parameters**:
- `config.MaxSessions` MUST be ≥ 1
- `config.StoreRaw` and `config.StoreProcessed` MUST NOT both be `false` simultaneously (at least one layer must be recorded)

**Side effects**: Persists updated config to the `workflow` table; if `MaxSessions` is decreased and the current session count exceeds the new limit, oldest sessions are trimmed immediately.

**Errors**:
- `"maxSessions must be at least 1"`
- `"at least one storage layer (raw or processed) must be enabled"`

**Logging**: `INFO session_config_updated store_raw=<bool> store_processed=<bool> max_sessions=<n>`

---

### `DeleteSession(sessionId string) error`

Permanently deletes a session and all its data from the SQLite store.

**Preconditions**: Cannot delete the currently active live session.

**Errors**:
- `"session not found: <sessionId>"`
- `"cannot delete the currently active session"`

**Logging**: `INFO session_deleted session_id=<id> data_volume=<bytes>`

---

### `GetStorageEstimate() StorageEstimate`

Returns an estimate of current and projected session storage, used to inform the user when changing session config (FR-033).

```go
type StorageEstimate struct {
    CurrentTotalBytes      int64
    AverageSessionBytes    int64   // Average of retained sessions; 0 if no sessions yet
    ProjectedMaxBytes      int64   // AverageSessionBytes × MaxSessions
    WarningThresholdBytes  int64   // Configured threshold (default: 1 GB)
    ExceedsThreshold       bool
}
```

---

## Events Emitted

### `session:view-changed`

Emitted when the active display session changes (live → historical or between historical sessions).

```typescript
type SessionViewChangedEvent = {
  sessionId: string;
  mode: "live" | "historical";
}
```

### `session:list-updated`

Emitted when the session list changes (new session created, session deleted, or session trimmed by retention policy).

```typescript
type SessionListUpdatedEvent = {
  sessions: SessionMeta[];
}
```

---

## Logging Contract

| Field | Type | Description |
|---|---|---|
| `session_id` | string | Session being operated on |
| `data_volume` | int64 | Bytes stored/freed |
| `max_sessions` | int | Retention limit |
| `error` | error | Error value on failure |
