import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import MovingAverageConfig from './MovingAverageConfig.vue'

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
  InputNumber: {
    inheritAttrs: false,
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: `<input type="number" :value="modelValue"
      @input="$emit('update:modelValue', +$event.target.value)" />`,
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

  it('save() calls workflowStore.updateBlockParams with current windowSize', async () => {
    const w = mountBlock({ windowSize: 5 })
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-ma', { windowSize: 5 })
  })

  it('save() sends updated windowSize after user changes value', async () => {
    const w = mountBlock()
    const input = w.find('input[type="number"]')
    await input.setValue(25)
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-ma', { windowSize: 25 })
  })

  it('exposes save() via defineExpose', () => {
    const w = mountBlock()
    expect(typeof (w.vm as any).save).toBe('function')
  })
})
