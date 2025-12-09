import type { AnalysisStatusType } from '../types'
import { ProgressBar } from './ProgressBar'

type AnalysisStatusProps = {
  status: AnalysisStatusType
  progress: number
  error: string | null
}

export function AnalysisStatus({ status, progress, error }: AnalysisStatusProps) {
  if (status === 'idle') {
    return (
      <section className="results">
        <div className="empty">Enter a username to start an analysis.</div>
      </section>
    )
  }

  return (
    <section className="results">
      <div className="panel" style={{ display: 'grid', gap: 12, padding: '8px 12px' }}>
        <ProgressBar progress={progress} status={status} />
        <div className={`status ${status === 'failed' ? 'error' : ''}`}>
          {status === 'starting' && 'Starting analysis…'}
          {status === 'running' && `Analyzing games (${progress.toFixed(1)}%)`}
          {status === 'completed' && 'Analysis complete!'}
          {status === 'failed' && 'Analysis failed.'}
        </div>
        {error && <div className="status error">{error}</div>}
      </div>
    </section>
  )
}
