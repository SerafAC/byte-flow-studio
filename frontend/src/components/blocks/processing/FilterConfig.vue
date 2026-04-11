<script setup lang="ts">
import '../../../assets/main.scss'
import { ref, computed } from 'vue'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'
import type { FilterParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: FilterParams
}>()

const mode = ref<string>(props.params.mode ?? 'lowpass')
const cutoffHz = ref<number>(props.params.cutoffHz ?? 100)
const cutoffLowHz = ref<number>(props.params.cutoffLowHz ?? 50)
const cutoffHighHz = ref<number>(props.params.cutoffHighHz ?? 200)
const order = ref<number>(props.params.order ?? 2)
const sampleRateHz = ref<number>(props.params.sampleRateHz ?? 1000)

const modeOptions = [
  { label: 'Low Pass', value: 'lowpass' },
  { label: 'High Pass', value: 'highpass' },
  { label: 'Band Pass', value: 'bandpass' },
  { label: 'Band Stop (Notch)', value: 'bandstop' },
]

const isBandMode = computed(() => mode.value === 'bandpass' || mode.value === 'bandstop')

const { save } = useBlockConfig(props.blockId, () => ({
  mode: mode.value,
  cutoffHz: cutoffHz.value,
  cutoffLowHz: cutoffLowHz.value,
  cutoffHighHz: cutoffHighHz.value,
  order: order.value,
  sampleRateHz: sampleRateHz.value,
}))

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Filter Mode</label>
      <Select
        v-model="mode"
        :options="modeOptions"
        option-label="label"
        option-value="value"
      />
    </div>
    <div v-if="!isBandMode" class="field">
      <label>Cutoff Frequency (Hz)</label>
      <InputNumber v-model="cutoffHz" :min="1" :max="sampleRateHz / 2 - 1" :step="10" suffix=" Hz" />
    </div>
    <div v-if="isBandMode" class="field">
      <label>Low Cutoff (Hz)</label>
      <InputNumber v-model="cutoffLowHz" :min="1" :max="cutoffHighHz - 1" :step="10" suffix=" Hz" />
    </div>
    <div v-if="isBandMode" class="field">
      <label>High Cutoff (Hz)</label>
      <InputNumber v-model="cutoffHighHz" :min="cutoffLowHz + 1" :max="sampleRateHz / 2 - 1" :step="10" suffix=" Hz" />
    </div>
    <div class="field">
      <label>Order ({{ order }})</label>
      <Slider v-model="order" :min="1" :max="8" :step="1" />
    </div>
    <div class="field">
      <label>Sample Rate (Hz)</label>
      <InputNumber v-model="sampleRateHz" :min="1" :step="100" suffix=" Hz" />
    </div>
  </div>
</template>
