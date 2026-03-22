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

const workflowStore = useWorkflowStore()
const pipelineStore = usePipelineStore()
const toast = useToast()

const { onNodesChange, onEdgesChange, onConnect, findNode, getSelectedNodes, getSelectedEdges } = useVueFlow()

const nodes = computed(() =>
  workflowStore.blocks.map(b => ({
    id: b.id,
    type: 'block',
    position: { x: b.positionX, y: b.positionY },
    data: b,
  }))
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

async function onConnectEdge(conn: Connection) {
  try {
    await workflowStore.addConnection(
      conn.source!, conn.sourceHandle!,
      conn.target!, conn.targetHandle!,
    )
  } catch (e: unknown) {
    const msg = String(e)
    if (msg.includes('cycle')) {
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
    workflowStore.undo()
  }
  if ((event.ctrlKey || event.metaKey) && (event.key === 'y' || (event.shiftKey && event.key === 'z'))) {
    workflowStore.redo()
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
      @connect="onConnectEdge"
      @node-drag-stop="onNodeDragStop"
    >
      <Background />
      <Controls />
      <MiniMap />
    </VueFlow>
  </div>
</template>

<style>
@import '@vue-flow/core/dist/style.css';
@import '@vue-flow/core/dist/theme-default.css';
@import '@vue-flow/controls/dist/style.css';
@import '@vue-flow/minimap/dist/style.css';

.canvas-wrapper {
  width: 100%;
  height: 100%;
  outline: none;
}

.node-error .vue-flow__node {
  border: 2px solid #f87171;
}
</style>
