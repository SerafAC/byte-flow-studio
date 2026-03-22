<script setup lang="ts">
import { computed, ref } from 'vue'
import { Handle, Position, useNode } from '@vue-flow/core'
import type { BlockDef } from '../../services/wails'
import QuickAddMenu from './QuickAddMenu.vue'

const props = defineProps<{ data: BlockDef }>()

const hovered = ref(false)
const quickAddMenu = ref()
const { node } = useNode()

function onQuickAdd(event: MouseEvent) {
  event.stopPropagation()
  quickAddMenu.value?.open(event)
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
</script>

<template>
  <div class="block-node" :style="{ borderColor: nodeColor }" @mouseenter="hovered = true" @mouseleave="hovered = false">
    <!-- Status dot -->
    <span class="status-dot" :style="{ background: statusColor }" :title="data.status" />

    <!-- Header -->
    <div class="block-header">
      <span class="block-label">{{ data.label || data.type }}</span>
      <span class="block-badge" :style="{ background: nodeColor }">{{ data.category }}</span>
    </div>

    <!-- Error tooltip -->
    <div v-if="data.status === 'error' && data.errorMessage" class="block-error-msg">
      {{ data.errorMessage }}
    </div>

    <!-- Input handles (left side) -->
    <Handle
      v-for="port in ['in', 'in-numeric', 'in-raw']"
      :key="'in-' + port"
      :id="port"
      type="target"
      :position="Position.Left"
    />

    <!-- Output handles (right side) with quick-add button -->
    <div v-for="port in ['out']" :key="'out-' + port" class="out-port-wrapper">
      <Handle
        :id="port"
        type="source"
        :position="Position.Right"
      />
      <button
        v-show="hovered"
        class="quick-add-btn"
        title="Quick add block"
        @mousedown.stop
        @click="onQuickAdd($event)"
      >+</button>
    </div>

    <QuickAddMenu
      ref="quickAddMenu"
      :source-block-id="data.id"
      source-port-id="out"
      :source-data-type="data.category === 'input' && data.type === 'uart' ? 'raw' : 'numeric'"
      :source-x="node?.position.x ?? 0"
      :source-y="node?.position.y ?? 0"
      @close="hovered = false"
    />
  </div>
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

.out-port-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.quick-add-btn {
  position: absolute;
  right: -36px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 50%;
  width: 20px;
  height: 20px;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
  padding: 0;
}
.quick-add-btn:hover { background: #2563eb; }
</style>
