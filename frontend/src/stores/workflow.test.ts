import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWorkflowStore } from './workflow'

// ── Hoisted mocks ─────────────────────────────────────────────────────────────

const mockWails = vi.hoisted(() => ({
  updateBlockSize: vi.fn(),
  getWorkflow: vi.fn(),
  addBlock: vi.fn(),
}))

vi.mock('../services/wails', () => mockWails)

// ── Helpers ───────────────────────────────────────────────────────────────────

function makeBlock(overrides = {}) {
  return {
    id: 'b1',
    type: 'line-chart',
    category: 'analysis' as const,
    label: 'Line Chart',
    params: {},
    positionX: 0,
    positionY: 0,
    ...overrides,
  }
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('workflow store – updateBlockSize (T010)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('calls wails.updateBlockSize with the correct arguments', async () => {
    mockWails.updateBlockSize.mockResolvedValue(undefined)

    const store = useWorkflowStore()
    // Pre-populate the store with a block so the in-memory update can be verified
    store.blocks = [makeBlock({ id: 'block-a', width: 200, height: 150 })]

    await store.updateBlockSize('block-a', 320, 200)

    expect(mockWails.updateBlockSize).toHaveBeenCalledOnce()
    expect(mockWails.updateBlockSize).toHaveBeenCalledWith('block-a', 320, 200)
  })

  it('updates the matching block width and height in the blocks array', async () => {
    mockWails.updateBlockSize.mockResolvedValue(undefined)

    const store = useWorkflowStore()
    store.blocks = [
      makeBlock({ id: 'block-a', width: 200, height: 150 }),
      makeBlock({ id: 'block-b', width: 300, height: 180 }),
    ]

    await store.updateBlockSize('block-a', 320, 240)

    const updated = store.blocks.find(b => b.id === 'block-a')
    expect(updated?.width).toBe(320)
    expect(updated?.height).toBe(240)

    // Other blocks must not be touched
    const other = store.blocks.find(b => b.id === 'block-b')
    expect(other?.width).toBe(300)
    expect(other?.height).toBe(180)
  })

  it('does not modify blocks if the id is not found', async () => {
    mockWails.updateBlockSize.mockResolvedValue(undefined)

    const store = useWorkflowStore()
    store.blocks = [makeBlock({ id: 'block-a', width: 200, height: 150 })]

    await store.updateBlockSize('nonexistent', 320, 240)

    // Block should be unchanged
    expect(store.blocks[0].width).toBe(200)
    expect(store.blocks[0].height).toBe(150)
  })
})
