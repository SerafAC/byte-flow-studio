import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import SimulatorBlockConfig from './SimulatorBlockConfig.vue'

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

const globalStubs = {
  Select: {
    inheritAttrs: false,
    props: ['modelValue', 'options', 'optionLabel', 'optionValue'],
    emits: ['update:modelValue'],
    template: `<select :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)">
      <option v-for="o in options" :key="o.value" :value="o.value">{{ o.label }}</option>
    </select>`,
  },
  InputNumber: {
    inheritAttrs: false,
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: `<input type="number" :value="modelValue"
      @input="$emit('update:modelValue', +$event.target.value)" />`,
  },
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('SimulatorBlockConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateBlockParams.mockResolvedValue(undefined)
  })

  function mountBlock(params: Record<string, unknown> = {}) {
    return mount(SimulatorBlockConfig, {
      props: { blockId: 'block-sim', params },
      global: { stubs: globalStubs },
    })
  }

  // ── Initialization ───────────────────────────────────────────────────────────

  it('defaults to sine waveform', () => {
    const w = mountBlock()
    expect((w.find('select').element as HTMLSelectElement).value).toBe('sine')
  })

  it('defaults to frequency 1.0 Hz', () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[0].element as HTMLInputElement).value).toBe('1')
  })

  it('defaults to amplitude 1.0', () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[1].element as HTMLInputElement).value).toBe('1')
  })

  it('defaults to offset 0.0', () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[2].element as HTMLInputElement).value).toBe('0')
  })

  it('defaults to sample rate 100 Hz', () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[3].element as HTMLInputElement).value).toBe('100')
  })

  it('initializes all fields from params', () => {
    const w = mountBlock({ waveform: 'square', frequencyHz: 5, amplitude: 2, offset: 1, sampleRateHz: 200 })
    expect((w.find('select').element as HTMLSelectElement).value).toBe('square')
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[0].element as HTMLInputElement).value).toBe('5')
    expect((inputs[1].element as HTMLInputElement).value).toBe('2')
    expect((inputs[2].element as HTMLInputElement).value).toBe('1')
    expect((inputs[3].element as HTMLInputElement).value).toBe('200')
  })

  it('renders all configuration field labels', () => {
    const w = mountBlock()
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('Waveform')
    expect(labels).toContain('Frequency (Hz)')
    expect(labels).toContain('Amplitude')
    expect(labels).toContain('DC Offset')
    expect(labels).toContain('Sample Rate (Hz)')
  })

  // ── Save behaviour ───────────────────────────────────────────────────────────

  it('save() calls workflowStore.updateBlockParams with all current params', async () => {
    const w = mountBlock({ waveform: 'square', frequencyHz: 10, amplitude: 3, offset: 0.5, sampleRateHz: 500 })
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-sim', {
      waveform: 'square',
      frequencyHz: 10,
      amplitude: 3,
      offset: 0.5,
      sampleRateHz: 500,
    })
  })

  it('save() sends updated waveform after user changes value', async () => {
    const w = mountBlock()
    await w.find('select').setValue('noise')
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-sim', expect.objectContaining({ waveform: 'noise' }))
  })

  it('does NOT auto-save on waveform change (requires explicit save)', async () => {
    const w = mountBlock()
    await w.find('select').setValue('sawtooth')
    await flushPromises()
    expect(mocks.updateBlockParams).not.toHaveBeenCalled()
  })

  it('does NOT call save on unmount (no implicit save on close)', async () => {
    const w = mountBlock()
    w.unmount()
    await flushPromises()
    expect(mocks.updateBlockParams).not.toHaveBeenCalled()
  })

  it('exposes save() via defineExpose', () => {
    const w = mountBlock()
    expect(typeof (w.vm as any).save).toBe('function')
  })
})
