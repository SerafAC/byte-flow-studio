<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import InputText from 'primevue/inputtext'
import { getAvailableBlockTypes, type BlockTypeDescriptor } from '../../services/wails'

const blockTypes = ref<BlockTypeDescriptor[]>([])
const search = ref('')

onMounted(async () => {
  blockTypes.value = await getAvailableBlockTypes() ?? []
})

const grouped = computed(() => {
  const filtered = blockTypes.value.filter(b =>
    b.label.toLowerCase().includes(search.value.toLowerCase()) ||
    b.description.toLowerCase().includes(search.value.toLowerCase())
  )
  const groups: Record<string, BlockTypeDescriptor[]> = {}
  for (const b of filtered) {
    if (!groups[b.category]) groups[b.category] = []
    groups[b.category].push(b)
  }
  return groups
})

const categoryOrder = ['input', 'processing', 'analysis']
const categoryLabel: Record<string, string> = {
  input: 'Input',
  processing: 'Processing',
  analysis: 'Analysis',
}
const categoryColor: Record<string, string> = {
  input: '#3b82f6',
  processing: '#8b5cf6',
  analysis: '#10b981',
}

function onDragStart(event: DragEvent, block: BlockTypeDescriptor) {
  event.dataTransfer?.setData('application/byteflow-block', JSON.stringify(block))
}
</script>

<template>
  <div class="block-library">
    <div class="library-header">
      <span class="library-title">Blocks</span>
      <InputText v-model="search" placeholder="Search…" size="small" class="search-input" />
    </div>

    <template v-for="cat in categoryOrder" :key="cat">
      <template v-if="grouped[cat]?.length">
        <div class="category-label" :style="{ borderLeftColor: categoryColor[cat] }">
          {{ categoryLabel[cat] }}
        </div>
        <div
          v-for="block in grouped[cat]"
          :key="block.type"
          class="block-card"
          draggable="true"
          @dragstart="onDragStart($event, block)"
        >
          <span class="block-dot" :style="{ background: categoryColor[cat] }"></span>
          <div class="block-info">
            <div class="block-label">{{ block.label }}</div>
            <div class="block-desc">{{ block.description }}</div>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.block-library {
  padding: 8px;
  font-size: 13px;
}

.library-header {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 10px;
}

.library-title {
  font-weight: bold;
  font-size: 14px;
}

.search-input {
  width: 100%;
}

.category-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #9ca3af;
  padding: 4px 0 2px 8px;
  border-left: 3px solid;
  margin: 8px 0 4px;
}

.block-card {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: grab;
  background: #2d3f52;
  margin-bottom: 4px;
  transition: background 0.15s;
}

.block-card:hover {
  background: #3a5068;
}

.block-card:active {
  cursor: grabbing;
}

.block-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-top: 4px;
}

.block-info {
  min-width: 0;
}

.block-label {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.block-desc {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 1px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
