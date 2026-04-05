# Implementation Plan: Nocturnal Redesign & Canvas-Inline Analysis

**Branch**: `003-nocturnal-redesign-canvas-analysis` | **Date**: 2026-04-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/003-nocturnal-redesign-canvas-analysis/spec.md`

---

## Summary

Applies the "Nocturnal Architect" design system to ByteFlow Studio (deep navy palette, glassmorphic floating panels, Space Grotesk font, no hard border lines) and changes analysis block nodes to render their live data visualization inline on the canvas. The full-screen detailed view is retained, triggered by double-click, and runs simultaneously with the inline view. Analysis node sizes are persisted to the workflow file via a new `Width`/`Height` field on `BlockDef`.

---

## Technical Context

**Language/Version**: Go 1.23+ (backend), TypeScript 5.x (frontend)
**Primary Dependencies**: Vue 3.2, Wails v3 alpha, Vue Flow 1.48, `@vue-flow/node-resizer` (new), uPlot 1.6, PrimeVue 4.5, Pinia 3, SCSS (sass)
**Storage**: SQLite via `modernc.org/sqlite` — workflow as JSON blob; `BlockDef` extended with `width`/`height float64` (backward-compatible, `omitempty`)
**Testing**: Vitest (frontend unit tests)
**Target Platform**: Desktop (macOS/Linux/Windows via Wails v3 WebView)
**Performance Goals**: 5+ simultaneous inline chart updates at live pipeline data rates without frame drops
**Constraints**: Fully offline-capable (Space Grotesk bundled locally, no CDN); `backdrop-filter` safe on all Wails v3 WebView targets
**Scale/Scope**: Single-user desktop app; ~10–20 canvas nodes typical; all 5 analysis block types

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| **I. Spec-Driven** | ✅ PASS | `spec.md` complete; all `[NEEDS CLARIFICATION]` markers resolved in clarification session 2026-04-05 |
| **II. Test-First** | ✅ PASS | Tasks will be structured with tests written before implementation; existing Vitest suite must stay green |
| **III. Component Modularity** | ✅ PASS | Analysis components unchanged; inline rendering reuses existing components via composition. Backend `UpdateBlockSize` is a thin new method following the existing `UpdateBlockPosition` pattern |
| **IV. Observability** | ✅ PASS | No new backend business logic; existing structured logging covers all service calls. No new error paths requiring additional observability |
| **V. YAGNI** | ✅ PASS | No abstractions added beyond concrete present need. `@vue-flow/node-resizer` is the purpose-built solution for this exact requirement. Block library toggle uses a single `ref<boolean>` — no Pinia store entry. Status bar CPU/RAM uses static placeholders per spec assumption |

**Post-design re-check**: ✅ PASS — No violations introduced in Phase 1 design. Data model extends `BlockDef` minimally with two `omitempty` fields. No new architectural layers added.

---

## Project Structure

### Documentation (this feature)

```text
specs/003-nocturnal-redesign-canvas-analysis/
├── plan.md                    # This file
├── research.md                # Phase 0 — technology decisions
├── data-model.md              # Phase 1 — BlockDef changes, design tokens
├── quickstart.md              # Phase 1 — developer guide
├── contracts/
│   ├── backend-api.md         # UpdateBlockSize method contract
│   └── component-interfaces.md # Vue component props/emits contracts
└── tasks.md                   # Phase 2 — /speckit.tasks output (not yet created)
```

### Source Code (affected files)

```text
# Backend
internal/workflow/types.go               # BlockDef: +Width, +Height float64
services/workflow_service.go             # +UpdateBlockSize() method

# Frontend — Assets
frontend/src/assets/_variables.scss      # Full palette replacement (Nocturnal Architect)
frontend/src/assets/_mixins.scss         # +inline-analysis-wrapper mixin
frontend/src/assets/main.scss            # +@font-face Space Grotesk, +uPlot overrides
frontend/src/assets/fonts/               # NEW — Space Grotesk woff2 (400, 500, 600)

# Frontend — Services & Store
frontend/src/services/wails.ts           # BlockDef +width/height; +updateBlockSize()
frontend/src/stores/workflow.ts          # +updateBlockSize() action

# Frontend — Components
frontend/src/components/canvas/BlockNode.vue          # Inline analysis, NodeResizer, HISTORICAL badge
frontend/src/components/canvas/WorkflowCanvas.vue     # +node-resize-stop handler, +node style
frontend/src/components/panels/BlockLibraryPanel.vue  # 2-col grid, ✕ emit, panel redesign
frontend/src/components/panels/SessionPanel.vue       # Style token updates

# Frontend — Views
frontend/src/views/MainView.vue          # Floating layout, libraryOpen toggle, new status bar

# Frontend — Dependencies
frontend/package.json                    # +@vue-flow/node-resizer (pinned)
```

**Structure Decision**: Single frontend application with Wails backend. Option 2 (frontend + backend separation) applies. No new directories beyond `src/assets/fonts/`.

---

## Complexity Tracking

> No Constitution violations requiring justification.

---

## Implementation Phases

### Phase A: Backend Extension (unblocks everything)

**Goal**: Add `Width`/`Height` to `BlockDef` and expose `UpdateBlockSize` via Wails.

1. `internal/workflow/types.go` — add `Width`, `Height float64` with `json:"width,omitempty"` / `json:"height,omitempty"` to `BlockDef`.
2. `services/workflow_service.go` — implement `UpdateBlockSize(blockId string, width, height float64) error` following the exact same pattern as `UpdateBlockPosition`. Validate `width > 0 && height > 0`.
3. Regenerate Wails bindings (`wails3 generate bindings` or equivalent) so the new method is available to the frontend.

**Tests**: Unit test `UpdateBlockSize` — verify it updates the in-memory struct, returns error on unknown blockId, and returns error on invalid dimensions.

---

### Phase B: Design Tokens & Font (unblocks all visual changes)

**Goal**: Replace the SCSS variable palette and bundle the font. This is the foundation that all component styling builds on.

1. Download Space Grotesk woff2 files (weights 400, 500, 600) from the Google Fonts open-source repo (OFL license). Place in `frontend/src/assets/fonts/`.
2. Replace `_variables.scss` with the Nocturnal Architect palette (see `data-model.md` token table). Preserve all variable names; only values change.
3. Add `$glass-bg`, `$glass-blur`, `$glass-shadow`, `$font-display`, `$font-body`, `$ghost-border` new tokens.
4. Add `@font-face` declarations for Space Grotesk (400, 500, 600) in `main.scss`.
5. Add uPlot dark theme CSS overrides in `main.scss` (`.uplot`, `.u-legend`, `.u-title`).
6. Update `_mixins.scss` `analysis-block` mixin to use new tokens; add `inline-analysis-wrapper` mixin.

**Tests**: Visual regression is manual. Automated: verify no SCSS compile errors (`npm run build:dev`).

---

### Phase C: Frontend Service & Store Layer

**Goal**: Wire `updateBlockSize` through the TypeScript service and Pinia store.

1. `wails.ts` — add `width?: number` / `height?: number` to `BlockDef` interface. Add `updateBlockSize(blockId, width, height)` function.
2. `workflow.ts` store — add `updateBlockSize(id: string, w: number, h: number)` action that calls `wails.updateBlockSize()` and updates the local `blocks` array (same pattern as `updateBlockPosition`).

**Tests**: Unit test the `updateBlockSize` store action — verify it calls the wails function with correct args and updates the store state.

---

### Phase D: Canvas-Inline Analysis & Node Resizer

**Goal**: Render analysis visualizations inside BlockNode; add resize support.

1. `package.json` — add `@vue-flow/node-resizer` (pin to latest 1.x stable version).
2. `BlockNode.vue`:
   - Import all 5 analysis components and build `analysisComponentMap`.
   - Import `NodeResizer` from `@vue-flow/node-resizer`.
   - Add `analysisComponent` computed property (maps `data.type` → component for analysis blocks; `null` otherwise).
   - In template: add `<NodeResizer>` (rendered only when `data.category === 'analysis'`, using per-type min-width/min-height from the table in `data-model.md`).
   - In template: add the inline `<component :is="analysisComponent">` mount below the block header, with `class="inline-analysis"`.
   - Remove the `"double-click to view"` hint text (`analysis-hint` element).
   - The existing `onDblClick` for analysis blocks already calls `enterFullscreen` — no change needed.
3. `WorkflowCanvas.vue`:
   - Add `@node-resize-stop="onNodeResizeStop"` to `<VueFlow>`.
   - Implement `onNodeResizeStop`: parse `node.style.width`/`height`, call `workflowStore.updateBlockSize()`.
   - In the `nodes` computed array, add `style: b.width ? { width: \`${b.width}px\`, height: \`${b.height}px\` } : undefined` per node.

**Tests**:
- `BlockNode.test.ts` — verify that an analysis block renders the inline component; non-analysis blocks do not.
- `WorkflowCanvas.test.ts` — verify `onNodeResizeStop` calls `updateBlockSize` with parsed pixel values.

---

### Phase E: MainView Layout Redesign

**Goal**: Restructure `MainView.vue` from docked layout to floating-panels-over-canvas.

1. Replace the flex-column layout with `position: relative; width: 100vw; height: 100vh; overflow: hidden`.
2. Canvas (`WorkflowCanvas`) fills the full viewport: `position: absolute; inset: 0`.
3. Header: `position: fixed; top: 12px; left: 12px; right: 12px; height: 56px` — glassmorphic, flex row with app title, Open/Save buttons (left), pipeline controls + state tag (right). Apply `$glass-bg`, `$glass-blur`, `$glass-shadow`, border-radius `4px`.
4. Slim toolbar: `position: fixed; top: 80px; bottom: 48px; left: 12px; width: 44px` — glassmorphic, flex column of icon buttons. First icon is the blocks-palette toggle; bottom icons are help/settings.
5. Block library panel: `position: fixed; top: 80px; bottom: 48px; left: 68px; width: 256px` — glassmorphic, `v-if="libraryOpen"`. Panel header includes ✕ button that emits `close` → sets `libraryOpen = false`. Slim toolbar blocks icon sets `libraryOpen = true`.
6. Sessions panel: `position: fixed; top: 80px; bottom: 48px; right: 12px; width: 256px` — glassmorphic.
7. Status bar: `position: fixed; bottom: 8px; left: 12px; right: 12px; height: 28px` — pill-shaped (`border-radius: 14px`), glassmorphic. Content: `State: {flowState}` | connection info (from `pipelineStore`) | `CPU: —` | `RAM: —` | version string. All in `$font-body`, `$font-size-xs`.
8. Remove all `1px solid` border-bottom/left/right from layout regions.
9. Preserve all existing pipeline control logic, keyboard shortcuts, error banners, and recovery dialog — only the layout and styling changes.
10. Update `startErrorBanner` and `error-banner` styling to use new token colors.
11. Update `recovery-dialog` styling to use new token colors.

**Tests**:
- `MainView.test.ts` — verify `libraryOpen` starts `true`; verify clicking ✕ emitted from BlockLibraryPanel sets it to `false`; verify toolbar icon click sets it to `true`.
- All existing `MainView.test.ts` pipeline tests must continue to pass.

---

### Phase F: Block Library Panel Redesign

**Goal**: Update `BlockLibraryPanel.vue` to 2-column card grid matching the design reference.

1. Add `defineEmits<{ (e: 'close'): void }>()`.
2. Add a panel header with "Blocks Palette" title and ✕ button that emits `'close'`.
3. Replace the current list layout with a `display: grid; grid-template-columns: 1fr 1fr; gap: 8px` per category section.
4. Each card: `border: 1px solid {categoryColor}50; border-radius: 4px; padding: 8px; height: 96px; display: flex; flex-direction: column` with a colored 32×32 icon badge (square, category color background), block name in `$font-display $font-size-md`, description in `$text-muted $font-size-xs`.
5. Category label: `$font-size-xs uppercase tracking-wider $text-secondary` with matching left color accent.
6. Apply `$glass-bg`, `$glass-blur` to the panel root.

**Tests**: `BlockLibraryPanel.test.ts` — verify card grid renders, ✕ click emits `'close'`, drag data is set correctly on `dragstart`.

---

### Phase G: Session Panel & Node Styling Cleanup

**Goal**: Update remaining components to Nocturnal Architect tokens.

1. `SessionPanel.vue` — update hardcoded color values to use new SCSS token variables (all structural logic unchanged).
2. `BlockNode.vue` styles — update `$bg-card` background, remove `box-shadow` glow on selected state (replace with `surface-bright` background shift `$bg-block`), ensure the 2px outline uses new `categoryColor` values aligned to the new palette.
3. `WorkflowCanvas.vue` — update the Vue Flow `Background` component pattern color to `#1a1e2b` (grid line color from design reference). Vue Flow default theme CSS overrides for handle colors, edge colors to match new palette.

**Tests**: No new tests; existing tests must pass (styling changes only for session panel and node).

---

## Risk Notes

- **Vue Flow NodeResizer event API**: The `node-resize-stop` event name and `node.style.width` format should be verified against the installed `@vue-flow/node-resizer` version. If the API differs, the `onNodeResizeStop` handler in `WorkflowCanvas.vue` needs adjustment.
- **uPlot canvas rendering**: uPlot renders to a `<canvas>` element; its internal background is white by default. Setting `.uplot { background: transparent }` exposes the node's dark background, which achieves the dark theme. Test visually to confirm no white flash on first render.
- **Wails bindings regeneration**: Adding `UpdateBlockSize` to `WorkflowService` requires regenerating Wails bindings. This step must happen before frontend Phase C work; document in task order.
