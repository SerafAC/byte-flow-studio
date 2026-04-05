<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import InputText from 'primevue/inputtext'
import { getAvailableBlockTypes, type BlockTypeDescriptor } from '../../services/wails'

defineEmits<{ (e: 'close'): void }>()

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
  input: '#4381cf',
  processing: '#9d50cf',
  analysis: '#49b393',
}

function onDragStart(event: DragEvent, block: BlockTypeDescriptor) {
  event.dataTransfer?.setData('application/byteflow-block', JSON.stringify(block))
}

function monogram(label: string) {
  return label.charAt(0).toUpperCase()
}
</script>

<template>
  <div class="block-library">
    <!-- Panel header -->
    <div class="library-header">
      <span class="library-title">Blocks</span>
      <button class="library-close" title="Close panel" @click="$emit('close')">✕</button>
    </div>

    <!-- Search -->
    <InputText v-model="search" placeholder="Search…" size="small" class="search-input" />

    <!-- Block cards per category -->
    <div class="library-scroll">
      <template v-for="cat in categoryOrder" :key="cat">
        <template v-if="grouped[cat]?.length">
          <div class="category-label" :style="{ color: categoryColor[cat] }">
            {{ categoryLabel[cat] }}
          </div>
          <div class="category-grid">
            <div
              v-for="block in grouped[cat]"
              :key="block.type"
              class="block-card"
              draggable="true"
              @dragstart="onDragStart($event, block)"
            >
              <div class="block-badge" :style="{ background: categoryColor[cat] }">
                {{ monogram(block.label) }}
              </div>
              <div class="block-info">
                <div class="block-label">{{ block.label }}</div>
                <div class="block-desc">{{ block.description }}</div>
              </div>
            </div>
          </div>
        </template>
      </template>
    </div>
  </div>
</template>

<style scoped lang="scss">
@use '../../assets/variables' as *;

.block-library {
  display: flex;
  flex-direction: column;
  background: $glass-bg;
  backdrop-filter: blur($glass-blur);
  -webkit-backdrop-filter: blur($glass-blur);
  box-shadow: $glass-shadow;
  border-radius: $radius-lg;
  border: 1px solid $ghost-border;
  overflow: hidden;
  font-family: $font-body;
  color: $text-primary;
}

.library-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $space-lg $space-lg $space-sm;
  flex-shrink: 0;
}

.library-title {
  font-family: $font-display;
  font-weight: 600;
  font-size: $font-size-lg;
  color: $text-primary;
}

.library-close {
  background: none;
  border: none;
  color: $text-muted;
  font-size: $font-size-md;
  cursor: pointer;
  padding: $space-xs $space-sm;
  border-radius: $radius-sm;
  line-height: 1;
  &:hover { background: rgba(255,255,255,0.08); color: $text-primary; }
}

.search-input {
  width: calc(100% - #{$space-xl});
  margin: 0 $space-md $space-md;
  flex-shrink: 0;
}

.library-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 0 $space-md $space-md;

  &::-webkit-scrollbar { width: 4px; }
  &::-webkit-scrollbar-track { background: transparent; }
  &::-webkit-scrollbar-thumb { background: $border-color; border-radius: 2px; }
}

.category-label {
  font-size: $font-size-xs;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.07em;
  padding: $space-md 0 $space-sm;
}

.category-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: $space-md;
  margin-bottom: $space-md;
}

.block-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: $space-sm;
  padding: $space-md;
  height: 96px;
  border-radius: $radius-md;
  cursor: grab;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid $ghost-border;
  transition: background $transition-fast, border-color $transition-fast;
  overflow: hidden;

  &:hover {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(43, 70, 128, 0.35);
  }

  &:active { cursor: grabbing; }
}

.block-badge {
  width: 32px;
  height: 32px;
  border-radius: $radius-md;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: $font-display;
  font-weight: 700;
  font-size: $font-size-lg;
  color: #fff;
  flex-shrink: 0;
}

.block-info {
  min-width: 0;
  width: 100%;
}

.block-label {
  font-family: $font-display;
  font-size: $font-size-md;
  font-weight: 500;
  color: $text-primary;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.block-desc {
  font-size: $font-size-xs;
  color: $text-muted;
  margin-top: 2px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
