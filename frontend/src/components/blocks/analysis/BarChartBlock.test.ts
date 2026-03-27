import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import BarChartBlock from './BarChartBlock.vue'

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
    },
    Off: (event: string, cb: (...args: unknown[]) => void) => {
      mocks.eventListeners[event] = (mocks.eventListeners[event] ?? []).filter(f => f !== cb)
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
      series: [{}],
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

describe('BarChartBlock', () => {
  let wrapper: ReturnType<typeof mount> | null = null

  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(mocks.eventListeners).forEach(k => delete mocks.eventListeners[k])
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
  })

  function mountBlock(blockId = 'bar-1') {
    return mount(BarChartBlock, {
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

  it('shows historical badge when mode is historical', async () => {
    wrapper = mountBlock()
    fireEvent('session:view-changed', { data: { sessionId: 's1', mode: 'historical' } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.historical-badge').exists()).toBe(true)
  })

  it('hides historical badge when mode returns to live', async () => {
    wrapper = mountBlock()
    fireEvent('session:view-changed', { data: { sessionId: 's1', mode: 'historical' } })
    fireEvent('session:view-changed', { data: { sessionId: 's1', mode: 'live' } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.historical-badge').exists()).toBe(false)
  })

  it('ignores data for a different blockId', () => {
    wrapper = mountBlock('bar-1')
    // Should not throw
    fireEvent('pipeline:data', {
      data: { blockId: 'not-bar-1', points: [{ timestamp: 1000, values: [1, 2, 3] }] },
    })
  })

  it('ignores replay points in live mode', () => {
    wrapper = mountBlock('bar-1')
    fireEvent('pipeline:data', {
      data: {
        blockId: 'bar-1',
        points: [{ timestamp: 1000, values: [1, 2], replay: true }],
      },
    })
    // latestValues not updated — no bar-latest div rendered
    expect(wrapper.find('.bar-latest').exists()).toBe(false)
  })

  it('renders latest values when data arrives for matching blockId', async () => {
    wrapper = mountBlock('bar-1')
    fireEvent('pipeline:data', {
      data: {
        blockId: 'bar-1',
        points: [{ timestamp: 1000, values: [3.5, 7.2] }],
      },
    })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.bar-latest').exists()).toBe(true)
    expect(wrapper.text()).toContain('Ch 1')
    expect(wrapper.text()).toContain('Ch 2')
  })
})
