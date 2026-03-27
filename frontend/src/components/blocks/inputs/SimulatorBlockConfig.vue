<script setup lang="ts">
import '../../../assets/main.scss'
import { ref } from 'vue'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import type { SimulatorParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: SimulatorParams
}>()

const waveform = ref<string>(props.params.waveform ?? 'sine')
const frequencyHz = ref<number>(props.params.frequencyHz ?? 1.0)
const amplitude = ref<number>(props.params.amplitude ?? 1.0)
const offset = ref<number>(props.params.offset ?? 0.0)
const sampleRateHz = ref<number>(props.params.sampleRateHz ?? 100.0)

const waveformOptions = [
  { label: 'Sine', value: 'sine' },
  { label: 'Square', value: 'square' },
  { label: 'Sawtooth', value: 'sawtooth' },
  { label: 'Noise', value: 'noise' },
]

const { save } = useBlockConfig(props.blockId, () => ({
  waveform: waveform.value,
  frequencyHz: frequencyHz.value,
  amplitude: amplitude.value,
  offset: offset.value,
  sampleRateHz: sampleRateHz.value,
}))

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Waveform</label>
      <Select v-model="waveform" :options="waveformOptions" option-label="label" option-value="value" />
    </div>
    <div class="field">
      <label>Frequency (Hz)</label>
      <InputNumber v-model="frequencyHz" :min="0.001" :max="1000" :step="0.1" :min-fraction-digits="1" />
    </div>
    <div class="field">
      <label>Amplitude</label>
      <InputNumber v-model="amplitude" :step="0.1" :min-fraction-digits="1" />
    </div>
    <div class="field">
      <label>DC Offset</label>
      <InputNumber v-model="offset" :step="0.1" :min-fraction-digits="1" />
    </div>
    <div class="field">
      <label>Sample Rate (Hz)</label>
      <InputNumber v-model="sampleRateHz" :min="1" :max="10000" :step="10" />
    </div>
  </div>
</template>

