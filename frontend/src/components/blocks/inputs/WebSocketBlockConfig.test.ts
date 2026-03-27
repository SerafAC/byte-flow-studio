import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import WebSocketBlockConfig from './WebSocketBlockConfig.vue'

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
  InputText: {
    inheritAttrs: false,
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: `<input type="text" :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)" />`,
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

describe('WebSocketBlockConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateBlockParams.mockResolvedValue(undefined)
  })

  function mountBlock(params: Record<string, unknown> = {}) {
    return mount(WebSocketBlockConfig, {
      props: { blockId: 'block-ws', params },
      global: { stubs: globalStubs },
    })
  }

  // ── Initialization ───────────────────────────────────────────────────────────

  it('renders with default ws:// URL', () => {
    const w = mountBlock()
    const input = w.find('input[type="text"]') as ReturnType<typeof w.find>
    expect((input.element as HTMLInputElement).value).toBe('ws://localhost:8080/data')
  })

  it('initializes URL from params', () => {
    const w = mountBlock({ url: 'wss://example.com/stream' })
    const input = w.find('input[type="text"]')
    expect((input.element as HTMLInputElement).value).toBe('wss://example.com/stream')
  })

  it('initializes reconnect interval from params', () => {
    const w = mountBlock({ reconnectIntervalMs: 5000 })
    const numInput = w.find('input[type="number"]')
    expect((numInput.element as HTMLInputElement).value).toBe('5000')
  })

  it('renders URL, Subprotocol, and Reconnect Interval fields', () => {
    const w = mountBlock()
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('URL')
    expect(labels).toContain('Subprotocol')
    expect(labels).toContain('Reconnect Interval (ms)')
  })

  // ── URL validation ───────────────────────────────────────────────────────────

  it('shows an error message for an http:// URL when save() is called', async () => {
    const w = mountBlock()
    const input = w.findAll('input[type="text"]')[0]
    await input.setValue('http://bad.url/data')
    await (w.vm as any).save()
    await flushPromises()
    expect(w.find('.error').exists()).toBe(true)
    expect(w.find('.error').text()).toContain('ws://')
  })

  it('shows an error message for a plain hostname when save() is called', async () => {
    const w = mountBlock()
    const input = w.findAll('input[type="text"]')[0]
    await input.setValue('localhost:8080')
    await (w.vm as any).save()
    await flushPromises()
    expect(w.find('.error').exists()).toBe(true)
  })

  it('does not call updateBlockParams for an invalid URL', async () => {
    const w = mountBlock()
    const input = w.findAll('input[type="text"]')[0]
    await input.setValue('ftp://wrong.protocol')
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).not.toHaveBeenCalled()
  })

  it('clears the error and saves for a valid ws:// URL', async () => {
    const w = mountBlock({ url: 'http://invalid' })
    // Trigger validation error
    await (w.vm as any).save()
    await flushPromises()
    expect(w.find('.error').exists()).toBe(true)

    // Fix the URL and save again
    const input = w.findAll('input[type="text"]')[0]
    await input.setValue('ws://valid.host:9000/data')
    await (w.vm as any).save()
    await flushPromises()
    expect(w.find('.error').exists()).toBe(false)
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-ws', expect.objectContaining({
      url: 'ws://valid.host:9000/data',
    }))
  })

  it('accepts and saves a wss:// URL', async () => {
    const w = mountBlock()
    const input = w.findAll('input[type="text"]')[0]
    await input.setValue('wss://secure.example.com/ws')
    await (w.vm as any).save()
    await flushPromises()
    expect(w.find('.error').exists()).toBe(false)
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-ws', expect.objectContaining({
      url: 'wss://secure.example.com/ws',
    }))
  })

  // ── Save behaviour ───────────────────────────────────────────────────────────

  it('save() sends all params together for a valid URL', async () => {
    const w = mountBlock({ url: 'ws://host/path', subprotocol: 'mqtt', reconnectIntervalMs: 3000 })
    await (w.vm as any).save()
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-ws', {
      url: 'ws://host/path',
      subprotocol: 'mqtt',
      reconnectIntervalMs: 3000,
    })
  })

  it('does NOT auto-save when URL input changes (requires explicit save)', async () => {
    const w = mountBlock()
    const input = w.findAll('input[type="text"]')[0]
    await input.setValue('ws://new.host/ws')
    await flushPromises()
    expect(mocks.updateBlockParams).not.toHaveBeenCalled()
  })

  it('exposes save() via defineExpose', () => {
    const w = mountBlock()
    expect(typeof (w.vm as any).save).toBe('function')
  })
})
