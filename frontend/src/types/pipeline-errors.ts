const TOPOLOGY_NO_INPUT = 'Input and one Analysis block'
const TOPOLOGY_UNREACHABLE = 'not reachable'
const CORRUPTED_FILE = 'not a valid byteflow file'
const CYCLE_DETECTED = 'cycle'

export function isPipelineTopologyError(message: string): boolean {
  return message.includes(TOPOLOGY_NO_INPUT) || message.includes(TOPOLOGY_UNREACHABLE)
}

export function isCorruptedFileError(message: string): boolean {
  return message.includes(CORRUPTED_FILE)
}

export function isCycleError(message: string): boolean {
  return message.includes(CYCLE_DETECTED)
}
