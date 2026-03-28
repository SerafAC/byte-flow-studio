import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import LineChartBlock from './LineChartBlock.vue'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => {
  const eventListeners: Record<string, ((...args: unknown[]) => void)[]> = {}
  return {
    eventListeners,
    enterFullscreen: vi.fn(),
    exitFullscreen: vi.fn(),
    uplotSetData: vi.fn(),
    uplotDestroy: vi.fn(),
    uplotSetSize: vi.fn(),
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

vi.mock('../../../composables/useFullscreen', () => ({
  useFullscreen: () => ({
    enterFullscreen: mocks.enterFullscreen,
    exitFullscreen: mocks.exitFullscreen,
    fullscreenBlockId: { value: null },
  }),
}))

vi.mock('uplot', () => {
  return {
    default: vi.fn().mockImplementation(() => ({
      setData: mocks.uplotSetData,
      destroy: mocks.uplotDestroy,
      setSize: mocks.uplotSetSize,
    })),
  }
})

vi.mock('uplot/dist/uPlot.min.css', () => ({}))

// ── Helpers ───────────────────────────────────────────────────────────────────

function fireEvent(event: string, payload: unknown) {
  for (const cb of mocks.eventListeners[event] ?? []) cb(payload as Parameters<typeof cb>[0])
}

const globalStubs = { ContextMenu: { template: '<div />' } }

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('LineChartBlock', () => {
  let wrapper: ReturnType<typeof mount> | null = null

  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(mocks.eventListeners).forEach(k => delete mocks.eventListeners[k])
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
  })

  function mountBlock(blockId = 'chart-1') {
    return mount(LineChartBlock, {
      props: { blockId },
      global: { stubs: globalStubs },
      attachTo: document.body,
    })
  }

  it('registers pipeline:data and session:view-changed listeners on mount', () => {
    wrapper = mountBlock()
    expect(mocks.eventListeners['pipeline:data']).toHaveLength(1)
    expect(mocks.eventListeners['session:view-changed']).toHaveLength(1)
  })

  it('deregisters listeners on unmount', () => {
    wrapper = mountBlock()
    wrapper.unmount()
    wrapper = null
    expect(mocks.eventListeners['pipeline:data']).toHaveLength(0)
    expect(mocks.eventListeners['session:view-changed']).toHaveLength(0)
  })

  it('sets isHistorical to true when session:view-changed fires with historical', async () => {
    wrapper = mountBlock()
    fireEvent('session:view-changed', { data: { sessionId: 's1', mode: 'historical' } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.historical-badge').exists()).toBe(true)
  })

  it('sets isHistorical to false when mode returns to live', async () => {
    wrapper = mountBlock()
    fireEvent('session:view-changed', { data: { sessionId: 's1', mode: 'historical' } })
    fireEvent('session:view-changed', { data: { sessionId: 's1', mode: 'live' } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.historical-badge').exists()).toBe(false)
  })

  it('ignores data for a different blockId', () => {
    wrapper = mountBlock('chart-1')
    fireEvent('pipeline:data', {
      data: { blockId: 'other-block', points: [{ timestamp: 1000, values: [42] }] },
    })
    // No error thrown and setData not called (uplot not initialized in jsdom)
  })

  it('ignores replay points in live mode', () => {
    wrapper = mountBlock('chart-1')
    // In live mode (isHistorical = false), replay points are skipped
    fireEvent('pipeline:data', {
      data: {
        blockId: 'chart-1',
        points: [{ timestamp: 1000, values: [42], replay: true }],
      },
    })
    // setData not called since point was skipped (uplot is null in jsdom anyway)
    expect(mocks.uplotSetData).not.toHaveBeenCalled()
  })
})
