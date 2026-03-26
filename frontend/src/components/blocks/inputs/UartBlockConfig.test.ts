import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import UartBlockConfig from './UartBlockConfig.vue'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => ({
  updateBlockParams: vi.fn(),
  listSerialPorts: vi.fn(),
}))

vi.mock('../../../services/wails', () => ({
  updateBlockParams: mocks.updateBlockParams,
  listSerialPorts: mocks.listSerialPorts,
}))

// ── Stubs ─────────────────────────────────────────────────────────────────────

const globalStubs = {
  Select: {
    inheritAttrs: false,
    props: ['modelValue', 'options', 'optionLabel', 'optionValue', 'placeholder'],
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
  Button: {
    inheritAttrs: false,
    emits: ['click'],
    template: '<button @click="$emit(\'click\')">Refresh</button>',
  },
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('UartBlockConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateBlockParams.mockResolvedValue(undefined)
    mocks.listSerialPorts.mockResolvedValue([])
  })

  function mountBlock(params: Record<string, unknown> = {}) {
    return mount(UartBlockConfig, {
      props: { blockId: 'block-uart', params },
      global: { stubs: globalStubs },
    })
  }

  // ── Initialization ───────────────────────────────────────────────────────────

  it('calls listSerialPorts on mount', async () => {
    mountBlock()
    await flushPromises()
    expect(mocks.listSerialPorts).toHaveBeenCalledOnce()
  })

  it('renders Port, Baud Rate, Data Bits, Stop Bits, and Parity fields', () => {
    const w = mountBlock()
    const labels = w.findAll('label').map(l => l.text())
    expect(labels).toContain('Port')
    expect(labels).toContain('Baud Rate')
    expect(labels).toContain('Data Bits')
    expect(labels).toContain('Stop Bits')
    expect(labels).toContain('Parity')
  })

  it('defaults baudRate to 115200', async () => {
    const w = mountBlock()
    await flushPromises()
    // Baud rate is the second select (index 1, after port)
    const baudSelect = w.findAll('select')[1]
    expect((baudSelect.element as HTMLSelectElement).value).toBe('115200')
  })

  it('initializes baudRate from params', async () => {
    const w = mountBlock({ baudRate: 9600 })
    await flushPromises()
    const baudSelect = w.findAll('select')[1]
    expect((baudSelect.element as HTMLSelectElement).value).toBe('9600')
  })

  it('defaults parity to none', async () => {
    const w = mountBlock()
    await flushPromises()
    // Parity is the last select (index 4)
    const paritySelect = w.findAll('select')[4]
    expect((paritySelect.element as HTMLSelectElement).value).toBe('none')
  })

  // ── Port list ────────────────────────────────────────────────────────────────

  it('populates port options returned by listSerialPorts', async () => {
    mocks.listSerialPorts.mockResolvedValue([
      { name: '/dev/ttyUSB0', description: 'USB Serial' },
      { name: '/dev/ttyUSB1', description: '' },
    ])
    const w = mountBlock()
    await flushPromises()
    const portOptions = w.findAll('select')[0].findAll('option')
    expect(portOptions.some(o => o.text().includes('/dev/ttyUSB0'))).toBe(true)
    expect(portOptions.some(o => o.text().includes('/dev/ttyUSB1'))).toBe(true)
  })

  it('includes description in port option label when present', async () => {
    mocks.listSerialPorts.mockResolvedValue([
      { name: 'COM3', description: 'Prolific USB' },
    ])
    const w = mountBlock()
    await flushPromises()
    const portOption = w.findAll('select')[0].findAll('option')[0]
    expect(portOption.text()).toContain('(Prolific USB)')
  })

  it('omits parentheses when description is empty', async () => {
    mocks.listSerialPorts.mockResolvedValue([
      { name: 'COM4', description: '' },
    ])
    const w = mountBlock()
    await flushPromises()
    const portOption = w.findAll('select')[0].findAll('option')[0]
    expect(portOption.text()).not.toContain('()')
  })

  // ── Refresh ──────────────────────────────────────────────────────────────────

  it('renders a refresh button', () => {
    const w = mountBlock()
    expect(w.find('button').exists()).toBe(true)
  })

  it('calls listSerialPorts again when refresh button is clicked', async () => {
    const w = mountBlock()
    await flushPromises()
    await w.find('button').trigger('click')
    await flushPromises()
    expect(mocks.listSerialPorts).toHaveBeenCalledTimes(2)
  })

  // ── Save behaviour ───────────────────────────────────────────────────────────

  it('calls updateBlockParams when a port is selected', async () => {
    mocks.listSerialPorts.mockResolvedValue([{ name: '/dev/ttyUSB0', description: '' }])
    const w = mountBlock()
    await flushPromises()
    await w.findAll('select')[0].setValue('/dev/ttyUSB0')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-uart', expect.objectContaining({
      port: '/dev/ttyUSB0',
    }))
  })

  it('calls updateBlockParams when baud rate changes', async () => {
    const w = mountBlock()
    await flushPromises()
    await w.findAll('select')[1].setValue('9600')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-uart', expect.objectContaining({
      baudRate: '9600',
    }))
  })

  it('calls updateBlockParams when parity changes', async () => {
    const w = mountBlock()
    await flushPromises()
    await w.findAll('select')[4].setValue('even')
    await flushPromises()
    expect(mocks.updateBlockParams).toHaveBeenCalledWith('block-uart', expect.objectContaining({
      parity: 'even',
    }))
  })
})
