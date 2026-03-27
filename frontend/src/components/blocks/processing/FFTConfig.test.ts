import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import FFTConfig from './FFTConfig.vue'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => ({
  updateBlockParams: vi.fn(),
}))

vi.mock('../../../stores/workflow', () => ({
  useWorkflowStore: () => ({
    updateBlockParams: mocks.updateBlockParams,
  }),
}))

// ── Stubs ─────────────────────────────────────────────────────────────────────

// The Select stub emits the raw string value. For numeric window sizes the
// component stores them as numbers, so we convert when the string is numeric.
const globalStubs = {
  Select: {
    inheritAttrs: false,
    props: ['modelValue', 'options', 'optionLabel', 'optionValue'],
    emits: ['update:modelValue'],
    template: `<select :value="modelValue"
      @change="$emit('update:modelValue', isNaN(Number($event.target.value)) ? $event.target.value : Number($event.target.value))">
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

  it('save() calls workflowStore.updateBlockParams with current params', async () => {
    const w = mountBlock({ windowSize: 1024, windowFunction: 'hamming' })
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-fft', {
      windowSize: 1024,
      windowFunction: 'hamming',
    })
  })

  it('save() sends updated windowSize after user changes value', async () => {
    const w = mountBlock()
    await w.findAll('select')[0].setValue('2048')
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-fft', expect.objectContaining({ windowSize: 2048 }))
  })

  it('save() sends updated windowFunction after user changes value', async () => {
    const w = mountBlock()
    await w.findAll('select')[1].setValue('hamming')
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-fft', expect.objectContaining({ windowFunction: 'hamming' }))
  })

  it('save() sends both params together', async () => {
    const w = mountBlock({ windowSize: 256, windowFunction: 'hann' })
    await w.findAll('select')[1].setValue('none')
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-fft', {
      windowSize: 256,
      windowFunction: 'none',
    })
  })

  it('exposes save() via defineExpose', () => {
    const w = mountBlock()
    expect(typeof (w.vm as any).save).toBe('function')
  })
})
