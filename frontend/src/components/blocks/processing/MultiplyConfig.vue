<script setup lang="ts">
import '../../../assets/main.scss'
import { ref } from 'vue'
import InputNumber from 'primevue/inputnumber'
import type { MultiplyParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: MultiplyParams
}>()

const inputCount = ref<number>(props.params.inputCount ?? 2)

const { save } = useBlockConfig(props.blockId, () => ({ inputCount: inputCount.value }))

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Number of Inputs (2-8)</label>
      <InputNumber v-model="inputCount" :min="2" :max="8" :step="1" />
    </div>
  </div>
</template>
