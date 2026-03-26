<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import { updateBlockParams } from '../../../services/wails'

const props = defineProps<{
  blockId: string
  params: Record<string, unknown>
}>()

const waveform = ref<string>((props.params.waveform as string) ?? 'sine')
const frequencyHz = ref<number>((props.params.frequencyHz as number) ?? 1.0)
const amplitude = ref<number>((props.params.amplitude as number) ?? 1.0)
const offset = ref<number>((props.params.offset as number) ?? 0.0)
const sampleRateHz = ref<number>((props.params.sampleRateHz as number) ?? 100.0)

const waveformOptions = [
  { label: 'Sine', value: 'sine' },
  { label: 'Square', value: 'square' },
  { label: 'Sawtooth', value: 'sawtooth' },
  { label: 'Noise', value: 'noise' },
]

async function save() {
  await updateBlockParams(props.blockId, {
    waveform: waveform.value,
    frequencyHz: frequencyHz.value,
    amplitude: amplitude.value,
    offset: offset.value,
    sampleRateHz: sampleRateHz.value,
  })
}

onUnmounted(save)
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Waveform</label>
      <Select v-model="waveform" :options="waveformOptions" option-label="label" option-value="value" @change="save" />
    </div>
    <div class="field">
      <label>Frequency (Hz)</label>
      <InputNumber v-model="frequencyHz" :min="0.001" :max="1000" :step="0.1" :min-fraction-digits="1" @update:model-value="save" />
    </div>
    <div class="field">
      <label>Amplitude</label>
      <InputNumber v-model="amplitude" :step="0.1" :min-fraction-digits="1" @update:model-value="save" />
    </div>
    <div class="field">
      <label>DC Offset</label>
      <InputNumber v-model="offset" :step="0.1" :min-fraction-digits="1" @update:model-value="save" />
    </div>
    <div class="field">
      <label>Sample Rate (Hz)</label>
      <InputNumber v-model="sampleRateHz" :min="1" :max="10000" :step="10" @update:model-value="save" />
    </div>
  </div>
</template>

<style scoped>
.config-form { display: flex; flex-direction: column; gap: 10px; padding: 8px; font-size: 13px; }
.field { display: flex; flex-direction: column; gap: 4px; }
label { font-size: 11px; color: #9ca3af; font-weight: 600; text-transform: uppercase; }
</style>
