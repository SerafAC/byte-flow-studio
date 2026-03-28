import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import DataTableBlock from './DataTableBlock.vue'

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

// Render the rows prop so we can assert on them
const globalStubs = {
  ContextMenu: { template: '<div />' },
  DataTable: {
    props: ['value'],
    template: `<table>
      <tbody>
        <tr v-for="(row, i) in value" :key="i">
          <td class="ts">{{ row.timestamp }}</td>
          <td v-for="(v, j) in row.values" :key="j" class="val">{{ v }}</td>
        </tr>
      </tbody>
    </table>`,
  },
  Column: { template: '<col />' },
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('DataTableBlock', () => {
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
    wrapper = mount(DataTableBlock, {
      props: { blockId: 'block-1', ...props },
      global: { stubs: globalStubs },
    })
    return wrapper
  }

  // ── Rendering ───────────────────────────────────────────────────────────────

  it('renders the outer container', () => {
    const w = mountBlock()
    expect(w.find('.data-table-block').exists()).toBe(true)
  })

  it('starts with no rows', () => {
    const w = mountBlock()
    expect(w.findAll('tr').length).toBe(0)
  })

  it('does not show HISTORICAL badge initially', () => {
    const w = mountBlock()
    expect(w.find('.historical-badge').exists()).toBe(false)
  })

  // ── Data events ─────────────────────────────────────────────────────────────

  it('appends a row for each incoming data point', async () => {
    const w = mountBlock()
    fireEvent('pipeline:data', {
      data: { blockId: 'block-1', points: [{ timestamp: 1_000_000, values: [1.0] }, { timestamp: 2_000_000, values: [2.0] }] },
    })
    await flushPromises()
    expect(w.findAll('tr').length).toBe(2)
  })

  it('ignores data for other blockIds', async () => {
    const w = mountBlock()
    fireEvent('pipeline:data', { data: { blockId: 'other', points: [{ timestamp: 1000, values: [99] }] } })
    await flushPromises()
    expect(w.findAll('tr').length).toBe(0)
  })

  it('formats values with the decimals prop', async () => {
    const w = mountBlock({ decimals: 4 })
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [3.14159] }] } })
    await flushPromises()
    expect(w.findAll('.val').some(td => td.text() === '3.1416')).toBe(true)
  })

  it('defaults to 2 decimal places', async () => {
    const w = mountBlock()
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [1.23456] }] } })
    await flushPromises()
    expect(w.findAll('.val').some(td => td.text() === '1.23')).toBe(true)
  })

  it('ignores replay points in live mode', async () => {
    const w = mountBlock()
    fireEvent('pipeline:data', {
      data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [1], replay: true }] },
    })
    await flushPromises()
    expect(w.findAll('tr').length).toBe(0)
  })

  // ── maxRows ──────────────────────────────────────────────────────────────────

  it('respects maxRows limit', async () => {
    const w = mountBlock({ maxRows: 3 })
    for (let i = 0; i < 5; i++) {
      fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: i * 1000, values: [i] }] } })
    }
    await flushPromises()
    expect(w.findAll('tr').length).toBe(3)
  })

  it('keeps the most recent rows when maxRows is exceeded', async () => {
    const w = mountBlock({ maxRows: 2 })
    for (let i = 1; i <= 4; i++) {
      fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: i * 1000, values: [i] }] } })
    }
    await flushPromises()
    const valueCells = w.findAll('.val')
    // Remaining rows should contain values 3 and 4
    const texts = valueCells.map(c => c.text())
    expect(texts).toContain('3.00')
    expect(texts).toContain('4.00')
    expect(texts).not.toContain('1.00')
  })

  // ── View change (historical / live) ─────────────────────────────────────────

  it('shows HISTORICAL badge when view switches to historical', async () => {
    const w = mountBlock()
    fireEvent('session:view-changed', { data: { mode: 'historical' } })
    await flushPromises()
    expect(w.find('.historical-badge').exists()).toBe(true)
  })

  it('clears rows on switch to historical', async () => {
    const w = mountBlock()
    fireEvent('pipeline:data', { data: { blockId: 'block-1', points: [{ timestamp: 1000, values: [1] }] } })
    await flushPromises()
    expect(w.findAll('tr').length).toBe(1)

    fireEvent('session:view-changed', { data: { mode: 'historical' } })
    await flushPromises()
    expect(w.findAll('tr').length).toBe(0)
  })

  it('hides HISTORICAL badge when view switches back to live', async () => {
    const w = mountBlock()
    fireEvent('session:view-changed', { data: { mode: 'historical' } })
    await flushPromises()
    fireEvent('session:view-changed', { data: { mode: 'live' } })
    await flushPromises()
    expect(w.find('.historical-badge').exists()).toBe(false)
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
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab' }))
    await flushPromises()
    expect(mocks.exitFullscreen).not.toHaveBeenCalled()
  })

  // ── Lifecycle ────────────────────────────────────────────────────────────────

  it('registers event listeners on mount', () => {
    mountBlock()
    expect((mocks.eventListeners['pipeline:data'] ?? []).length).toBeGreaterThan(0)
    expect((mocks.eventListeners['session:view-changed'] ?? []).length).toBeGreaterThan(0)
  })

  it('deregisters pipeline:data listener on unmount', () => {
    const w = mountBlock()
    const before = (mocks.eventListeners['pipeline:data'] ?? []).length
    w.unmount()
    wrapper = null
    expect((mocks.eventListeners['pipeline:data'] ?? []).length).toBe(before - 1)
  })
})
