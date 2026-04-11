<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import ContextMenu from 'primevue/contextmenu'
import Select from 'primevue/select'
import { useAnalysisBlock } from '../../../composables/useAnalysisBlock'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  colorMap?: string
  minFreqHz?: number
  maxFreqHz?: number
  minAmplitude?: number
  maxAmplitude?: number
}>()

const emit = defineEmits<{
  (e: 'fullscreen:enter', payload: { blockId: string }): void
  (e: 'fullscreen:exit'): void
}>()

interface DataPoint { timestamp: number; values?: number[] }

const canvasEl = ref<HTMLCanvasElement | null>(null)
const containerEl = ref<HTMLDivElement | null>(null)
let ro: ResizeObserver | null = null

const selectedColorMap = ref(props.colorMap ?? 'viridis')
const colorMapOptions = [
  { label: 'Viridis', value: 'viridis' },
  { label: 'Magma', value: 'magma' },
  { label: 'Inferno', value: 'inferno' },
  { label: 'Plasma', value: 'plasma' },
  { label: 'Grayscale', value: 'grayscale' },
]

const { save } = useBlockConfig(props.blockId, () => ({
  colorMap: selectedColorMap.value,
  minFreqHz: props.minFreqHz ?? 0,
  maxFreqHz: props.maxFreqHz ?? 0,
  minAmplitude: props.minAmplitude ?? 0,
  maxAmplitude: props.maxAmplitude ?? 0,
}))

watch(selectedColorMap, () => { save() })

// Color map lookup tables (256 entries each): [r, g, b]
const colorMaps: Record<string, (t: number) => [number, number, number]> = {
  viridis: (t) => {
    const r = Math.round(68 + t * (253 - 68))
    const g = Math.round(1 + t * (231 - 1))
    const b = Math.round(84 + (t < 0.5 ? t * 2 * (170 - 84) : (170 + (t - 0.5) * 2 * (37 - 170))))
    return [Math.max(0, Math.min(255, r)), Math.max(0, Math.min(255, g)), Math.max(0, Math.min(255, b))]
  },
  magma: (t) => {
    const r = Math.round(t < 0.5 ? t * 2 * 180 : 180 + (t - 0.5) * 2 * 75)
    const g = Math.round(t < 0.3 ? t * 10 : t < 0.7 ? 3 + (t - 0.3) * 300 : 123 + (t - 0.7) * 440)
    const b = Math.round(4 + t * 100 + (t > 0.5 ? (t - 0.5) * 300 : 0))
    return [Math.max(0, Math.min(255, r)), Math.max(0, Math.min(255, g)), Math.max(0, Math.min(255, b))]
  },
  inferno: (t) => {
    const r = Math.round(t < 0.5 ? t * 2 * 210 : 210 + (t - 0.5) * 2 * 45)
    const g = Math.round(t < 0.4 ? t * 2.5 * 20 : 20 + (t - 0.4) * 392)
    const b = Math.round(t < 0.3 ? 4 + t * 3.33 * 140 : 144 - (t - 0.3) * 206)
    return [Math.max(0, Math.min(255, r)), Math.max(0, Math.min(255, g)), Math.max(0, Math.min(255, b))]
  },
  plasma: (t) => {
    const r = Math.round(13 + t * 227)
    const g = Math.round(t < 0.5 ? 8 + t * 2 * 130 : 138 - (t - 0.5) * 2 * 90 + (t - 0.5) * 2 * 200)
    const b = Math.round(t < 0.5 ? 135 + t * 2 * 120 : 255 - (t - 0.5) * 2 * 255)
    return [Math.max(0, Math.min(255, r)), Math.max(0, Math.min(255, g)), Math.max(0, Math.min(255, b))]
  },
  grayscale: (t) => {
    const v = Math.round(t * 255)
    return [v, v, v]
  },
}

let minAmp = props.minAmplitude ?? 0
let maxAmp = props.maxAmplitude ?? 0
let autoRange = maxAmp <= minAmp

function onData(event: { data: { blockId: string; points: DataPoint[] } }) {
  if (event.data.blockId !== props.blockId) return
  const canvas = canvasEl.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  for (const point of event.data.points) {
    if (!point.values || point.values.length === 0) continue
    const mags = point.values
    const w = canvas.width
    const h = canvas.height

    // Auto-range amplitude
    if (autoRange) {
      for (const v of mags) {
        if (v > maxAmp) maxAmp = v
      }
      if (maxAmp <= 0) maxAmp = 1
    }

    // Scroll: copy existing content up by 1 pixel
    ctx.drawImage(canvas, 0, 1, w, h - 1, 0, 0, w, h - 1)

    // Draw new row at bottom
    const imgData = ctx.createImageData(w, 1)
    const colorFn = colorMaps[selectedColorMap.value] ?? colorMaps.viridis
    for (let x = 0; x < w; x++) {
      const binIdx = Math.floor((x / w) * mags.length)
      const amplitude = mags[Math.min(binIdx, mags.length - 1)]
      const normalized = Math.max(0, Math.min(1, (amplitude - minAmp) / (maxAmp - minAmp || 1)))
      const [r, g, b] = colorFn(normalized)
      const i = x * 4
      imgData.data[i] = r
      imgData.data[i + 1] = g
      imgData.data[i + 2] = b
      imgData.data[i + 3] = 255
    }
    ctx.putImageData(imgData, 0, h - 1)
  }
}

function onHistoricalMode() {
  // Clear canvas for historical replay
  const canvas = canvasEl.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  ctx?.clearRect(0, 0, canvas.width, canvas.height)
}

const { contextMenu, contextMenuItems } = useAnalysisBlock({
  blockId: props.blockId,
  onData,
  onHistoricalMode,
  emit,
})

function resizeCanvas() {
  const canvas = canvasEl.value
  const container = containerEl.value
  if (!canvas || !container) return
  canvas.width = container.clientWidth
  canvas.height = container.clientHeight
}

onMounted(async () => {
  await nextTick()
  resizeCanvas()
  ro = new ResizeObserver(resizeCanvas)
  if (containerEl.value) ro.observe(containerEl.value)
})

onUnmounted(() => {
  ro?.disconnect()
})
</script>

<template>
  <div class="spectrum-viewer" ref="containerEl" @contextmenu.prevent="contextMenu?.show($event)">
    <canvas ref="canvasEl" class="waterfall-canvas" />
    <div class="sv-controls">
      <Select
        v-model="selectedColorMap"
        :options="colorMapOptions"
        option-label="label"
        option-value="value"
        class="colormap-select"
      />
    </div>
    <ContextMenu ref="contextMenu" :model="contextMenuItems" />
  </div>
</template>

<style scoped lang="scss">
.spectrum-viewer {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

.waterfall-canvas {
  flex: 1;
  width: 100%;
  min-height: 0;
}

.sv-controls {
  position: absolute;
  top: 4px;
  right: 4px;
  z-index: 2;
}

.colormap-select {
  font-size: 11px;
  min-width: 90px;
}
</style>
