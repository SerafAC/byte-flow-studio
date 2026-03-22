<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Events } from '@wailsio/runtime'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import ContextMenu from 'primevue/contextmenu'
import { useFullscreen } from '../../../composables/useFullscreen'

const props = defineProps<{
  blockId: string
  xLabel?: string
  yLabel?: string
  decimals?: number
  logScaleY?: boolean
}>()

const emit = defineEmits<{
  (e: 'fullscreen:enter', payload: { blockId: string }): void
  (e: 'fullscreen:exit'): void
}>()

const { enterFullscreen: fsEnter, exitFullscreen: fsExit } = useFullscreen()

const contextMenu = ref()
const contextMenuItems = [
  { label: 'Full Screen', icon: 'pi pi-window-maximize', command: () => { fsEnter(props.blockId); emit('fullscreen:enter', { blockId: props.blockId }) } },
]

function onEscapeKey(e: KeyboardEvent) { if (e.key === 'Escape') { fsExit(); emit('fullscreen:exit') } }

interface DataPoint { timestamp: number; values?: number[] }

const chartEl = ref<HTMLDivElement | null>(null)
const isHistorical = ref(false)
const logScale = ref(props.logScaleY ?? false)

let uplot: uPlot | null = null
let lastBinCount = 0

function buildOpts(binCount: number): uPlot.Options {
  const bins = new Float64Array(binCount).map((_, i) => i)
  return {
    title: '',
    width: chartEl.value?.clientWidth || 300,
    height: 180,
    series: [
      {},
      {
        label: props.yLabel ?? 'Magnitude',
        stroke: '#3b82f6',
        fill: 'rgba(59, 130, 246, 0.15)',
        width: 1.5,
      },
    ],
    axes: [
      { label: props.xLabel ?? 'Frequency Bin' },
      { label: props.yLabel ?? 'Magnitude', ...(logScale.value ? { scale: 'log' } : {}) },
    ],
    scales: {
      x: { time: false, range: [0, binCount - 1] },
      y: logScale.value ? { distr: 3, log: 10 } : {},
    },
  }
}

function renderSpectrum(values: number[]) {
  if (!chartEl.value) return
  if (!uplot || lastBinCount !== values.length) {
    uplot?.destroy()
    lastBinCount = values.length
    const bins = Array.from({ length: values.length }, (_, i) => i)
    uplot = new uPlot(buildOpts(values.length), [new Float64Array(bins), new Float64Array(values)], chartEl.value)
  } else {
    const bins = new Float64Array(values.length).map((_, i) => i)
    uplot.setData([new Float64Array(Array.from({ length: values.length }, (_, i) => i)), new Float64Array(values)])
  }
}

function onData(event: { data: { blockId: string; points: DataPoint[] } }) {
  if (event.data.blockId !== props.blockId) return
  for (const pt of event.data.points) {
    const replay = (pt as unknown as { replay?: boolean }).replay
    if (replay && !isHistorical.value) continue
    if (pt.values && pt.values.length > 0) {
      renderSpectrum(pt.values)
    }
  }
}

function onViewChanged(event: { data: { sessionId: string; mode: 'live' | 'historical' } }) {
  isHistorical.value = event.data.mode === 'historical'
  if (isHistorical.value) {
    uplot?.destroy()
    uplot = null
    lastBinCount = 0
  }
}

onMounted(() => {
  Events.On('pipeline:data', onData)
  Events.On('session:view-changed', onViewChanged)
  document.addEventListener('keydown', onEscapeKey)
})

onUnmounted(() => {
  Events.Off('pipeline:data', onData)
  Events.Off('session:view-changed', onViewChanged)
  document.removeEventListener('keydown', onEscapeKey)
  uplot?.destroy()
  uplot = null
})
</script>

<template>
  <div class="fft-spectrum-block" @contextmenu.prevent="contextMenu?.show($event)">
    <ContextMenu ref="contextMenu" :model="contextMenuItems" />
    <div v-if="isHistorical" class="historical-badge">HISTORICAL</div>
    <div class="toolbar">
      <label class="log-toggle">
        <input v-model="logScale" type="checkbox" @change="() => { uplot?.destroy(); uplot = null; lastBinCount = 0 }" />
        Log scale
      </label>
    </div>
    <div ref="chartEl" class="chart-container" />
  </div>
</template>

<style scoped>
.fft-spectrum-block {
  background: #1e2d3d;
  border-radius: 8px;
  padding: 8px;
  position: relative;
}
.toolbar {
  display: flex; align-items: center; gap: 8px;
  font-size: 11px; color: #9ca3af; margin-bottom: 4px;
}
.log-toggle { display: flex; align-items: center; gap: 4px; cursor: pointer; }
.chart-container { width: 100%; }
.historical-badge {
  position: absolute; top: 6px; right: 8px;
  font-size: 9px; color: #f59e0b; font-weight: bold; z-index: 1;
}
</style>
