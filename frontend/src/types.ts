export type AnalysisStatusType = 'idle' | 'starting' | 'running' | 'completed' | 'failed'

export type JobStatus = {
  id: string
  status: AnalysisStatusType
  completed_batches: number
  total_batches: number
}
