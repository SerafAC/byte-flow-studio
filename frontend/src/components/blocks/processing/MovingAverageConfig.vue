<script setup lang="ts">
import '../../../assets/config-form.css'
import { ref } from 'vue'
import InputNumber from 'primevue/inputnumber'
import type { MovingAverageParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: MovingAverageParams
}>()

const windowSize = ref<number>(props.params.windowSize ?? 10)

const { save } = useBlockConfig(props.blockId, () => ({ windowSize: windowSize.value }))

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Window Size</label>
      <InputNumber v-model="windowSize" :min="1" :max="10000" :step="1" />
    </div>
  </div>
</template>

