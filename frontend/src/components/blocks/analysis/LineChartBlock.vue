<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import ContextMenu from 'primevue/contextmenu'
import { useAnalysisBlock } from '../../../composables/useAnalysisBlock'

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

const MAX_SAMPLES = props.bufferSamples ?? 500

const tsRing: number[] = []
const valRing: number[] = []

let uplot: uPlot | null = null
let ro: ResizeObserver | null = null

function pushPoint(ts: number, v: number) {
  tsRing.push(ts / 1000)
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

const { isHistorical, contextMenu, contextMenuItems } = useAnalysisBlock({
  blockId: props.blockId,
  onData,
  onHistoricalMode: () => { tsRing.length = 0; valRing.length = 0; redraw() },
  emit,
})

function buildOpts(width: number, height: number): uPlot.Options {
  return {
    title: props.title ?? '',
    width,
    height,
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
}

onMounted(async () => {
  // Wait for layout so clientWidth/clientHeight are accurate
  await nextTick()
  if (!chartEl.value) return

  const w = chartEl.value.clientWidth || 400
  const h = chartEl.value.clientHeight || 300

  uplot = new uPlot(buildOpts(w, h), [new Float64Array(), new Float64Array()], chartEl.value)

  // Resize uPlot whenever the container changes size
  ro = new ResizeObserver(entries => {
    for (const entry of entries) {
      const { width, height } = entry.contentRect
      if (width > 0 && height > 0) {
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
  /* Fill the flex parent (.fullscreen-block has flex:1; min-height:0) */
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
</style>
