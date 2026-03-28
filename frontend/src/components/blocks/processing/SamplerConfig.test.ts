import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import SamplerConfig from './SamplerConfig.vue'

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

describe('SamplerConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateBlockParams.mockResolvedValue(undefined)
  })

  function mountBlock(params: Record<string, unknown> = {}) {
    return mount(SamplerConfig, {
      props: { blockId: 'block-sampler', params },
      global: { stubs: globalStubs },
    })
  }

  // ── Initialization ────────────────────────────────────────────────────────────

  it('defaults to every-n-samples mode', () => {
    const w = mountBlock()
    const select = w.find('select')
    expect((select.element as HTMLSelectElement).value).toBe('every-n-samples')
  })

  it('defaults N to 10', () => {
    const w = mountBlock()
    const input = w.find('input[type="number"]')
    expect((input.element as HTMLInputElement).value).toBe('10')
  })

  it('initializes mode from params', () => {
    const w = mountBlock({ mode: 'first-in-window', intervalMs: 200 })
    expect((w.find('select').element as HTMLSelectElement).value).toBe('first-in-window')
  })

  it('initializes n from params', () => {
    const w = mountBlock({ mode: 'every-n-samples', n: 25 })
    expect((w.find('input[type="number"]').element as HTMLInputElement).value).toBe('25')
  })

  // ── Conditional field visibility ──────────────────────────────────────────────

  it('shows N field and hides interval field for every-n-samples', () => {
    const w = mountBlock({ mode: 'every-n-samples' })
    const inputs = w.findAll('input[type="number"]')
    expect(inputs).toHaveLength(1)
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('Sample Interval (N)')
    expect(labels).not.toContain('Time Window (ms)')
  })

  it('shows interval field and hides N field for first-in-window', async () => {
    const w = mountBlock({ mode: 'first-in-window', intervalMs: 100 })
    await flushPromises()
    const inputs = w.findAll('input[type="number"]')
    expect(inputs).toHaveLength(1)
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('Time Window (ms)')
    expect(labels).not.toContain('Sample Interval (N)')
  })

  it('shows interval field for last-in-window', async () => {
    const w = mountBlock({ mode: 'last-in-window', intervalMs: 500 })
    await flushPromises()
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('Time Window (ms)')
  })

  it('hides both N and interval fields for first mode', async () => {
    const w = mountBlock({ mode: 'first' })
    await flushPromises()
    expect(w.findAll('input[type="number"]')).toHaveLength(0)
  })

  it('shows N field after mode changes from first-in-window to every-n-samples', async () => {
    const w = mountBlock({ mode: 'first-in-window', intervalMs: 100 })
    await w.find('select').setValue('every-n-samples')
    await flushPromises()
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('Sample Interval (N)')
    expect(labels).not.toContain('Time Window (ms)')
  })

  // ── Save behaviour ────────────────────────────────────────────────────────────

  it('save() sends every-n-samples params', async () => {
    const w = mountBlock({ mode: 'every-n-samples', n: 5, intervalMs: 100 })
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-sampler', {
      mode: 'every-n-samples',
      n: 5,
      intervalMs: 100,
    })
  })

  it('save() sends updated n after user changes value', async () => {
    const w = mountBlock({ mode: 'every-n-samples', n: 10, intervalMs: 100 })
    await w.find('input[type="number"]').setValue(50)
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-sampler', {
      mode: 'every-n-samples',
      n: 50,
      intervalMs: 100,
    })
  })

  it('save() sends first-in-window params with intervalMs', async () => {
    const w = mountBlock({ mode: 'first-in-window', n: 10, intervalMs: 250 })
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-sampler', {
      mode: 'first-in-window',
      n: 10,
      intervalMs: 250,
    })
  })

  it('exposes save() via defineExpose', () => {
    const w = mountBlock()
    expect(typeof (w.vm as any).save).toBe('function')
  })
})
