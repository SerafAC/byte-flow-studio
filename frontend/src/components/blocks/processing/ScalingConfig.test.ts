import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ScalingConfig from './ScalingConfig.vue'

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

describe('ScalingConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateBlockParams.mockResolvedValue(undefined)
  })

  function mountBlock(params: Record<string, unknown> = {}) {
    return mount(ScalingConfig, {
      props: { blockId: 'block-scale', params },
      global: { stubs: globalStubs },
    })
  }

  // ── Initialization ───────────────────────────────────────────────────────────

  it('defaults scale to 1.0', () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[0].element as HTMLInputElement).value).toBe('1')
  })

  it('defaults offset to 0.0', () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[1].element as HTMLInputElement).value).toBe('0')
  })

  it('initializes scale and offset from params', () => {
    const w = mountBlock({ scale: 2.5, offset: -3 })
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[0].element as HTMLInputElement).value).toBe('2.5')
    expect((inputs[1].element as HTMLInputElement).value).toBe('-3')
  })

  it('renders Scale and Offset labels', () => {
    const w = mountBlock()
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('Scale')
    expect(labels).toContain('Offset')
  })

  // ── Save behaviour ───────────────────────────────────────────────────────────

  it('save() calls workflowStore.updateBlockParams with current scale and offset', async () => {
    const w = mountBlock({ scale: 2, offset: 1 })
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-scale', { scale: 2, offset: 1 })
  })

  it('save() sends updated scale after user changes value', async () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    await inputs[0].setValue(10)
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-scale', expect.objectContaining({ scale: 10 }))
  })

  it('save() sends updated offset after user changes value', async () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    await inputs[1].setValue(-5)
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-scale', expect.objectContaining({ offset: -5 }))
  })

  it('exposes save() via defineExpose', () => {
    const w = mountBlock()
    expect(typeof (w.vm as any).save).toBe('function')
  })
})
