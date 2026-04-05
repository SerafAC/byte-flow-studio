<script setup lang="ts">
import { computed, ref, type Component } from 'vue'
import { Handle, Position, useNode } from '@vue-flow/core'
import { NodeResizer } from '@vue-flow/node-resizer'
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
import LineChartBlock from '../blocks/analysis/LineChartBlock.vue'
import BarChartBlock from '../blocks/analysis/BarChartBlock.vue'
import FftSpectrumBlock from '../blocks/analysis/FftSpectrumBlock.vue'
import ValueDisplayBlock from '../blocks/analysis/ValueDisplayBlock.vue'
import DataTableBlock from '../blocks/analysis/DataTableBlock.vue'

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

const analysisComponentMap: Record<string, Component> = {
  'line-chart':   LineChartBlock,
  'bar-chart':    BarChartBlock,
  'fft-spectrum': FftSpectrumBlock,
  'value-display': ValueDisplayBlock,
  'data-table':   DataTableBlock,
}

const analysisComponent = computed(() =>
  props.data.category === 'analysis' ? (analysisComponentMap[props.data.type] ?? null) : null
)

const isSelected = computed(() => !!(node?.selected))

const analysisMinDimensions: Record<string, { minWidth: number; minHeight: number }> = {
  'line-chart':    { minWidth: 240, minHeight: 160 },
  'value-display': { minWidth: 140, minHeight: 100 },
  'bar-chart':     { minWidth: 240, minHeight: 160 },
  'fft-spectrum':  { minWidth: 240, minHeight: 200 },
  'data-table':    { minWidth: 200, minHeight: 140 },
}

const resizerMinWidth = computed(() => analysisMinDimensions[props.data.type]?.minWidth ?? 240)
const resizerMinHeight = computed(() => analysisMinDimensions[props.data.type]?.minHeight ?? 160)

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
  input:      '#4381cf',
  processing: '#9d50cf',
  analysis:   '#49b393',
}

const statusColor = computed(() => {
  switch (props.data.status) {
    case 'connected': return '#49b393'
    case 'error':     return '#e07a90'
    default:          return '#4a6090'
  }
})

const nodeColor = computed(() =>
  props.data.status === 'error' ? '#f97316' : (categoryColor[props.data.category] ?? '#6b7280')
)

async function onResizeEnd(_: unknown, params: { width: number; height: number }) {
  await workflowStore.updateBlockSize(props.data.id, params.width, params.height)
}

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
  <div
    class="block-node"
    :class="{ 'block-node--selected': isSelected, 'block-node--analysis': data.category === 'analysis' }"
    :style="{ borderColor: nodeColor }"
    @dblclick="onDblClick"
  >
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

    <!-- Inline analysis rendering: chart/table/value mounts directly inside the node -->
    <div v-if="data.category === 'analysis'" class="inline-analysis">
      <component
        v-if="analysisComponent"
        :is="analysisComponent"
        :block-id="data.id"
        v-bind="data.params ?? {}"
      />
      <div v-else class="inline-analysis-placeholder">Waiting for data…</div>
    </div>

    <!-- NodeResizer: allows dragging the node corner to resize analysis blocks -->
    <NodeResizer
      v-if="data.category === 'analysis'"
      :min-width="resizerMinWidth"
      :min-height="resizerMinHeight"
      :is-visible="isSelected"
      @resize-end="onResizeEnd"
    />

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

<style scoped lang="scss">
@use '../../assets/variables' as *;

.block-node {
  position: relative;
  min-width: 140px;
  background: $bg-card;
  border: 1px solid;
  border-radius: $radius-lg;
  padding: $space-md $space-lg;
  font-size: $font-size-base;
  font-family: $font-body;
  color: $text-primary;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  // Non-analysis nodes size to their content — no explicit height needed
}

// Analysis nodes always receive an explicit width+height via the Vue Flow node style prop,
// so height:100% correctly fills the VueFlow node container without causing a growth loop
.block-node--analysis {
  width: 100%;
  height: 100%;
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
  gap: $space-sm;
  flex-wrap: wrap;
}

.block-label {
  font-family: $font-display;
  font-weight: 600;
  font-size: $font-size-lg;
  color: $text-primary;
}

.block-badge {
  font-size: $font-size-xs;
  font-weight: 700;
  text-transform: uppercase;
  padding: 1px 5px;
  border-radius: $radius-sm;
  color: white;
}

.block-error-msg {
  margin-top: $space-xs;
  font-size: $font-size-sm;
  color: $color-danger-light;
  word-break: break-word;
}

// Selected node: background-shift instead of glow
.block-node--selected {
  background: $bg-block;
}

// Action row: floats above the block, revealed on hover.
// padding-bottom bridges the gap so the mouse can travel from block to buttons
// without leaving the hover zone.
.action-row {
  position: absolute;
  top: -30px;
  left: 0;
  display: flex;
  gap: $space-xs;
  padding-bottom: $space-md;
  visibility: hidden;
  z-index: $z-node;
}
.block-node:hover .action-row { visibility: visible; }

.action-btn {
  background: rgba(255,255,255,0.07);
  color: $text-muted;
  border: none;
  border-radius: $radius-sm;
  width: 22px;
  height: 22px;
  font-size: $font-size-md;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  &:hover { background: rgba(255,255,255,0.14); color: $text-primary; }
}
.action-btn--delete:hover { background: rgba($color-danger, 0.25); color: $color-danger-light; }

.inline-analysis {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  width: 100%;
  margin-top: 6px;
}

.inline-analysis-placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  color: #6b7280;
  font-style: italic;
}

.out-port-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

// Output handle: styled as a large clickable "+" button.
// Click → open QuickAddMenu; drag → Vue Flow connection drag.
:deep(.output-handle.vue-flow__handle) {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: $color-primary;
  border: none;
  right: -12px;
  color: white;
  font-size: 16px;
  font-weight: bold;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: $z-node;
  padding: 0;
  &:hover { background: darken($color-primary, 10%); }
}

// Input handles: larger target area for easier clicking
:deep(.input-handle.vue-flow__handle) {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: $text-muted;
  border: none;
  left: -7px;
  &:hover { background: $text-secondary; }
}
</style>
