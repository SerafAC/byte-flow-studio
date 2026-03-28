import { defineStore } from 'pinia'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Events } from '@wailsio/runtime'
import * as wails from '../services/wails'

export type FlowState = 'idle' | 'running' | 'paused' | 'error'

export const usePipelineStore = defineStore('pipeline', () => {
  const flowState = ref<FlowState>('idle')
  const blockErrors = ref<Record<string, string>>({})
  const sessionId = ref<string | null>(null)

  function onStateChanged(event: { data: { state: FlowState; blockErrors: Record<string, string>; sessionId: string | null } }) {
    flowState.value = event.data.state
    blockErrors.value = event.data.blockErrors ?? {}
    sessionId.value = event.data.sessionId ?? null
  }

  let offStateChanged: (() => void) | null = null

  function subscribe() {
    offStateChanged = Events.On('pipeline:state-changed', onStateChanged)
  }

  function unsubscribe() {
    offStateChanged?.()
    offStateChanged = null
  }

  async function startFlow() {
    await wails.startPipeline()
  }

  async function pauseFlow() {
    await wails.pausePipeline()
  }

  async function resumeFlow() {
    await wails.resumePipeline()
  }

  async function stopFlow() {
    await wails.stopPipeline()
  }

  const canStart = computed(() => flowState.value === 'idle' || flowState.value === 'error')
  const canPause = computed(() => flowState.value === 'running')
  const canResume = computed(() => flowState.value === 'paused')
  const canStop = computed(() => flowState.value === 'running' || flowState.value === 'paused' || flowState.value === 'error')

  return {
    flowState,
    blockErrors,
    sessionId,
    subscribe,
    unsubscribe,
    startFlow,
    pauseFlow,
    resumeFlow,
    stopFlow,
    canStart,
    canPause,
    canResume,
    canStop,
  }
})
