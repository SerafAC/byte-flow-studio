import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ByteParserConfig from './ByteParserConfig.vue'

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
  // Renders a real <select> so setValue() can trigger the watcher via v-model
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

describe('ByteParserConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateBlockParams.mockResolvedValue(undefined)
  })

  function mountBlock(params: Record<string, unknown> = {}) {
    return mount(ByteParserConfig, {
      props: { blockId: 'block-parser', params },
      global: { stubs: globalStubs },
    })
  }

  // ── Initialization ───────────────────────────────────────────────────────────

  it('defaults to float32-le format', () => {
    const w = mountBlock()
    const select = w.find('select')
    expect((select.element as HTMLSelectElement).value).toBe('float32-le')
  })

  it('defaults to 1 channel', () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[0].element as HTMLInputElement).value).toBe('1')
  })

  it('defaults to frameSize of 4', () => {
    const w = mountBlock()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[1].element as HTMLInputElement).value).toBe('4')
  })

  it('initializes from params', () => {
    const w = mountBlock({ format: 'int16-be', channels: 4, frameSize: 8 })
    const select = w.find('select')
    expect((select.element as HTMLSelectElement).value).toBe('int16-be')
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[0].element as HTMLInputElement).value).toBe('4')
    expect((inputs[1].element as HTMLInputElement).value).toBe('8')
  })

  it('renders Format, Channels, and Frame Size fields', () => {
    const w = mountBlock()
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('Format')
    expect(labels).toContain('Channels')
    expect(labels).toContain('Frame Size (bytes)')
  })

  // ── Auto frameSize calculation ───────────────────────────────────────────────

  it('sets frameSize to 2 when format changes to int16-le (1 channel)', async () => {
    const w = mountBlock() // float32-le, 1 ch, frameSize=4
    const select = w.find('select')
    await select.setValue('int16-le')
    await flushPromises()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[1].element as HTMLInputElement).value).toBe('2')
  })

  it('sets frameSize to 1 when format changes to uint8 (1 channel)', async () => {
    const w = mountBlock()
    const select = w.find('select')
    await select.setValue('uint8')
    await flushPromises()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[1].element as HTMLInputElement).value).toBe('1')
  })

  it('sets frameSize to 4 when format changes from int16-le to float32-be (1 channel)', async () => {
    const w = mountBlock({ format: 'int16-le', channels: 1, frameSize: 2 })
    const select = w.find('select')
    await select.setValue('float32-be')
    await flushPromises()
    const inputs = w.findAll('input[type="number"]')
    expect((inputs[1].element as HTMLInputElement).value).toBe('4')
  })

  // ── Save behaviour ───────────────────────────────────────────────────────────

  it('save() calls workflowStore.updateBlockParams with all current params', async () => {
    const w = mountBlock({ format: 'int16-le', channels: 2, frameSize: 4 })
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-parser', {
      format: 'int16-le',
      channels: 2,
      frameSize: 4,
    })
  })

  it('save() includes watcher-updated frameSize after format change', async () => {
    const w = mountBlock({ format: 'int16-le', channels: 2, frameSize: 4 })
    await w.find('select').setValue('float32-le')
    await flushPromises() // watcher updates frameSize to 4*2=8
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-parser', {
      format: 'float32-le',
      channels: 2,
      frameSize: 8,
    })
  })

  it('exposes save() via defineExpose', () => {
    const w = mountBlock()
    expect(typeof (w.vm as any).save).toBe('function')
  })
})
