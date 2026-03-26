<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
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
  fftWindowSize?: number
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
let ro: ResizeObserver | null = null
let lastBinCount = 0
// Track current container size for rebuilds (e.g. log scale toggle)
let currentW = 400
let currentH = 300

// Internal FFT: accumulate single-float samples when no FFT processing block is used.
// Must be a power of 2; defaults to 512.
function nextPow2(n: number): number { let p = 1; while (p < n) p <<= 1; return p }
const FFT_WIN = nextPow2(props.fftWindowSize ?? 512)
const sampleBuf: number[] = []

/** Compute FFT magnitudes for the positive-frequency bins (N/2 bins) using an
 *  iterative Cooley-Tukey radix-2 FFT with a Hann window applied to the samples. */
function computeFFTMagnitudes(samples: number[]): number[] {
  const n = samples.length
  const re = new Float64Array(n)
  const im = new Float64Array(n)

  // Apply Hann window
  for (let i = 0; i < n; i++) {
    re[i] = samples[i] * 0.5 * (1 - Math.cos(2 * Math.PI * i / (n - 1)))
  }

  // Bit-reversal permutation
  let j = 0
  for (let i = 1; i < n; i++) {
    let bit = n >> 1
    for (; j & bit; bit >>= 1) j ^= bit
    j ^= bit
    if (i < j) {
      let t = re[i]; re[i] = re[j]; re[j] = t
      t = im[i]; im[i] = im[j]; im[j] = t
    }
  }

  // Butterfly passes
  for (let len = 2; len <= n; len <<= 1) {
    const halfLen = len >> 1
    const ang = -Math.PI / halfLen
    const wRe = Math.cos(ang), wIm = Math.sin(ang)
    for (let i = 0; i < n; i += len) {
      let curRe = 1, curIm = 0
      for (let k = 0; k < halfLen; k++) {
        const uRe = re[i + k], uIm = im[i + k]
        const vRe = re[i + k + halfLen] * curRe - im[i + k + halfLen] * curIm
        const vIm = re[i + k + halfLen] * curIm + im[i + k + halfLen] * curRe
        re[i + k] = uRe + vRe
        im[i + k] = uIm + vIm
        re[i + k + halfLen] = uRe - vRe
        im[i + k + halfLen] = uIm - vIm
        const newCurRe = curRe * wRe - curIm * wIm
        curIm = curRe * wIm + curIm * wRe
        curRe = newCurRe
      }
    }
  }

  const halfN = n >> 1
  const mags: number[] = new Array(halfN)
  for (let i = 0; i < halfN; i++) {
    mags[i] = Math.sqrt(re[i] * re[i] + im[i] * im[i])
  }
  return mags
}

function buildOpts(binCount: number, width: number, height: number): uPlot.Options {
  return {
    title: '',
    width,
    height,
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
      x: { time: false, range: [0, Math.max(binCount - 1, 1)] },
      y: logScale.value ? { distr: 3, log: 10 } : {},
    },
  }
}

function renderSpectrum(values: number[]) {
  if (!chartEl.value) return
  // Read current container dimensions directly to avoid stale cached values on first render
  const w = chartEl.value.clientWidth || currentW
  const h = chartEl.value.clientHeight || currentH
  if (!uplot || lastBinCount !== values.length) {
    uplot?.destroy()
    lastBinCount = values.length
    const bins = Array.from({ length: values.length }, (_, i) => i)
    uplot = new uPlot(
      buildOpts(values.length, w, h),
      [new Float64Array(bins), new Float64Array(values)],
      chartEl.value,
    )
    currentW = w
    currentH = h
  } else {
    const bins = new Float64Array(values.length).map((_, i) => i)
    uplot.setData([bins, new Float64Array(values)])
  }
}

function onData(event: { data: { blockId: string; points: DataPoint[] } }) {
  if (event.data.blockId !== props.blockId) return
  for (const pt of event.data.points) {
    const replay = (pt as unknown as { replay?: boolean }).replay
    if (replay && !isHistorical.value) continue
    if (!pt.values || pt.values.length === 0) continue

    if (pt.values.length === 1) {
      // Single-value input: buffer samples and compute FFT internally.
      // This allows the FFT spectrum block to work without a separate FFT processing block.
      sampleBuf.push(pt.values[0])
      if (sampleBuf.length > FFT_WIN) sampleBuf.shift()
      if (sampleBuf.length === FFT_WIN) {
        renderSpectrum(computeFFTMagnitudes(sampleBuf))
      }
    } else {
      // Multi-value input: pre-computed spectrum (e.g. from FFT processing block) — render directly.
      renderSpectrum(pt.values)
    }
  }
}

function onViewChanged(event: { data: { sessionId: string; mode: 'live' | 'historical' } }) {
  isHistorical.value = event.data.mode === 'historical'
  sampleBuf.length = 0
  uplot?.destroy()
  uplot = null
  lastBinCount = 0
  if (!isHistorical.value) {
    nextTick(() => initEmptyChart())
  }
}

function onLogScaleToggle() {
  uplot?.destroy()
  uplot = null
  lastBinCount = 0
  // Re-render immediately with whatever data we already have
  if (sampleBuf.length === FFT_WIN) {
    renderSpectrum(computeFFTMagnitudes(sampleBuf))
  } else {
    nextTick(() => initEmptyChart())
  }
}

function initEmptyChart() {
  if (!chartEl.value || uplot) return
  const w = chartEl.value.clientWidth || currentW
  const h = chartEl.value.clientHeight || currentH
  const emptyBins = 1
  uplot = new uPlot(
    buildOpts(emptyBins, w, h),
    [new Float64Array([0]), new Float64Array([0])],
    chartEl.value,
  )
  lastBinCount = emptyBins
}

onMounted(async () => {
  Events.On('pipeline:data', onData)
  Events.On('session:view-changed', onViewChanged)
  document.addEventListener('keydown', onEscapeKey)

  // Wait for layout so container dimensions are accurate
  await nextTick()
  if (!chartEl.value) return

  currentW = chartEl.value.clientWidth || 400
  currentH = chartEl.value.clientHeight || 300

  initEmptyChart()

  // Resize uPlot whenever the container changes size
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
  <div class="fft-spectrum-block" @contextmenu.prevent="contextMenu?.show($event)">
    <ContextMenu ref="contextMenu" :model="contextMenuItems" />
    <div v-if="isHistorical" class="historical-badge">HISTORICAL</div>
    <div class="toolbar">
      <label class="log-toggle">
        <input v-model="logScale" type="checkbox" @change="onLogScaleToggle" />
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
  /* Fill the flex parent (.fullscreen-block has flex:1; min-height:0) */
  height: 100%;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  color: #9ca3af;
  margin-bottom: 4px;
  flex-shrink: 0;
}
.log-toggle { display: flex; align-items: center; gap: 4px; cursor: pointer; }
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
