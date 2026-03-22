// Package workflow manages workflow persistence and the SQLite schema.
package workflow

// BlockDef is the persisted representation of a block in a workflow.
type BlockDef struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Category     string         `json:"category"`
	Label        string         `json:"label"`
	Params       map[string]any `json:"params"`
	PositionX    float64        `json:"positionX"`
	PositionY    float64        `json:"positionY"`
	Status       string         `json:"status,omitempty"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
}

// ConnectionDef is the persisted representation of a connection in a workflow.
type ConnectionDef struct {
	ID          string `json:"id"`
	FromBlockID string `json:"fromBlockId"`
	FromPortID  string `json:"fromPortId"`
	ToBlockID   string `json:"toBlockId"`
	ToPortID    string `json:"toPortId"`
}

// SessionConfig holds per-workflow session recording configuration.
type SessionConfig struct {
	StoreRaw       bool `json:"storeRaw"`
	StoreProcessed bool `json:"storeProcessed"`
	MaxSessions    int  `json:"maxSessions"`
}

// DefaultSessionConfig returns a SessionConfig with sensible defaults.
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		StoreRaw:       true,
		StoreProcessed: true,
		MaxSessions:    10,
	}
}

// Workflow is the top-level container for the canvas definition, JSON-serialisable.
type Workflow struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	CreatedAt     int64           `json:"createdAt"`
	UpdatedAt     int64           `json:"updatedAt"`
	Blocks        []BlockDef      `json:"blocks"`
	Connections   []ConnectionDef `json:"connections"`
	SessionConfig SessionConfig   `json:"sessionConfig"`
}
