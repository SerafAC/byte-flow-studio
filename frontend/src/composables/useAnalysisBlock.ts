import { ref, onMounted, onUnmounted } from 'vue'
import { Events } from '@wailsio/runtime'
import { useFullscreen } from './useFullscreen'

interface AnalysisBlockEmit {
  (e: 'fullscreen:enter', payload: { blockId: string }): void
  (e: 'fullscreen:exit'): void
}

interface AnalysisBlockOptions {
  blockId: string
  onData: (event: { data: { blockId: string; points: unknown[] } }) => void
  onHistoricalMode: () => void
  emit: AnalysisBlockEmit
}

export function useAnalysisBlock(opts: AnalysisBlockOptions) {
  const isHistorical = ref(false)
  const contextMenu = ref()
  const { enterFullscreen: fsEnter, exitFullscreen: fsExit } = useFullscreen()

  const contextMenuItems = [
    {
      label: 'Full Screen',
      icon: 'pi pi-window-maximize',
      command: () => {
        fsEnter(opts.blockId)
        opts.emit('fullscreen:enter', { blockId: opts.blockId })
      },
    },
  ]

  function onEscapeKey(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      fsExit()
      opts.emit('fullscreen:exit')
    }
  }

  function onViewChanged(event: { data: { sessionId: string; mode: 'live' | 'historical' } }) {
    isHistorical.value = event.data.mode === 'historical'
    if (isHistorical.value) {
      opts.onHistoricalMode()
    }
  }

  let offData: (() => void) | null = null
  let offView: (() => void) | null = null

  onMounted(() => {
    offData = Events.On('pipeline:data', opts.onData)
    offView = Events.On('session:view-changed', onViewChanged)
    document.addEventListener('keydown', onEscapeKey)
  })

  onUnmounted(() => {
    offData?.()
    offView?.()
    document.removeEventListener('keydown', onEscapeKey)
  })

  return { isHistorical, contextMenu, contextMenuItems }
}
