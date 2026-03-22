<script setup lang="ts">
import { ref } from 'vue'
import InputNumber from 'primevue/inputnumber'
import { updateBlockParams } from '../../../services/wails'

const props = defineProps<{
  blockId: string
  params: Record<string, unknown>
}>()

const windowSize = ref<number>((props.params.windowSize as number) ?? 10)

async function save() {
  await updateBlockParams(props.blockId, { windowSize: windowSize.value })
}
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Window Size</label>
      <InputNumber v-model="windowSize" :min="1" :max="10000" :step="1" @blur="save" />
    </div>
  </div>
</template>

<style scoped>
.config-form { display: flex; flex-direction: column; gap: 10px; padding: 8px; font-size: 13px; }
.field { display: flex; flex-direction: column; gap: 4px; }
label { font-size: 11px; color: #9ca3af; font-weight: 600; text-transform: uppercase; }
</style>
