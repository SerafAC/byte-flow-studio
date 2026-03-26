import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import MovingAverageConfig from './MovingAverageConfig.vue'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => ({
  updateBlockParams: vi.fn(),
}))

vi.mock('../../../services/wails', () => ({
  updateBlockParams: mocks.updateBlockParams,
}))

// ── Stubs ─────────────────────────────────────────────────────────────────────

const globalStubs = {
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

describe('MovingAverageConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateBlockParams.mockResolvedValue(undefined)
  })

  function mountBlock(params: Record<string, unknown> = {}) {
    return mount(MovingAverageConfig, {
      props: { blockId: 'block-ma', params },
      global: { stubs: globalStubs },
    })
  }

  // ── Initialization ───────────────────────────────────────────────────────────

  it('defaults to windowSize of 10', () => {
    const w = mountBlock()
    expect((w.find('input[type="number"]').element as HTMLInputElement).value).toBe('10')
  })

  it('initializes windowSize from params', () => {
    const w = mountBlock({ windowSize: 50 })
    expect((w.find('input[type="number"]').element as HTMLInputElement).value).toBe('50')
  })

  it('renders the Window Size label', () => {
    const w = mountBlock()
    expect(w.find('label').text()).toBe('Window Size')
  })

  // ── Save behaviour ───────────────────────────────────────────────────────────

  it('calls updateBlockParams on blur with current windowSize', async () => {
    const w = mountBlock({ windowSize: 5 })
    await w.find('input[type="number"]').trigger('blur')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-ma', { windowSize: 5 })
  })

  it('saves updated windowSize after input and blur', async () => {
    const w = mountBlock()
    const input = w.find('input[type="number"]')
    await input.setValue(25)
    await input.trigger('blur')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-ma', { windowSize: 25 })
  })
})
