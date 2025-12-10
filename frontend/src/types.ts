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
