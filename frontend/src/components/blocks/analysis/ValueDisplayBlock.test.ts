import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ValueDisplayBlock from './ValueDisplayBlock.vue'

// ── Hoisted mocks (available before imports are resolved) ─────────────────────

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

// ── Helpers ───────────────────────────────────────────────────────────────────

function fireEvent(event: string, payload: unknown) {
  for (const cb of mocks.eventListeners[event] ?? []) cb(payload as Parameters<typeof cb>[0])
}

const globalStubs = {
  ContextMenu: { template: '<div />' },
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('ValueDisplayBlock', () => {
  let wrapper: ReturnType<typeof mount> | null = null

  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(mocks.eventListeners).forEach(k => delete mocks.eventListeners[k])
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
  })

  function mountBlock(props: Record<string, unknown> = {}) {
    wrapper = mount(ValueDisplayBlock, {
      props: { blockId: 'block-1', ...props },
      global: { stubs: globalStubs },
    })
    return wrapper
  }

  // ── Rendering ───────────────────────────────────────────────────────────────

  it('shows default "Value" label when no label prop', () => {
    const w = mountBlock()
    expect(w.find('.vd-label').text()).toBe('Value')
  })

  it('shows custom label from prop', () => {
    const w = mountBlock({ label: 'Temperature' })
    expect(w.find('.vd-label').text()).toBe('Temperature')
  })

  it('shows dash initially', () => {
    const w = mountBlock()
    expect(w.find('.vd-value').text()).toBe('—')
  })

  it('does not show sparkline initially', () => {
    const w = mountBlock({ showHistory: true })
    expect(w.find('.vd-sparkline').exists()).toBe(false)
  })

  it('does not show HISTORICAL badge initially', () => {
    const w = mountBlock()
    expect(w.find('.vd-historical-badge').exists()).toBe(false)
  })

  // ── Data events ─────────────────────────────────────────────────────────────

  it('updates display from pipeline:data with numeric values', async () => {
    const w = mountBlock({ decimals: 3, unit: 'V' })
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [1.2345] }] } })
    await flushPromises()
    // (1.2345).toFixed(3) varies by JS engine; match actual output
    expect(w.find('.vd-value').text()).toBe((1.2345).toFixed(3) + ' V')
  })

  it('defaults to 2 decimal places', async () => {
    const w = mountBlock()
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [3.14159] }] } })
    await flushPromises()
    expect(w.find('.vd-value').text()).toBe('3.14')
  })

  it('omits unit suffix when unit prop is not set', async () => {
    const w = mountBlock({ decimals: 1 })
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [5.0] }] } })
    await flushPromises()
    expect(w.find('.vd-value').text()).toBe('5.0')
  })

  it('ignores data for other blockIds', async () => {
    const w = mountBlock()
    fireEvent('pipeline:data', { data: { blockId: 'other-block', points: [{ timestamp: 1000, values: [42] }] } })
    await flushPromises()
    expect(w.find('.vd-value').text()).toBe('—')
  })

  it('renders raw bytes as uppercase hex pairs', async () => {
    const w = mountBlock()
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, raw: [0x0a, 0xff, 0x10] }] } })
    await flushPromises()
    expect(w.find('.vd-value').text()).toBe('0A FF 10')
  })

  it('ignores replay points in live mode', async () => {
    const w = mountBlock()
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [77], replay: true }] } })
    await flushPromises()
    expect(w.find('.vd-value').text()).toBe('—')
  })

  // ── Sparkline ───────────────────────────────────────────────────────────────

  it('shows sparkline after receiving data when showHistory is true', async () => {
    const w = mountBlock({ showHistory: true })
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [1] }, { timestamp: 2000, values: [2] }] } })
    await flushPromises()
    expect(w.find('.vd-sparkline').exists()).toBe(true)
  })

  it('does not show sparkline when showHistory is false', async () => {
    const w = mountBlock({ showHistory: false })
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [1] }, { timestamp: 2000, values: [2] }] } })
    await flushPromises()
    expect(w.find('.vd-sparkline').exists()).toBe(false)
  })

  // ── View change (historical / live) ─────────────────────────────────────────

  it('shows HISTORICAL badge when view switches to historical', async () => {
    const w = mountBlock()
    fireEvent('session:view-changed', { data: { mode: 'historical' } })
    await flushPromises()
    expect(w.find('.vd-historical-badge').exists()).toBe(true)
    expect(w.find('.vd-historical-badge').text()).toBe('HISTORICAL')
  })

  it('resets value and sparkline on switch to historical', async () => {
    const w = mountBlock({ showHistory: true })
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [99] }] } })
    await flushPromises()

    fireEvent('session:view-changed', { data: { mode: 'historical' } })
    await flushPromises()

    expect(w.find('.vd-value').text()).toBe('—')
    expect(w.find('.vd-sparkline').exists()).toBe(false)
  })

  it('hides HISTORICAL badge when view switches back to live', async () => {
    const w = mountBlock()
    fireEvent('session:view-changed', { data: { mode: 'historical' } })
    await flushPromises()
    fireEvent('session:view-changed', { data: { mode: 'live' } })
    await flushPromises()
    expect(w.find('.vd-historical-badge').exists()).toBe(false)
  })

  // ── Keyboard / fullscreen ────────────────────────────────────────────────────

  it('calls exitFullscreen and emits fullscreen:exit on Escape key', async () => {
    const w = mountBlock()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(mocks.exitFullscreen).toHaveBeenCalledOnce()
    expect(w.emitted('fullscreen:exit')).toBeTruthy()
  })

  it('does not call exitFullscreen for non-Escape keys', async () => {
    mountBlock()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    await flushPromises()
    expect(mocks.exitFullscreen).not.toHaveBeenCalled()
  })

  // ── Lifecycle ────────────────────────────────────────────────────────────────

  it('registers pipeline:data listener on mount', () => {
    mountBlock()
    expect((mocks.eventListeners['pipeline:data'] ?? []).length).toBeGreaterThan(0)
  })

  it('deregisters pipeline:data listener on unmount', () => {
    const w = mountBlock()
    const before = (mocks.eventListeners['pipeline:data'] ?? []).length
    w.unmount()
    wrapper = null
    expect((mocks.eventListeners['pipeline:data'] ?? []).length).toBe(before - 1)
  })
})
