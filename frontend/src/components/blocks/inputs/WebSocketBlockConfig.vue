<script setup lang="ts">
import { ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import { updateBlockParams } from '../../../services/wails'

const props = defineProps<{
  blockId: string
  params: Record<string, unknown>
}>()

const url = ref<string>((props.params.url as string) ?? 'ws://localhost:8080/data')
const subprotocol = ref<string>((props.params.subprotocol as string) ?? '')
const reconnectIntervalMs = ref<number>((props.params.reconnectIntervalMs as number) ?? 2000)

const urlError = ref('')

function validateUrl(v: string): boolean {
  return v.startsWith('ws://') || v.startsWith('wss://')
}

async function save() {
  if (!validateUrl(url.value)) {
    urlError.value = 'URL must start with ws:// or wss://'
    return
  }
  urlError.value = ''
  await updateBlockParams(props.blockId, {
    url: url.value,
    subprotocol: subprotocol.value,
    reconnectIntervalMs: reconnectIntervalMs.value,
  })
}
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>URL</label>
      <InputText v-model="url" placeholder="ws://host:port/path" @blur="save" />
      <span v-if="urlError" class="error">{{ urlError }}</span>
    </div>
    <div class="field">
      <label>Subprotocol</label>
      <InputText v-model="subprotocol" placeholder="Optional" @blur="save" />
    </div>
    <div class="field">
      <label>Reconnect Interval (ms)</label>
      <InputNumber v-model="reconnectIntervalMs" :min="100" :max="30000" :step="100" @blur="save" />
    </div>
  </div>
</template>

<style scoped>
.config-form { display: flex; flex-direction: column; gap: 10px; padding: 8px; font-size: 13px; }
.field { display: flex; flex-direction: column; gap: 4px; }
label { font-size: 11px; color: #9ca3af; font-weight: 600; text-transform: uppercase; }
.error { font-size: 11px; color: #ef4444; }
</style>
