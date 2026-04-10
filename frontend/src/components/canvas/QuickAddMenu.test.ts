import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import QuickAddMenu from './QuickAddMenu.vue'

// ── Hoisted mocks ────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => ({
  addBlock: vi.fn().mockResolvedValue({ id: 'new-block-1' }),
  addConnection: vi.fn().mockResolvedValue({}),
  getAvailableBlockTypes: vi.fn().mockResolvedValue([]),
}))

vi.mock('../../services/wails', () => ({
  getAvailableBlockTypes: mocks.getAvailableBlockTypes,
  addBlock: mocks.addBlock,
  addConnection: mocks.addConnection,
}))

vi.mock('../../stores/workflow', () => ({
  useWorkflowStore: () => ({
    addBlock: mocks.addBlock,
    addConnection: mocks.addConnection,
  }),
}))

// ── Test data ────────────────────────────────────────────────────────────────

const allBlocks = [
  { type: 'uart', label: 'UART Input', category: 'input', description: '', defaults: {} },
  { type: 'simulator', label: 'Signal Simulator', category: 'input', description: '', defaults: {} },
  { type: 'moving-average', label: 'Moving Average', category: 'processing', description: '', defaults: {} },
  { type: 'byte-parser', label: 'Byte Parser', category: 'processing', description: '', defaults: {} },
  { type: 'passthrough', label: 'Passthrough', category: 'processing', description: '', defaults: {} },
  { type: 'fft', label: 'FFT', category: 'processing', description: '', defaults: {} },
  { type: 'scaling', label: 'Scaling', category: 'processing', description: '', defaults: {} },
  { type: 'summation', label: 'Summation', category: 'processing', description: '', defaults: {} },
  { type: 'value-display', label: 'Value Display', category: 'analysis', description: '', defaults: {} },
  { type: 'line-chart', label: 'Line Chart', category: 'analysis', description: '', defaults: {} },
  { type: 'data-table', label: 'Data Table', category: 'analysis', description: '', defaults: {} },
  { type: 'bar-chart', label: 'Bar Chart', category: 'analysis', description: '', defaults: {} },
  { type: 'fft-spectrum', label: 'FFT Spectrum', category: 'analysis', description: '', defaults: {} },
]

// ── Helpers ──────────────────────────────────────────────────────────────────

const PopoverStub = {
  template: '<div class="popover-stub"><slot /></div>',
  methods: { show: vi.fn(), hide: vi.fn() },
}

const InputTextStub = {
  template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  props: ['modelValue', 'placeholder', 'size', 'autofocus'],
  emits: ['update:modelValue'],
}

function mountMenu(sourceDataType = 'numeric') {
  return mount(QuickAddMenu, {
    props: {
      sourceBlockId: 'src-1',
      sourcePortId: 'out',
      sourceDataType,
      sourceX: 100,
      sourceY: 200,
    },
    global: {
      stubs: {
        Popover: PopoverStub,
        InputText: InputTextStub,
      },
    },
  })
}

// ── Setup ────────────────────────────────────────────────────────────────────

beforeEach(() => {
  vi.clearAllMocks()
  mocks.getAvailableBlockTypes.mockResolvedValue([...allBlocks])
  mocks.addBlock.mockResolvedValue({ id: 'new-block-1' })
})

// ── Tests ────────────────────────────────────────────────────────────────────

describe('QuickAddMenu', () => {
  describe('compatibility filter (T115)', () => {
    it('excludes input blocks (they have no input ports)', async () => {
      const wrapper = mountMenu('numeric')
      await flushPromises()

      const items = wrapper.findAll('.quick-add-item')
      const types = items.map(i => i.find('.qa-category').text())
      expect(types.every(t => t !== 'input')).toBe(true)
    })

    it('hides analysis blocks for raw data sources', async () => {
      const wrapper = mountMenu('raw')
      await flushPromises()

      const items = wrapper.findAll('.quick-add-item')
      const categories = items.map(i => i.find('.qa-category').text())
      expect(categories).not.toContain('analysis')
    })

    it('hides byte-parser for numeric data sources', async () => {
      const wrapper = mountMenu('numeric')
      await flushPromises()

      const items = wrapper.findAll('.quick-add-item')
      const labels = items.map(i => i.find('.qa-label').text())
      expect(labels).not.toContain('Byte Parser')
    })

    it('shows byte-parser for raw data sources', async () => {
      const wrapper = mountMenu('raw')
      await flushPromises()

      const items = wrapper.findAll('.quick-add-item')
      const labels = items.map(i => i.find('.qa-label').text())
      expect(labels).toContain('Byte Parser')
    })

    it('shows processing and analysis blocks for numeric sources', async () => {
      const wrapper = mountMenu('numeric')
      await flushPromises()

      const items = wrapper.findAll('.quick-add-item')
      const labels = items.map(i => i.find('.qa-label').text())
      expect(labels).toContain('Moving Average')
      expect(labels).toContain('Line Chart')
      expect(labels).toContain('Value Display')
    })
  })

  describe('search filtering', () => {
    it('filters by label text', async () => {
      const wrapper = mountMenu('numeric')
      await flushPromises()

      const input = wrapper.find('input')
      await input.setValue('chart')
      await flushPromises()

      const items = wrapper.findAll('.quick-add-item')
      const labels = items.map(i => i.find('.qa-label').text())
      expect(labels).toContain('Line Chart')
      expect(labels).toContain('Bar Chart')
      expect(labels).not.toContain('Moving Average')
    })
  })

  describe('block creation', () => {
    it('creates a block at +250px offset and auto-connects', async () => {
      const wrapper = mountMenu('numeric')
      await flushPromises()

      const firstItem = wrapper.find('.quick-add-item')
      await firstItem.trigger('click')
      await flushPromises()

      expect(mocks.addBlock).toHaveBeenCalledWith(
        expect.any(String),
        350, // sourceX(100) + 250
        200, // sourceY
      )
      expect(mocks.addConnection).toHaveBeenCalledWith('src-1', 'out', 'new-block-1', 'in')
    })

    it('emits close after selection', async () => {
      const wrapper = mountMenu('numeric')
      await flushPromises()

      const firstItem = wrapper.find('.quick-add-item')
      await firstItem.trigger('click')
      await flushPromises()

      expect(wrapper.emitted('close')).toBeTruthy()
    })
  })

  describe('empty state', () => {
    it('shows "No compatible blocks" when all are filtered out', async () => {
      mocks.getAvailableBlockTypes.mockResolvedValue([
        { type: 'uart', label: 'UART Input', category: 'input', description: '', defaults: {} },
      ])

      const wrapper = mountMenu('numeric')
      await flushPromises()

      expect(wrapper.find('.qa-empty').text()).toBe('No compatible blocks')
    })
  })
})
