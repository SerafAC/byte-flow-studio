<script setup lang="ts">
import { ref } from 'vue'
import ContextMenu from 'primevue/contextmenu'
import { useAnalysisBlock } from '../../../composables/useAnalysisBlock'

const props = defineProps<{
  blockId: string
  decimals?: number
  unit?: string
  label?: string
  showHistory?: boolean
}>()

interface DataPoint { timestamp: number; values?: number[]; raw?: number[]; mode?: string }

const emit = defineEmits<{
  (e: 'fullscreen:enter', payload: { blockId: string }): void
  (e: 'fullscreen:exit'): void
}>()

const latest = ref<string>('—')
const sparkline = ref<number[]>([])

function onData(event: { data: { blockId: string; points: DataPoint[] } }) {
  if (event.data.blockId !== props.blockId) return

  for (const pt of event.data.points) {
    const replay = (pt as unknown as { replay?: boolean }).replay
    if (replay && !isHistorical.value) continue

    if (pt.mode === 'raw' || (pt.raw && pt.raw.length > 0)) {
      const bytes: number[] = pt.raw ?? []
      latest.value = bytes.map(b => b.toString(16).padStart(2, '0').toUpperCase()).join(' ')
    } else if (pt.values && pt.values.length > 0) {
      const v = pt.values[0]
      latest.value = v.toFixed(props.decimals ?? 2) + (props.unit ? ' ' + props.unit : '')
      if (props.showHistory) {
        sparkline.value = [...sparkline.value.slice(-19), v]
      }
    }
  }
}

const { isHistorical, contextMenu, contextMenuItems } = useAnalysisBlock({
  blockId: props.blockId,
  onData,
  onHistoricalMode: () => { latest.value = '—'; sparkline.value = [] },
  emit,
})
</script>

<template>
  <div class="value-display" @contextmenu.prevent="contextMenu?.show($event)">
    <ContextMenu ref="contextMenu" :model="contextMenuItems" />
    <div class="vd-label">{{ label || 'Value' }}</div>
    <div class="vd-value">{{ latest }}</div>
    <div v-if="showHistory && sparkline.length > 0" class="vd-sparkline">
      <svg width="100%" height="30" viewBox="0 0 100 30" preserveAspectRatio="none">
        <polyline
          :points="sparkline.map((v, i) => `${(i / (sparkline.length - 1)) * 100},${30 - ((v - Math.min(...sparkline)) / (Math.max(...sparkline) - Math.min(...sparkline) || 1)) * 25}`).join(' ')"
          fill="none"
          stroke="#3b82f6"
          stroke-width="2"
        />
      </svg>
    </div>
    <div v-if="isHistorical" class="vd-historical-badge">HISTORICAL</div>
  </div>
</template>

<style scoped>
.value-display {
  background: #1e2d3d;
  border-radius: 8px;
  padding: 12px;
  min-width: 120px;
  text-align: center;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
}

.vd-label {
  font-size: 11px;
  color: #9ca3af;
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.vd-value {
  font-size: 24px;
  font-weight: 700;
  color: #e2e8f0;
  font-family: monospace;
}

.vd-sparkline {
  margin-top: 6px;
}

.vd-historical-badge {
  font-size: 9px;
  color: #f59e0b;
  margin-top: 4px;
  font-weight: bold;
}
</style>
