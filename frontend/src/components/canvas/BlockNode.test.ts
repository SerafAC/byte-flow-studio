import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import BlockNode from './BlockNode.vue'
import type { BlockDef } from '../../services/wails'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => ({
  enterFullscreen: vi.fn(),
  removeBlock: vi.fn(),
  nodeSelected: false,
  configSave: vi.fn(),
}))

vi.mock('@vue-flow/core', () => ({
  Handle: { template: '<div />' },
  Position: { Left: 'left', Right: 'right' },
  useNode: () => ({ node: { selected: mocks.nodeSelected, position: { x: 0, y: 0 } } }),
}))

vi.mock('../../composables/useFullscreen', () => ({
  useFullscreen: () => ({
    enterFullscreen: mocks.enterFullscreen,
    exitFullscreen: vi.fn(),
    fullscreenBlockId: { value: null },
  }),
}))

vi.mock('../../stores/workflow', () => ({
  useWorkflowStore: () => ({
    removeBlock: mocks.removeBlock,
  }),
}))

// ── Stubs ─────────────────────────────────────────────────────────────────────

// Dialog stub: renders both the default slot and the footer slot so that
// Save/Cancel buttons placed in #footer are reachable in tests.
const DialogStub = {
  template: '<div class="dialog-stub"><slot /><slot name="footer" /></div>',
}

// Button stub: renders a plain <button> so we can find and click it by label.
const ButtonStub = {
  template: '<button @click="$emit(\'click\')">{{ label }}</button>',
  props: ['label', 'severity'],
  emits: ['click'],
}

// Config component stubs that expose a `save` method — mirrors the interface
// required by BlockNode after the fix (defineExpose({ save })).
const makeConfigStub = () =>
  defineComponent({
    setup(_, { expose }) {
      expose({ save: mocks.configSave })
      return {}
    },
    template: '<div class="config-stub" />',
  })

const globalStubs = {
  Dialog: DialogStub,
  Button: ButtonStub,
  QuickAddMenu: { template: '<div />' },
  UartBlockConfig: makeConfigStub(),
  SimulatorBlockConfig: makeConfigStub(),
  WebSocketBlockConfig: makeConfigStub(),
  MovingAverageConfig: makeConfigStub(),
  FFTConfig: makeConfigStub(),
  ScalingConfig: makeConfigStub(),
  ByteParserConfig: makeConfigStub(),
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function makeBlock(overrides: Partial<BlockDef> = {}): BlockDef {
  return {
    id: 'b1',
    type: 'simulator',
    category: 'input',
    label: 'Simulator',
    params: {},
    positionX: 0,
    positionY: 0,
    ...overrides,
  }
}

function mountNode(data: BlockDef) {
  return mount(BlockNode, {
    props: { data },
    global: { stubs: globalStubs },
  })
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('BlockNode', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.configSave.mockResolvedValue(undefined)
  })

  // ── Rendering ────────────────────────────────────────────────────────────────

  it('renders block label', () => {
    const w = mountNode(makeBlock({ label: 'My Block' }))
    expect(w.text()).toContain('My Block')
  })

  it('renders block type as fallback when label is empty', () => {
    const w = mountNode(makeBlock({ label: '', type: 'uart' }))
    expect(w.text()).toContain('uart')
  })

  it('renders category badge', () => {
    const w = mountNode(makeBlock({ category: 'processing' }))
    expect(w.text()).toContain('processing')
  })

  it('shows error message when status is error', () => {
    const w = mountNode(makeBlock({ status: 'error', errorMessage: 'Port not found' }))
    expect(w.text()).toContain('Port not found')
  })

  it('does not show error message when status is not error', () => {
    const w = mountNode(makeBlock({ status: 'connected', errorMessage: 'Port not found' }))
    expect(w.find('.block-error-msg').exists()).toBe(false)
  })

  it('shows analysis hint for analysis blocks', () => {
    const w = mountNode(makeBlock({ category: 'analysis', type: 'line-chart' }))
    expect(w.find('.analysis-hint').exists()).toBe(true)
  })

  it('does not show analysis hint for input blocks', () => {
    const w = mountNode(makeBlock({ category: 'input' }))
    expect(w.find('.analysis-hint').exists()).toBe(false)
  })

  // ── Config button ─────────────────────────────────────────────────────────────

  it('shows config button for blocks with a config component', () => {
    const w = mountNode(makeBlock({ type: 'uart', category: 'input' }))
    expect(w.find('button[title="Open configuration"]').exists()).toBe(true)
  })

  it('does not show config button for analysis blocks (no config component)', () => {
    const w = mountNode(makeBlock({ type: 'line-chart', category: 'analysis' }))
    expect(w.find('button[title="Open configuration"]').exists()).toBe(false)
  })

  // ── Delete ────────────────────────────────────────────────────────────────────

  it('calls removeBlock when delete button is clicked', async () => {
    const w = mountNode(makeBlock({ id: 'block-xyz' }))
    await w.find('button[title="Delete block"]').trigger('click')
    expect(mocks.removeBlock).toHaveBeenCalledWith('block-xyz')
  })

  // ── Double-click behaviour ────────────────────────────────────────────────────

  it('double-click on analysis block calls enterFullscreen', async () => {
    const w = mountNode(makeBlock({ id: 'chart-1', category: 'analysis', type: 'line-chart' }))
    await w.find('.block-node').trigger('dblclick')
    expect(mocks.enterFullscreen).toHaveBeenCalledWith('chart-1')
  })

  it('double-click on input block with config does not enter fullscreen', async () => {
    const w = mountNode(makeBlock({ category: 'input', type: 'uart' }))
    await w.find('.block-node').trigger('dblclick')
    expect(mocks.enterFullscreen).not.toHaveBeenCalled()
  })

  it('double-click on analysis block does NOT open dialog', async () => {
    const w = mountNode(makeBlock({ category: 'analysis', type: 'line-chart' }))
    await w.find('.block-node').trigger('dblclick')
    expect(mocks.enterFullscreen).toHaveBeenCalled()
    // No config dialog for analysis blocks
    expect(w.find('[header]').exists()).toBe(false)
  })

  // ── Config dialog Save / Cancel (bug reproduction) ───────────────────────────
  // These tests reproduce the reported issue: changes were not reliably preserved
  // because (a) there were no explicit Save/Cancel buttons and (b) auto-save via
  // @blur did not fire when the dialog closed.

  it('config dialog is not rendered before it is opened', () => {
    const w = mountNode(makeBlock({ type: 'simulator', category: 'input' }))
    expect(w.find('.dialog-stub').exists()).toBe(false)
  })

  it('config dialog renders after clicking the config button', async () => {
    const w = mountNode(makeBlock({ type: 'simulator', category: 'input' }))
    await w.find('button[title="Open configuration"]').trigger('click')
    expect(w.find('.dialog-stub').exists()).toBe(true)
  })

  it('config dialog renders after double-clicking an input block', async () => {
    const w = mountNode(makeBlock({ type: 'uart', category: 'input' }))
    await w.find('.block-node').trigger('dblclick')
    expect(w.find('.dialog-stub').exists()).toBe(true)
  })

  it('config dialog has a Save button', async () => {
    const w = mountNode(makeBlock({ type: 'simulator', category: 'input' }))
    await w.find('button[title="Open configuration"]').trigger('click')
    const buttons = w.findAll('button')
    const labels = buttons.map(b => b.text())
    expect(labels).toContain('Save')
  })

  it('config dialog has a Cancel button', async () => {
    const w = mountNode(makeBlock({ type: 'simulator', category: 'input' }))
    await w.find('button[title="Open configuration"]').trigger('click')
    const buttons = w.findAll('button')
    const labels = buttons.map(b => b.text())
    expect(labels).toContain('Cancel')
  })

  it('clicking Save calls the config component save and closes the dialog', async () => {
    const w = mountNode(makeBlock({ type: 'simulator', category: 'input' }))
    await w.find('button[title="Open configuration"]').trigger('click')
    expect(w.find('.dialog-stub').exists()).toBe(true)

    const saveBtn = w.findAll('button').find(b => b.text() === 'Save')
    expect(saveBtn).toBeDefined()
    await saveBtn!.trigger('click')
    await flushPromises()

    expect(mocks.configSave).toHaveBeenCalledOnce()
    // Dialog should close after save
    expect(w.find('.dialog-stub').exists()).toBe(false)
  })

  it('clicking Cancel closes the dialog without calling config save', async () => {
    const w = mountNode(makeBlock({ type: 'simulator', category: 'input' }))
    await w.find('button[title="Open configuration"]').trigger('click')
    expect(w.find('.dialog-stub').exists()).toBe(true)

    const cancelBtn = w.findAll('button').find(b => b.text() === 'Cancel')
    expect(cancelBtn).toBeDefined()
    await cancelBtn!.trigger('click')

    expect(mocks.configSave).not.toHaveBeenCalled()
    // Dialog should close after cancel
    expect(w.find('.dialog-stub').exists()).toBe(false)
  })

  it('config dialog can be reopened after being closed', async () => {
    const w = mountNode(makeBlock({ type: 'simulator', category: 'input' }))

    // Open
    await w.find('button[title="Open configuration"]').trigger('click')
    expect(w.find('.dialog-stub').exists()).toBe(true)

    // Cancel (close)
    const cancelBtn = w.findAll('button').find(b => b.text() === 'Cancel')!
    await cancelBtn.trigger('click')
    expect(w.find('.dialog-stub').exists()).toBe(false)

    // Re-open — config component must be freshly mounted with current props
    await w.find('button[title="Open configuration"]').trigger('click')
    expect(w.find('.dialog-stub').exists()).toBe(true)
    expect(w.find('.config-stub').exists()).toBe(true)
  })
})
