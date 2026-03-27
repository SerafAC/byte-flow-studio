<script setup lang="ts">
import '../../../assets/config-form.css'
import { ref } from 'vue'
import InputNumber from 'primevue/inputnumber'
import type { ScalingParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: ScalingParams
}>()

const scale = ref<number>(props.params.scale ?? 1.0)
const offset = ref<number>(props.params.offset ?? 0.0)

const { save } = useBlockConfig(props.blockId, () => ({ scale: scale.value, offset: offset.value }))

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Scale</label>
      <InputNumber v-model="scale" :step="0.1" :min-fraction-digits="1" />
    </div>
    <div class="field">
      <label>Offset</label>
      <InputNumber v-model="offset" :step="0.1" :min-fraction-digits="1" />
    </div>
  </div>
</template>

