<script setup lang="ts">
import '../../../assets/main.scss'
import { ref, computed } from 'vue'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import type { SamplerParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: SamplerParams
}>()

const mode = ref<string>(props.params.mode ?? 'every-n-samples')
const n = ref<number>(props.params.n ?? 10)
const intervalMs = ref<number>(props.params.intervalMs ?? 100)

const modeOptions = [
  { label: 'Every N Samples', value: 'every-n-samples' },
  { label: 'First in Time Window', value: 'first-in-window' },
  { label: 'Last in Time Window', value: 'last-in-window' },
  { label: 'First Value Only', value: 'first' },
]

const showN = computed(() => mode.value === 'every-n-samples')
const showInterval = computed(() => mode.value === 'first-in-window' || mode.value === 'last-in-window')

const { save } = useBlockConfig(props.blockId, () => ({
  mode: mode.value,
  n: n.value,
  intervalMs: intervalMs.value,
}))

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Mode</label>
      <Select v-model="mode" :options="modeOptions" option-label="label" option-value="value" />
    </div>
    <div v-if="showN" class="field">
      <label>Sample Interval (N)</label>
      <InputNumber v-model="n" :min="1" :max="100000" :step="1" />
    </div>
    <div v-if="showInterval" class="field">
      <label>Time Window (ms)</label>
      <InputNumber v-model="intervalMs" :min="1" :max="60000" :step="10" />
    </div>
  </div>
</template>
