<script setup lang="ts">
import { computed, ref, type Component } from 'vue'
import { Handle, Position, useNode } from '@vue-flow/core'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import type { BlockDef } from '../../services/wails'
import QuickAddMenu from './QuickAddMenu.vue'
import { useFullscreen } from '../../composables/useFullscreen'
import { useWorkflowStore } from '../../stores/workflow'
import UartBlockConfig from '../blocks/inputs/UartBlockConfig.vue'
import SimulatorBlockConfig from '../blocks/inputs/SimulatorBlockConfig.vue'
import WebSocketBlockConfig from '../blocks/inputs/WebSocketBlockConfig.vue'
import MovingAverageConfig from '../blocks/processing/MovingAverageConfig.vue'
import FFTConfig from '../blocks/processing/FFTConfig.vue'
import ScalingConfig from '../blocks/processing/ScalingConfig.vue'
import ByteParserConfig from '../blocks/processing/ByteParserConfig.vue'
import SamplerConfig from '../blocks/processing/SamplerConfig.vue'

const props = defineProps<{ data: BlockDef }>()

const quickAddMenu = ref()
const configOpen = ref(false)
const { node } = useNode()
const { enterFullscreen } = useFullscreen()
const workflowStore = useWorkflowStore()

// Ref to the mounted config component instance — exposes { save() }
const configRef = ref<{ save: () => Promise<void> } | null>(null)

async function saveConfig() {
  await configRef.value?.save()
  configOpen.value = false
}

const configComponentMap: Record<string, Component> = {
  'uart': UartBlockConfig,
  'simulator': SimulatorBlockConfig,
  'websocket': WebSocketBlockConfig,
  'moving-average': MovingAverageConfig,
  'fft': FFTConfig,
  'scaling': ScalingConfig,
  'byte-parser': ByteParserConfig,
  'sampler': SamplerConfig,
}

const configComponent = computed(() => configComponentMap[props.data.type] ?? null)
const isSelected = computed(() => !!(node?.selected))

// Double-click: analysis → fullscreen view; input/processing → open config
function onDblClick(event: MouseEvent) {
  if (props.data.category === 'analysis') {
    event.stopPropagation()
    enterFullscreen(props.data.id)
  } else if (configComponent.value) {
    event.stopPropagation()
    configOpen.value = true
  }
}

function openConfig(event: MouseEvent) {
  event.stopPropagation()
  configOpen.value = true
}

function deleteBlock(event: MouseEvent) {
  event.stopPropagation()
  workflowStore.removeBlock(props.data.id)
}

const categoryColor: Record<string, string> = {
  input:      '#3b82f6',
  processing: '#8b5cf6',
  analysis:   '#10b981',
}

const statusColor = computed(() => {
  switch (props.data.status) {
    case 'connected': return '#34d399'
    case 'error':     return '#f87171'
    default:          return '#6b7280'
  }
})

const nodeColor = computed(() =>
  props.data.status === 'error' ? '#f97316' : (categoryColor[props.data.category] ?? '#6b7280')
)

let portDragStart = { x: 0, y: 0 }

function onOutputPortMouseDown(event: MouseEvent) {
  portDragStart = { x: event.clientX, y: event.clientY }
}

function onOutputPortMouseUp(event: MouseEvent) {
  const dx = Math.abs(event.clientX - portDragStart.x)
  const dy = Math.abs(event.clientY - portDragStart.y)
  if (dx < 5 && dy < 5) {
    quickAddMenu.value?.open(event)
  }
}
</script>

<template>
  <div class="block-node" :class="{ 'block-node--selected': isSelected }" :style="{ borderColor: nodeColor }" @dblclick="onDblClick">
    <!-- Status dot -->
    <span class="status-dot" :style="{ background: statusColor }" :title="data.status" />

    <!-- Header -->
    <div class="block-header">
      <span class="block-label">{{ data.label || data.type }}</span>
      <span class="block-badge" :style="{ background: nodeColor }">{{ data.category }}</span>
    </div>

    <!-- Action row: config + delete buttons; floats above the block on hover -->
    <div class="action-row">
      <button
        v-if="configComponent"
        class="action-btn"
        title="Open configuration"
        @mousedown.stop
        @click.stop="openConfig($event)"
      >⚙</button>
      <button
        class="action-btn action-btn--delete"
        title="Delete block"
        @mousedown.stop
        @click.stop="deleteBlock($event)"
      >✕</button>
    </div>

    <!-- Error tooltip -->
    <div v-if="data.status === 'error' && data.errorMessage" class="block-error-msg">
      {{ data.errorMessage }}
    </div>

    <!-- Double-click hint for analysis blocks -->
    <div v-if="data.category === 'analysis'" class="analysis-hint">double-click to view</div>

    <!-- Input handle (left side) — hidden for input-category blocks -->
    <Handle
      v-if="data.category !== 'input'"
      id="in"
      type="target"
      :position="Position.Left"
      class="input-handle"
    />

    <!-- Output handle (right side) — hidden for analysis-category blocks; merged with quick-add -->
    <div v-if="data.category !== 'analysis'" class="out-port-wrapper">
      <Handle
        id="out"
        type="source"
        :position="Position.Right"
        class="output-handle"
        @mousedown="onOutputPortMouseDown($event)"
        @mouseup="onOutputPortMouseUp($event)"
      >+</Handle>
    </div>

    <QuickAddMenu
      ref="quickAddMenu"
      :source-block-id="data.id"
      source-port-id="out"
      :source-data-type="data.category === 'input' && data.type === 'uart' ? 'raw' : 'numeric'"
      :source-x="node?.position.x ?? 0"
      :source-y="node?.position.y ?? 0"
    />
  </div>

  <!-- Config dialog: v-if on configOpen ensures the component is freshly
       mounted each time the dialog opens, so it reads the latest saved params. -->
  <Dialog
    v-if="configOpen && configComponent"
    v-model:visible="configOpen"
    :header="(data.label || data.type) + ' Configuration'"
    :modal="true"
    :draggable="false"
    :style="{ width: '380px' }"
    append-to="body"
  >
    <component
      :is="configComponent"
      ref="configRef"
      :block-id="data.id"
      :params="data.params ?? {}"
    />
    <template #footer>
      <Button label="Cancel" severity="secondary" @click="configOpen = false" />
      <Button label="Save" @click="saveConfig" />
    </template>
  </Dialog>
</template>

<style scoped>
.block-node {
  position: relative;
  min-width: 140px;
  background: #1e2d3d;
  border: 2px solid;
  border-radius: 8px;
  padding: 8px 12px;
  font-size: 12px;
  color: #e2e8f0;
}

.status-dot {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.block-header {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.block-label {
  font-weight: 600;
  font-size: 13px;
}

.block-badge {
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  padding: 1px 5px;
  border-radius: 4px;
  color: white;
}

.block-error-msg {
  margin-top: 4px;
  font-size: 10px;
  color: #f87171;
  word-break: break-word;
}

/* Selected node indicator */
.block-node--selected {
  box-shadow: 0 0 0 2px #f59e0b;
}

/* Action row: floats above the block, revealed on hover.
   padding-bottom bridges the gap so the mouse can travel from block to buttons
   without leaving the hover zone. */
.action-row {
  position: absolute;
  top: -30px;
  left: 0;
  display: flex;
  gap: 4px;
  padding-bottom: 8px;
  visibility: hidden;
  z-index: 10;
}
.block-node:hover .action-row { visibility: visible; }

.action-btn {
  background: rgba(255,255,255,0.1);
  color: #9ca3af;
  border: none;
  border-radius: 4px;
  width: 22px;
  height: 22px;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}
.action-btn:hover { background: rgba(255,255,255,0.2); color: #e2e8f0; }
.action-btn--delete:hover { background: rgba(239,68,68,0.3); color: #f87171; }

/* BUG3: analysis hint text */
.analysis-hint {
  margin-top: 4px;
  font-size: 9px;
  color: #6b7280;
  font-style: italic;
  visibility: hidden;
}
.block-node:hover .analysis-hint { visibility: visible; }

.out-port-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

/* Output handle: styled as a large clickable "+" button.
   Click → open QuickAddMenu; drag → Vue Flow connection drag. */
:deep(.output-handle.vue-flow__handle) {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #3b82f6;
  border: none;
  right: -12px;
  color: white;
  font-size: 16px;
  font-weight: bold;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 10;
  padding: 0;
}
:deep(.output-handle.vue-flow__handle:hover) { background: #2563eb; }

/* Input handles: larger target area for easier clicking */
:deep(.input-handle.vue-flow__handle) {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: #6b7280;
  border: none;
  left: -7px;
}
:deep(.input-handle.vue-flow__handle:hover) { background: #9ca3af; }
</style>
