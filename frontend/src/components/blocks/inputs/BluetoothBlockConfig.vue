<script setup lang="ts">
import '../../../assets/main.scss'
import { ref, onMounted } from 'vue'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { listBluetoothDevices, type BluetoothDeviceInfo } from '../../../services/wails'
import type { BluetoothParams } from '../../../types/block-params'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: BluetoothParams
}>()

const devices = ref<BluetoothDeviceInfo[]>([])
const selectedDevice = ref<string>(props.params.deviceAddress ?? '')
const serialPort = ref<string>(props.params.serialPort ?? '')

async function refreshDevices() {
  devices.value = await listBluetoothDevices() ?? []
}

const { save } = useBlockConfig(props.blockId, () => ({
  deviceAddress: selectedDevice.value,
  serialPort: serialPort.value,
}))

defineExpose({ save })

onMounted(refreshDevices)
</script>

<template>
  <div class="config-form">
    <div class="field">
      <label>Bluetooth Device</label>
      <div class="port-row">
        <Select
          v-model="selectedDevice"
          :options="devices.map(d => ({ label: d.name + ' (' + d.address + ')', value: d.address }))"
          option-label="label"
          option-value="value"
          placeholder="Select paired device"
          class="flex-1"
        />
        <Button icon="pi pi-refresh" size="small" severity="secondary" @click="refreshDevices" />
      </div>
    </div>
    <div class="field">
      <label>Serial Port Override (optional)</label>
      <InputText v-model="serialPort" placeholder="e.g. /dev/rfcomm0" />
    </div>
  </div>
</template>

<style scoped>
.port-row { display: flex; gap: 6px; align-items: center; }
.flex-1 { flex: 1; }
</style>
