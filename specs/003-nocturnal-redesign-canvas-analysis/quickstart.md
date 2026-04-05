# Developer Quickstart: Nocturnal Redesign & Canvas-Inline Analysis

**Branch**: `003-nocturnal-redesign-canvas-analysis`
**Date**: 2026-04-05

---

## What This Feature Changes

Two distinct changes shipped together:

1. **New Visual Design** ("Nocturnal Architect") — deep navy palette, glassmorphic floating panels, no hard border lines, Space Grotesk font, redesigned status bar, toggleable block library panel.

2. **Canvas-Inline Analysis** — analysis block nodes embed their live visualization directly on the canvas. Double-click still opens the full-screen detailed view.

---

## File Change Map

### Backend (Go)

| File | Change |
|------|--------|
| `internal/workflow/types.go` | Add `Width`, `Height float64` fields to `BlockDef` |
| `services/workflow_service.go` | Add `UpdateBlockSize(blockId string, width, height float64) error` method |

### Frontend (TypeScript/Vue/SCSS)

| File | Change |
|------|--------|
| `frontend/src/assets/_variables.scss` | Replace all color/token values with Nocturnal Architect palette; add glass/font tokens |
| `frontend/src/assets/main.scss` | Add `@font-face` for Space Grotesk; add uPlot dark theme overrides |
| `frontend/src/assets/fonts/` | **New** — bundle Space Grotesk woff2 files (400, 500, 600) |
| `frontend/src/assets/_mixins.scss` | Add `.canvas-node-inline` mixin for inline analysis wrapper |
| `frontend/src/services/wails.ts` | Add `width?`/`height?` to `BlockDef`; add `updateBlockSize()` function |
| `frontend/src/stores/workflow.ts` | Add `updateBlockSize(id, w, h)` action |
| `frontend/src/views/MainView.vue` | Full layout restructure: floating header, floating sidebars, floating status bar, `libraryOpen` toggle state |
| `frontend/src/components/canvas/BlockNode.vue` | Add inline analysis rendering, `NodeResizer`, HISTORICAL badge passthrough, remove analysis hint text |
| `frontend/src/components/canvas/WorkflowCanvas.vue` | Add `@node-resize-stop` handler; pass node `style` with persisted width/height |
| `frontend/src/components/panels/BlockLibraryPanel.vue` | 2-column card grid layout, ✕ close button, `'close'` emit |
| `frontend/src/components/panels/SessionPanel.vue` | Update styles to glassmorphic tokens |
| `frontend/package.json` | Add `@vue-flow/node-resizer` (pinned version) |

### No-change files
- All analysis block components (`LineChartBlock.vue`, etc.) — mounted as-is inside `BlockNode`
- `useAnalysisBlock.ts` — no changes required (event subscription works for multiple instances)
- `useFullscreen.ts` — no changes required
- All input/processing block config components — styles will update automatically via SCSS token changes

---

## Key Implementation Notes

### 1. Node size persistence flow
```
User drags resize handle
  → VueFlow fires `node-resize-stop`
  → WorkflowCanvas.onNodeResizeStop()
  → workflowStore.updateBlockSize(id, w, h)
  → wails.updateBlockSize(id, w, h)
  → Go WorkflowService.UpdateBlockSize()
  → saveState() flushes to SQLite
```
On load, `BlockDef.width`/`height` come back from `getWorkflow()` and are applied as `node.style` in the VueFlow node array.

### 2. Inline analysis mounting
Inside `BlockNode.vue`, the analysis component is mounted only when `data.category === 'analysis'`:
```vue
<component
  v-if="analysisComponent"
  :is="analysisComponent"
  :block-id="data.id"
  v-bind="data.params ?? {}"
  class="inline-analysis"
/>
```
The `inline-analysis` class constrains height to fill the node body below the header row.

### 3. Simultaneous inline + fullscreen (FR-017)
Both the inline node and the fullscreen overlay mount the same component type for the same `blockId`. Both instances independently subscribe to `'pipeline:data'` events. This works correctly — Wails events are pub/sub; multiple listeners for the same event all fire. No special handling needed.

### 4. HISTORICAL badge (FR-012a)
The inline component already renders its own `historical-badge` via `isHistorical` from `useAnalysisBlock`. No changes to analysis components needed — the badge appears automatically when `session:view-changed` sets `isHistorical = true`.

### 5. Layout structure change
`MainView.vue` changes from a flexbox column layout with docked sidebars to a full-viewport canvas with `position: fixed` floating panels. The canvas fills `100vw × 100vh`. All panels overlay it.

### 6. Block library panel toggle
```typescript
const libraryOpen = ref(true)  // open by default
```
The slim toolbar icon emits `libraryOpen.value = true`. The panel emits `'close'` when ✕ is clicked.

---

## Running the Feature Locally

```bash
# Install new dependency
cd frontend && npm install @vue-flow/node-resizer@<pinned-version>

# Run frontend dev server
npm run dev

# Or full Wails dev build
cd .. && wails3 dev
```

After adding Space Grotesk font files to `frontend/src/assets/fonts/`, rebuild so Vite bundles them.
