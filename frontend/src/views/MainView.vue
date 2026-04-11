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
import SpectrumViewerBlock from '../components/blocks/analysis/SpectrumViewerBlock.vue'
import HexViewerBlock from '../components/blocks/analysis/HexViewerBlock.vue'
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
  'spectrum-viewer': SpectrumViewerBlock,
  'hex-viewer': HexViewerBlock,
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

// T107: persistent FR-004 error banner (dismissed on successful start)
const startErrorBanner = ref<string | null>(null)

// T106: corrupted file recovery state
const corruptedFilePath = ref<string | null>(null)

function onGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && fullscreenBlockId.value) {
    exitFullscreen()
    return
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'o') {
    e.preventDefault()
    onOpen()
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault()
    onSave()
  }
}

onMounted(async () => {
  pipelineStore.subscribe()
  sessionStore.subscribe()
  document.addEventListener('keydown', onGlobalKeydown)
  await workflowStore.loadWorkflow()
  await sessionStore.loadSessions()
})

onUnmounted(() => {
  pipelineStore.unsubscribe()
  sessionStore.unsubscribe()
  document.removeEventListener('keydown', onGlobalKeydown)
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

// Block library toggle state (starts visible, closeable via ✕, reopenable via toolbar)
const libraryOpen = ref(true)
</script>

<template>
  <div class="main-layout">
    <ConfirmDialog />

    <!-- Canvas fills the full viewport as the background layer -->
    <WorkflowCanvas class="canvas-fill" />

    <!-- Floating header bar -->
    <div class="menu-bar">
      <span class="app-title">ByteFlow Studio</span>
      <div class="menu-actions">
        <button class="ctrl-btn" @click="onOpen">Open</button>
        <button class="ctrl-btn" @click="onSave">Save</button>
      </div>
      <!-- Pipeline controls -->
      <div class="pipeline-controls">
        <button class="ctrl-btn ctrl-btn--start" :disabled="!pipelineStore.canStart" @click="onStart">Start</button>
        <button class="ctrl-btn ctrl-btn--pause" :disabled="!pipelineStore.canPause" @click="onPause">Pause</button>
        <button class="ctrl-btn ctrl-btn--resume" :disabled="!pipelineStore.canResume" @click="onResume">Resume</button>
        <button class="ctrl-btn ctrl-btn--stop" :disabled="!pipelineStore.canStop" @click="onStop">Stop</button>
        <Tag :value="pipelineStore.flowState.toUpperCase()" :severity="stateColor()" />
      </div>
    </div>

    <!-- T107: FR-004 persistent topology error banner -->
    <div v-if="startErrorBanner" class="error-banner error-banner--topology">
      <span>{{ startErrorBanner }}</span>
      <button class="error-banner-dismiss" @click="startErrorBanner = null">✕</button>
    </div>

    <!-- Error banner for block errors -->
    <div v-if="pipelineStore.flowState === 'error' && blockErrorEntries().length > 0" class="error-banner">
      <span>Pipeline error — </span>
      <span v-for="[id, msg] in blockErrorEntries()" :key="id">
        <strong>{{ id }}</strong>: {{ msg }};
      </span>
    </div>

    <!-- Slim toolbar (left, below header) — toggles block library -->
    <div class="toolbar-strip">
      <button
        class="toolbar-btn"
        data-testid="toggle-library"
        title="Toggle block library"
        @click="libraryOpen = !libraryOpen"
      >⊞</button>
    </div>

    <!-- Floating block library panel (toggleable) -->
    <BlockLibraryPanel
      v-if="libraryOpen"
      class="panel-library"
      @close="libraryOpen = false"
    />

    <!-- Floating sessions panel (right) -->
    <SessionPanel class="panel-sessions" />

    <!-- Pill-shaped status bar (floating, bottom) -->
    <div class="status-bar">
      <span :class="'state-dot state-dot--' + pipelineStore.flowState" />
      <span>State: {{ pipelineStore.flowState }}</span>
      <span v-if="pipelineStore.sessionId"> | Session: {{ pipelineStore.sessionId }}</span>
      <span class="status-spacer" />
      <span class="status-version">v0.1.0-dev</span>
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

<style scoped lang="scss">
@use '../assets/variables' as *;

// ── Root layout ────────────────────────────────────────────────────────────────

.main-layout {
  position: fixed;
  inset: 0;
  background: $bg-root;
  color: $text-primary;
  font-family: $font-body;
}

// Canvas fills the entire viewport behind the floating panels
.canvas-fill {
  position: absolute;
  inset: 0;
}

// ── Floating header ────────────────────────────────────────────────────────────

.menu-bar {
  position: fixed;
  top: 12px;
  left: 12px;
  right: 12px;
  height: 56px;
  z-index: $z-overlay;
  display: flex;
  align-items: center;
  gap: $space-md;
  padding: 0 $space-xl;
  background: $glass-bg;
  backdrop-filter: blur($glass-blur);
  -webkit-backdrop-filter: blur($glass-blur);
  box-shadow: $glass-shadow;
  border-radius: $radius-xl;
  border: 1px solid $ghost-border;
}

.app-title {
  font-family: $font-display;
  font-weight: 600;
  font-size: $font-size-xl;
  color: $text-primary;
  margin-right: $space-md;
  white-space: nowrap;
}

.menu-actions {
  display: flex;
  gap: $space-sm;
}

.pipeline-controls {
  display: flex;
  align-items: center;
  gap: $space-sm;
  margin-left: auto;
}

// ── Control buttons ────────────────────────────────────────────────────────────

.ctrl-btn {
  height: 28px;
  padding: 0 $space-lg;
  border-radius: $radius-md;
  font-family: $font-display;
  font-size: $font-size-base;
  font-weight: 500;
  cursor: pointer;
  background: transparent;
  border: 1px solid $border-color;
  color: $text-secondary;
  transition: background $transition-fast, color $transition-fast;

  &:disabled { opacity: 0.4; cursor: default; }
  &:not(:disabled):hover { background: rgba(255,255,255,0.06); color: $text-primary; }
}

.ctrl-btn--start {
  border-color: $color-success;
  color: $color-success;
  &:not(:disabled):hover { background: rgba($color-success, 0.12); }
}

.ctrl-btn--pause {
  border-color: $color-warning;
  color: $color-warning;
  &:not(:disabled):hover { background: rgba($color-warning, 0.12); }
}

.ctrl-btn--resume {
  background: $color-primary;
  border-color: $color-primary;
  color: #fff;
  &:not(:disabled):hover { background: darken($color-primary, 8%); }
}

.ctrl-btn--stop {
  border-color: $color-danger;
  background: rgba($color-danger, 0.15);
  color: $color-danger;
  &:not(:disabled):hover { background: rgba($color-danger, 0.25); }
}

// ── Error banners ──────────────────────────────────────────────────────────────

.error-banner {
  position: fixed;
  top: 80px;
  left: 12px;
  right: 12px;
  z-index: $z-overlay;
  background: rgba($color-danger, 0.2);
  border: 1px solid $color-danger;
  color: $color-danger-light;
  padding: $space-sm $space-xl;
  font-size: $font-size-md;
  border-radius: $radius-md;
}

.error-banner--topology {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba($color-warning, 0.15);
  border-color: $color-warning;
  color: $color-warning;
}

.error-banner-dismiss {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: $font-size-xl;
  padding: 0 $space-xs;
  border-radius: $radius-sm;
  &:hover { background: rgba(255,255,255,0.1); }
}

// ── Toolbar strip ──────────────────────────────────────────────────────────────

.toolbar-strip {
  position: fixed;
  top: 80px;
  bottom: 48px;
  left: 12px;
  width: 44px;
  z-index: $z-overlay;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $space-md 0;
  gap: $space-sm;
  background: $glass-bg;
  backdrop-filter: blur($glass-blur);
  -webkit-backdrop-filter: blur($glass-blur);
  box-shadow: $glass-shadow;
  border-radius: $radius-lg;
  border: 1px solid $ghost-border;
}

.toolbar-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  border-radius: $radius-md;
  color: $text-secondary;
  font-size: 16px;
  cursor: pointer;
  &:hover { background: rgba(255,255,255,0.08); color: $text-primary; }
}

// ── Floating panels ────────────────────────────────────────────────────────────

.panel-library {
  position: fixed;
  top: 80px;
  bottom: 48px;
  left: 68px;
  width: 256px;
  z-index: $z-overlay;
}

.panel-sessions {
  position: fixed;
  top: 80px;
  bottom: 48px;
  right: 12px;
  width: 256px;
  z-index: $z-overlay;
}

// ── Status bar pill ────────────────────────────────────────────────────────────

.status-bar {
  position: fixed;
  bottom: 8px;
  left: 12px;
  right: 12px;
  height: 28px;
  z-index: $z-overlay;
  display: flex;
  align-items: center;
  gap: $space-md;
  padding: 0 $space-xl;
  background: $glass-bg;
  backdrop-filter: blur($glass-blur);
  -webkit-backdrop-filter: blur($glass-blur);
  box-shadow: $glass-shadow;
  border-radius: 14px;
  border: 1px solid $ghost-border;
  font-size: $font-size-sm;
  color: $text-muted;
}

.state-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
  background: $status-default;

  &--running  { background: $status-connected; }
  &--error    { background: $status-error; }
  &--paused   { background: $color-warning; }
}

.status-spacer { flex: 1; }

.status-version {
  color: $text-muted;
  font-size: $font-size-xs;
  opacity: 0.6;
}
</style>

<style lang="scss">
@use '../assets/variables' as v;

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
  background: v.$bg-card;
  border-radius: v.$radius-xl;
  border: 1px solid v.$ghost-border;
  padding: 16px;
  display: flex;
  flex-direction: column;
  box-shadow: v.$glass-shadow;
}

.fullscreen-close {
  position: absolute;
  top: 10px;
  right: 12px;
  background: none;
  border: none;
  color: v.$text-muted;
  font-size: 18px;
  cursor: pointer;
  z-index: 1;
  line-height: 1;
  padding: 4px 8px;
  border-radius: v.$radius-md;
}
.fullscreen-close:hover { background: rgba(255,255,255,0.1); color: v.$text-primary; }

.fullscreen-block {
  flex: 1;
  min-height: 0;
}

.recovery-overlay {
  position: fixed;
  inset: 0;
  z-index: v.$z-recovery;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
}

.recovery-dialog {
  background: v.$bg-card;
  border: 1px solid v.$color-warning;
  border-radius: v.$radius-xl;
  padding: 24px;
  max-width: 480px;
  color: v.$text-primary;
}

.recovery-dialog h3 { margin: 0 0 12px; color: v.$color-warning; }
.recovery-dialog p { font-size: 13px; margin: 0 0 16px; color: v.$text-secondary; }
.recovery-dialog code { font-size: 11px; word-break: break-all; color: v.$color-cyan; }

.recovery-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
