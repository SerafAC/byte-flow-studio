<script setup lang="ts">
import { ref } from 'vue'
import InputNumber from 'primevue/inputnumber'
import { updateBlockParams } from '../../../services/wails'

const props = defineProps<{
  blockId: string
  params: Record<string, unknown>
}>()

const scale = ref<number>((props.params.scale as number) ?? 1.0)
const offset = ref<number>((props.params.offset as number) ?? 0.0)

async function save() {
  await updateBlockParams(props.blockId, { scale: scale.value, offset: offset.value })
}
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Scale</label>
      <InputNumber v-model="scale" :step="0.1" :min-fraction-digits="1" @blur="save" />
    </div>
    <div class="field">
      <label>Offset</label>
      <InputNumber v-model="offset" :step="0.1" :min-fraction-digits="1" @blur="save" />
    </div>
  </div>
</template>

<style scoped>
.config-form { display: flex; flex-direction: column; gap: 10px; padding: 8px; font-size: 13px; }
.field { display: flex; flex-direction: column; gap: 4px; }
label { font-size: 11px; color: #9ca3af; font-weight: 600; text-transform: uppercase; }
</style>
