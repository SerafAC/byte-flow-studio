<script setup lang="ts">
import '../../../assets/main.scss'
import { useBlockConfig } from '../../../composables/useBlockConfig'

const props = defineProps<{
  blockId: string
  params: Record<string, unknown>
}>()

const { save } = useBlockConfig(props.blockId, () => ({}))

defineExpose({ save })
</script>

<template>
  <div class="config-form">
    <p class="info-text">
      Computes the rate of change (derivative) of the input signal over time using backward finite difference.
    </p>
    <p class="info-text">No configurable parameters.</p>
  </div>
</template>

<style scoped>
.info-text {
  font-size: 13px;
  color: #9ca3af;
  margin: 4px 0;
}
</style>
