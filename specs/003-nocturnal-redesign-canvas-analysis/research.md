# Research: Nocturnal Redesign & Canvas-Inline Analysis

**Branch**: `003-nocturnal-redesign-canvas-analysis`
**Phase**: 0 — Pre-design research
**Date**: 2026-04-05

---

## 1. Vue Flow Node Resizing

**Decision**: Add `@vue-flow/node-resizer` as a new pinned dependency.

**Rationale**: The package is the first-party Vue Flow resizer that integrates cleanly with Vue Flow's internal node dimension tracking. It provides a `<NodeResizer>` component that renders drag handles and updates `node.style.width`/`node.style.height` automatically. Vue Flow then passes those dimensions back through `onNodeResizeStop` events. The alternative (CSS `resize: both` on the wrapper) does not fire Vue Flow events, making it impossible to read the final dimensions for persistence.

**Alternatives considered**:
- `resize: both` CSS on node wrapper — rejected: no Vue Flow event integration, dimensions not accessible reactively.
- Custom drag-resize implementation — rejected: violates Principle V (YAGNI); `@vue-flow/node-resizer` is purpose-built for this.

**Usage pattern**:
```vue
<NodeResizer :min-width="240" :min-height="160" :is-resizable="true" />
```
`@node-resize-stop` on the `<VueFlow>` element fires `{ node: Node }` where `node.style.width` and `node.style.height` contain the final dimensions as CSS strings (e.g., `"320px"`). Parse with `parseFloat()`.

---

## 2. Space Grotesk Font Bundling

**Decision**: Download Space Grotesk woff2 files (weights 400, 500, 600) and place them in `frontend/src/assets/fonts/`. Declare via `@font-face` in `main.scss`.

**Rationale**: Wails v3 runs as a desktop app — no guarantee of internet access. Bundling ensures the font renders identically offline and in CI/CD builds. woff2 is universally supported by Chromium (Windows/Linux) and WebKit (macOS), the two WebView engines Wails v3 targets. No woff fallback needed for desktop.

**Source**: Google Fonts open-source repository (OFL license — free to bundle).

**Alternatives considered**:
- Google Fonts CDN `@import` — rejected: requires internet, fails offline.
- System font stack only — rejected: Space Grotesk is specified in the design reference as the brand identity font; falling back to system fonts breaks the design intent.

**`@font-face` pattern**:
```scss
@font-face {
  font-family: 'Space Grotesk';
  src: url('./fonts/SpaceGrotesk-Regular.woff2') format('woff2');
  font-weight: 400;
  font-display: swap;
}
// Repeat for 500, 600
```

---

## 3. Glassmorphism & `backdrop-filter` Support

**Decision**: Use `backdrop-filter: blur(12px)` on floating panels. Safe to use without a fallback.

**Rationale**: Wails v3 on Windows uses WebView2 (Chromium-based, version 88+, fully supports `backdrop-filter`). On macOS it uses WebKit (Safari engine, supports `backdrop-filter` since Safari 9). On Linux it uses WebKitGTK (supports `backdrop-filter` since GTK 4 + WebKitGTK 2.36). All current Wails v3 targets support this property. No fallback (solid background) is needed since the minimum supported WebView versions all include it.

**Panel implementation**:
```scss
.floating-panel {
  background: rgba(30, 34, 46, 0.85);    // surface-variant at ~85% opacity
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);   // WebKit prefix for macOS
  border-radius: 4px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
}
```

---

## 4. uPlot Dark Theme Integration

**Decision**: Apply global CSS overrides targeting uPlot's internal class names to match the Nocturnal Architect palette. Do not patch uPlot source.

**Rationale**: uPlot does not have a built-in theming API — it renders to a canvas and uses CSS classes for legend/axis text. The correct approach is scoped global overrides added to `main.scss` after the `uplot/dist/uPlot.min.css` import.

**Key overrides**:
```scss
// uPlot theming — applied globally after uplot import
.uplot {
  background: transparent;
}
.u-legend {
  color: #91aaeb;  // on-surface-variant
  font-size: 10px;
}
.u-title {
  color: #dee5ff;  // on-surface
  font-size: 11px;
}
```

Axis tick/line colors are set via the `axes[].stroke` option in the uPlot options object (already `'#3b82f6'` pattern in the codebase) — these stay in the component; only label colors are global CSS overrides.

---

## 5. Inline Analysis Rendering Architecture

**Decision**: Mount analysis components directly inside `BlockNode.vue` using a `v-if="data.category === 'analysis'"` guard and a component map matching the existing `analysisComponentMap` in `MainView.vue`.

**Rationale**: Analysis components (`LineChartBlock.vue`, etc.) already subscribe independently to `'pipeline:data'` Wails events via `useAnalysisBlock` composable. Each mounted instance registers its own event listener and filters by `blockId`. Mounting the same component for the same `blockId` both inline (in `BlockNode`) and in the fullscreen overlay (in `MainView`) works correctly — both receive events simultaneously, satisfying FR-017. No data subscription architecture changes are required.

**Simultaneous rendering validation**:
- `useAnalysisBlock` calls `Events.On('pipeline:data', opts.onData)` — this is a pub/sub pattern; multiple subscribers for the same event are fully supported by `@wailsio/runtime`.
- Each instance's `onData` handler filters `event.data.blockId !== props.blockId` — no cross-contamination.

---

## 6. `BlockDef` Width/Height Extension

**Decision**: Add `Width` and `Height float64` fields to the Go `BlockDef` struct with `json:"width,omitempty"` and `json:"height,omitempty"` tags. Add corresponding optional `width?: number` and `height?: number` fields to the TypeScript `BlockDef` interface. Add a `UpdateBlockSize(blockId string, width, height float64)` method to `WorkflowService`.

**Rationale**: The workflow definition is stored as a JSON blob — adding new `omitempty` fields is fully backward-compatible. Existing workflow files without width/height fields will deserialize with zero values, which maps to "use default size" on the frontend. No SQL schema migration needed.

**Persistence flow**:
1. Node resize stops → `VueFlow` fires `node-resize-stop` with `node.style.width`/`height`
2. `WorkflowCanvas.vue` calls `workflowStore.updateBlockSize(id, w, h)`
3. Store dispatches `wails.updateBlockSize(id, w, h)` → Go `UpdateBlockSize()`
4. Go updates `BlockDef` in memory and calls `saveState()` (same pattern as `UpdateBlockPosition`)
5. On next `loadWorkflow()`, width/height are restored and passed to Vue Flow node `style` prop

**Node style injection pattern** in `WorkflowCanvas.vue`:
```typescript
workflowStore.blocks.map(b => ({
  id: b.id,
  type: 'block',
  position: { x: b.positionX, y: b.positionY },
  data: b,
  style: b.width ? { width: `${b.width}px`, height: `${b.height}px` } : undefined,
}))
```

---

## 7. Block Library Panel Toggle

**Decision**: Manage `libraryOpen` as a `ref<boolean>` in `MainView.vue`. Pass it as a prop to `BlockLibraryPanel` and emit a `'close'` event. The slim toolbar icon calls `libraryOpen.value = true`.

**Rationale**: The panel open/close state is UI-only, ephemeral, and does not need to be persisted. A simple reactive boolean in the parent component is the simplest correct implementation (Principle V). No Pinia store entry is needed for this.

---

## 8. Status Bar System Metrics

**Decision**: Use static placeholder values for CPU and RAM in the initial implementation. Display the pipeline state (from `pipelineStore`) and serial connection info (from `pipelineStore`) as live data; CPU/RAM as static `—` or `N/A` strings.

**Rationale**: The spec assumption explicitly states "placeholder/static data initially if the backend does not yet expose these metrics; the visual structure is required, not the live data source." Adding a Go system-metrics polling API is out of scope for this feature (Principle V). The status bar layout and styling is the deliverable.
