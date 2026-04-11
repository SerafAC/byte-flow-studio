export interface UartParams {
  port: string
  baudRate: number
  dataBits: number
  stopBits: number
  parity: 'none' | 'odd' | 'even'
}

export interface SimulatorParams {
  waveform: 'sine' | 'square' | 'sawtooth' | 'noise'
  frequencyHz: number
  amplitude: number
  offset: number
  sampleRateHz: number
}

export interface WebSocketParams {
  url: string
  subprotocol: string
  reconnectIntervalMs: number
}

export interface MovingAverageParams {
  windowSize: number
}

export interface FFTParams {
  windowSize: number
  windowFunction: 'none' | 'hann' | 'hamming'
}

export interface ScalingParams {
  scale: number
  offset: number
}

export interface ByteParserParams {
  format: 'float32-le' | 'float32-be' | 'int16-le' | 'int16-be' | 'uint8'
  channels: number
  frameSize: number
}

export interface SamplerParams {
  mode: 'every-n-samples' | 'first-in-window' | 'last-in-window' | 'first'
  n: number
  intervalMs: number
}

export interface FilterParams {
  mode: 'lowpass' | 'highpass' | 'bandpass' | 'bandstop'
  cutoffHz: number
  cutoffLowHz: number
  cutoffHighHz: number
  order: number
  sampleRateHz: number
}

export interface MultiplyParams {
  inputCount: number
}

export interface BluetoothParams {
  deviceAddress: string
  serialPort: string
}
