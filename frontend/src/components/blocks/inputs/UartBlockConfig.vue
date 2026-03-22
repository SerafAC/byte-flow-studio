<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import Button from 'primevue/button'
import { listSerialPorts, updateBlockParams, type SerialPortInfo } from '../../../services/wails'

const props = defineProps<{
  blockId: string
  params: Record<string, unknown>
}>()

const ports = ref<SerialPortInfo[]>([])
const selectedPort = ref<string>((props.params.port as string) ?? '')
const baudRate = ref<number>((props.params.baudRate as number) ?? 115200)
const dataBits = ref<number>((props.params.dataBits as number) ?? 8)
const stopBits = ref<number>((props.params.stopBits as number) ?? 1)
const parity = ref<string>((props.params.parity as string) ?? 'none')

const baudRateOptions = [1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600]
const parityOptions = [{ label: 'None', value: 'none' }, { label: 'Odd', value: 'odd' }, { label: 'Even', value: 'even' }]
const stopBitsOptions = [{ label: '1', value: 1 }, { label: '1.5', value: 1.5 }, { label: '2', value: 2 }]
const dataBitsOptions = [7, 8]

async function refreshPorts() {
  ports.value = await listSerialPorts() ?? []
}

async function save() {
  await updateBlockParams(props.blockId, {
    port: selectedPort.value,
    baudRate: baudRate.value,
    dataBits: dataBits.value,
    stopBits: stopBits.value,
    parity: parity.value,
  })
}

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
          @change="save"
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
        @change="save"
      />
    </div>
    <div class="field">
      <label>Data Bits</label>
      <Select
        v-model="dataBits"
        :options="dataBitsOptions.map(d => ({ label: String(d), value: d }))"
        option-label="label"
        option-value="value"
        @change="save"
      />
    </div>
    <div class="field">
      <label>Stop Bits</label>
      <Select v-model="stopBits" :options="stopBitsOptions" option-label="label" option-value="value" @change="save" />
    </div>
    <div class="field">
      <label>Parity</label>
      <Select v-model="parity" :options="parityOptions" option-label="label" option-value="value" @change="save" />
    </div>
  </div>
</template>

<style scoped>
.config-form { display: flex; flex-direction: column; gap: 10px; padding: 8px; font-size: 13px; }
.field { display: flex; flex-direction: column; gap: 4px; }
label { font-size: 11px; color: #9ca3af; font-weight: 600; text-transform: uppercase; }
.port-row { display: flex; gap: 6px; align-items: center; }
.flex-1 { flex: 1; }
</style>
