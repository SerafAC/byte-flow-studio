# Backend API Contract: Nocturnal Redesign & Canvas-Inline Analysis

**Type**: Wails v3 Go service method (exposed to frontend via generated bindings)
**Date**: 2026-04-05

---

## New Method: `WorkflowService.UpdateBlockSize`

**Purpose**: Persist the canvas width and height of a block after the user resizes it.

**Signature**:
```go
func (s *WorkflowService) UpdateBlockSize(blockId string, width, height float64) error
```

**Parameters**:

| Parameter | Type | Constraints | Description |
|-----------|------|-------------|-------------|
| `blockId` | `string` | Non-empty; must match an existing block ID | The block whose size is being updated |
| `width` | `float64` | `> 0` | New canvas width in logical pixels |
| `height` | `float64` | `> 0` | New canvas height in logical pixels |

**Returns**: `error` — non-nil if the blockId is not found in the current workflow.

**Side effects**: Updates the in-memory `BlockDef.Width` and `BlockDef.Height`, then calls `saveState()` to flush the full workflow JSON to SQLite. Identical pattern to `UpdateBlockPosition`.

**Error cases**:
- Block ID not found → return `fmt.Errorf("block %s not found", blockId)`
- Invalid dimensions (≤ 0) → return `fmt.Errorf("invalid size: width=%f height=%f", width, height)`

---

## Changed Interface: `BlockDef`

New optional fields added to the shared Go struct and TypeScript interface. Fully backward-compatible.

**Go**:
```go
Width  float64 `json:"width,omitempty"`
Height float64 `json:"height,omitempty"`
```

**TypeScript** (`wails.ts`):
```typescript
width?:  number
height?: number
```

**Semantics**: A zero/absent value means "no persisted size — use the block type's default minimum dimensions." The frontend is responsible for applying the type-specific defaults when these fields are absent.

---

## Unchanged Methods (referenced for context)

These existing methods are called by the resize flow but are not modified:

- `UpdateBlockPosition(blockId string, x, y float64) error` — unchanged
- `GetWorkflow() (*Workflow, error)` — returns `BlockDef` with new width/height fields; zero values for existing blocks
- `RestoreBlocks(blocks []BlockDef, connections []ConnectionDef) error` — accepts new fields transparently (used by undo/redo)
