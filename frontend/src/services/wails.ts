/**
 * Typed wrappers over Wails v3 generated bindings.
 * All service methods are exposed here so components never import bindings directly.
 */

// ── Types ─────────────────────────────────────────────────────────────────────

export interface BlockDef {
  id: string
  type: string
  category: 'input' | 'processing' | 'analysis'
  label: string
  params: Record<string, unknown>
  positionX: number
  positionY: number
  status?: string
  errorMessage?: string
}

export interface ConnectionDef {
  id: string
  fromBlockId: string
  fromPortId: string
  toBlockId: string
  toPortId: string
}

export interface SessionConfig {
  storeRaw: boolean
  storeProcessed: boolean
  maxSessions: number
}

export interface Workflow {
  id: string
  name: string
  createdAt: number
  updatedAt: number
  blocks: BlockDef[]
  connections: ConnectionDef[]
  sessionConfig: SessionConfig
}

export interface BlockTypeDescriptor {
  type: string
  category: 'input' | 'processing' | 'analysis'
  label: string
  description: string
  defaultParams: Record<string, unknown>
}

export interface SerialPortInfo {
  name: string
  vendorId: string
  productId: string
  description: string
}

export interface SessionMeta {
  id: string
  startTime: number
  endTime: number
  endReason: string
  rawLayerEnabled: boolean
  processedLayerEnabled: boolean
  dataVolume: number
}

export interface StorageEstimate {
  currentTotalBytes: number
  averageSessionBytes: number
  projectedMaxBytes: number
  warningThresholdBytes: number
  exceedsThreshold: boolean
}

// ── WorkflowService bindings ──────────────────────────────────────────────────

let _WorkflowService: Record<string, (...args: unknown[]) => Promise<unknown>>

async function workflowSvc() {
  if (!_WorkflowService) {
    // Wails v3 generates bindings at frontend/bindings/
    // We use dynamic import so the app builds even before bindings are generated.
    try {
      const mod = await import('../../bindings/byteflow-studio/services/workflowservice')
      _WorkflowService = mod
    } catch {
      // Bindings not generated yet (dev mode before first wails3 build)
      _WorkflowService = createStubService()
    }
  }
  return _WorkflowService
}

let _PipelineService: Record<string, (...args: unknown[]) => Promise<unknown>>

async function pipelineSvc() {
  if (!_PipelineService) {
    try {
      const mod = await import('../../bindings/byteflow-studio/services/pipelineservice')
      _PipelineService = mod
    } catch {
      _PipelineService = createStubService()
    }
  }
  return _PipelineService
}

let _SessionService: Record<string, (...args: unknown[]) => Promise<unknown>>

async function sessionSvc() {
  if (!_SessionService) {
    try {
      const mod = await import('../../bindings/byteflow-studio/services/sessionservice')
      _SessionService = mod
    } catch {
      _SessionService = createStubService()
    }
  }
  return _SessionService
}

function createStubService(): Record<string, (...args: unknown[]) => Promise<unknown>> {
  return new Proxy({}, {
    get: (_target, prop) => async (..._args: unknown[]) => {
      console.warn(`[wails.ts] Binding not available: ${String(prop)}`)
      return null
    },
  }) as Record<string, (...args: unknown[]) => Promise<unknown>>
}

// ── WorkflowService API ───────────────────────────────────────────────────────

export async function getWorkflow(): Promise<Workflow> {
  const svc = await workflowSvc()
  return svc.GetWorkflow() as Promise<Workflow>
}

export async function saveWorkflow(path: string): Promise<void> {
  const svc = await workflowSvc()
  return svc.SaveWorkflow(path) as Promise<void>
}

export async function loadWorkflow(path: string): Promise<Workflow> {
  const svc = await workflowSvc()
  return svc.LoadWorkflow(path) as Promise<Workflow>
}

export async function recoverWorkflow(path: string): Promise<Workflow> {
  const svc = await workflowSvc()
  return svc.RecoverWorkflow(path) as Promise<Workflow>
}

export async function addBlock(blockType: string, x: number, y: number): Promise<BlockDef> {
  const svc = await workflowSvc()
  return svc.AddBlock(blockType, x, y) as Promise<BlockDef>
}

export async function removeBlock(blockId: string): Promise<void> {
  const svc = await workflowSvc()
  return svc.RemoveBlock(blockId) as Promise<void>
}

export async function addConnection(
  fromBlockId: string, fromPortId: string,
  toBlockId: string, toPortId: string
): Promise<ConnectionDef> {
  const svc = await workflowSvc()
  return svc.AddConnection(fromBlockId, fromPortId, toBlockId, toPortId) as Promise<ConnectionDef>
}

export async function removeConnection(connectionId: string): Promise<void> {
  const svc = await workflowSvc()
  return svc.RemoveConnection(connectionId) as Promise<void>
}

export async function updateBlockParams(blockId: string, params: Record<string, unknown>): Promise<void> {
  const svc = await workflowSvc()
  return svc.UpdateBlockParams(blockId, params) as Promise<void>
}

export async function updateBlockPosition(blockId: string, x: number, y: number): Promise<void> {
  const svc = await workflowSvc()
  return svc.UpdateBlockPosition(blockId, x, y) as Promise<void>
}

export async function getAvailableBlockTypes(): Promise<BlockTypeDescriptor[]> {
  const svc = await workflowSvc()
  return (svc.GetAvailableBlockTypes() as Promise<BlockTypeDescriptor[]>)
}

export async function listSerialPorts(): Promise<SerialPortInfo[]> {
  const svc = await workflowSvc()
  return svc.ListSerialPorts() as Promise<SerialPortInfo[]>
}

// ── PipelineService API ───────────────────────────────────────────────────────

export async function startPipeline(): Promise<void> {
  const svc = await pipelineSvc()
  return svc.Start() as Promise<void>
}

export async function pausePipeline(): Promise<void> {
  const svc = await pipelineSvc()
  return svc.Pause() as Promise<void>
}

export async function resumePipeline(): Promise<void> {
  const svc = await pipelineSvc()
  return svc.Resume() as Promise<void>
}

export async function stopPipeline(): Promise<void> {
  const svc = await pipelineSvc()
  return svc.Stop() as Promise<void>
}

export async function getPipelineState(): Promise<string> {
  const svc = await pipelineSvc()
  return svc.GetState() as Promise<string>
}

export async function getBlockErrors(): Promise<Record<string, string>> {
  const svc = await pipelineSvc()
  return svc.GetBlockErrors() as Promise<Record<string, string>>
}

// ── SessionService API ────────────────────────────────────────────────────────

export async function listSessions(): Promise<SessionMeta[]> {
  const svc = await sessionSvc()
  return (svc.ListSessions() as Promise<SessionMeta[]>)
}

export async function getActiveSessionId(): Promise<string> {
  const svc = await sessionSvc()
  return svc.GetActiveSessionID() as Promise<string>
}

export async function setViewSession(sessionId: string): Promise<void> {
  const svc = await sessionSvc()
  return svc.SetViewSession(sessionId) as Promise<void>
}

export async function deleteSession(sessionId: string): Promise<void> {
  const svc = await sessionSvc()
  return svc.DeleteSession(sessionId) as Promise<void>
}

export async function getStorageEstimate(): Promise<StorageEstimate> {
  const svc = await sessionSvc()
  return svc.GetStorageEstimate() as Promise<StorageEstimate>
}

export async function getSessionConfig(): Promise<SessionConfig> {
  const svc = await sessionSvc()
  return svc.GetSessionConfig() as Promise<SessionConfig>
}

export async function updateSessionConfig(cfg: SessionConfig): Promise<void> {
  const svc = await sessionSvc()
  return svc.UpdateSessionConfig(cfg) as Promise<void>
}
