# Data Model: Nocturnal Redesign & Canvas-Inline Analysis

**Branch**: `003-nocturnal-redesign-canvas-analysis`
**Phase**: 1 — Design
**Date**: 2026-04-05

---

## Changed Entities

### BlockDef — Extended with Size Fields

The `BlockDef` struct/interface gains two optional fields to support resizable analysis nodes. All other fields are unchanged.

**Go (`internal/workflow/types.go`)**:
```go
type BlockDef struct {
    ID           string         `json:"id"`
    Type         string         `json:"type"`
    Category     string         `json:"category"`
    Label        string         `json:"label"`
    Params       map[string]any `json:"params"`
    PositionX    float64        `json:"positionX"`
    PositionY    float64        `json:"positionY"`
    Width        float64        `json:"width,omitempty"`         // NEW
    Height       float64        `json:"height,omitempty"`        // NEW
    Status       string         `json:"status,omitempty"`
    ErrorMessage string         `json:"errorMessage,omitempty"`
}
```

**TypeScript (`frontend/src/services/wails.ts`)**:
```typescript
export interface BlockDef {
  id: string
  type: string
  category: 'input' | 'processing' | 'analysis'
  label: string
  params: Record<string, unknown>
  positionX: number
  positionY: number
  width?: number    // NEW — undefined means use type default
  height?: number   // NEW — undefined means use type default
  status?: string
  errorMessage?: string
}
```

**Backward compatibility**: `omitempty` in Go means existing `.byteflow` files without these fields deserialize with zero values (`0`). The frontend treats `0` / `undefined` as "use default minimum size" — no migration needed.

---

### Analysis Block Default & Minimum Sizes

Per FR-011, each analysis block type has a defined minimum and default canvas size:

| Block Type      | Min Width | Min Height | Default Width | Default Height |
|-----------------|-----------|------------|---------------|----------------|
| `line-chart`    | 240px     | 160px      | 320px         | 200px          |
| `value-display` | 140px     | 100px      | 180px         | 130px          |
| `bar-chart`     | 240px     | 160px      | 320px         | 200px          |
| `fft-spectrum`  | 240px     | 200px      | 360px         | 240px          |
| `data-table`    | 200px     | 140px      | 280px         | 180px          |

Non-analysis blocks retain their current CSS-driven size (no explicit width/height stored).

---

## New Service Method

### Go: `UpdateBlockSize`

Added to `WorkflowService` alongside the existing `UpdateBlockPosition`:

```go
// UpdateBlockSize persists a new canvas width and height for a block.
// Only meaningful for analysis-category blocks; safe to call on any block type.
func (s *WorkflowService) UpdateBlockSize(blockId string, width, height float64) error
```

- Updates `BlockDef.Width` and `BlockDef.Height` in the in-memory workflow.
- Calls `s.saveState()` to flush the change to the SQLite JSON blob.
- Returns an error if `blockId` is not found (same pattern as `UpdateBlockPosition`).

### TypeScript: `updateBlockSize`

Added to `frontend/src/services/wails.ts`:

```typescript
export async function updateBlockSize(blockId: string, width: number, height: number): Promise<void> {
  const svc = await workflowSvc()
  return cast<void>(svc.UpdateBlockSize(blockId, width, height))
}
```

---

## Design Token Changes

The SCSS variable file (`frontend/src/assets/_variables.scss`) is fully replaced with the Nocturnal Architect palette. Canonical token names are preserved where semantically equivalent; values change.

### Color Tokens

| Token | Old Value | New Value | Usage |
|-------|-----------|-----------|-------|
| `$bg-root` | `#1b2636` | `#060e20` | App/canvas background |
| `$bg-surface` | `#243447` | `#06122d` | Secondary sidebars, toolbar |
| `$bg-card` | `#1e2d3d` | `#0d1b38` | Node bodies, dialogs |
| `$bg-block` | `#2d3f52` | `#002867` | Active/lifted surfaces |
| `$bg-hover` | `#3a5068` | `#00225a` | Hover/highlighted nodes |
| `$border-color` | `#3a4a5c` | `#2b4680` | Ghost borders (15% opacity use) |
| `$text-primary` | `#e2e8f0` | `#dee5ff` | Primary text |
| `$text-secondary` | `#9ca3af` | `#91aaeb` | Muted metadata |
| `$text-muted` | `#6b7280` | `#4a6090` | Disabled / placeholder |
| `$color-primary` | `#3b82f6` | `#4381cf` | Blue accent (input category) |
| `$color-secondary` | `#8b5cf6` | `#9d50cf` | Purple accent (processing category) |
| `$color-success` | `#10b981` | `#49b393` | Green accent (analysis category) |
| `$color-warning` | `#f59e0b` | `#d1b452` | Yellow accent (pause/warn) |
| `$color-danger` | `#ef4444` | `#cf546c` | Red accent (stop/error) |

### New Tokens

```scss
// Glassmorphism
$glass-bg:     rgba(13, 27, 56, 0.85);   // floating panel fill
$glass-blur:   16px;                      // backdrop blur
$glass-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);

// Typography
$font-display: 'Space Grotesk', system-ui, sans-serif;  // headings, labels, node names
$font-body:    'Inter', system-ui, sans-serif;           // data, forms, status bar

// Ghost border (accessibility boundary, barely visible)
$ghost-border: rgba(43, 70, 128, 0.15);
```

---

## UI State (Frontend-only, not persisted)

| State | Location | Type | Description |
|-------|----------|------|-------------|
| `libraryOpen` | `MainView.vue` | `ref<boolean>` | Whether block library panel is visible |
| `fullscreenBlockId` | `useFullscreen.ts` | `ref<string \| null>` | Which block (if any) is in detailed view |

These are ephemeral UI state — no persistence required.
