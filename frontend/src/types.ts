export type AnalysisStatusType = 'idle' | 'starting' | 'running' | 'completed' | 'failed'

export type JobStatus = {
  id: string
  status: AnalysisStatusType
  completed_batches: number
  total_batches: number
}

export type ErrorPosition = {
  BadFen: {
    NormalizedFenBefore: string
    TimesSeen: number
    SuboptimalCount: number
    InaccuracyCount: number
    MistakeCount: number
    ErrorCount: number
    ErrorRate: number
    SideToMove: string
  }
  Moves: {
    Move: string
    PlayedBy: string
    MoveNumber: number
    Ply: number
    Color: string
    Opponent: string
    URL: string
    ECO?: string
  }[]
}

export type ErrorsResponse = {
  username: string
  count: number
  positions: ErrorPosition[]
}

export type EngineScore = { type: 'cp'; value: number } | { type: 'mate'; value: number }

export type EngineLine = {
  multipv: number
  depth: number
  pv: string[]
  san: string[]
  score: EngineScore | null
  nodes?: number
  nps?: number
}
