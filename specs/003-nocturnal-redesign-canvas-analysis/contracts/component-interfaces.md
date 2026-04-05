# Component Interface Contracts: Nocturnal Redesign & Canvas-Inline Analysis

**Type**: Vue 3 component props/emits contracts
**Date**: 2026-04-05

---

## BlockNode.vue — Updated Interface

The BlockNode component gains the ability to render analysis visualizations inline and to emit a size-change event.

**Props** (unchanged from current):
```typescript
defineProps<{ data: BlockDef }>()
```

**New internal behavior**:
- When `data.category === 'analysis'`: renders the matched analysis component inline inside the node body.
- The `NodeResizer` component is rendered only for analysis-category nodes.
- On double-click: still calls `enterFullscreen(data.id)` (behavior unchanged).

**No new emits** — size changes are dispatched directly to the Pinia store via `workflowStore.updateBlockSize()`.

---

## BlockLibraryPanel.vue — Updated Interface

**Props**:
```typescript
defineProps<{}>()  // unchanged — no props
```

**New emits**:
```typescript
defineEmits<{
  (e: 'close'): void  // NEW — emitted when user clicks the ✕ button
}>()
```

**Usage in MainView.vue**:
```vue
<BlockLibraryPanel v-if="libraryOpen" @close="libraryOpen = false" />
```

---

## Analysis Block Components — Unchanged Interface

All analysis components (`LineChartBlock`, `BarChartBlock`, `FftSpectrumBlock`, `ValueDisplayBlock`, `DataTableBlock`) retain their existing props/emits contracts. They are mounted directly inside `BlockNode.vue` using the same `blockId` prop.

**Shared props pattern** (example — LineChartBlock):
```typescript
defineProps<{
  blockId: string
  title?: string
  xLabel?: string
  yLabel?: string
  unit?: string
  decimals?: number
  bufferSamples?: number
}>()
```

When mounted inline in `BlockNode.vue`, props are spread from `data.params`:
```vue
<component
  :is="analysisComponent"
  :block-id="data.id"
  v-bind="data.params ?? {}"
  class="inline-analysis"
/>
```

The `fullscreen:enter` and `fullscreen:exit` emits from analysis components are **not used** in the inline context (double-click on the node handles fullscreen entry directly). The context menu "Full Screen" item in `useAnalysisBlock` remains functional via right-click.

---

## WorkflowCanvas.vue — New Event Handler

New event added to the `<VueFlow>` element:

```typescript
// Handler signature
function onNodeResizeStop(event: NodeResizeStopEvent): void

// Calls:
workflowStore.updateBlockSize(
  event.node.id,
  parseFloat(event.node.style.width as string),
  parseFloat(event.node.style.height as string)
)
```

Template addition:
```vue
<VueFlow
  ...existing props...
  @node-resize-stop="onNodeResizeStop"
>
```
