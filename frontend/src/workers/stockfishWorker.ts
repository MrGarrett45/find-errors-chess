/// <reference lib="webworker" />

type WorkerRequest =
  | { type: 'analyze'; fen: string; depth?: number; multipv?: number }
  | { type: 'stop' }

type WorkerResponse =
  | { type: 'info'; data: string }
  | { type: 'ready' }
  | { type: 'error'; error: string }

type EngineEvent = { data: string }

type EngineInstance = Worker

const DEFAULT_DEPTH = 14
const DEFAULT_MULTIPV = 3

let engine: EngineInstance | null = null
let readyResolvers: Array<() => void> = []

const sendMessage = (message: WorkerResponse) => {
  postMessage(message)
}

const flushReadyResolvers = () => {
  readyResolvers.forEach((resolve) => resolve())
  readyResolvers = []
}

const handleEngineMessage = (event: EngineEvent | string) => {
  const text = typeof event === 'string' ? event : event.data
  sendMessage({ type: 'info', data: text })

  if (text === 'readyok') {
    sendMessage({ type: 'ready' })
    flushReadyResolvers()
  }
}

const ensureEngine = (): EngineInstance => {
  if (engine) return engine

  try {
    const engineUrl = new URL('/stockfish.wasm.js', self.location.origin)
    engine = new Worker(engineUrl.toString())
    engine.onmessage = handleEngineMessage
    engine.onerror = () => {
      sendMessage({ type: 'error', error: 'Stockfish worker failed to load' })
    }
    engine.postMessage('uci')
  } catch (err) {
    const error =
      err instanceof Error ? err.message : 'Failed to initialize Stockfish engine'
    sendMessage({ type: 'error', error })
    throw err
  }

  return engine
}

const waitForReady = (inst: EngineInstance): Promise<void> =>
  new Promise((resolve) => {
    readyResolvers.push(resolve)
    inst.postMessage('isready')
  })

const handleAnalyze = async (msg: Extract<WorkerRequest, { type: 'analyze' }>) => {
  const inst = ensureEngine()
  inst.postMessage('stop')
  await waitForReady(inst)

  const multipv = Math.max(1, msg.multipv ?? DEFAULT_MULTIPV)
  const depth = Math.max(1, msg.depth ?? DEFAULT_DEPTH)

  inst.postMessage(`setoption name MultiPV value ${multipv}`)
  inst.postMessage('ucinewgame')
  inst.postMessage(`position fen ${msg.fen}`)
  inst.postMessage(`go depth ${depth}`)
}

onmessage = (event: MessageEvent<WorkerRequest>) => {
  const msg = event.data
  if (!msg) return

  if (msg.type === 'analyze') {
    handleAnalyze(msg).catch(() => {
      // Errors already surfaced via sendMessage
    })
    return
  }

  if (msg.type === 'stop') {
    engine?.postMessage('stop')
  }
}
