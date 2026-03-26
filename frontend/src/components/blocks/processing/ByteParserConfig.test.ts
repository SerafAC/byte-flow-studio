import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ByteParserConfig from './ByteParserConfig.vue'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => ({
  updateBlockParams: vi.fn(),
}))

vi.mock('../../../services/wails', () => ({
  updateBlockParams: mocks.updateBlockParams,
}))

// ── Stubs ─────────────────────────────────────────────────────────────────────

const globalStubs = {
  // Renders a real <select> so setValue() can trigger the watcher via v-model
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

  it('calls updateBlockParams with updated format on change', async () => {
    const w = mountBlock()
    const select = w.find('select')
    await select.setValue('int16-le')
    await flushPromises()
    // save() fires on @change before the async watcher updates frameSize
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-parser', expect.objectContaining({
      format: 'int16-le',
    }))
  })

  it('saves watcher-updated frameSize when frame size input is blurred after format change', async () => {
    const w = mountBlock({ format: 'int16-le', channels: 2, frameSize: 4 })
    const select = w.find('select')
    await select.setValue('float32-le')
    await flushPromises()
    // Watcher has now updated frameSize to 4*2=8; trigger save via blur
    const inputs = w.findAll('input[type="number"]')
    await inputs[1].trigger('blur')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-parser', {
      format: 'float32-le',
      channels: 2,
      frameSize: 8,
    })
  })
})
