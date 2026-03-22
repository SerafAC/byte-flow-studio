import { ref } from 'vue'

const fullscreenBlockId = ref<string | null>(null)

export function useFullscreen() {
  function enterFullscreen(blockId: string) {
    fullscreenBlockId.value = blockId
  }

  function exitFullscreen() {
    fullscreenBlockId.value = null
  }

  return { fullscreenBlockId, enterFullscreen, exitFullscreen }
}
