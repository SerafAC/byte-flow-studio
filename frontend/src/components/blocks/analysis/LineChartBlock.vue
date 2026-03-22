<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Events } from '@wailsio/runtime'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import ContextMenu from 'primevue/contextmenu'
import { useFullscreen } from '../../../composables/useFullscreen'

const props = defineProps<{
  blockId: string
  title?: string
  xLabel?: string
  yLabel?: string
  unit?: string
  decimals?: number
  bufferSamples?: number
}>()

const emit = defineEmits<{
  (e: 'fullscreen:enter', payload: { blockId: string }): void
  (e: 'fullscreen:exit'): void
}>()

interface DataPoint { timestamp: number; values?: number[] }

const chartEl = ref<HTMLDivElement | null>(null)
const isHistorical = ref(false)
const { enterFullscreen: fsEnter, exitFullscreen: fsExit } = useFullscreen()

const contextMenu = ref()
const contextMenuItems = [
  { label: 'Full Screen', icon: 'pi pi-window-maximize', command: () => enterFullscreen() },
]

function enterFullscreen() {
  fsEnter(props.blockId)
  emit('fullscreen:enter', { blockId: props.blockId })
}

function onEscapeKey(e: KeyboardEvent) {
  if (e.key === 'Escape') { fsExit(); emit('fullscreen:exit') }
}

const MAX_SAMPLES = props.bufferSamples ?? 500

// Ring buffer: timestamps and values
const tsRing: number[] = []
const valRing: number[] = []

let uplot: uPlot | null = null

function pushPoint(ts: number, v: number) {
  tsRing.push(ts / 1000) // uPlot expects seconds
  valRing.push(v)
  while (tsRing.length > MAX_SAMPLES) {
    tsRing.shift()
    valRing.shift()
  }
}

function redraw() {
  if (!uplot) return
  uplot.setData([new Float64Array(tsRing), new Float64Array(valRing)])
}

function onData(event: { data: { blockId: string; points: DataPoint[] } }) {
  if (event.data.blockId !== props.blockId) return
  for (const pt of event.data.points) {
    const replay = (pt as unknown as { replay?: boolean }).replay
    if (replay && !isHistorical.value) continue
    if (pt.values && pt.values.length > 0) {
      pushPoint(pt.timestamp, pt.values[0])
    }
  }
  redraw()
}

function onViewChanged(event: { data: { sessionId: string; mode: 'live' | 'historical' } }) {
  isHistorical.value = event.data.mode === 'historical'
  if (isHistorical.value) {
    tsRing.length = 0
    valRing.length = 0
    redraw()
  }
}

onMounted(() => {
  document.addEventListener('keydown', onEscapeKey)
  if (!chartEl.value) return

  const opts: uPlot.Options = {
    title: props.title ?? '',
    width: chartEl.value.clientWidth || 300,
    height: 180,
    series: [
      {},
      {
        label: props.yLabel ?? 'Value',
        stroke: '#3b82f6',
        width: 1.5,
        value: (_u, v) => (v == null ? '—' : v.toFixed(props.decimals ?? 2) + (props.unit ? ' ' + props.unit : '')),
      },
    ],
    axes: [
      { label: props.xLabel ?? 'Time' },
      { label: props.yLabel ?? '' },
    ],
    scales: {
      x: { time: true },
    },
  }

  uplot = new uPlot(opts, [new Float64Array(), new Float64Array()], chartEl.value)

  Events.On('pipeline:data', onData)
  Events.On('session:view-changed', onViewChanged)
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
  <div class="line-chart-block" @contextmenu.prevent="contextMenu?.show($event)">
    <ContextMenu ref="contextMenu" :model="contextMenuItems" />
    <div v-if="isHistorical" class="historical-badge">HISTORICAL</div>
    <div ref="chartEl" class="chart-container" />
  </div>
</template>

<style scoped>
.line-chart-block {
  background: #1e2d3d;
  border-radius: 8px;
  padding: 8px;
  position: relative;
}
.chart-container {
  width: 100%;
}
.historical-badge {
  position: absolute;
  top: 6px;
  right: 8px;
  font-size: 9px;
  color: #f59e0b;
  font-weight: bold;
  z-index: 1;
}
</style>
