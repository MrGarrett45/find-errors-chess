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
        data.positions.map((pos) => (
          <article key={pos.BadFen.NormalizedFenBefore} className="game-card">
            <div>
              <strong>Position</strong>
              <div className="meta">{pos.BadFen.NormalizedFenBefore}</div>
            </div>
            <div>
              <strong>Seen</strong>
              <div className="meta">{pos.BadFen.TimesSeen} times</div>
            </div>
            <div>
              <strong>Error rate</strong>
              <div className="meta">{(pos.BadFen.ErrorRate * 100).toFixed(0)}%</div>
            </div>
            {pos.Moves.slice(0, 2).map((mv, idx) => (
              <div key={mv.URL + idx}>
                <strong>Example {idx + 1}</strong>
                <div className="meta">
                  {mv.Color} played {mv.Move} vs {mv.Opponent}
                  {mv.ECO ? ` · ${mv.ECO}` : ''}
                </div>
                <a
                  href={mv.URL}
                  target="_blank"
                  rel="noreferrer"
                  className="button"
                  style={{ padding: '8px 10px', fontSize: 14 }}
                >
                  View game
                </a>
              </div>
            ))}
          </article>
        ))
      )}
    </section>
  )
}
