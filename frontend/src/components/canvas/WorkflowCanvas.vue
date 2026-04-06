<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { VueFlow, useVueFlow, type NodeMouseEvent, type Connection, type NodeDragEvent, type Node, type Edge } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { useToast } from 'primevue/usetoast'
import { useWorkflowStore } from '../../stores/workflow'
import { usePipelineStore } from '../../stores/pipeline'
import BlockNode from './BlockNode.vue'
import type { BlockTypeDescriptor } from '../../services/wails'
import { isCycleError } from '../../types/pipeline-errors'

const workflowStore = useWorkflowStore()
const pipelineStore = usePipelineStore()
const toast = useToast()

const { onNodesChange, onEdgesChange, onConnect, findNode, getSelectedNodes, getSelectedEdges } = useVueFlow()

// Default dimensions for analysis blocks that haven't been resized yet
const analysisDefaultDimensions: Record<string, { width: number; height: number }> = {
  'line-chart':    { width: 320, height: 200 },
  'value-display': { width: 180, height: 130 },
  'bar-chart':     { width: 320, height: 200 },
  'fft-spectrum':  { width: 360, height: 240 },
  'data-table':    { width: 280, height: 180 },
}

const nodes = computed(() =>
  workflowStore.blocks.map(b => {
    if (b.category === 'analysis') {
      const defaults = analysisDefaultDimensions[b.type] ?? { width: 320, height: 200 }
      const w = b.width || defaults.width
      const h = b.height || defaults.height
      return {
        id: b.id,
        type: 'block',
        position: { x: b.positionX, y: b.positionY },
        data: b,
        style: { width: `${w}px`, height: `${h}px` },
      }
    }
    return {
      id: b.id,
      type: 'block',
      position: { x: b.positionX, y: b.positionY },
      data: b,
    }
  })
)

const edges = computed(() =>
  workflowStore.connections.map(c => ({
    id: c.id,
    source: c.fromBlockId,
    target: c.toBlockId,
    sourceHandle: c.fromPortId,
    targetHandle: c.toPortId,
    type: 'default',
  }))
)

function isValidConnection(conn: Connection): boolean {
  // Prevent self-connections
  if (conn.source === conn.target) return false
  // Only allow source handle ("out") → input handle ("in"); blocks output→output
  return conn.sourceHandle === 'out' && conn.targetHandle === 'in'
}

async function onConnectEdge(conn: Connection) {
  try {
    await workflowStore.addConnection(
      conn.source!, conn.sourceHandle!,
      conn.target!, conn.targetHandle!,
    )
  } catch (e: unknown) {
    const msg = String(e)
    if (isCycleError(msg)) {
      toast.add({ severity: 'warn', summary: 'Circular connection is not allowed', life: 3000 })
    } else {
      toast.add({ severity: 'error', summary: 'Connection failed', detail: msg, life: 3000 })
    }
  }
}

async function onNodeDragStop(event: NodeDragEvent) {
  await workflowStore.updateBlockPosition(event.node.id, event.node.position.x, event.node.position.y)
}


async function onDropBlock(event: DragEvent) {
  event.preventDefault()
  const raw = event.dataTransfer?.getData('application/byteflow-block')
  if (!raw) return
  const block: BlockTypeDescriptor = JSON.parse(raw)
  const canvas = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const x = event.clientX - canvas.left
  const y = event.clientY - canvas.top
  await workflowStore.addBlock(block.type, x, y)
}

function onDragOver(event: DragEvent) {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'copy'
  }
}

async function onKeyDown(event: KeyboardEvent) {
  if (event.key === 'Delete' || event.key === 'Backspace') {
    // T108: batch delete all selected nodes and edges
    const selectedNodes = getSelectedNodes.value as Node[]
    const selectedEdges = getSelectedEdges.value as Edge[]
    if (selectedNodes.length > 0 || selectedEdges.length > 0) {
      event.preventDefault()
      for (const edge of selectedEdges) {
        await workflowStore.removeConnection(edge.id)
      }
      for (const node of selectedNodes) {
        await workflowStore.removeBlock(node.id)
      }
    }
  }
  if ((event.ctrlKey || event.metaKey) && event.key === 'z') {
    await workflowStore.undo()
  }
  if ((event.ctrlKey || event.metaKey) && (event.key === 'y' || (event.shiftKey && event.key === 'z'))) {
    await workflowStore.redo()
  }
}

// Error styling: blocks with status="error" get red border class
function nodeClass(nodeId: string) {
  const block = workflowStore.blocks.find(b => b.id === nodeId)
  return block?.status === 'error' ? 'node-error' : ''
}
</script>

<template>
  <div
    class="canvas-wrapper"
    @drop="onDropBlock"
    @dragover="onDragOver"
    @keydown="onKeyDown"
    tabindex="0"
  >
    <VueFlow
      :nodes="nodes"
      :edges="edges"
      :node-types="{ block: BlockNode }"
      fit-view-on-init
      multi-selection-key-code="Shift"
      :is-valid-connection="isValidConnection"
      @connect="onConnectEdge"
      @node-drag-stop="onNodeDragStop"
    >
      <Background />
      <Controls position="top-right" />
      <MiniMap position="bottom-right" />
    </VueFlow>
  </div>
</template>

<style lang="scss">
@use '../../assets/variables' as v;
@import '@vue-flow/core/dist/style.css';
@import '@vue-flow/core/dist/theme-default.css';
@import '@vue-flow/controls/dist/style.css';
@import '@vue-flow/minimap/dist/style.css';
@import '@vue-flow/node-resizer/dist/style.css';

.canvas-wrapper {
  width: 100%;
  height: 100%;
  outline: none;
}

.node-error .vue-flow__node {
  border: 2px solid v.$color-danger !important;
}

// Vue Flow background: override dot/pattern color to stay on-brand
.vue-flow__background {
  background-color: v.$bg-root !important;
}

.vue-flow__background pattern circle,
.vue-flow__background pattern rect {
  fill: #1a1e2b !important;
}

// Edges: subdued tonal color
.vue-flow__edge-path {
  stroke: v.$border-color !important;
  stroke-width: 1.5px;
}
.vue-flow__edge:hover .vue-flow__edge-path,
.vue-flow__edge.selected .vue-flow__edge-path {
  stroke: v.$color-primary !important;
}

// Controls: glassmorphic + reposition below menu bar, left of sessions panel
.vue-flow__controls {
  background: v.$glass-bg !important;
  backdrop-filter: blur(v.$glass-blur) !important;
  border: 1px solid v.$ghost-border !important;
  border-radius: v.$radius-lg !important;
  box-shadow: v.$glass-shadow !important;
  // Keep away from left sidebar; sessions panel is 256px + 12px right = 268px from right
  right: 280px !important;
  left: auto !important;
  top: 80px !important; // below menu bar (12px top + 56px height + 12px gap)
  bottom: auto !important;

  button {
    background: transparent !important;
    border: none !important;
    border-bottom: 1px solid v.$ghost-border !important;
    color: v.$text-secondary !important;

    &:last-child { border-bottom: none !important; }
    &:hover { background: rgba(255,255,255,0.06) !important; color: v.$text-primary !important; }
  }
}

// MiniMap: dark tones + reposition above status bar, left of sessions panel
.vue-flow__minimap {
  background: v.$bg-card !important;
  border: 1px solid v.$ghost-border !important;
  border-radius: v.$radius-md !important;
  right: 280px !important;
  left: auto !important;
  bottom: 52px !important; // above status bar (8px bottom + 28px height + 16px gap)
  top: auto !important;
}
</style>
