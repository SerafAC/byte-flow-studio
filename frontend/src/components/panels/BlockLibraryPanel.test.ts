import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BlockLibraryPanel from './BlockLibraryPanel.vue'

vi.mock('../../services/wails', () => ({
  getAvailableBlockTypes: vi.fn().mockResolvedValue([]),
}))

const globalStubs = {
  InputText: { template: '<input />' },
}

function mountPanel() {
  return mount(BlockLibraryPanel, { global: { stubs: globalStubs } })
}

describe('BlockLibraryPanel', () => {
  // ── Close button (T021) ───────────────────────────────────────────────────
  // These tests MUST FAIL until T025 is implemented (✕ button + defineEmits close).

  it('renders a close button', () => {
    const wrapper = mountPanel()
    expect(wrapper.find('button[title="Close panel"]').exists()).toBe(true)
  })

  it('emits "close" when the close button is clicked', async () => {
    const wrapper = mountPanel()
    await wrapper.find('button[title="Close panel"]').trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
