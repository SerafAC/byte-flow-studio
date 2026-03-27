<script setup lang="ts">
import '../../../assets/main.scss'
import { ref, onMounted } from 'vue'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import Button from 'primevue/button'
import { listSerialPorts, type SerialPortInfo } from '../../../services/wails'
import type { UartParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: UartParams
}>()

const ports = ref<SerialPortInfo[]>([])
const selectedPort = ref<string>(props.params.port ?? '')
const baudRate = ref<number>(props.params.baudRate ?? 115200)
const dataBits = ref<number>(props.params.dataBits ?? 8)
const stopBits = ref<number>(props.params.stopBits ?? 1)
const parity = ref<string>(props.params.parity ?? 'none')

const baudRateOptions = [1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600]
const parityOptions = [{ label: 'None', value: 'none' }, { label: 'Odd', value: 'odd' }, { label: 'Even', value: 'even' }]
const stopBitsOptions = [{ label: '1', value: 1 }, { label: '1.5', value: 1.5 }, { label: '2', value: 2 }]
const dataBitsOptions = [7, 8]

async function refreshPorts() {
  ports.value = await listSerialPorts() ?? []
}

const { save } = useBlockConfig(props.blockId, () => ({
  port: selectedPort.value,
  baudRate: baudRate.value,
  dataBits: dataBits.value,
  stopBits: stopBits.value,
  parity: parity.value,
}))

defineExpose({ save })

onMounted(refreshPorts)
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Port</label>
      <div class="port-row">
        <Select
          v-model="selectedPort"
          :options="ports.map(p => ({ label: p.name + (p.description ? ' (' + p.description + ')' : ''), value: p.name }))"
          option-label="label"
          option-value="value"
          placeholder="Select port"
          class="flex-1"
        />
        <Button icon="pi pi-refresh" size="small" severity="secondary" @click="refreshPorts" />
      </div>
    </div>
    <div class="field">
      <label>Baud Rate</label>
      <Select
        v-model="baudRate"
        :options="baudRateOptions.map(r => ({ label: String(r), value: r }))"
        option-label="label"
        option-value="value"
      />
    </div>
    <div class="field">
      <label>Data Bits</label>
      <Select
        v-model="dataBits"
        :options="dataBitsOptions.map(d => ({ label: String(d), value: d }))"
        option-label="label"
        option-value="value"
      />
    </div>
    <div class="field">
      <label>Stop Bits</label>
      <Select v-model="stopBits" :options="stopBitsOptions" option-label="label" option-value="value" />
    </div>
    <div class="field">
      <label>Parity</label>
      <Select v-model="parity" :options="parityOptions" option-label="label" option-value="value" />
    </div>
  </div>
</template>

<style scoped>
.port-row { display: flex; gap: 6px; align-items: center; }
.flex-1 { flex: 1; }
</style>
