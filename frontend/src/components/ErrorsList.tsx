import { useMemo, useState } from 'react'
import { ErrorPositionCard } from './ErrorPositionCard'
import type { ErrorsResponse } from '../types'

type ErrorsListProps = {
  data: ErrorsResponse | null
  isLoading: boolean
  error: string | null
}

export function ErrorsList({ data, isLoading, error }: ErrorsListProps) {
  const [activeFilter, setActiveFilter] = useState<
    'all' | 'suboptimal' | 'inaccuracy' | 'mistake' | 'blunder'
  >('all')

  const counts = useMemo(() => {
    const positions = data?.positions ?? []
    return {
      all: positions.length,
      suboptimal: positions.filter((p) => p.BadFen.SuboptimalCount > 0).length,
      inaccuracy: positions.filter((p) => p.BadFen.InaccuracyCount > 0).length,
      mistake: positions.filter((p) => p.BadFen.MistakeCount > 0).length,
      blunder: positions.filter((p) => p.BadFen.BlunderCount > 0).length,
    }
  }, [data?.positions])

  const filteredPositions = useMemo(() => {
    const positions = data?.positions ?? []
    switch (activeFilter) {
      case 'inaccuracy':
        return positions.filter((p) => p.BadFen.InaccuracyCount > 0)
      case 'mistake':
        return positions.filter((p) => p.BadFen.MistakeCount > 0)
      case 'blunder':
        return positions.filter((p) => p.BadFen.BlunderCount > 0)
      case 'suboptimal':
        return positions.filter((p) => p.BadFen.SuboptimalCount > 0)
      default:
        return positions
    }
  }, [activeFilter, data?.positions])

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

  if (!data.positions) {
    return (
      <section className="results">
        <div className="empty">No error positions found.</div>
      </section>
    )
  }

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
      <div className="pill-row" style={{ alignItems: 'center' }}>
        <button
          className={`pill pill-button ${activeFilter === 'all' ? 'pill-active' : ''}`}
          type="button"
          onClick={() => setActiveFilter('all')}
        >
          All ({counts.all})
        </button>
        <button
          className={`pill pill-button ${activeFilter === 'suboptimal' ? 'pill-active' : ''}`}
          type="button"
          onClick={() => setActiveFilter('suboptimal')}
        >
          Suboptimal ({counts.suboptimal})
        </button>
        <button
          className={`pill pill-button ${activeFilter === 'inaccuracy' ? 'pill-active' : ''}`}
          type="button"
          onClick={() => setActiveFilter('inaccuracy')}
        >
          Inaccuracy ({counts.inaccuracy})
        </button>
        <button
          className={`pill pill-button ${activeFilter === 'mistake' ? 'pill-active' : ''}`}
          type="button"
          onClick={() => setActiveFilter('mistake')}
        >
          Mistake ({counts.mistake})
        </button>
        <button
          className={`pill pill-button ${activeFilter === 'blunder' ? 'pill-active' : ''}`}
          type="button"
          onClick={() => setActiveFilter('blunder')}
        >
          Blunder ({counts.blunder})
        </button>
        <div className="info-tooltip" aria-label="Error category info">
          <span className="info-badge" role="img" aria-label="Info">
            i
          </span>
          <div className="info-tooltip__content" role="tooltip">
            <div className="info-tooltip__text">
              Suboptimal = small loss, Inaccuracy = notable loss, Mistake = major loss,
              Blunder = decisive loss. Depending on engine settings, a position can
              appear in multiple categories. An accurate move can also occasionally appear as an error (see error rate)
            </div>
          </div>
        </div>
      </div>
      {data.positions.length === 0 ? (
        <div className="empty">No error positions found.</div>
      ) : (
        filteredPositions.map((pos) => (
          <ErrorPositionCard key={pos.BadFen.NormalizedFenBefore} position={pos} username={data.username} />
        ))
      )}
    </section>
  )
}
