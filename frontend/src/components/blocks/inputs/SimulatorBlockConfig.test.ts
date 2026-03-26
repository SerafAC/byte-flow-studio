import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import SimulatorBlockConfig from './SimulatorBlockConfig.vue'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => ({
  updateBlockParams: vi.fn(),
}))

vi.mock('../../../services/wails', () => ({
  updateBlockParams: mocks.updateBlockParams,
}))

// ── Stubs ─────────────────────────────────────────────────────────────────────

const globalStubs = {
  Select: {
    inheritAttrs: false,
    props: ['modelValue', 'options', 'optionLabel', 'optionValue'],
    emits: ['update:modelValue', 'change'],
    template: `<select :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value); $emit('change', $event.target.value)">
      <option v-for="o in options" :key="o.value" :value="o.value">{{ o.label }}</option>
    </select>`,
  },
  InputNumber: {
    inheritAttrs: false,
    props: ['modelValue'],
    emits: ['update:modelValue', 'blur'],
    template: `<input type="number" :value="modelValue"
      @input="$emit('update:modelValue', +$event.target.value)"
      @blur="$emit('blur')" />`,
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

  it('calls updateBlockParams when waveform changes', async () => {
    const w = mountBlock()
    await w.find('select').setValue('noise')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-sim', expect.objectContaining({ waveform: 'noise' }))
  })

  it('calls updateBlockParams when waveform changes to sawtooth', async () => {
    const w = mountBlock()
    await w.find('select').setValue('sawtooth')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-sim', expect.objectContaining({ waveform: 'sawtooth' }))
  })

  it('saves all current param values on waveform change', async () => {
    const w = mountBlock({ frequencyHz: 10, amplitude: 3, offset: 0.5, sampleRateHz: 500 })
    await w.find('select').setValue('square')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-sim', {
      waveform: 'square',
      frequencyHz: 10,
      amplitude: 3,
      offset: 0.5,
      sampleRateHz: 500,
    })
  })

  it('calls save on unmount (via onUnmounted hook)', async () => {
    const w = mountBlock()
    w.unmount()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledOnce()
  })
})
