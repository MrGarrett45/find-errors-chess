import type { ErrorPosition } from '../types'

type PositionGamesListProps = {
  position: ErrorPosition | null
  isLoading: boolean
  error: string | null
}

export function PositionGamesList({ position, isLoading, error }: PositionGamesListProps) {
  if (isLoading) {
    return (
      <div className="panel">
        <div className="status loading">Loading games for this position…</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="panel">
        <div className="status error">{error}</div>
      </div>
    )
  }

  if (!position) {
    return (
      <div className="panel empty">
        No recorded games for this position yet.
      </div>
    )
  }

  return (
    <div className="panel">
      <div className="headline" style={{ fontSize: 20, gap: 8 }}>
        Games reaching this position
      </div>
      <div className="meta" style={{ marginBottom: 8 }}>
        Seen {position.BadFen.TimesSeen} times; inaccuracies in {position.BadFen.ErrorCount} games
      </div>
      <div className="engine-lines">
        {position.Moves.map((mv) => (
          <div key={`${mv.URL}-${mv.Ply}`} className="engine-line">
            <div className="engine-line__meta">
              <span className="pill">{mv.Color === 'w' ? 'White' : 'Black'}</span>
              <span className="engine-line__depth">Move {mv.MoveNumber}</span>
              <span className="engine-line__score">Your Move: {mv.Move}</span>
            </div>
            <div
              className="engine-line__pv"
              style={{
                display: 'flex',
                gap: 12,
                alignItems: 'center',
                width: '100%',
                marginTop: 10,
              }}
            >
              <span className="pill" style={{ flexShrink: 0 }}>
                <strong>Opponent:</strong>&nbsp;{mv.Opponent}
              </span>
              <span
                className="pill"
                style={{
                  background: '#eef2ff',
                  flex: '0 1 auto',
                  textAlign: 'center',
                  marginLeft: 'auto',
                  marginRight: 'auto',
                  alignSelf: 'center',
                }}
              >
                <strong>Opening:</strong>&nbsp;{mv.ECO || '—'}
              </span>
              <a
                href={mv.URL}
                target="_blank"
                rel="noreferrer"
                className="button"
                style={{
                  padding: '8px 12px',
                  minWidth: 120,
                  justifyContent: 'center',
                  marginLeft: 'auto',
                }}
              >
                View game
              </a>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
