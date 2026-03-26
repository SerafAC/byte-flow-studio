<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Events } from '@wailsio/runtime'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import ContextMenu from 'primevue/contextmenu'
import { useFullscreen } from '../../../composables/useFullscreen'

const props = defineProps<{
  blockId: string
  maxRows?: number
  decimals?: number
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

interface TableRow { timestamp: string; values: string[] }
interface DataPoint { timestamp: number; values?: number[] }

const MAX_ROWS = props.maxRows ?? 200
const rows = ref<TableRow[]>([])
const isHistorical = ref(false)
const numChannels = ref(1)

function formatTs(ms: number): string {
  return new Date(ms).toISOString().replace('T', ' ').slice(0, 23)
}

function onData(event: { data: { blockId: string; points: DataPoint[] } }) {
  if (event.data.blockId !== props.blockId) return
  for (const pt of event.data.points) {
    const replay = (pt as unknown as { replay?: boolean }).replay
    if (replay && !isHistorical.value) continue
    if (pt.values && pt.values.length > 0) {
      if (pt.values.length > numChannels.value) numChannels.value = pt.values.length
      rows.value.push({
        timestamp: formatTs(pt.timestamp),
        values: pt.values.map(v => v.toFixed(props.decimals ?? 2)),
      })
      while (rows.value.length > MAX_ROWS) rows.value.shift()
    }
  }
}

function onViewChanged(event: { data: { sessionId: string; mode: 'live' | 'historical' } }) {
  isHistorical.value = event.data.mode === 'historical'
  if (isHistorical.value) {
    rows.value = []
    numChannels.value = 1
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
})
</script>

<template>
  <div class="data-table-block" @contextmenu.prevent="contextMenu?.show($event)">
    <ContextMenu ref="contextMenu" :model="contextMenuItems" />
    <div v-if="isHistorical" class="historical-badge">HISTORICAL</div>
    <DataTable
      :value="rows"
      scroll-height="flex"
      scrollable
      class="p-datatable-sm dt-fill"
    >
      <Column field="timestamp" header="Timestamp" style="min-width: 160px; font-family: monospace; font-size: 11px;" />
      <Column
        v-for="ch in numChannels"
        :key="ch"
        :field="`values[${ch - 1}]`"
        :header="`Ch ${ch}`"
        style="font-family: monospace; font-size: 12px; text-align: right;"
      />
    </DataTable>
  </div>
</template>

<style scoped>
.data-table-block {
  background: #1e2d3d;
  border-radius: 8px;
  padding: 8px;
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}
:deep(.dt-fill) {
  flex: 1;
  min-height: 0;
}
:deep(.dt-fill .p-datatable-wrapper) {
  flex: 1;
  min-height: 0;
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
