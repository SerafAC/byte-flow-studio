<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { Events } from '@wailsio/runtime'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import ContextMenu from 'primevue/contextmenu'
import { useFullscreen } from '../../../composables/useFullscreen'

const props = defineProps<{
  blockId: string
  title?: string
  yLabel?: string
  unit?: string
  decimals?: number
  bufferSamples?: number
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

const MAX_SAMPLES = props.bufferSamples ?? 50

const latestValues = ref<number[]>([])

let uplot: uPlot | null = null
let ro: ResizeObserver | null = null
let currentW = 400
let currentH = 300

const tsRing: number[] = []
const valRings: number[][] = []

function buildSeries(numChannels: number): uPlot.Series[] {
  const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#06b6d4']
  const series: uPlot.Series[] = [{}]
  for (let i = 0; i < numChannels; i++) {
    series.push({ label: `Ch ${i + 1}`, stroke: colors[i % colors.length], width: 2 })
  }
  return series
}

function pushPoint(ts: number, values: number[]) {
  if (valRings.length < values.length) {
    for (let i = valRings.length; i < values.length; i++) valRings.push([])
  }
  tsRing.push(ts / 1000)
  values.forEach((v, i) => valRings[i].push(v))
  while (tsRing.length > MAX_SAMPLES) {
    tsRing.shift()
    valRings.forEach(r => r.shift())
  }
  latestValues.value = values
}

function redraw() {
  if (!uplot || valRings.length === 0) return
  const data: uPlot.AlignedData = [new Float64Array(tsRing)]
  valRings.forEach(r => data.push(new Float64Array(r)))
  uplot.setData(data)
}

function rebuildChart(numChannels: number) {
  uplot?.destroy()
  if (!chartEl.value) return
  const opts: uPlot.Options = {
    title: props.title ?? '',
    width: currentW,
    height: currentH,
    series: buildSeries(numChannels),
    axes: [{ label: 'Time' }, { label: props.yLabel ?? '' }],
    scales: { x: { time: true } },
  }
  const data: uPlot.AlignedData = [new Float64Array()]
  for (let i = 0; i < numChannels; i++) data.push(new Float64Array())
  uplot = new uPlot(opts, data, chartEl.value)
}

function onData(event: { data: { blockId: string; points: DataPoint[] } }) {
  if (event.data.blockId !== props.blockId) return
  for (const pt of event.data.points) {
    const replay = (pt as unknown as { replay?: boolean }).replay
    if (replay && !isHistorical.value) continue
    if (pt.values && pt.values.length > 0) {
      if (!uplot || uplot.series.length - 1 !== pt.values.length) {
        rebuildChart(pt.values.length)
      }
      pushPoint(pt.timestamp, pt.values)
    }
  }
  redraw()
}

function onViewChanged(event: { data: { sessionId: string; mode: 'live' | 'historical' } }) {
  isHistorical.value = event.data.mode === 'historical'
  if (isHistorical.value) {
    tsRing.length = 0
    valRings.length = 0
    latestValues.value = []
    redraw()
  }
}

onMounted(async () => {
  Events.On('pipeline:data', onData)
  Events.On('session:view-changed', onViewChanged)
  document.addEventListener('keydown', onEscapeKey)

  await nextTick()
  if (!chartEl.value) return

  currentW = chartEl.value.clientWidth || 400
  currentH = chartEl.value.clientHeight || 300

  ro = new ResizeObserver(entries => {
    for (const entry of entries) {
      const { width, height } = entry.contentRect
      if (width > 0 && height > 0) {
        currentW = width
        currentH = height
        uplot?.setSize({ width, height })
      }
    }
  })
  ro.observe(chartEl.value)
})

onUnmounted(() => {
  Events.Off('pipeline:data', onData)
  Events.Off('session:view-changed', onViewChanged)
  document.removeEventListener('keydown', onEscapeKey)
  ro?.disconnect()
  ro = null
  uplot?.destroy()
  uplot = null
})
</script>

<template>
  <div class="bar-chart-block" @contextmenu.prevent="contextMenu?.show($event)">
    <ContextMenu ref="contextMenu" :model="contextMenuItems" />
    <div v-if="isHistorical" class="historical-badge">HISTORICAL</div>
    <div ref="chartEl" class="chart-container" />
    <div v-if="latestValues.length > 0" class="bar-latest">
      <div v-for="(v, i) in latestValues" :key="i" class="bar-item">
        <span class="bar-label">Ch {{ i + 1 }}</span>
        <span class="bar-value">{{ v.toFixed(decimals ?? 2) }}{{ unit ? ' ' + unit : '' }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.bar-chart-block {
  background: #1e2d3d;
  border-radius: 8px;
  padding: 8px;
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}
.chart-container {
  flex: 1;
  min-height: 0;
  min-width: 0;
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
.bar-latest {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 6px;
  font-family: monospace;
  font-size: 12px;
  flex-shrink: 0;
}
.bar-label { color: #9ca3af; margin-right: 4px; }
.bar-value { color: #e2e8f0; font-weight: 600; }
</style>
