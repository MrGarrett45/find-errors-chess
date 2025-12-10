import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Chessboard } from 'react-chessboard'

function decodeId(id: string): { username: string; fen: string } | null {
  try {
    const decoded = decodeURIComponent(id)
    const parts = decoded.split('::')
    if (parts.length !== 2) return null
    const [username, fen] = parts
    if (!username || !fen) return null
    return { username, fen }
  } catch {
    return null
  }
}

export function PositionPage() {
  const params = useParams<{ id: string }>()
  const navigate = useNavigate()
  const decoded = useMemo(() => (params.id ? decodeId(params.id) : null), [params.id])

  if (!decoded) {
    return (
      <main className="page">
        <div className="panel" style={{ padding: 12 }}>
          <div className="status error">Invalid position link.</div>
          <button className="button" type="button" onClick={() => navigate('/')}>
            Go back
          </button>
        </div>
      </main>
    )
  }

  const { username, fen } = decoded

  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Position View</div>
          <div className="headline">Error position for {username}</div>
          <p className="summary">Review the FEN and related games for this position.</p>
        </div>
      </section>

      <section className="results">
          <Chessboard
            options={{
              id: `position-${params.id}`,
              position: fen,
              allowDragging: true,
              showNotation: true,
              boardStyle: { borderRadius: 8, boxShadow: '0 4px 18px rgba(0,0,0,0.12)' },
            }}
          />
          <div className="meta" style={{ marginTop: 12, wordBreak: 'break-all' }}>
            {fen}
          </div>
        <button className="button" type="button" onClick={() => navigate(-1)} style={{ marginTop: 12 }}>
          Back
        </button>
      </section>
    </main>
  )
}
