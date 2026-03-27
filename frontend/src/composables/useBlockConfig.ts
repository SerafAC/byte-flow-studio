import { useWorkflowStore } from '../stores/workflow'

export function useBlockConfig(
  blockId: string,
  getParams: () => Record<string, unknown>
) {
  const workflowStore = useWorkflowStore()

  async function save(): Promise<void> {
    await workflowStore.updateBlockParams(blockId, getParams())
  }

  return { save }
}
