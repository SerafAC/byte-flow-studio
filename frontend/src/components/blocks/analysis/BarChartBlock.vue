<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import ContextMenu from 'primevue/contextmenu'
import { useAnalysisBlock } from '../../../composables/useAnalysisBlock'

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

interface DataPoint { timestamp: number; values?: number[] }

const chartEl = ref<HTMLDivElement | null>(null)

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

const { isHistorical, contextMenu, contextMenuItems } = useAnalysisBlock({
  blockId: props.blockId,
  onData,
  onHistoricalMode: () => { tsRing.length = 0; valRings.length = 0; latestValues.value = []; redraw() },
  emit,
})

onMounted(async () => {
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

<style lang="scss" scoped>
@use '../../../assets/variables' as *;
@use '../../../assets/mixins' as *;

.bar-chart-block { @include analysis-block; }
.chart-container  { @include chart-fill; }
.historical-badge { @include historical-badge; }

.bar-latest {
  display: flex;
  gap: $space-lg;
  flex-wrap: wrap;
  margin-top: $space-sm;
  font-family: $font-mono;
  font-size: $font-size-md;
  flex-shrink: 0;
}
.bar-label { color: $text-secondary; margin-right: $space-xs; }
.bar-value { color: $text-primary; font-weight: 600; }
</style>
