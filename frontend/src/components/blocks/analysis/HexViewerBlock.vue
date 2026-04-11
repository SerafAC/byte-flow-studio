<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import ContextMenu from 'primevue/contextmenu'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { useAnalysisBlock } from '../../../composables/useAnalysisBlock'

const props = defineProps<{
  blockId: string
  maxBytes?: number
  bytesPerRow?: number
}>()

const emit = defineEmits<{
  (e: 'fullscreen:enter', payload: { blockId: string }): void
  (e: 'fullscreen:exit'): void
}>()

interface RawDataPoint { timestamp: number; raw?: number[]; mode?: string }

const containerEl = ref<HTMLDivElement | null>(null)
const scrollEl = ref<HTMLDivElement | null>(null)
const paused = ref(false)
const searchPattern = ref('')
const searchMatches = ref<number[]>([])

const BYTES_PER_ROW = props.bytesPerRow ?? 16
const MAX_BYTES = props.maxBytes ?? 65536
const ROW_HEIGHT = 18

// Ring buffer of raw bytes
const byteBuffer = ref<number[]>([])

function onData(event: { data: { blockId: string; points: RawDataPoint[] } }) {
  if (event.data.blockId !== props.blockId) return

  for (const point of event.data.points) {
    if (!point.raw || point.raw.length === 0) continue
    byteBuffer.value.push(...point.raw)
    // Enforce max buffer
    while (byteBuffer.value.length > MAX_BYTES) {
      byteBuffer.value.shift()
    }
  }

  if (!paused.value && scrollEl.value) {
    nextTick(() => {
      if (scrollEl.value) {
        scrollEl.value.scrollTop = scrollEl.value.scrollHeight
      }
    })
  }
}

function onHistoricalMode() {
  byteBuffer.value = []
}

const { contextMenu, contextMenuItems } = useAnalysisBlock({
  blockId: props.blockId,
  onData,
  onHistoricalMode,
  emit,
})

const rows = computed(() => {
  const data = byteBuffer.value
  const result: { offset: number; hex: string[]; ascii: string }[] = []
  for (let i = 0; i < data.length; i += BYTES_PER_ROW) {
    const slice = data.slice(i, i + BYTES_PER_ROW)
    const hex = slice.map(b => b.toString(16).padStart(2, '0').toUpperCase())
    const ascii = slice.map(b => (b >= 0x20 && b <= 0x7e) ? String.fromCharCode(b) : '.').join('')
    result.push({ offset: i, hex, ascii })
  }
  return result
})

// Visible rows for virtual scrolling
const scrollTop = ref(0)
const containerHeight = ref(300)

const visibleRows = computed(() => {
  const startIdx = Math.floor(scrollTop.value / ROW_HEIGHT)
  const endIdx = Math.min(rows.value.length, startIdx + Math.ceil(containerHeight.value / ROW_HEIGHT) + 2)
  return rows.value.slice(startIdx, endIdx).map((row, i) => ({ ...row, index: startIdx + i }))
})

const totalHeight = computed(() => rows.value.length * ROW_HEIGHT)

function onScroll(e: Event) {
  const el = e.target as HTMLDivElement
  scrollTop.value = el.scrollTop
}

function togglePause() {
  paused.value = !paused.value
}

function doSearch() {
  const pattern = searchPattern.value.trim()
  if (!pattern) {
    searchMatches.value = []
    return
  }
  // Parse hex pattern (e.g., "CA FE" or "CAFE")
  const hexStr = pattern.replace(/\s+/g, '')
  if (hexStr.length % 2 !== 0 || !/^[0-9A-Fa-f]+$/.test(hexStr)) {
    searchMatches.value = []
    return
  }
  const patternBytes: number[] = []
  for (let i = 0; i < hexStr.length; i += 2) {
    patternBytes.push(parseInt(hexStr.slice(i, i + 2), 16))
  }

  const data = byteBuffer.value
  const matches: number[] = []
  for (let i = 0; i <= data.length - patternBytes.length; i++) {
    let match = true
    for (let j = 0; j < patternBytes.length; j++) {
      if (data[i + j] !== patternBytes[j]) {
        match = false
        break
      }
    }
    if (match) matches.push(i)
  }
  searchMatches.value = matches
}

function isHighlighted(byteOffset: number): boolean {
  const pattern = searchPattern.value.trim().replace(/\s+/g, '')
  const patternLen = pattern.length / 2
  for (const m of searchMatches.value) {
    if (byteOffset >= m && byteOffset < m + patternLen) return true
  }
  return false
}

let ro: ResizeObserver | null = null

onMounted(() => {
  if (containerEl.value) {
    containerHeight.value = containerEl.value.clientHeight
    ro = new ResizeObserver(() => {
      if (containerEl.value) containerHeight.value = containerEl.value.clientHeight
    })
    ro.observe(containerEl.value)
  }
})

onUnmounted(() => {
  ro?.disconnect()
})
</script>

<template>
  <div class="hex-viewer" ref="containerEl" @contextmenu.prevent="contextMenu?.show($event)">
    <div class="hv-toolbar">
      <Button
        :icon="paused ? 'pi pi-play' : 'pi pi-pause'"
        size="small"
        severity="secondary"
        @click="togglePause"
        :title="paused ? 'Resume' : 'Pause'"
      />
      <InputText
        v-model="searchPattern"
        placeholder="Search hex (e.g. CA FE)"
        class="hv-search-input"
        @keydown.enter="doSearch"
      />
      <Button icon="pi pi-search" size="small" severity="secondary" @click="doSearch" />
    </div>
    <div class="hv-scroll" ref="scrollEl" @scroll="onScroll">
      <div class="hv-virtual-spacer" :style="{ height: totalHeight + 'px' }">
        <div
          v-for="row in visibleRows"
          :key="row.index"
          class="hv-row"
          :style="{ top: row.index * ROW_HEIGHT + 'px' }"
        >
          <span class="hv-offset">{{ row.offset.toString(16).padStart(8, '0').toUpperCase() }}</span>
          <span class="hv-hex">
            <span
              v-for="(b, j) in row.hex"
              :key="j"
              :class="{ 'hv-match': isHighlighted(row.offset + j) }"
            >{{ b }} </span>
          </span>
          <span class="hv-ascii">{{ row.ascii }}</span>
        </div>
      </div>
    </div>
    <ContextMenu ref="contextMenu" :model="contextMenuItems" />
  </div>
</template>

<style scoped lang="scss">
.hex-viewer {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  font-family: 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
  font-size: 11px;
  overflow: hidden;
}

.hv-toolbar {
  display: flex;
  gap: 4px;
  padding: 4px;
  align-items: center;
  flex-shrink: 0;
}

.hv-search-input {
  flex: 1;
  font-size: 11px;
  min-width: 100px;
}

.hv-scroll {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;
}

.hv-virtual-spacer {
  position: relative;
  width: 100%;
}

.hv-row {
  position: absolute;
  left: 0;
  right: 0;
  height: 18px;
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 0 4px;
  white-space: nowrap;
}

.hv-offset {
  color: #6b7280;
  min-width: 64px;
}

.hv-hex {
  color: #d1d5db;
  flex: 1;
}

.hv-ascii {
  color: #9ca3af;
  min-width: 0;
}

.hv-match {
  background: rgba(251, 191, 36, 0.3);
  color: #fbbf24;
  border-radius: 2px;
}
</style>
