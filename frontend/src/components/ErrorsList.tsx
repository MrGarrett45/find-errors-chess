import { ErrorPositionCard } from './ErrorPositionCard'
import type { ErrorsResponse } from '../types'

type ErrorsListProps = {
  data: ErrorsResponse | null
  isLoading: boolean
  error: string | null
}

export function ErrorsList({ data, isLoading, error }: ErrorsListProps) {
  if (isLoading) {
    return (
      <section className="results">
        <div className="panel" style={{ padding: '8px 12px' }}>
          <span className="status loading">Loading errors…</span>
        </div>
      </section>
    )
  }

  if (error) {
    return (
      <section className="results">
        <div className="panel" style={{ padding: '8px 12px' }}>
          <span className="status error">{error}</span>
        </div>
      </section>
    )
  }

  if (!data) return null

  return (
    <section className="results" aria-live="polite">
      <div className="pill-row">
        <div className="pill">
          Errors found: <strong>{data.count}</strong>
        </div>
        <div className="pill">
          User: <strong>{data.username}</strong>
        </div>
      </div>
      {data.positions.length === 0 ? (
        <div className="empty">No error positions found.</div>
      ) : (
        data.positions.map((pos) => <ErrorPositionCard key={pos.BadFen.NormalizedFenBefore} position={pos} />)
      )}
    </section>
  )
}
