# Feature Specification: Nocturnal Redesign & Canvas-Inline Analysis

**Feature Branch**: `003-nocturnal-redesign-canvas-analysis`
**Created**: 2026-04-05
**Status**: Draft
**Input**: Apply a new design aligned with design-example/. Make analysis blocks display their content directly on the canvas without opening a full view. Full view is for the detailed view.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Analysis Data Visible on Canvas (Priority: P1)

As a pipeline operator, when I drop an analysis block (Line Chart, Bar Chart, FFT Spectrum, Value Display, Data Table) onto the canvas, I want to see the live data visualization rendered directly inside the node — without having to open any overlay or fullscreen panel — so I can monitor multiple signals at a glance while designing the workflow.

**Why this priority**: The current workflow requires a double-click to view any analysis output. This adds friction to the core use case: monitoring live or historical data. Inline rendering eliminates that friction and is the most visible behavioral change requested.

**Independent Test**: Add a Line Chart block to the canvas, connect a data source, start the pipeline. The chart updates live inside the node on the canvas with no user interaction beyond connection setup.

**Acceptance Scenarios**:

1. **Given** a Line Chart block exists on the canvas and the pipeline is running, **When** the pipeline emits data, **Then** the chart renders visually inside the node frame without requiring a double-click.
2. **Given** a Value Display block exists on the canvas, **When** new values arrive, **Then** the current value and optional sparkline are visible directly in the node body.
3. **Given** a Data Table block on the canvas, **When** data arrives, **Then** a compact scrollable table is visible inside the node.
4. **Given** any analysis block on the canvas before data arrives, **When** viewed, **Then** a muted "Waiting for data…" placeholder is shown inside the node frame.
5. **Given** multiple analysis blocks on the canvas, **When** all are receiving data simultaneously, **Then** all visualizations update without layout breakage.

---

### User Story 2 – Detailed View via Double-Click (Priority: P2)

As a pipeline operator, when I want a larger, more precise view of a single analysis block, I can double-click the node to open a full-screen detailed overlay. Pressing Escape or clicking the close button returns me to the canvas.

**Why this priority**: The inline canvas view is compact by necessity. The detailed view gives operators a full-resolution picture when precision matters. Both modes must coexist.

**Independent Test**: Double-click a Line Chart node. A full-screen overlay appears showing only that chart at maximum size. Press Escape; the overlay closes and the inline canvas chart continues updating.

**Acceptance Scenarios**:

1. **Given** an analysis block on the canvas, **When** the user double-clicks it, **Then** a full-screen overlay opens showing the block's visualization at full size.
2. **Given** the full-screen overlay is open, **When** the user presses Escape or clicks the close button, **Then** the overlay closes and the canvas reappears with the inline visualization still running.
3. **Given** the full-screen overlay is open and the pipeline is running, **When** data arrives, **Then** the full-screen visualization continues to update live.
4. **Given** the full-screen overlay is open, **When** the user clicks the backdrop outside the content panel, **Then** the overlay closes.

---

### User Story 3 – Nocturnal Architect Visual Design (Priority: P3)

As a user of ByteFlow Studio, I want the interface to feel like a precision engineering environment: deep navy surfaces, floating glassmorphic panels, no harsh divider lines, and a clear typographic hierarchy — matching the design reference provided in `design-example/`.

**Why this priority**: This is a visual layer that significantly improves perceived quality and supports long focused work sessions, but does not affect data correctness or core interaction.

**Independent Test**: Open the application. The background is deep navy. The header, sidebars, and status bar float as glassmorphic overlays. No hard `1px solid` lines divide major layout sections. Block nodes have a subdued tonal outline, not a bright glow.

**Acceptance Scenarios**:

1. **Given** the application is open, **When** viewed, **Then** the canvas background is deep navy matching the design reference, with a subtle grid/dot pattern.
2. **Given** the application is open, **When** viewed, **Then** the top header floats above the canvas with glassmorphic treatment (translucent + blur + rounded corners).
3. **Given** the block library panel is open, **When** viewed, **Then** it appears as a floating glassmorphic panel, not docked with a hard border.
4. **Given** the sessions panel, **When** viewed, **Then** it is a floating glassmorphic panel on the right side.
5. **Given** the status bar, **When** viewed, **Then** it is a floating pill-shaped strip at the bottom with state, connection, CPU, RAM, and version info in small label text.
6. **Given** block nodes on the canvas, **When** viewed, **Then** nodes use a dark surface background with a subdued 2px outline in the category's functional color — no outer glow.
7. **Given** the block library panel, **When** viewed, **Then** blocks are presented in a 2-column card grid per category with a colored icon/badge, name, and short description.
8. **Given** the pipeline control buttons, **When** viewed, **Then** Start uses a green outline, Pause a yellow outline, Resume a filled blue, and Stop a red-tinted style — matching the design reference.

---

### Edge Cases

- What happens when an analysis node is at minimum size (just dropped)? The visualization degrades gracefully: show a minimal placeholder or the compact version rather than overflowing or crashing.
- What happens when no data has arrived for an analysis block? A "Waiting for data…" muted placeholder is shown inside the node.
- What happens when an analysis block receives data while the detailed view is closed? The canvas-inline visualization continues to update independently.
- What happens when the window is resized? Floating panels reposition correctly; analysis nodes do not overflow the canvas viewport.
- What happens if a user resizes an analysis node to be very large? The embedded visualization should scale up to fill the available space.

---

## Requirements *(mandatory)*

### Functional Requirements

**Design System — Visual**

- **FR-001**: The application background canvas MUST use the deep navy color defined in the design reference (approximately `#060e20`), replacing the current blue-grey background.
- **FR-002**: The top header bar, block library panel, and sessions panel MUST use glassmorphic styling: semi-transparent dark fill, backdrop blur of 12–20px, rounded corners, and a subtle shadow — matching the design reference floating panel treatment.
- **FR-003**: Major layout region boundaries (header ↔ canvas, canvas ↔ sidebars, canvas ↔ status bar) MUST NOT use `1px solid` border lines; boundaries MUST be defined through background color shifts between tonal surface layers.
- **FR-004**: Space Grotesk MUST be loaded and applied as the primary font for labels, block names, headings, and the app title. A high-legibility sans-serif (Inter or system fallback) MUST be used for data-dense areas (config forms, session tables, status bar content).
- **FR-005**: The status bar MUST be rendered as a floating, pill-shaped strip at the bottom of the screen containing: pipeline state, serial connection info, CPU usage, RAM usage, and version string, using small label-size text.
- **FR-006**: Block nodes on the canvas MUST use a dark surface-container background with a 2px outline in the category's functional color (blue for input, purple for processing, green for analysis). Outer glow or neon shadow effects MUST be removed.
- **FR-007**: The block library panel MUST display blocks in a 2-column card grid per category, with each card showing a colored icon/badge square, block name, and a short description — matching the design reference layout.
- **FR-008**: Pipeline control buttons (Start, Pause, Resume, Stop) MUST be rendered in the floating header. Start MUST use a green outline style; Pause a yellow outline style; Resume a filled blue style; Stop a red-tinted outline style.
- **FR-009**: The slim icon toolbar (currently the left icon column) MUST be a narrow floating panel to the left of the block library, showing navigation/tool icons vertically. The toolbar icon corresponding to the block library MUST toggle the panel open when it is closed.
- **FR-009a**: The block library panel MUST be visible by default when the application opens. The user MAY close it via an ✕ button on the panel header. When closed, clicking the corresponding toolbar icon MUST reopen it.

**Canvas-Inline Analysis**

- **FR-010**: Analysis block nodes (Line Chart, Bar Chart, FFT Spectrum, Value Display, Data Table) MUST render their visualization component directly inside the node body on the canvas, visible without any user interaction.
- **FR-011**: Each analysis block node type MUST have a defined minimum canvas size sufficient to render its embedded visualization in a meaningful way (e.g., Line Chart: ~240×160px minimum; Value Display: ~140×100px minimum).
- **FR-012**: When no data has been received by an analysis block, the node MUST display a muted placeholder label (e.g., "Waiting for data…") inside the visualization area.
- **FR-012a**: When a session replay (historical mode) is active, a HISTORICAL badge MUST appear on the inline canvas node in the same position as in the full-screen detailed view. The badge MUST be visible in both the inline and full-screen contexts.
- **FR-013**: The embedded visualization inside an analysis node MUST receive and display live pipeline data at the same update rate as the current fullscreen implementation.
- **FR-014**: Analysis block nodes MUST support resizing on the canvas so users can increase or decrease the size of the inline visualization by dragging the node's resize handle. The node's width and height MUST be persisted to the workflow file and restored on next open, alongside the existing position fields.
- **FR-015**: Double-clicking an analysis block node MUST open the existing full-screen detailed view overlay (current behavior retained), functioning as the "detailed view" mode.
- **FR-016**: The full-screen detailed overlay MUST only be triggered by a double-click (or an explicit expand button on the node), not by a single click.
- **FR-017**: When the full-screen detailed overlay is open, the inline canvas visualization of the same block MUST continue to run and update simultaneously — both views share the live data stream without pausing or unmounting the inline view.

### Key Entities

- **Analysis Block Node**: A canvas node of category `analysis` that embeds a live visualization component (chart, value display, table) directly within its node body on the workflow canvas. Has persisted position (x, y) and size (width, height) in the workflow file.
- **Design Token**: A named color, spacing, or typography value from the design reference palette, stored in the SCSS variables file, applied consistently across all components.
- **Floating Panel**: A UI surface (header, sidebar, status bar) that overlays the canvas using glassmorphic styling rather than occupying a fixed docked slot in the layout grid.
- **Detailed View Overlay**: The existing full-screen overlay mode for analysis blocks, retained as the high-fidelity view accessible via double-click.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All 5 analysis block types display their data inline on the canvas node with no interaction required beyond connecting a data source and starting the pipeline.
- **SC-002**: Double-clicking any analysis node opens the detailed full-screen overlay, maintaining behavioral parity with the current implementation.
- **SC-003**: The visual appearance matches the design reference: floating glassmorphic header/sidebars, deep navy canvas background, no hard divider lines between layout regions, subdued node outlines.
- **SC-004**: All existing functionality continues to work correctly after the redesign — block drag-and-drop, connection drawing, config dialogs for input/processing blocks, session management, pipeline controls (Start/Pause/Resume/Stop), undo/redo, and keyboard shortcuts.
- **SC-005**: With 5 or more analysis blocks simultaneously receiving and rendering data on the canvas, the interface remains visually smooth with no layout overflow or breakage.
- **SC-006**: All existing automated tests pass with no regressions introduced by the visual or behavioral changes.

---

## Clarifications

### Session 2026-04-05

- Q: When the full-screen detailed view is open, what should the inline canvas visualization do? → A: Both run simultaneously — inline view continues updating while the overlay is open.
- Q: How should Space Grotesk be loaded given the desktop/offline context? → A: Bundle the font files locally in the frontend assets (no CDN dependency).
- Q: Should the HISTORICAL badge appear on the inline canvas node during session replay? → A: Show on both the inline canvas node and the full-screen overlay.
- Q: Should the block library panel be always visible or toggleable? → A: Visible by default, closeable via its ✕ button, reopened via the toolbar icon.
- Q: Should analysis node resize dimensions be persisted to the workflow file? → A: Yes — persist width and height alongside position (positionX, positionY).

---

## Assumptions

- Space Grotesk font files MUST be bundled locally within the frontend assets (e.g., `src/assets/fonts/`) and loaded via `@font-face` in the global SCSS. No CDN or internet connection must be required for font rendering.
- The existing SCSS design token system (`_variables.scss`) will be updated with the new palette values. No new CSS methodology (Tailwind, CSS-in-JS) will be introduced.
- The existing `useFullscreen` composable and fullscreen overlay in `MainView.vue` remain the mechanism for detailed view (FR-015); this spec does not change how the overlay works, only what triggers it.
- Vue Flow's built-in node resize capability or a CSS `resize` wrapper will be used for FR-014; a fully custom drag-resize system is out of scope.
- Analysis block components (LineChartBlock.vue, etc.) will be mounted directly inside the BlockNode template for the inline view, reusing their existing data subscription logic.
- The "slim icon toolbar" from the design reference maps to the existing left icon column in `MainView.vue` and will be restyled as a narrow floating panel.
- CPU and RAM stats shown in the status bar will use placeholder/static data initially if the backend does not yet expose these metrics; the visual structure is required, not the live data source.
