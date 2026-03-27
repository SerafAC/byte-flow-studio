<script setup lang="ts">
import '../../../assets/config-form.css'
import { ref } from 'vue'
import Select from 'primevue/select'
import type { FFTParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: FFTParams
}>()

const windowSize = ref<number>(props.params.windowSize ?? 512)
const windowFunction = ref<string>(props.params.windowFunction ?? 'hann')

const windowSizeOptions = [128, 256, 512, 1024, 2048].map(n => ({ label: String(n), value: n }))
const windowFuncOptions = [
  { label: 'None', value: 'none' },
  { label: 'Hann', value: 'hann' },
  { label: 'Hamming', value: 'hamming' },
]

const { save } = useBlockConfig(props.blockId, () => ({
  windowSize: windowSize.value,
  windowFunction: windowFunction.value,
}))

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Window Size</label>
      <Select v-model="windowSize" :options="windowSizeOptions" option-label="label" option-value="value" />
    </div>
    <div class="field">
      <label>Window Function</label>
      <Select v-model="windowFunction" :options="windowFuncOptions" option-label="label" option-value="value" />
    </div>
  </div>
</template>

