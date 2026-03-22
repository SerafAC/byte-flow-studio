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

onMounted(async () => {
  await loadConfig()
  Events.On('session:storage-warning', onStorageWarning)
})

onUnmounted(() => {
  Events.Off('session:storage-warning', onStorageWarning)
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

<style scoped>
.session-panel { padding: 10px; font-size: 12px; }
.panel-title { font-weight: bold; font-size: 14px; margin-bottom: 10px; }
.empty-state { color: #6b7280; font-style: italic; font-size: 11px; }

.storage-warning {
  background: #7c2d12; color: #fed7aa;
  padding: 6px 8px; border-radius: 4px; margin-bottom: 8px;
  font-size: 11px; display: flex; align-items: center; justify-content: space-between;
}

.config-section { margin-bottom: 10px; }
.config-row {
  display: flex; gap: 10px; align-items: center;
  margin-bottom: 6px; font-size: 11px; flex-wrap: wrap;
}
.config-label { color: #9ca3af; }

.storage-section { margin-bottom: 10px; }
.storage-label {
  font-size: 10px; color: #9ca3af; margin-bottom: 3px;
  display: flex; gap: 6px; align-items: center;
}
.exceeds-badge {
  font-size: 9px; color: #ef4444; font-weight: bold;
  background: #450a0a; padding: 1px 4px; border-radius: 3px;
}
.storage-bar-track {
  height: 4px; background: #374151; border-radius: 2px; overflow: hidden;
}
.storage-bar-fill {
  height: 100%; background: #3b82f6; transition: width 0.3s;
}
.bar-danger { background: #ef4444; }

.session-row {
  border-bottom: 1px solid #3a4a5c; padding: 6px 0;
  display: flex; flex-direction: column; gap: 4px;
}
.session-time { color: #d1d5db; font-size: 11px; }
.session-meta { display: flex; gap: 6px; align-items: center; }
.tag-done {
  font-size: 10px; background: #374151; padding: 1px 5px; border-radius: 4px; color: #9ca3af;
}
.tag-active {
  font-size: 10px; background: #064e3b; padding: 1px 5px; border-radius: 4px; color: #34d399;
}
.session-size { color: #9ca3af; font-size: 10px; margin-left: auto; }
.session-actions { display: flex; gap: 4px; }
</style>
