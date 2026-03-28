import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { useAnalysisBlock } from './useAnalysisBlock'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => {
  const eventListeners: Record<string, ((...args: unknown[]) => void)[]> = {}
  return {
    eventListeners,
    enterFullscreen: vi.fn(),
    exitFullscreen: vi.fn(),
  }
})

vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: (event: string, cb: (...args: unknown[]) => void) => {
      if (!mocks.eventListeners[event]) mocks.eventListeners[event] = []
      mocks.eventListeners[event].push(cb)
      return () => {
        mocks.eventListeners[event] = (mocks.eventListeners[event] ?? []).filter(f => f !== cb)
      }
    },
    Off: (...eventNames: string[]) => {
      eventNames.forEach(name => { delete mocks.eventListeners[name] })
    },
  },
}))

vi.mock('./useFullscreen', () => ({
  useFullscreen: () => ({
    enterFullscreen: mocks.enterFullscreen,
    exitFullscreen: mocks.exitFullscreen,
    fullscreenBlockId: { value: null },
  }),
}))

// ── Helpers ───────────────────────────────────────────────────────────────────

function fireEvent(event: string, payload: unknown) {
  for (const cb of mocks.eventListeners[event] ?? []) cb(payload as Parameters<typeof cb>[0])
}

function mountComposable(opts: {
  blockId?: string
  onData?: (e: unknown) => void
  onHistoricalMode?: () => void
  emit?: (...args: unknown[]) => void
}) {
  const onData = opts.onData ?? vi.fn()
  const onHistoricalMode = opts.onHistoricalMode ?? vi.fn()
  const emit = opts.emit ?? vi.fn()
  const blockId = opts.blockId ?? 'block-1'

  const Wrapper = defineComponent({
    setup() {
      return useAnalysisBlock({
        blockId,
        onData: onData as (event: { data: { blockId: string; points: unknown[] } }) => void,
        onHistoricalMode,
        emit: emit as ReturnType<typeof useAnalysisBlock>['contextMenuItems'] extends never ? never : Parameters<typeof useAnalysisBlock>[0]['emit'],
      })
    },
    template: '<div />',
  })

  return mount(Wrapper)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('useAnalysisBlock', () => {
  let wrapper: ReturnType<typeof mount> | null = null

  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(mocks.eventListeners).forEach(k => delete mocks.eventListeners[k])
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
  })

  it('registers pipeline:data listener on mount', () => {
    wrapper = mountComposable({})
    expect(mocks.eventListeners['pipeline:data']).toHaveLength(1)
  })

  it('registers session:view-changed listener on mount', () => {
    wrapper = mountComposable({})
    expect(mocks.eventListeners['session:view-changed']).toHaveLength(1)
  })

  it('deregisters both listeners on unmount', () => {
    wrapper = mountComposable({})
    expect(mocks.eventListeners['pipeline:data']).toHaveLength(1)
    expect(mocks.eventListeners['session:view-changed']).toHaveLength(1)
    wrapper.unmount()
    wrapper = null
    expect(mocks.eventListeners['pipeline:data']).toHaveLength(0)
    expect(mocks.eventListeners['session:view-changed']).toHaveLength(0)
  })

  it('sets isHistorical to true and calls onHistoricalMode when mode is historical', async () => {
    const onHistoricalMode = vi.fn()
    wrapper = mountComposable({ onHistoricalMode })
    fireEvent('session:view-changed', { data: { sessionId: 's1', mode: 'historical' } })
    expect(wrapper.vm.isHistorical).toBe(true)
    expect(onHistoricalMode).toHaveBeenCalledOnce()
  })

  it('sets isHistorical to false and does NOT call onHistoricalMode when mode is live', () => {
    const onHistoricalMode = vi.fn()
    wrapper = mountComposable({ onHistoricalMode })
    // First go historical, then back to live
    fireEvent('session:view-changed', { data: { sessionId: 's1', mode: 'historical' } })
    fireEvent('session:view-changed', { data: { sessionId: 's1', mode: 'live' } })
    expect(wrapper.vm.isHistorical).toBe(false)
    expect(onHistoricalMode).toHaveBeenCalledOnce() // only called for historical
  })

  it('escape key calls exitFullscreen and emits fullscreen:exit', () => {
    const emit = vi.fn()
    wrapper = mountComposable({ emit })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(mocks.exitFullscreen).toHaveBeenCalled()
    expect(emit).toHaveBeenCalledWith('fullscreen:exit')
  })

  it('non-escape key does nothing', () => {
    const emit = vi.fn()
    wrapper = mountComposable({ emit })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(mocks.exitFullscreen).not.toHaveBeenCalled()
    expect(emit).not.toHaveBeenCalled()
  })

  it('removes keydown listener on unmount', () => {
    const emit = vi.fn()
    wrapper = mountComposable({ emit })
    wrapper.unmount()
    wrapper = null
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(mocks.exitFullscreen).not.toHaveBeenCalled()
  })

  it('context menu item calls enterFullscreen and emits fullscreen:enter', () => {
    const emit = vi.fn()
    wrapper = mountComposable({ blockId: 'block-abc', emit })
    const item = wrapper.vm.contextMenuItems[0]
    item.command()
    expect(mocks.enterFullscreen).toHaveBeenCalledWith('block-abc')
    expect(emit).toHaveBeenCalledWith('fullscreen:enter', { blockId: 'block-abc' })
  })
})
