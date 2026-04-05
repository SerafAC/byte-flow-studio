<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Events } from '@wailsio/runtime'
import Button from 'primevue/button'
import InputNumber from 'primevue/inputnumber'
import Checkbox from 'primevue/checkbox'
import { useSessionStore } from '../../stores/session'
import { usePipelineStore } from '../../stores/pipeline'
import * as wails from '../../services/wails'

const sessionStore = useSessionStore()
const pipelineStore = usePipelineStore()

const storeRaw = ref(true)
const storeProcessed = ref(true)
const maxSessions = ref(10)
const storageWarning = ref(false)
const storageEstimate = ref<wails.StorageEstimate | null>(null)

function formatTime(ms: number) {
  if (!ms) return '—'
  return new Date(ms).toLocaleString()
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

async function loadConfig() {
  try {
    const cfg = await wails.getSessionConfig()
    if (cfg) {
      storeRaw.value = cfg.storeRaw
      storeProcessed.value = cfg.storeProcessed
      maxSessions.value = cfg.maxSessions
    }
    storageEstimate.value = await wails.getStorageEstimate()
  } catch { /* ignore */ }
}

async function saveConfig() {
  try {
    await wails.updateSessionConfig({
      storeRaw: storeRaw.value,
      storeProcessed: storeProcessed.value,
      maxSessions: maxSessions.value,
    })
    storageEstimate.value = await wails.getStorageEstimate()
  } catch { /* ignore */ }
}

function onStorageWarning(event: { data: wails.StorageEstimate }) {
  storageWarning.value = true
  storageEstimate.value = event.data
}

async function onView(id: string) {
  await sessionStore.switchSession(id)
}

async function onDelete(id: string) {
  await sessionStore.deleteSession(id)
}

let offStorageWarning: (() => void) | null = null

onMounted(async () => {
  await loadConfig()
  offStorageWarning = Events.On('session:storage-warning', onStorageWarning)
})

onUnmounted(() => {
  offStorageWarning?.()
})
</script>

<template>
  <div class="session-panel">
    <div class="panel-title">Sessions</div>

    <!-- Storage warning -->
    <div v-if="storageWarning" class="storage-warning">
      Storage threshold exceeded! Consider reducing MaxSessions.
      <Button label="Dismiss" size="small" text @click="storageWarning = false" />
    </div>

    <!-- Recording config -->
    <div class="config-section">
      <div class="config-row">
        <label><Checkbox v-model="storeRaw" :binary="true" @change="saveConfig" /> Store Raw</label>
        <label><Checkbox v-model="storeProcessed" :binary="true" @change="saveConfig" /> Store Processed</label>
      </div>
      <div class="config-row">
        <span class="config-label">Max Sessions</span>
        <InputNumber v-model="maxSessions" :min="1" :max="100" :step="1" size="small" @blur="saveConfig" />
      </div>
    </div>

    <!-- Storage estimate -->
    <div v-if="storageEstimate" class="storage-section">
      <div class="storage-label">
        {{ formatBytes(storageEstimate.currentTotalBytes) }}
        / {{ formatBytes(storageEstimate.projectedMaxBytes) }}
        <span v-if="storageEstimate.exceedsThreshold" class="exceeds-badge">EXCEEDS LIMIT</span>
      </div>
      <div class="storage-bar-track">
        <div
          class="storage-bar-fill"
          :style="{ width: Math.min(100, (storageEstimate.currentTotalBytes / Math.max(1, storageEstimate.projectedMaxBytes)) * 100) + '%' }"
          :class="{ 'bar-danger': storageEstimate.exceedsThreshold }"
        />
      </div>
    </div>

    <!-- Session list -->
    <div v-if="!sessionStore.sessions.length" class="empty-state">
      No sessions yet. Start the pipeline to record data.
    </div>

    <div v-for="s in sessionStore.sessions" :key="s.id" class="session-row">
      <div class="session-time">{{ formatTime(s.startTime) }}</div>
      <div class="session-meta">
        <span :class="s.endTime ? 'tag-done' : 'tag-active'">
          {{ s.endTime ? s.endReason : 'Active' }}
        </span>
        <span class="session-size">{{ formatBytes(s.dataVolume) }}</span>
      </div>
      <div class="session-actions">
        <Button
          v-if="s.endTime"
          label="View"
          size="small"
          severity="info"
          :disabled="pipelineStore.flowState === 'running'"
          @click="onView(s.id)"
        />
        <Button
          v-if="s.endTime"
          label="Delete"
          size="small"
          severity="danger"
          @click="onDelete(s.id)"
        />
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
@use '../../assets/variables' as *;
@use '../../assets/mixins' as *;

.session-panel {
  padding: $space-lg;
  font-size: $font-size-base;
  font-family: $font-body;
  color: $text-primary;
  background: $glass-bg;
  backdrop-filter: blur($glass-blur);
  -webkit-backdrop-filter: blur($glass-blur);
  box-shadow: $glass-shadow;
  border-radius: $radius-lg;
  border: 1px solid $ghost-border;
  overflow-y: auto;
  box-sizing: border-box;

  &::-webkit-scrollbar { width: 4px; }
  &::-webkit-scrollbar-track { background: transparent; }
  &::-webkit-scrollbar-thumb { background: $border-color; border-radius: 2px; }
}

.panel-title {
  font-family: $font-display;
  font-weight: 600;
  font-size: $font-size-lg;
  margin-bottom: $space-lg;
  color: $text-primary;
}

.empty-state { color: $text-muted; font-style: italic; font-size: $font-size-sm; }

.storage-warning {
  background: rgba($color-warning, 0.15);
  color: $color-warning;
  border: 1px solid rgba($color-warning, 0.3);
  padding: $space-sm $space-md;
  border-radius: $radius-md;
  margin-bottom: $space-md;
  font-size: $font-size-sm;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.config-section { margin-bottom: $space-lg; }
.config-row {
  display: flex;
  gap: $space-lg;
  align-items: center;
  margin-bottom: $space-sm;
  font-size: $font-size-sm;
  flex-wrap: wrap;
}
.config-label { color: $text-secondary; }

.storage-section { margin-bottom: $space-lg; }
.storage-label {
  font-size: $font-size-xs;
  color: $text-secondary;
  margin-bottom: $space-xs;
  display: flex;
  gap: $space-sm;
  align-items: center;
}
.exceeds-badge {
  font-size: $font-size-xs;
  color: $color-danger;
  font-weight: bold;
  background: rgba($color-danger, 0.15);
  padding: 1px 4px;
  border-radius: $radius-sm;
}
.storage-bar-track {
  height: 4px;
  background: $bg-block;
  border-radius: 2px;
  overflow: hidden;
}
.storage-bar-fill {
  height: 100%;
  background: $color-primary;
  transition: width 0.3s;
}
.bar-danger { background: $color-danger; }

.session-row {
  border-bottom: 1px solid $ghost-border;
  padding: $space-sm 0;
  display: flex;
  flex-direction: column;
  gap: $space-xs;
}
.session-time { color: $text-primary; font-size: $font-size-sm; }
.session-meta { display: flex; gap: $space-sm; align-items: center; }
.tag-done {
  @include tag;
}
.tag-active {
  @include tag($bg: rgba(73, 179, 147, 0.15), $color: $color-success);
}
.session-size { color: $text-muted; font-size: $font-size-xs; margin-left: auto; }
.session-actions { display: flex; gap: $space-xs; }
</style>
