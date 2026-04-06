<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Popover from 'primevue/popover'
import InputText from 'primevue/inputtext'
import * as wails from '../../services/wails'
import type { BlockTypeDescriptor } from '../../services/wails'
import { useWorkflowStore } from '../../stores/workflow'

const props = defineProps<{
  sourceBlockId: string
  sourcePortId: string
  sourceDataType: string
  sourceX: number
  sourceY: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const popover = ref()
const search = ref('')
const allBlockTypes = ref<BlockTypeDescriptor[]>([])
const workflowStore = useWorkflowStore()

onMounted(async () => {
  allBlockTypes.value = await wails.getAvailableBlockTypes()
})

// Filter to blocks with at least one input port that matches sourceDataType
const compatibleBlocks = computed(() => {
  const q = search.value.toLowerCase()
  return allBlockTypes.value.filter(bt => {
    if (bt.category === 'input') return false // inputs have no input ports
    // Check if block type has compatible input port (use label/type hints)
    // For raw data sources, exclude numeric-only blocks and vice versa
    const isRawSource = props.sourceDataType === 'raw'
    if (isRawSource && bt.category === 'analysis') return false // analysis expects numeric
    if (!isRawSource && bt.type === 'byte-parser') return false // byte-parser expects raw
    if (q && !bt.label.toLowerCase().includes(q) && !bt.type.toLowerCase().includes(q)) return false
    return true
  })
})

async function selectBlock(bt: BlockTypeDescriptor) {
  try {
    const newBlock = await workflowStore.addBlock(bt.type, props.sourceX + 250, props.sourceY)
    await workflowStore.addConnection(props.sourceBlockId, props.sourcePortId, newBlock.id, 'in')
  } catch {
    // Best-effort — ignore if connection fails
  }
  emit('close')
}

function open(event: MouseEvent) {
  popover.value?.show(event)
}

defineExpose({ open })
</script>

<template>
  <Popover ref="popover" @hide="emit('close')">
    <div class="quick-add-menu">
      <InputText
        v-model="search"
        placeholder="Search blocks…"
        size="small"
        class="quick-add-search"
        autofocus
      />
      <div class="quick-add-list">
        <button
          v-for="bt in compatibleBlocks"
          :key="bt.type"
          class="quick-add-item"
          @click="selectBlock(bt)"
        >
          <span class="qa-label">{{ bt.label }}</span>
          <span class="qa-category">{{ bt.category }}</span>
        </button>
        <div v-if="compatibleBlocks.length === 0" class="qa-empty">No compatible blocks</div>
      </div>
    </div>
  </Popover>
</template>

<style scoped>
.quick-add-menu {
  min-width: 200px;
  max-width: 260px;
}

.quick-add-search {
  width: 100%;
  margin-bottom: 6px;
  font-size: 12px;
}

.quick-add-list {
  max-height: 240px;
  overflow-y: auto;
}

.quick-add-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 6px 10px;
  background: none;
  border: none;
  cursor: pointer;
  color: #e2e8f0;
  font-size: 12px;
  border-radius: 4px;
  text-align: left;
}

.quick-add-item:hover {
  background: rgba(59, 130, 246, 0.2);
}

.qa-label {
  font-weight: 500;
}

.qa-category {
  font-size: 10px;
  color: #9ca3af;
  text-transform: uppercase;
}

.qa-empty {
  padding: 8px 10px;
  color: #6b7280;
  font-size: 12px;
}
</style>
