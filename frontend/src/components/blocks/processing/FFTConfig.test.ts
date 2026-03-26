import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import FFTConfig from './FFTConfig.vue'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => ({
  updateBlockParams: vi.fn(),
}))

vi.mock('../../../services/wails', () => ({
  updateBlockParams: mocks.updateBlockParams,
}))

// ── Stubs ─────────────────────────────────────────────────────────────────────

// The Select stub emits the raw string value. For numeric window sizes the
// component stores them as numbers, so we convert when the string is numeric.
const globalStubs = {
  Select: {
    inheritAttrs: false,
    props: ['modelValue', 'options', 'optionLabel', 'optionValue'],
    emits: ['update:modelValue', 'change'],
    template: `<select :value="modelValue"
      @change="$emit('update:modelValue', isNaN(Number($event.target.value)) ? $event.target.value : Number($event.target.value));
               $emit('change', $event.target.value)">
      <option v-for="o in options" :key="o.value" :value="o.value">{{ o.label }}</option>
    </select>`,
  },
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('FFTConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateBlockParams.mockResolvedValue(undefined)
  })

  function mountBlock(params: Record<string, unknown> = {}) {
    return mount(FFTConfig, {
      props: { blockId: 'block-fft', params },
      global: { stubs: globalStubs },
    })
  }

  // ── Initialization ───────────────────────────────────────────────────────────

  it('defaults windowSize to 512', () => {
    const w = mountBlock()
    const selects = w.findAll('select')
    expect((selects[0].element as HTMLSelectElement).value).toBe('512')
  })

  it('defaults windowFunction to hann', () => {
    const w = mountBlock()
    const selects = w.findAll('select')
    expect((selects[1].element as HTMLSelectElement).value).toBe('hann')
  })

  it('initializes windowSize from params', () => {
    const w = mountBlock({ windowSize: 1024, windowFunction: 'hamming' })
    const selects = w.findAll('select')
    expect((selects[0].element as HTMLSelectElement).value).toBe('1024')
  })

  it('initializes windowFunction from params', () => {
    const w = mountBlock({ windowSize: 256, windowFunction: 'none' })
    const selects = w.findAll('select')
    expect((selects[1].element as HTMLSelectElement).value).toBe('none')
  })

  it('renders Window Size and Window Function labels', () => {
    const w = mountBlock()
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('Window Size')
    expect(labels).toContain('Window Function')
  })

  it('exposes all valid window size options (128 – 2048)', () => {
    const w = mountBlock()
    const options = w.findAll('select')[0].findAll('option').map(o => o.element.value)
    expect(options).toEqual(['128', '256', '512', '1024', '2048'])
  })

  it('exposes None, Hann, and Hamming window function options', () => {
    const w = mountBlock()
    const options = w.findAll('select')[1].findAll('option').map(o => o.element.value)
    expect(options).toEqual(['none', 'hann', 'hamming'])
  })

  // ── Save behaviour ───────────────────────────────────────────────────────────

  it('calls updateBlockParams when windowSize changes', async () => {
    const w = mountBlock()
    await w.findAll('select')[0].setValue('2048')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-fft', expect.objectContaining({ windowSize: 2048 }))
  })

  it('calls updateBlockParams when windowFunction changes', async () => {
    const w = mountBlock()
    await w.findAll('select')[1].setValue('hamming')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-fft', expect.objectContaining({ windowFunction: 'hamming' }))
  })

  it('saves both params together', async () => {
    const w = mountBlock({ windowSize: 256, windowFunction: 'hann' })
    await w.findAll('select')[1].setValue('none')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-fft', {
      windowSize: 256,
      windowFunction: 'none',
    })
  })
})
