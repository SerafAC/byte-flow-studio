import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import MainView from './MainView.vue'

// ── Hoisted mock state (available before any imports are resolved) ─────────────

const mocks = vi.hoisted(() => ({
  toastAdd: vi.fn(),
  confirmRequire: vi.fn(),
  openFile: vi.fn(),
  saveFile: vi.fn(),
  recoverWorkflow: vi.fn(),
  saveWorkflow: vi.fn(),
  exitFullscreen: vi.fn(),
  // Plain object — value is read at setup() call time, so set before mount()
  fullscreenBlockId: { value: null as string | null },
  workflowStore: {
    blocks: [] as { id: string; type: string; params?: Record<string, unknown> }[],
    isDirty: false,
    loadWorkflow: vi.fn(),
    loadFromFile: vi.fn(),
  },
  pipelineStore: {
    flowState: 'idle' as string,
    blockErrors: {} as Record<string, string>,
    sessionId: null as string | null,
    canStart: true,
    canPause: false,
    canResume: false,
    canStop: false,
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
    startFlow: vi.fn(),
    pauseFlow: vi.fn(),
    resumeFlow: vi.fn(),
    stopFlow: vi.fn(),
  },
  sessionStore: {
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
    loadSessions: vi.fn(),
  },
}))

vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: mocks.toastAdd }) }))
vi.mock('primevue/useconfirm', () => ({ useConfirm: () => ({ require: mocks.confirmRequire }) }))

vi.mock('@wailsio/runtime', () => ({
  Dialogs: { OpenFile: mocks.openFile, SaveFile: mocks.saveFile },
}))

vi.mock('../services/wails', () => ({
  recoverWorkflow: mocks.recoverWorkflow,
  saveWorkflow: mocks.saveWorkflow,
}))

vi.mock('../composables/useFullscreen', () => ({
  useFullscreen: () => ({
    fullscreenBlockId: mocks.fullscreenBlockId,
    exitFullscreen: mocks.exitFullscreen,
  }),
}))

vi.mock('../stores/workflow', () => ({ useWorkflowStore: () => mocks.workflowStore }))
vi.mock('../stores/pipeline', () => ({ usePipelineStore: () => mocks.pipelineStore }))
vi.mock('../stores/session', () => ({ useSessionStore: () => mocks.sessionStore }))

// ── Test helpers ──────────────────────────────────────────────────────────────

const globalStubs = {
  // Stub heavy child components
  BlockLibraryPanel: { template: '<div class="block-library" />' },
  WorkflowCanvas: { template: '<div class="workflow-canvas" />' },
  SessionPanel: { template: '<div class="session-panel" />' },
  LineChartBlock: { template: '<div class="line-chart-block" />' },
  ValueDisplayBlock: { template: '<div />' },
  DataTableBlock: { template: '<div />' },
  BarChartBlock: { template: '<div />' },
  FftSpectrumBlock: { template: '<div />' },
  // Stub PrimeVue components with testable HTML
  ConfirmDialog: { template: '<div />' },
  Button: {
    inheritAttrs: false,
    template: '<button v-bind="$attrs">{{ $attrs.label }}</button>',
  },
  Tag: {
    inheritAttrs: false,
    template: '<span class="tag">{{ $attrs.value }}</span>',
  },
  // Render Teleport content inline so wrapper.find() works
  Teleport: true,
}

function findButton(wrapper: ReturnType<typeof mountView>, label: string) {
  return wrapper.findAll('button').find(b => b.text() === label)
}

// ── Setup ─────────────────────────────────────────────────────────────────────

// Track the current wrapper so we can unmount it after each test, preventing
// stale keydown listeners from accumulating on the shared document.
let currentWrapper: ReturnType<typeof mountView> | null = null

function mountView() {
  currentWrapper = mount(MainView, { global: { stubs: globalStubs } })
  return currentWrapper
}

afterEach(() => {
  currentWrapper?.unmount()
  currentWrapper = null
})

beforeEach(() => {
  vi.clearAllMocks()

  // Reset store state
  Object.assign(mocks.workflowStore, {
    blocks: [],
    isDirty: false,
  })
  mocks.workflowStore.loadWorkflow.mockResolvedValue(undefined)
  mocks.workflowStore.loadFromFile.mockResolvedValue(undefined)

  Object.assign(mocks.pipelineStore, {
    flowState: 'idle',
    blockErrors: {},
    sessionId: null,
    canStart: true,
    canPause: false,
    canResume: false,
    canStop: false,
  })
  mocks.pipelineStore.subscribe.mockReturnValue(undefined)
  mocks.pipelineStore.unsubscribe.mockReturnValue(undefined)
  mocks.pipelineStore.startFlow.mockResolvedValue(undefined)
  mocks.pipelineStore.pauseFlow.mockResolvedValue(undefined)
  mocks.pipelineStore.resumeFlow.mockResolvedValue(undefined)
  mocks.pipelineStore.stopFlow.mockResolvedValue(undefined)

  mocks.sessionStore.subscribe.mockReturnValue(undefined)
  mocks.sessionStore.unsubscribe.mockReturnValue(undefined)
  mocks.sessionStore.loadSessions.mockResolvedValue(undefined)

  mocks.openFile.mockResolvedValue(null)
  mocks.saveFile.mockResolvedValue(null)
  mocks.recoverWorkflow.mockResolvedValue(undefined)
  mocks.saveWorkflow.mockResolvedValue(undefined)
  mocks.fullscreenBlockId.value = null
})

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('MainView', () => {

  // ── Rendering ──────────────────────────────────────────────────────────────

  describe('rendering', () => {
    it('shows the app title', () => {
      const wrapper = mountView()
      expect(wrapper.find('.menu-bar').text()).toContain('ByteFlow Studio')
    })

    it('renders the three-panel layout', () => {
      const wrapper = mountView()
      expect(wrapper.find('.block-library').exists()).toBe(true)
      expect(wrapper.find('.workflow-canvas').exists()).toBe(true)
      expect(wrapper.find('.session-panel').exists()).toBe(true)
    })

    it('shows the flow state in the status bar', () => {
      mocks.pipelineStore.flowState = 'running'
      const wrapper = mountView()
      expect(wrapper.find('.status-bar').text()).toContain('running')
    })

    it('shows session id in the status bar when present', () => {
      mocks.pipelineStore.sessionId = 'sess-abc'
      const wrapper = mountView()
      expect(wrapper.find('.status-bar').text()).toContain('sess-abc')
    })

    it('omits session id from status bar when absent', () => {
      mocks.pipelineStore.sessionId = null
      const wrapper = mountView()
      expect(wrapper.find('.status-bar').text()).not.toContain('Session:')
    })

    it('shows the pipeline state tag in uppercase', () => {
      mocks.pipelineStore.flowState = 'running'
      const wrapper = mountView()
      expect(wrapper.find('.tag').text()).toBe('RUNNING')
    })
  })

  // ── Pipeline controls ──────────────────────────────────────────────────────

  describe('pipeline controls', () => {
    it('Start button is enabled when canStart is true', () => {
      mocks.pipelineStore.canStart = true
      const wrapper = mountView()
      expect(findButton(wrapper, 'Start')?.attributes('disabled')).toBeUndefined()
    })

    it('Start button is disabled when canStart is false', () => {
      mocks.pipelineStore.canStart = false
      const wrapper = mountView()
      expect(findButton(wrapper, 'Start')?.attributes('disabled')).toBeDefined()
    })

    it('Pause button is disabled when canPause is false', () => {
      mocks.pipelineStore.canPause = false
      const wrapper = mountView()
      expect(findButton(wrapper, 'Pause')?.attributes('disabled')).toBeDefined()
    })

    it('Resume button is disabled when canResume is false', () => {
      mocks.pipelineStore.canResume = false
      const wrapper = mountView()
      expect(findButton(wrapper, 'Resume')?.attributes('disabled')).toBeDefined()
    })

    it('Stop button is disabled when canStop is false', () => {
      mocks.pipelineStore.canStop = false
      const wrapper = mountView()
      expect(findButton(wrapper, 'Stop')?.attributes('disabled')).toBeDefined()
    })

    it('clicking Start calls pipelineStore.startFlow', async () => {
      const wrapper = mountView()
      await findButton(wrapper, 'Start')?.trigger('click')
      await flushPromises()
      expect(mocks.pipelineStore.startFlow).toHaveBeenCalled()
    })

    it('clicking Pause calls pipelineStore.pauseFlow', async () => {
      mocks.pipelineStore.canPause = true
      const wrapper = mountView()
      await findButton(wrapper, 'Pause')?.trigger('click')
      await flushPromises()
      expect(mocks.pipelineStore.pauseFlow).toHaveBeenCalled()
    })

    it('clicking Resume calls pipelineStore.resumeFlow', async () => {
      mocks.pipelineStore.canResume = true
      const wrapper = mountView()
      await findButton(wrapper, 'Resume')?.trigger('click')
      await flushPromises()
      expect(mocks.pipelineStore.resumeFlow).toHaveBeenCalled()
    })

    it('clicking Stop calls pipelineStore.stopFlow', async () => {
      mocks.pipelineStore.canStop = true
      const wrapper = mountView()
      await findButton(wrapper, 'Stop')?.trigger('click')
      await flushPromises()
      expect(mocks.pipelineStore.stopFlow).toHaveBeenCalled()
    })
  })

  // ── onStart error handling ─────────────────────────────────────────────────

  describe('onStart', () => {
    it('shows topology banner for "Input and one Analysis block" error', async () => {
      mocks.pipelineStore.startFlow.mockRejectedValue(new Error('Input and one Analysis block required'))
      const wrapper = mountView()
      await findButton(wrapper, 'Start')?.trigger('click')
      await flushPromises()
      expect(wrapper.find('.error-banner--topology').exists()).toBe(true)
      expect(wrapper.find('.error-banner--topology').text()).toContain('Input and one Analysis block')
    })

    it('shows topology banner for "not reachable" error', async () => {
      mocks.pipelineStore.startFlow.mockRejectedValue(new Error('Block B is not reachable from input'))
      const wrapper = mountView()
      await findButton(wrapper, 'Start')?.trigger('click')
      await flushPromises()
      expect(wrapper.find('.error-banner--topology').exists()).toBe(true)
    })

    it('shows a toast for non-topology start errors', async () => {
      mocks.pipelineStore.startFlow.mockRejectedValue(new Error('port not found'))
      const wrapper = mountView()
      await findButton(wrapper, 'Start')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'error', summary: 'Start failed' }),
      )
      expect(wrapper.find('.error-banner--topology').exists()).toBe(false)
    })

    it('clears the topology banner on a successful start', async () => {
      mocks.pipelineStore.startFlow.mockRejectedValueOnce(new Error('Input and one Analysis block required'))
      const wrapper = mountView()
      await findButton(wrapper, 'Start')?.trigger('click')
      await flushPromises()
      expect(wrapper.find('.error-banner--topology').exists()).toBe(true)

      mocks.pipelineStore.startFlow.mockResolvedValueOnce(undefined)
      await findButton(wrapper, 'Start')?.trigger('click')
      await flushPromises()
      expect(wrapper.find('.error-banner--topology').exists()).toBe(false)
    })

    it('topology banner can be dismissed with the ✕ button', async () => {
      mocks.pipelineStore.startFlow.mockRejectedValue(new Error('Input and one Analysis block required'))
      const wrapper = mountView()
      await findButton(wrapper, 'Start')?.trigger('click')
      await flushPromises()
      await wrapper.find('.error-banner-dismiss').trigger('click')
      expect(wrapper.find('.error-banner--topology').exists()).toBe(false)
    })
  })

  // ── Pause / Resume / Stop error toasts ────────────────────────────────────

  describe('pipeline error toasts', () => {
    it('shows a toast when pauseFlow throws', async () => {
      mocks.pipelineStore.canPause = true
      mocks.pipelineStore.pauseFlow.mockRejectedValue(new Error('pause err'))
      const wrapper = mountView()
      await findButton(wrapper, 'Pause')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'error', summary: 'Pause failed' }),
      )
    })

    it('shows a toast when resumeFlow throws', async () => {
      mocks.pipelineStore.canResume = true
      mocks.pipelineStore.resumeFlow.mockRejectedValue(new Error('resume err'))
      const wrapper = mountView()
      await findButton(wrapper, 'Resume')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'error', summary: 'Resume failed' }),
      )
    })

    it('shows a toast when stopFlow throws', async () => {
      mocks.pipelineStore.canStop = true
      mocks.pipelineStore.stopFlow.mockRejectedValue(new Error('stop err'))
      const wrapper = mountView()
      await findButton(wrapper, 'Stop')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'error', summary: 'Stop failed' }),
      )
    })
  })

  // ── Block error banner ─────────────────────────────────────────────────────

  describe('block error banner', () => {
    it('shows block errors when flowState is error and blockErrors are present', () => {
      mocks.pipelineStore.flowState = 'error'
      mocks.pipelineStore.blockErrors = { 'blk-1': 'overflow detected' }
      const wrapper = mountView()
      const banner = wrapper.find('.error-banner:not(.error-banner--topology)')
      expect(banner.exists()).toBe(true)
      expect(banner.text()).toContain('blk-1')
      expect(banner.text()).toContain('overflow detected')
    })

    it('does not show block error banner when flowState is not error', () => {
      mocks.pipelineStore.flowState = 'idle'
      mocks.pipelineStore.blockErrors = { 'blk-1': 'overflow detected' }
      const wrapper = mountView()
      expect(wrapper.find('.error-banner:not(.error-banner--topology)').exists()).toBe(false)
    })

    it('does not show block error banner when blockErrors is empty', () => {
      mocks.pipelineStore.flowState = 'error'
      mocks.pipelineStore.blockErrors = {}
      const wrapper = mountView()
      expect(wrapper.find('.error-banner:not(.error-banner--topology)').exists()).toBe(false)
    })
  })

  // ── Lifecycle ──────────────────────────────────────────────────────────────

  describe('lifecycle', () => {
    it('subscribes to pipeline and session stores on mount', async () => {
      mountView()
      await flushPromises()
      expect(mocks.pipelineStore.subscribe).toHaveBeenCalled()
      expect(mocks.sessionStore.subscribe).toHaveBeenCalled()
    })

    it('loads workflow and sessions on mount', async () => {
      mountView()
      await flushPromises()
      expect(mocks.workflowStore.loadWorkflow).toHaveBeenCalled()
      expect(mocks.sessionStore.loadSessions).toHaveBeenCalled()
    })

    it('unsubscribes from stores on unmount', async () => {
      const wrapper = mountView()
      await flushPromises()
      wrapper.unmount()
      expect(mocks.pipelineStore.unsubscribe).toHaveBeenCalled()
      expect(mocks.sessionStore.unsubscribe).toHaveBeenCalled()
    })
  })

  // ── Open file ──────────────────────────────────────────────────────────────

  describe('onOpen', () => {
    it('opens a file dialog when workflow is not dirty', async () => {
      mocks.workflowStore.isDirty = false
      const wrapper = mountView()
      await findButton(wrapper, 'Open')?.trigger('click')
      await flushPromises()
      expect(mocks.openFile).toHaveBeenCalled()
    })

    it('requires confirmation when workflow is dirty', async () => {
      mocks.workflowStore.isDirty = true
      const wrapper = mountView()
      await findButton(wrapper, 'Open')?.trigger('click')
      await flushPromises()
      expect(mocks.confirmRequire).toHaveBeenCalledWith(
        expect.objectContaining({ header: 'Unsaved Changes' }),
      )
      expect(mocks.openFile).not.toHaveBeenCalled()
    })

    it('shows a success toast after a file is loaded', async () => {
      mocks.openFile.mockResolvedValue('/my/workflow.byteflow')
      const wrapper = mountView()
      await findButton(wrapper, 'Open')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'success', summary: 'Opened' }),
      )
    })

    it('shows the recovery dialog for a corrupted file', async () => {
      mocks.openFile.mockResolvedValue('/bad.byteflow')
      mocks.workflowStore.loadFromFile.mockRejectedValue(new Error('not a valid byteflow file'))
      const wrapper = mountView()
      await findButton(wrapper, 'Open')?.trigger('click')
      await flushPromises()
      expect(wrapper.find('.recovery-dialog').exists()).toBe(true)
      expect(wrapper.text()).toContain('/bad.byteflow')
    })

    it('shows an error toast for a non-corruption load failure', async () => {
      mocks.openFile.mockResolvedValue('/bad.byteflow')
      mocks.workflowStore.loadFromFile.mockRejectedValue(new Error('permission denied'))
      const wrapper = mountView()
      await findButton(wrapper, 'Open')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'error', summary: 'Open failed' }),
      )
      expect(wrapper.find('.recovery-dialog').exists()).toBe(false)
    })

    it('does nothing if the file dialog is cancelled', async () => {
      mocks.openFile.mockResolvedValue(null)
      const wrapper = mountView()
      await findButton(wrapper, 'Open')?.trigger('click')
      await flushPromises()
      expect(mocks.workflowStore.loadFromFile).not.toHaveBeenCalled()
    })
  })

  // ── Recovery dialog ────────────────────────────────────────────────────────

  describe('recovery dialog', () => {
    async function triggerCorruptedOpen(wrapper: ReturnType<typeof mountView>) {
      mocks.openFile.mockResolvedValue('/corrupted.byteflow')
      mocks.workflowStore.loadFromFile.mockRejectedValue(new Error('not a valid byteflow file'))
      await findButton(wrapper, 'Open')?.trigger('click')
      await flushPromises()
    }

    it('Cancel hides the recovery dialog', async () => {
      const wrapper = mountView()
      await triggerCorruptedOpen(wrapper)
      await findButton(wrapper, 'Cancel')?.trigger('click')
      await flushPromises()
      expect(wrapper.find('.recovery-dialog').exists()).toBe(false)
    })

    it('Recovery Mode calls wails.recoverWorkflow with the file path', async () => {
      const wrapper = mountView()
      await triggerCorruptedOpen(wrapper)
      await findButton(wrapper, 'Recovery Mode')?.trigger('click')
      await flushPromises()
      expect(mocks.recoverWorkflow).toHaveBeenCalledWith('/corrupted.byteflow')
    })

    it('Recovery Mode shows a warn toast on success', async () => {
      const wrapper = mountView()
      await triggerCorruptedOpen(wrapper)
      await findButton(wrapper, 'Recovery Mode')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'warn', summary: 'Recovery Mode' }),
      )
    })

    it('Recovery Mode hides the dialog on completion', async () => {
      const wrapper = mountView()
      await triggerCorruptedOpen(wrapper)
      await findButton(wrapper, 'Recovery Mode')?.trigger('click')
      await flushPromises()
      expect(wrapper.find('.recovery-dialog').exists()).toBe(false)
    })

    it('Recovery Mode shows an error toast on failure', async () => {
      mocks.recoverWorkflow.mockRejectedValue(new Error('recovery fail'))
      const wrapper = mountView()
      await triggerCorruptedOpen(wrapper)
      await findButton(wrapper, 'Recovery Mode')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'error', summary: 'Recovery failed' }),
      )
    })

    it('Recovery Mode hides the dialog even when recovery fails', async () => {
      mocks.recoverWorkflow.mockRejectedValue(new Error('recovery fail'))
      const wrapper = mountView()
      await triggerCorruptedOpen(wrapper)
      await findButton(wrapper, 'Recovery Mode')?.trigger('click')
      await flushPromises()
      expect(wrapper.find('.recovery-dialog').exists()).toBe(false)
    })
  })

  // ── Save file ──────────────────────────────────────────────────────────────

  describe('onSave', () => {
    it('calls wails.saveWorkflow with the chosen path', async () => {
      mocks.saveFile.mockResolvedValue('/out/workflow.byteflow')
      const wrapper = mountView()
      await findButton(wrapper, 'Save')?.trigger('click')
      await flushPromises()
      expect(mocks.saveWorkflow).toHaveBeenCalledWith('/out/workflow.byteflow')
    })

    it('shows a success toast after saving', async () => {
      mocks.saveFile.mockResolvedValue('/out/workflow.byteflow')
      const wrapper = mountView()
      await findButton(wrapper, 'Save')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'success', summary: 'Saved' }),
      )
    })

    it('does nothing if the save dialog is cancelled', async () => {
      mocks.saveFile.mockResolvedValue(null)
      const wrapper = mountView()
      await findButton(wrapper, 'Save')?.trigger('click')
      await flushPromises()
      expect(mocks.saveWorkflow).not.toHaveBeenCalled()
    })

    it('shows an error toast when saveWorkflow throws', async () => {
      mocks.saveFile.mockResolvedValue('/out/workflow.byteflow')
      mocks.saveWorkflow.mockRejectedValue(new Error('disk full'))
      const wrapper = mountView()
      await findButton(wrapper, 'Save')?.trigger('click')
      await flushPromises()
      expect(mocks.toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ severity: 'error', summary: 'Save failed' }),
      )
    })
  })

  // ── Fullscreen overlay ─────────────────────────────────────────────────────

  describe('fullscreen', () => {
    it('does not show the fullscreen overlay when no block is fullscreened', () => {
      mocks.fullscreenBlockId.value = null
      const wrapper = mountView()
      expect(wrapper.find('.fullscreen-overlay').exists()).toBe(false)
    })

    it('shows the fullscreen overlay when a block is fullscreened', () => {
      mocks.workflowStore.blocks = [{ id: 'b1', type: 'line-chart', params: {} }]
      mocks.fullscreenBlockId.value = 'b1'
      const wrapper = mountView()
      expect(wrapper.find('.fullscreen-overlay').exists()).toBe(true)
    })

    it('does not show overlay for an unknown block type', () => {
      mocks.workflowStore.blocks = [{ id: 'b1', type: 'unknown-type', params: {} }]
      mocks.fullscreenBlockId.value = 'b1'
      const wrapper = mountView()
      expect(wrapper.find('.fullscreen-overlay').exists()).toBe(false)
    })

    it('Escape key calls exitFullscreen when a block is fullscreened', () => {
      mocks.fullscreenBlockId.value = 'b1'
      mountView()
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      expect(mocks.exitFullscreen).toHaveBeenCalled()
    })

    it('Escape key does nothing when no block is fullscreened', () => {
      mocks.fullscreenBlockId.value = null
      mountView()
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      expect(mocks.exitFullscreen).not.toHaveBeenCalled()
    })

    it('removes the Escape listener on unmount', () => {
      mocks.fullscreenBlockId.value = 'b1'
      const wrapper = mountView()
      wrapper.unmount()
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      expect(mocks.exitFullscreen).not.toHaveBeenCalled()
    })
  })
})
