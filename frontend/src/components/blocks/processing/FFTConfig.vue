<script setup lang="ts">
import { ref } from 'vue'
import Select from 'primevue/select'
import { updateBlockParams } from '../../../services/wails'

const props = defineProps<{
  blockId: string
  params: Record<string, unknown>
}>()

const windowSize = ref<number>((props.params.windowSize as number) ?? 512)
const windowFunction = ref<string>((props.params.windowFunction as string) ?? 'hann')

const windowSizeOptions = [128, 256, 512, 1024, 2048].map(n => ({ label: String(n), value: n }))
const windowFuncOptions = [
  { label: 'None', value: 'none' },
  { label: 'Hann', value: 'hann' },
  { label: 'Hamming', value: 'hamming' },
]

async function save() {
  await updateBlockParams(props.blockId, {
    windowSize: windowSize.value,
    windowFunction: windowFunction.value,
  })
}
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Window Size</label>
      <Select v-model="windowSize" :options="windowSizeOptions" option-label="label" option-value="value" @change="save" />
    </div>
    <div class="field">
      <label>Window Function</label>
      <Select v-model="windowFunction" :options="windowFuncOptions" option-label="label" option-value="value" @change="save" />
    </div>
  </div>
</template>

<style scoped>
.config-form { display: flex; flex-direction: column; gap: 10px; padding: 8px; font-size: 13px; }
.field { display: flex; flex-direction: column; gap: 4px; }
label { font-size: 11px; color: #9ca3af; font-weight: 600; text-transform: uppercase; }
</style>
