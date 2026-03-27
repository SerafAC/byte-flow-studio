<script setup lang="ts">
import { onMounted, onUnmounted, computed, ref, type Component } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import ConfirmDialog from 'primevue/confirmdialog'
import BlockLibraryPanel from '../components/panels/BlockLibraryPanel.vue'
import WorkflowCanvas from '../components/canvas/WorkflowCanvas.vue'
import SessionPanel from '../components/panels/SessionPanel.vue'
import LineChartBlock from '../components/blocks/analysis/LineChartBlock.vue'
import ValueDisplayBlock from '../components/blocks/analysis/ValueDisplayBlock.vue'
import DataTableBlock from '../components/blocks/analysis/DataTableBlock.vue'
import BarChartBlock from '../components/blocks/analysis/BarChartBlock.vue'
import FftSpectrumBlock from '../components/blocks/analysis/FftSpectrumBlock.vue'
import { useWorkflowStore } from '../stores/workflow'
import { usePipelineStore } from '../stores/pipeline'
import { useSessionStore } from '../stores/session'
import { useFullscreen } from '../composables/useFullscreen'
import * as wails from '../services/wails'
import { isPipelineTopologyError, isCorruptedFileError } from '../types/pipeline-errors'

const analysisComponentMap: Record<string, Component> = {
  'line-chart': LineChartBlock,
  'value-display': ValueDisplayBlock,
  'data-table': DataTableBlock,
  'bar-chart': BarChartBlock,
  'fft-spectrum': FftSpectrumBlock,
}

const toast = useToast()
const confirm = useConfirm()
const workflowStore = useWorkflowStore()
const pipelineStore = usePipelineStore()
const sessionStore = useSessionStore()
const { fullscreenBlockId, exitFullscreen } = useFullscreen()

const fullscreenBlock = computed(() =>
  fullscreenBlockId.value ? workflowStore.blocks.find(b => b.id === fullscreenBlockId.value) ?? null : null
)

const fullscreenComponent = computed(() =>
  fullscreenBlock.value ? (analysisComponentMap[fullscreenBlock.value.type] ?? null) : null
)

function onFullscreenEscape(e: KeyboardEvent) {
  if (e.key === 'Escape' && fullscreenBlockId.value) exitFullscreen()
}

// T107: persistent FR-004 error banner (dismissed on successful start)
const startErrorBanner = ref<string | null>(null)

// T106: corrupted file recovery state
const corruptedFilePath = ref<string | null>(null)

onMounted(async () => {
  pipelineStore.subscribe()
  sessionStore.subscribe()
  document.addEventListener('keydown', onFullscreenEscape)
  await workflowStore.loadWorkflow()
  await sessionStore.loadSessions()
})

onUnmounted(() => {
  pipelineStore.unsubscribe()
  sessionStore.unsubscribe()
  document.removeEventListener('keydown', onFullscreenEscape)
})

async function onStart() {
  try {
    await pipelineStore.startFlow()
    startErrorBanner.value = null // clear on successful start (T107)
  } catch (e: unknown) {
    const msg = String(e)
    // T107: show persistent banner for FR-004 topology errors
    if (isPipelineTopologyError(msg)) {
      startErrorBanner.value = msg
    } else {
      toast.add({ severity: 'error', summary: 'Start failed', detail: msg, life: 5000 })
    }
  }
}

async function onPause() {
  try {
    await pipelineStore.pauseFlow()
  } catch (e: unknown) {
    toast.add({ severity: 'error', summary: 'Pause failed', detail: String(e), life: 3000 })
  }
}

async function onResume() {
  try {
    await pipelineStore.resumeFlow()
  } catch (e: unknown) {
    toast.add({ severity: 'error', summary: 'Resume failed', detail: String(e), life: 3000 })
  }
}

async function onStop() {
  try {
    await pipelineStore.stopFlow()
  } catch (e: unknown) {
    toast.add({ severity: 'error', summary: 'Stop failed', detail: String(e), life: 3000 })
  }
}

async function openFileDialog(): Promise<string | null> {
  try {
    const { Dialogs } = await import('@wailsio/runtime')
    const result = await Dialogs.OpenFile({
      Title: 'Open Workflow',
      Filters: [{ DisplayName: 'ByteFlow Workflow (*.byteflow)', Pattern: '*.byteflow' }],
    })
    return result ?? null
  } catch {
    return null
  }
}

async function saveFileDialog(): Promise<string | null> {
  try {
    const { Dialogs } = await import('@wailsio/runtime')
    const result = await Dialogs.SaveFile({
      Title: 'Save Workflow',
      DefaultFilename: 'workflow.byteflow',
      Filters: [{ DisplayName: 'ByteFlow Workflow (*.byteflow)', Pattern: '*.byteflow' }],
    })
    return result ?? null
  } catch {
    return null
  }
}

async function onOpen() {
  if (workflowStore.isDirty) {
    confirm.require({
      message: 'You have unsaved changes. Open a new file anyway?',
      header: 'Unsaved Changes',
      accept: async () => {
        const path = await openFileDialog()
        if (path) await doLoad(path)
      },
    })
    return
  }
  const path = await openFileDialog()
  if (path) await doLoad(path)
}

async function doLoad(path: string) {
  try {
    await workflowStore.loadFromFile(path)
    await sessionStore.loadSessions()
    toast.add({ severity: 'success', summary: 'Opened', detail: path, life: 2000 })
  } catch (e: unknown) {
    const msg = String(e)
    // T106: corrupted file recovery — offer Recovery Mode dialog
    if (isCorruptedFileError(msg)) {
      corruptedFilePath.value = path
    } else {
      toast.add({ severity: 'error', summary: 'Open failed', detail: msg, life: 5000 })
    }
  }
}

async function doRecoveryLoad() {
  if (!corruptedFilePath.value) return
  try {
    await wails.recoverWorkflow(corruptedFilePath.value)
    await sessionStore.loadSessions()
    toast.add({ severity: 'warn', summary: 'Recovery Mode', detail: 'Workflow opened with sessions discarded', life: 3000 })
  } catch (e: unknown) {
    toast.add({ severity: 'error', summary: 'Recovery failed', detail: String(e), life: 5000 })
  } finally {
    corruptedFilePath.value = null
  }
}

async function onSave() {
  try {
    const path = await saveFileDialog()
    if (!path) return
    await wails.saveWorkflow(path)
    toast.add({ severity: 'success', summary: 'Saved', detail: path, life: 2000 })
  } catch (e: unknown) {
    toast.add({ severity: 'error', summary: 'Save failed', detail: String(e), life: 3000 })
  }
}

function stateColor() {
  switch (pipelineStore.flowState) {
    case 'running': return 'success'
    case 'paused':  return 'warn'
    case 'error':   return 'danger'
    default:        return 'secondary'
  }
}

const blockErrorEntries = () => Object.entries(pipelineStore.blockErrors)
</script>

<template>
  <div class="main-layout">
    <ConfirmDialog />

  <!-- Menu bar -->
    <div class="menu-bar">
      <span class="app-title">ByteFlow Studio</span>
      <div class="menu-actions">
        <Button label="Open" size="small" severity="secondary" @click="onOpen" />
        <Button label="Save" size="small" severity="secondary" @click="onSave" />
      </div>
      <!-- Pipeline controls -->
      <div class="pipeline-controls">
        <Button label="Start" size="small" severity="success" :disabled="!pipelineStore.canStart" @click="onStart" />
        <Button label="Pause" size="small" severity="warn" :disabled="!pipelineStore.canPause" @click="onPause" />
        <Button label="Resume" size="small" severity="info" :disabled="!pipelineStore.canResume" @click="onResume" />
        <Button label="Stop" size="small" severity="danger" :disabled="!pipelineStore.canStop" @click="onStop" />
        <Tag :value="pipelineStore.flowState.toUpperCase()" :severity="stateColor()" />
      </div>
    </div>

    <!-- T107: FR-004 persistent topology error banner (dismissed on successful start) -->
    <div v-if="startErrorBanner" class="error-banner error-banner--topology">
      <span>{{ startErrorBanner }}</span>
      <button class="error-banner-dismiss" @click="startErrorBanner = null">✕</button>
    </div>

    <!-- Error banner for FR-004 / block errors -->
    <div v-if="pipelineStore.flowState === 'error' && blockErrorEntries().length > 0" class="error-banner">
      <span>Pipeline error — </span>
      <span v-for="[id, msg] in blockErrorEntries()" :key="id">
        <strong>{{ id }}</strong>: {{ msg }};
      </span>
    </div>

    <!-- T106: Corrupted file recovery dialog -->
    <Teleport to="body">
      <div v-if="corruptedFilePath" class="recovery-overlay">
        <div class="recovery-dialog">
          <h3>Cannot Open File</h3>
          <p>The file <code>{{ corruptedFilePath }}</code> is corrupted or not a valid ByteFlow file.</p>
          <div class="recovery-actions">
            <Button label="Recovery Mode" severity="warn" size="small" @click="doRecoveryLoad" />
            <Button label="Cancel" severity="secondary" size="small" @click="corruptedFilePath = null" />
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Three-panel body -->
    <div class="body">
      <BlockLibraryPanel class="sidebar-left" />
      <WorkflowCanvas class="canvas-area" />
      <SessionPanel class="sidebar-right" />
    </div>

    <!-- Status bar -->
    <div class="status-bar">
      <span>State: {{ pipelineStore.flowState }}</span>
      <span v-if="pipelineStore.sessionId"> | Session: {{ pipelineStore.sessionId }}</span>
    </div>
  </div>

  <!-- Fullscreen overlay -->
  <Teleport to="body">
    <div v-if="fullscreenComponent && fullscreenBlock" class="fullscreen-overlay" @click.self="exitFullscreen">
      <div class="fullscreen-content">
        <button class="fullscreen-close" @click="exitFullscreen" title="Close (Esc)">✕</button>
        <component
          :is="fullscreenComponent"
          :block-id="fullscreenBlock.id"
          v-bind="fullscreenBlock.params ?? {}"
          class="fullscreen-block"
        />
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.main-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #1b2636;
  color: #e0e0e0;
}

.menu-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: #243447;
  border-bottom: 1px solid #3a4a5c;
  flex-shrink: 0;
}

.app-title {
  font-weight: bold;
  font-size: 14px;
  margin-right: 12px;
}

.menu-actions {
  display: flex;
  gap: 6px;
}

.pipeline-controls {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}

.error-banner {
  background: #7f1d1d;
  color: #fca5a5;
  padding: 6px 12px;
  font-size: 13px;
  flex-shrink: 0;
}

.error-banner--topology {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #78350f;
  color: #fde68a;
}

.error-banner-dismiss {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 14px;
  padding: 0 4px;
}

.body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.sidebar-left {
  width: 220px;
  flex-shrink: 0;
  border-right: 1px solid #3a4a5c;
  overflow-y: auto;
}

.canvas-area {
  flex: 1;
  overflow: hidden;
}

.sidebar-right {
  width: 240px;
  flex-shrink: 0;
  border-left: 1px solid #3a4a5c;
  overflow-y: auto;
}

.status-bar {
  padding: 4px 12px;
  font-size: 12px;
  background: #243447;
  border-top: 1px solid #3a4a5c;
  flex-shrink: 0;
  color: #9ca3af;
}
</style>

<style>
.fullscreen-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
}

.fullscreen-content {
  position: relative;
  width: 90vw;
  height: 85vh;
  background: #1e2d3d;
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
}

.fullscreen-close {
  position: absolute;
  top: 10px;
  right: 12px;
  background: none;
  border: none;
  color: #9ca3af;
  font-size: 18px;
  cursor: pointer;
  z-index: 1;
  line-height: 1;
  padding: 4px 8px;
  border-radius: 4px;
}
.fullscreen-close:hover { background: rgba(255,255,255,0.1); color: #e2e8f0; }

.fullscreen-block {
  flex: 1;
  min-height: 0;
}

.recovery-overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
}

.recovery-dialog {
  background: #1e2d3d;
  border: 1px solid #f59e0b;
  border-radius: 10px;
  padding: 24px;
  max-width: 480px;
  color: #e2e8f0;
}

.recovery-dialog h3 { margin: 0 0 12px; color: #f59e0b; }
.recovery-dialog p { font-size: 13px; margin: 0 0 16px; }
.recovery-dialog code { font-size: 11px; word-break: break-all; color: #93c5fd; }

.recovery-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
