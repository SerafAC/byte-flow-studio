import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import FftSpectrumBlock from './FftSpectrumBlock.vue'
import uPlotLib from 'uplot'

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

vi.mock('uplot', () => ({
  default: vi.fn().mockImplementation(() => ({
    setData: mocks.uplotSetData,
    destroy: mocks.uplotDestroy,
    setSize: mocks.uplotSetSize,
  })),
}))

vi.mock('uplot/dist/uPlot.min.css', () => ({}))

// ── Helpers ───────────────────────────────────────────────────────────────────

function fireEvent(event: string, payload: unknown) {
  for (const cb of mocks.eventListeners[event] ?? []) cb(payload as Parameters<typeof cb>[0])
}

const globalStubs = { ContextMenu: { template: '<div />' } }

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('FftSpectrumBlock', () => {
  let wrapper: ReturnType<typeof mount> | null = null

  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(mocks.eventListeners).forEach(k => delete mocks.eventListeners[k])
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
  })

  function mountBlock(blockId = 'fft-1', fftWindowSize = 4) {
    return mount(FftSpectrumBlock, {
      props: { blockId, fftWindowSize },
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

  it('renders log scale toggle checkbox', () => {
    wrapper = mountBlock()
    expect(wrapper.find('input[type="checkbox"]').exists()).toBe(true)
  })

  it('ignores data for a different blockId', () => {
    wrapper = mountBlock('fft-1')
    // Should not throw
    fireEvent('pipeline:data', {
      data: { blockId: 'not-fft-1', points: [{ timestamp: 1000, values: [1, 2, 3, 4] }] },
    })
  })

  it('ignores replay points in live mode', () => {
    wrapper = mountBlock('fft-1')
    fireEvent('pipeline:data', {
      data: {
        blockId: 'fft-1',
        points: [{ timestamp: 1000, values: [1, 2, 3, 4], replay: true }],
      },
    })
    // Replay points skipped in live mode — no render triggered
    expect(mocks.uplotSetData).not.toHaveBeenCalled()
  })

  it('buffers single-value samples and only renders when FFT window is full', async () => {
    // fftWindowSize=4 means FFT_WIN=4 (nextPow2(4)=4)
    wrapper = mountBlock('fft-1', 4)
    await wrapper.vm.$nextTick()

    // Record constructor calls after mount (initEmptyChart creates one instance)
    const constructorCallsBefore = vi.mocked(uPlotLib).mock.calls.length

    // Send 3 samples — buffer not full yet, no new uPlot created
    for (let i = 1; i <= 3; i++) {
      fireEvent('pipeline:data', {
        data: { blockId: 'fft-1', points: [{ timestamp: i * 1000, values: [i] }] },
      })
    }
    await wrapper.vm.$nextTick()
    expect(vi.mocked(uPlotLib).mock.calls.length).toBe(constructorCallsBefore)

    // Send 4th sample — buffer full, FFT computed, new uPlot created with spectrum data
    fireEvent('pipeline:data', {
      data: { blockId: 'fft-1', points: [{ timestamp: 4000, values: [4] }] },
    })
    await wrapper.vm.$nextTick()
    expect(vi.mocked(uPlotLib).mock.calls.length).toBeGreaterThan(constructorCallsBefore)
  })

  it('renders multi-value (pre-computed spectrum) directly without buffering', async () => {
    wrapper = mountBlock('fft-1', 4)
    await wrapper.vm.$nextTick()
    const constructorCallsBefore = vi.mocked(uPlotLib).mock.calls.length

    // Multi-value input bypasses the internal FFT buffer — new uPlot constructed immediately
    fireEvent('pipeline:data', {
      data: { blockId: 'fft-1', points: [{ timestamp: 1000, values: [0.1, 0.5, 0.3, 0.2] }] },
    })
    await wrapper.vm.$nextTick()
    expect(vi.mocked(uPlotLib).mock.calls.length).toBeGreaterThan(constructorCallsBefore)
  })
})
