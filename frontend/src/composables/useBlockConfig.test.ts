import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { useBlockConfig } from './useBlockConfig'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mocks = vi.hoisted(() => ({
  storeUpdateBlockParams: vi.fn(),
  wailsUpdateBlockParams: vi.fn(),
}))

// Bug reproduction: useBlockConfig was calling wails.updateBlockParams directly,
// bypassing the Pinia workflow store.  As a result the store's blocks[].params
// was never updated after a save, so the next time the config dialog opened it
// received stale props from the store and showed old/default values.
vi.mock('../stores/workflow', () => ({
  useWorkflowStore: () => ({
    updateBlockParams: mocks.storeUpdateBlockParams,
  }),
}))

// Keep the wails mock to confirm it is NOT called directly from this composable.
vi.mock('../services/wails', () => ({
  updateBlockParams: mocks.wailsUpdateBlockParams,
}))

// ── Helper: mount composable inside a component ────────────────────────────

function mountComposable(blockId: string, getParams: () => Record<string, unknown>) {
  const Wrapper = defineComponent({
    setup() {
      return useBlockConfig(blockId, getParams)
    },
    template: '<div />',
  })
  return mount(Wrapper)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('useBlockConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.storeUpdateBlockParams.mockResolvedValue(undefined)
    mocks.wailsUpdateBlockParams.mockResolvedValue(undefined)
  })

  // Reproduces: "changed values are reflected in the processing flow, however
  // opening a configuration of the block again shows old/default values"
  // Root cause: save bypassed the Pinia store, so store.blocks[].params stayed
  // stale and the next dialog open received the old props.
  it('routes save through the workflow store so store params stay in sync', async () => {
    const wrapper = mountComposable('block-1', () => ({ scale: 2, offset: 0 }))
    await wrapper.vm.save()
    await flushPromises()

    expect(mocks.storeUpdateBlockParams).toHaveBeenCalledWith('block-1', { scale: 2, offset: 0 })
    // Must NOT call the wails service directly — that path bypasses the store.
    expect(mocks.wailsUpdateBlockParams).not.toHaveBeenCalled()
  })

  it('calls getParams at save time, not at construction time', async () => {
    const value = ref(1)
    const wrapper = mountComposable('block-2', () => ({ value: value.value }))

    value.value = 99
    await wrapper.vm.save()
    await flushPromises()

    expect(mocks.storeUpdateBlockParams).toHaveBeenCalledWith('block-2', { value: 99 })
  })

  it('calls updateBlockParams once per save call', async () => {
    const wrapper = mountComposable('block-3', () => ({ x: 1 }))
    await wrapper.vm.save()
    await wrapper.vm.save()
    await flushPromises()
    expect(mocks.storeUpdateBlockParams).toHaveBeenCalledTimes(2)
  })

  it('calls updateBlockParams with correct blockId', async () => {
    const wrapper = mountComposable('my-block-id', () => ({ windowSize: 512 }))
    await wrapper.vm.save()
    await flushPromises()
    expect(mocks.storeUpdateBlockParams).toHaveBeenCalledWith('my-block-id', expect.anything())
  })
})
