<script setup lang="ts">
import '../../../assets/main.scss'
import { ref, watch } from 'vue'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import type { ByteParserParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: ByteParserParams
}>()

const format = ref<string>(props.params.format ?? 'float32-le')
const channels = ref<number>(props.params.channels ?? 1)
const frameSize = ref<number>(props.params.frameSize ?? 4)

const formatOptions = [
  { label: 'Float32 LE', value: 'float32-le' },
  { label: 'Float32 BE', value: 'float32-be' },
  { label: 'Int16 LE', value: 'int16-le' },
  { label: 'Int16 BE', value: 'int16-be' },
  { label: 'UInt8', value: 'uint8' },
]

const bytesPerSampleMap: Record<string, number> = {
  'float32-le': 4,
  'float32-be': 4,
  'int16-le': 2,
  'int16-be': 2,
  'uint8': 1,
}

// Auto-calculate frameSize when format or channels change
watch([format, channels], () => {
  frameSize.value = (bytesPerSampleMap[format.value] ?? 4) * channels.value
})

const { save } = useBlockConfig(props.blockId, () => ({
  format: format.value,
  channels: channels.value,
  frameSize: frameSize.value,
}))

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Format</label>
      <Select v-model="format" :options="formatOptions" option-label="label" option-value="value" />
    </div>
    <div class="field">
      <label>Channels</label>
      <InputNumber v-model="channels" :min="1" :max="32" :step="1" />
    </div>
    <div class="field">
      <label>Frame Size (bytes)</label>
      <InputNumber v-model="frameSize" :min="1" :max="1024" :step="1" />
    </div>
  </div>
</template>

