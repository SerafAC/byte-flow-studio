<script setup lang="ts">
import '../../../assets/main.scss'
import { ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import type { WebSocketParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: WebSocketParams
}>()

const url = ref<string>(props.params.url ?? 'ws://localhost:8080/data')
const subprotocol = ref<string>(props.params.subprotocol ?? '')
const reconnectIntervalMs = ref<number>(props.params.reconnectIntervalMs ?? 2000)

const urlError = ref('')

function validateUrl(v: string): boolean {
  return v.startsWith('ws://') || v.startsWith('wss://')
}

const { save: saveParams } = useBlockConfig(props.blockId, () => ({
  url: url.value,
  subprotocol: subprotocol.value,
  reconnectIntervalMs: reconnectIntervalMs.value,
}))

async function save() {
  if (!validateUrl(url.value)) {
    urlError.value = 'URL must start with ws:// or wss://'
    return
  }
  urlError.value = ''
  await saveParams()
}

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>URL</label>
      <InputText v-model="url" placeholder="ws://host:port/path" />
      <span v-if="urlError" class="error">{{ urlError }}</span>
    </div>
    <div class="field">
      <label>Subprotocol</label>
      <InputText v-model="subprotocol" placeholder="Optional" />
    </div>
    <div class="field">
      <label>Reconnect Interval (ms)</label>
      <InputNumber v-model="reconnectIntervalMs" :min="100" :max="30000" :step="100" />
    </div>
  </div>
</template>

<style scoped>
.error { font-size: 11px; color: #ef4444; }
</style>
