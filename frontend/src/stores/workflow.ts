import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { BlockDef, ConnectionDef, Workflow } from '../services/wails'
import * as wails from '../services/wails'

export const useWorkflowStore = defineStore('workflow', () => {
  const blocks = ref<BlockDef[]>([])
  const connections = ref<ConnectionDef[]>([])
  const workflowId = ref<string>('')
  const workflowName = ref<string>('Untitled Workflow')

  // Undo/redo history
  type HistoryEntry = { blocks: BlockDef[]; connections: ConnectionDef[] }
  const history = ref<HistoryEntry[]>([])
  const historyIndex = ref(-1)

  function _snapshot() {
    return {
      blocks: JSON.parse(JSON.stringify(blocks.value)),
      connections: JSON.parse(JSON.stringify(connections.value)),
    }
  }

  function _pushHistory() {
    history.value = history.value.slice(0, historyIndex.value + 1)
    history.value.push(_snapshot())
    historyIndex.value = history.value.length - 1
  }

  const savedHistoryIndex = ref(-1)
  const isDirty = computed(() => historyIndex.value !== savedHistoryIndex.value)

  async function loadWorkflow() {
    const wf: Workflow = await wails.getWorkflow()
    workflowId.value = wf.id
    workflowName.value = wf.name
    blocks.value = wf.blocks ?? []
    connections.value = wf.connections ?? []
    history.value = []
    historyIndex.value = -1
    _pushHistory()
    savedHistoryIndex.value = historyIndex.value
  }

  async function loadFromFile(path: string) {
    const wf = await wails.loadWorkflow(path)
    workflowId.value = wf.id
    workflowName.value = wf.name
    blocks.value = wf.blocks ?? []
    connections.value = wf.connections ?? []
    history.value = []
    historyIndex.value = -1
    _pushHistory()
    savedHistoryIndex.value = historyIndex.value
  }

  async function addBlock(blockType: string, x: number, y: number) {
    const def = await wails.addBlock(blockType, x, y)
    _pushHistory()
    blocks.value = await (async () => {
      const wf = await wails.getWorkflow()
      connections.value = wf.connections ?? []
      return wf.blocks ?? []
    })()
    return def
  }

  async function removeBlock(blockId: string) {
    await wails.removeBlock(blockId)
    _pushHistory()
    const wf = await wails.getWorkflow()
    blocks.value = wf.blocks ?? []
    connections.value = wf.connections ?? []
  }

  async function addConnection(fromBlockId: string, fromPortId: string, toBlockId: string, toPortId: string) {
    const conn = await wails.addConnection(fromBlockId, fromPortId, toBlockId, toPortId)
    _pushHistory()
    connections.value = [...connections.value, conn]
    return conn
  }

  async function removeConnection(connectionId: string) {
    await wails.removeConnection(connectionId)
    _pushHistory()
    connections.value = connections.value.filter(c => c.id !== connectionId)
  }

  async function updateBlockParams(blockId: string, params: Record<string, unknown>) {
    _pushHistory()
    await wails.updateBlockParams(blockId, params)
    const idx = blocks.value.findIndex(b => b.id === blockId)
    if (idx >= 0) {
      blocks.value[idx] = { ...blocks.value[idx], params: { ...blocks.value[idx].params, ...params } }
    }
  }

  async function updateBlockPosition(blockId: string, x: number, y: number) {
    _pushHistory()
    await wails.updateBlockPosition(blockId, x, y)
    const idx = blocks.value.findIndex(b => b.id === blockId)
    if (idx >= 0) {
      blocks.value[idx] = { ...blocks.value[idx], positionX: x, positionY: y }
    }
  }

  function undo() {
    if (historyIndex.value > 0) {
      historyIndex.value--
      const snap = history.value[historyIndex.value]
      blocks.value = snap.blocks
      connections.value = snap.connections
    }
  }

  function redo() {
    if (historyIndex.value < history.value.length - 1) {
      historyIndex.value++
      const snap = history.value[historyIndex.value]
      blocks.value = snap.blocks
      connections.value = snap.connections
    }
  }

  const canUndo = computed(() => historyIndex.value > 0)
  const canRedo = computed(() => historyIndex.value < history.value.length - 1)

  return {
    blocks,
    connections,
    workflowId,
    workflowName,
    isDirty,
    canUndo,
    canRedo,
    loadWorkflow,
    loadFromFile,
    addBlock,
    removeBlock,
    addConnection,
    removeConnection,
    updateBlockParams,
    updateBlockPosition,
    undo,
    redo,
  }
})
