// Package session manages session lifecycle and data storage.
package session

// SessionConfig holds per-workflow session recording configuration.
type SessionConfig struct {
	StoreRaw       bool
	StoreProcessed bool
	MaxSessions    int
}

// Session represents a bounded recording from one Start → Stop run.
type Session struct {
	ID                    string
	WorkflowID            string
	StartTime             int64  // Unix ms
	EndTime               int64  // Unix ms; 0 if active
	EndReason             string // "user-stop" | "error" | "active"
	RawLayerEnabled       bool
	ProcessedLayerEnabled bool
	DataVolume            int64 // Total bytes stored
}

// SessionMeta is a lightweight view of a session for listing purposes.
type SessionMeta struct {
	ID                    string
	StartTime             int64
	EndTime               int64
	EndReason             string
	RawLayerEnabled       bool
	ProcessedLayerEnabled bool
	DataVolume            int64
}

// RawDataRecord is one record in the raw byte layer.
type RawDataRecord struct {
	SessionID string
	BlockID   string
	Timestamp int64
	Data      []byte
}

// ProcessedDataRecord is one record in the processed value layer.
type ProcessedDataRecord struct {
	SessionID string
	BlockID   string
	Timestamp int64
	Values    []float64
}
