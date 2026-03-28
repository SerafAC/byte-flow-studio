import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import * as wails from '../services/wails'
import type { SessionMeta } from '../services/wails'

export type ViewMode = 'live' | 'historical'

export const useSessionStore = defineStore('session', () => {
  const sessions = ref<SessionMeta[]>([])
  const activeSessionId = ref<string | null>(null)
  const viewMode = ref<ViewMode>('live')

  function onViewChanged(event: { data: { sessionId: string; mode: ViewMode } }) {
    activeSessionId.value = event.data.sessionId
    viewMode.value = event.data.mode
  }

  function onListUpdated(event: { data: { sessions: SessionMeta[] } }) {
    sessions.value = event.data.sessions ?? []
  }

  let offViewChanged: (() => void) | null = null
  let offListUpdated: (() => void) | null = null

  function subscribe() {
    offViewChanged = Events.On('session:view-changed', onViewChanged)
    offListUpdated = Events.On('session:list-updated', onListUpdated)
  }

  function unsubscribe() {
    offViewChanged?.()
    offListUpdated?.()
    offViewChanged = null
    offListUpdated = null
  }

  async function loadSessions() {
    sessions.value = await wails.listSessions()
    activeSessionId.value = await wails.getActiveSessionId()
  }

  async function switchSession(id: string) {
    await wails.setViewSession(id)
  }

  async function deleteSession(id: string) {
    await wails.deleteSession(id)
    sessions.value = sessions.value.filter(s => s.id !== id)
  }

  return {
    sessions,
    activeSessionId,
    viewMode,
    subscribe,
    unsubscribe,
    loadSessions,
    switchSession,
    deleteSession,
  }
})
