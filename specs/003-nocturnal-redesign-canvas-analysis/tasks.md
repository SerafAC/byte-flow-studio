# Tasks: Nocturnal Redesign & Canvas-Inline Analysis

**Input**: Design documents from `specs/003-nocturnal-redesign-canvas-analysis/`
**Branch**: `003-nocturnal-redesign-canvas-analysis`
**Date**: 2026-04-05

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to ([US1], [US2], [US3])
- **Test-First**: Per Constitution Principle II — test tasks MUST be completed (and failing) before their corresponding implementation tasks

---

## Phase 1: Setup

**Purpose**: Install new dependency and bundle font assets. No user story work begins until this is done.

- [ ] T001 Add `@vue-flow/node-resizer` to `frontend/package.json` dependencies (pin to latest 1.x stable — e.g. `"@vue-flow/node-resizer": "1.4.0"`) and run `npm install` in `frontend/`
- [ ] T002 [P] Download Space Grotesk woff2 files (weights 400, 500, 600) from the Google Fonts open-source repository (OFL license) and place them in `frontend/src/assets/fonts/` as `SpaceGrotesk-Regular.woff2`, `SpaceGrotesk-Medium.woff2`, `SpaceGrotesk-SemiBold.woff2`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Backend data model extension, SCSS token replacement, and font wiring. Every user story depends on these.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T003 [P] Write failing Go unit test for `UpdateBlockSize` in `services/workflow_service_test.go` — test: valid call updates `BlockDef.Width`/`Height` and returns nil; unknown blockId returns error; width ≤ 0 returns error
- [ ] T004 [P] Add `Width float64 \`json:"width,omitempty"\`` and `Height float64 \`json:"height,omitempty"\`` fields to the `BlockDef` struct in `internal/workflow/types.go`
- [ ] T005 Implement `UpdateBlockSize(blockId string, width, height float64) error` method on `WorkflowService` in `services/workflow_service.go`, following the same pattern as `UpdateBlockPosition` (find block by ID, validate width/height > 0, update fields, call saveState) — satisfies T003
- [ ] T006 Regenerate Wails v3 bindings by running `wails3 generate bindings` (or project-equivalent command) from repo root so `UpdateBlockSize` is available to the frontend TypeScript layer
- [ ] T007 [P] Replace all design token values in `frontend/src/assets/_variables.scss` with the Nocturnal Architect palette per the token table in `data-model.md`; add new tokens `$glass-bg`, `$glass-blur`, `$glass-shadow`, `$font-display`, `$font-body`, `$ghost-border`
- [X] T008 [P] Add `@font-face` declarations for Space Grotesk (400/500/600) pointing to `./fonts/SpaceGrotesk-*.woff2` in `frontend/src/assets/main.scss`; add uPlot dark theme CSS overrides (`.uplot`, `.u-legend`, `.u-title`); update `_mixins.scss` `analysis-block` mixin to use updated tokens and add `inline-analysis-wrapper` mixin (flex column, flex: 1, min-height: 0, overflow: hidden)

**Checkpoint**: Backend extension compiled and tested ✓, Wails bindings regenerated ✓, SCSS tokens replaced ✓, font wired ✓ — user story implementation can now begin

---

## Phase 3: User Story 1 – Analysis Data Visible on Canvas (Priority: P1) 🎯 MVP

**Goal**: All 5 analysis block types render their live visualization directly inside the canvas node — no interaction required.

**Independent Test**: Drop a Line Chart block, connect a data source, start the pipeline. A chart renders and updates live inside the node on the canvas without any click. Value Display, Bar Chart, FFT Spectrum, and Data Table behave identically.

### Tests for User Story 1 ⚠️ Write FIRST — must FAIL before implementation begins

- [X] T009 [P] [US1] Write failing test: when `data.category === 'analysis'` and `data.type === 'line-chart'`, `BlockNode` renders the inline analysis component; when category is `'input'` or `'processing'`, it does not — in `frontend/src/components/canvas/BlockNode.test.ts`
- [X] T010 [P] [US1] Write failing test: `updateBlockSize(id, 320, 200)` action in the workflow store calls `wails.updateBlockSize` with the correct arguments and updates the matching block's `width`/`height` in `blocks` array — in `frontend/src/stores/workflow.test.ts` (create file if it does not exist)

### Implementation for User Story 1

- [X] T011 [US1] Add `width?: number` and `height?: number` optional fields to the `BlockDef` interface and add `export async function updateBlockSize(blockId: string, width: number, height: number): Promise<void>` to `frontend/src/services/wails.ts` (calls `svc.UpdateBlockSize`)
- [X] T012 [US1] Add `async updateBlockSize(id: string, w: number, h: number)` action to the workflow Pinia store in `frontend/src/stores/workflow.ts` — calls `wails.updateBlockSize`, then updates the matching block's `width`/`height` in `this.blocks` array — satisfies T010
- [X] T013 [P] [US1] Import all 5 analysis components (`LineChartBlock`, `BarChartBlock`, `FftSpectrumBlock`, `ValueDisplayBlock`, `DataTableBlock`) and build `analysisComponentMap: Record<string, Component>` and `analysisComponent` computed prop in `frontend/src/components/canvas/BlockNode.vue`
- [X] T014 [US1] Add `<NodeResizer>` (from `@vue-flow/node-resizer`, rendered only when `data.category === 'analysis'`) with per-type `min-width`/`min-height` values from `data-model.md`; add `<component :is="analysisComponent" :block-id="data.id" v-bind="data.params ?? {}" class="inline-analysis" />` below the block header; add `v-else` "Waiting for data…" placeholder when no data has arrived; remove the existing `analysis-hint` element — in `frontend/src/components/canvas/BlockNode.vue` — satisfies T009
- [X] T015 [US1] Add `@node-resize-stop="onNodeResizeStop"` to the `<VueFlow>` element and implement `onNodeResizeStop(event)` handler that calls `workflowStore.updateBlockSize(event.node.id, parseFloat(event.node.style.width), parseFloat(event.node.style.height))` in `frontend/src/components/canvas/WorkflowCanvas.vue`; update the `nodes` computed array to include `style: b.width ? { width: \`${b.width}px\`, height: \`${b.height}px\` } : undefined` per node

**Checkpoint**: US1 complete — analysis nodes show live data inline; resize and persistence work; tests green ✓

---

## Phase 4: User Story 2 – Detailed View via Double-Click (Priority: P2)

**Goal**: Double-clicking an analysis node opens the full-screen detailed overlay; the inline canvas view continues running simultaneously.

**Independent Test**: Double-click a Line Chart node — full-screen overlay opens. While open, pipeline data continues to update both the overlay chart and the inline node chart. Press Escape — overlay closes, inline chart still running.

### Tests for User Story 2 ⚠️ Write FIRST — must FAIL before implementation begins

- [X] T016 [P] [US2] Write failing test: double-clicking a `BlockNode` with `data.category === 'analysis'` calls `enterFullscreen(data.id)` AND the inline analysis component remains mounted (not unmounted/destroyed) — in `frontend/src/components/canvas/BlockNode.test.ts`
- [X] T017 [P] [US2] Write failing test: in `MainView`, when `fullscreenBlockId` is set to a valid block ID, the fullscreen overlay renders while the `WorkflowCanvas` (and its nodes) remains in the DOM — in `frontend/src/views/MainView.test.ts`

### Implementation for User Story 2

- [X] T018 [US2] Verify `onDblClick` in `frontend/src/components/canvas/BlockNode.vue` still calls `enterFullscreen(data.id)` for analysis blocks after the Phase 3 changes; if the inline component `v-if` accidentally guards it, fix so the `@dblclick` handler fires regardless of inline content — satisfies T016
- [X] T019 [US2] Confirm `MainView.vue` fullscreen overlay uses `v-if` (not `v-show`) on the overlay itself but does NOT hide or `v-if`-out the `<WorkflowCanvas>` beneath it; the canvas and all its nodes (including inline analysis components) remain mounted while the overlay is open — satisfies T017; adjust if needed

**Checkpoint**: US2 complete — full-screen detail view works; both inline and overlay receive live data simultaneously; tests green ✓

---

## Phase 5: User Story 3 – Nocturnal Architect Visual Design (Priority: P3)

**Goal**: The full application matches the Nocturnal Architect design reference — deep navy canvas, glassmorphic floating panels, Space Grotesk typography, no hard border dividers, redesigned status bar, toggleable block library.

**Independent Test**: Open the application. Background is deep navy. Header, sidebars, and status bar float as glassmorphic panels. No `1px solid` lines separate layout regions. Block nodes use subdued tonal outlines with no glow. Block library is visible by default, closeable, reopenable via toolbar icon.

### Tests for User Story 3 ⚠️ Write FIRST — must FAIL before implementation begins

- [X] T020 [P] [US3] Write failing test: `MainView` mounts with `libraryOpen === true`; when `BlockLibraryPanel` emits `'close'`, `libraryOpen` becomes `false`; when the toolbar blocks-icon is clicked, `libraryOpen` becomes `true` — in `frontend/src/views/MainView.test.ts`
- [X] T021 [P] [US3] Write failing test: `BlockLibraryPanel` emits `'close'` when the ✕ button is clicked — in `frontend/src/components/panels/BlockLibraryPanel.test.ts`

### Implementation for User Story 3

- [X] T022 [US3] Restructure `frontend/src/views/MainView.vue` layout: replace flex-column docked layout with full-viewport canvas + `position: fixed` floating panels; canvas (`<WorkflowCanvas>`) fills `position: absolute; inset: 0`; header fixed at `top: 12px, left: 12px, right: 12px, height: 56px` with `$glass-bg/$glass-blur/$glass-shadow`; slim toolbar fixed at `top: 80px, bottom: 48px, left: 12px, width: 44px` with toolbar icon that toggles `libraryOpen`; block library panel `v-if="libraryOpen"` at `top: 80px, bottom: 48px, left: 68px, width: 256px`; sessions panel at `top: 80px, bottom: 48px, right: 12px, width: 256px`; pill-shaped status bar at `bottom: 8px, left: 12px, right: 12px, height: 28px, border-radius: 14px`; add `const libraryOpen = ref(true)` — satisfies T020
- [X] T023 [US3] Update pipeline control buttons in `MainView.vue` header: Start → green outline (`border: 1px solid $color-success; color: $color-success`); Pause → yellow outline; Resume → filled `$color-primary` background; Stop → red-tinted outline (`border: 1px solid $color-danger; background: rgba($color-danger, 0.15)`)
- [X] T024 [US3] Update status bar content in `MainView.vue`: show `State: {flowState}` using `$color-success`/`$color-warning`/`$color-danger` per state; show connection info from `pipelineStore.sessionId` or a serial port label if available; show `CPU: —  RAM: —` as static placeholders; show hardcoded version string `v0.1.0-dev` (or read from a build constant if one exists)
- [X] T025 [US3] Redesign `frontend/src/components/panels/BlockLibraryPanel.vue`: add `defineEmits<{ (e: 'close'): void }>()` and ✕ button in panel header that calls `emit('close')`; replace list layout with `display: grid; grid-template-columns: 1fr 1fr; gap: 8px` per category; each card: 96px height, flex column, 32×32 colored badge square (category color background, white icon placeholder or first-letter monogram), block name in `$font-display $font-size-md`, description in `$text-muted $font-size-xs`; apply `$glass-bg/$glass-blur` to panel root — satisfies T021
- [X] T026 [P] [US3] Update `frontend/src/components/panels/SessionPanel.vue` styles: replace all hardcoded hex values with SCSS token variables; apply `$glass-bg/$glass-blur/$glass-shadow` to panel root; use `$font-body` for session data text; preserve all existing markup and behavior
- [X] T027 [P] [US3] Update `frontend/src/components/canvas/BlockNode.vue` styles: set `background` to `$bg-card`; replace `box-shadow: 0 0 0 2px #f59e0b` selected state with `background: $bg-block` background-shift (no shadow glow); update `output-handle` background to `$color-primary`; update `input-handle` background to `$text-muted`; apply `$font-display` to `.block-label`; apply `$font-body` to body text
- [X] T028 [P] [US3] Update `frontend/src/components/canvas/WorkflowCanvas.vue`: override Vue Flow `Background` dot/grid pattern color to `#1a1e2b`; override `.vue-flow__edge-path` stroke to use gradient matching source/target category colors (or solid `$color-primary` as a fallback); override `.vue-flow__controls` button styles to match glassmorphic panel tokens

**Checkpoint**: US3 complete — full visual redesign matches reference; library toggle works; tests green ✓

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verify zero regressions, fix any SCSS compile issues, and run the full test suite.

- [X] T029 [P] Update error banner and recovery dialog styles in `frontend/src/views/MainView.vue` to use new token colors (`$color-danger`, `$color-warning`, `$bg-card`) — currently uses hardcoded hex values
- [X] T030 [P] Run `npm run build:dev` from `frontend/` and fix any SCSS compile errors, TypeScript type errors, or missing import errors introduced by the redesign
- [X] T031 Run `npm test` from `frontend/` and ensure all tests pass with no regressions (target: all pre-existing tests green + new tests from T009, T010, T016, T017, T020, T021 green)
- [ ] T032 Visual review: run the Wails dev build (`wails3 dev` from repo root) and compare the running application side-by-side against `design-example/screen.png` for all three user stories; note and fix any material visual discrepancies

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately; T001 and T002 can run in parallel
- **Phase 2 (Foundational)**: Depends on Phase 1 completion
  - T003/T004 can run in parallel (different files)
  - T005 depends on T003 (test must exist and fail first) and T004 (struct field must exist)
  - T006 depends on T005 (bindings generated after method exists)
  - T007/T008 can run in parallel with T003–T006 (different files — SCSS/assets vs Go)
- **Phase 3 (US1)**: Depends on Phase 2 complete (T006 done for bindings, T007/T008 done for tokens)
  - T009/T010 can run in parallel (tests for different files)
  - T011 depends on T006 (bindings available)
  - T012 depends on T011 (calls wails function)
  - T013 can run in parallel with T011/T012 (just imports, no runtime dependency)
  - T014 depends on T013 (uses analysisComponentMap) and T009 (test must fail first)
  - T015 depends on T012 (calls workflowStore.updateBlockSize) and T010 (test must fail first)
- **Phase 4 (US2)**: Depends on Phase 3 complete (inline rendering must exist for simultaneous test)
  - T016/T017 can run in parallel (tests for different files)
  - T018 depends on T016 (test must fail first)
  - T019 depends on T017 (test must fail first)
- **Phase 5 (US3)**: Can start in parallel with Phase 3 after Phase 2 completes (different files)
  - T020/T021 can run in parallel (tests for different files)
  - T022 depends on T020
  - T023/T024 depends on T022 (pipeline buttons and status bar are part of the same MainView restructure)
  - T025 depends on T021
  - T026/T027/T028 can run in parallel with T022–T025 (different component files)
- **Phase 6 (Polish)**: Depends on all user story phases complete

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2 — no dependency on US2 or US3
- **US2 (P2)**: Depends on US1 Phase 3 complete (inline rendering must exist to test simultaneous rendering)
- **US3 (P3)**: Can start after Phase 2 — independent of US1/US2 (different files); though node styling in T027 touches the same file as US1's T014/T015, sequence T014→T015 before T027 to avoid conflicts

### Parallel Opportunities

Within Phase 2 (can run simultaneously):
- T003 + T004 (Go test + Go struct field)
- T007 + T008 (SCSS tokens + font/mixin setup) — these can run in parallel with T003–T006

Within Phase 3 (can run simultaneously):
- T009 + T010 (two independent test files)
- T011 + T013 (wails.ts + BlockNode imports — different aspects, no runtime conflict)

Across phases (can run simultaneously after Phase 2):
- Phase 3 (US1) + Phase 5 (US3) tasks T020/T021/T025/T026/T028 — different component files

---

## Parallel Example: User Story 1

```bash
# Kick off tests first (both in parallel):
T009: Write failing test in BlockNode.test.ts
T010: Write failing test in workflow.test.ts

# Then implementation in parallel where possible:
T011: wails.ts updateBlockSize wrapper
T013: BlockNode analysis imports (parallel with T011)

# Sequential after T013:
T014: BlockNode inline rendering template (depends on T013)

# Sequential after T011/T012:
T015: WorkflowCanvas resize handler (depends on T012)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T002)
2. Complete Phase 2: Foundational (T003–T008) — **BLOCKS everything**
3. Complete Phase 3: US1 (T009–T015)
4. **STOP and VALIDATE**: Drop a Line Chart block, connect data, start pipeline — chart visible inline ✓
5. Ship as preview / demo

### Incremental Delivery

1. Setup + Foundational → foundation ready
2. US1 complete → inline analysis works → demo
3. US2 complete → fullscreen detail view verified alongside inline → demo
4. US3 complete → full visual redesign → release candidate
5. Polish → all tests green, visual review done → release

### Single-Developer Sequence

1. T001 → T002 (setup)
2. T003 → T004 → T005 → T006 (backend, sequential)
3. T007 + T008 (SCSS, parallel if desired)
4. T009 + T010 (US1 tests, write together)
5. T011 → T012 → T013 → T014 → T015 (US1 implementation, mostly sequential)
6. T016 + T017 (US2 tests) → T018 → T019 (US2 implementation)
7. T020 + T021 (US3 tests) → T022 → T023 → T024 → T025 → T026 → T027 → T028 (US3 implementation)
8. T029 → T030 → T031 → T032 (polish)

---

## Notes

- **Test-First is mandatory** (Constitution Principle II): All test tasks (T003, T009, T010, T016, T017, T020, T021) MUST be completed and confirmed failing before their corresponding implementation tasks run
- **Wails bindings** (T006): Must regenerate bindings before any frontend code that calls `UpdateBlockSize` is written; failing to do so will cause TypeScript import errors
- **Font files** (T002): Must be present before running any dev build that references `@font-face`; otherwise Space Grotesk silently falls back to system font
- **SCSS token rename risk**: T007 changes all token values; all existing components using the old values will immediately pick up the new palette. Visual regressions in components not touched in US3 are expected and are fixed in T032
- `[P]` tasks can be delegated to parallel Claude Code sessions or worked on simultaneously by a developer
- Commit after each completed task or at each `**Checkpoint**`
