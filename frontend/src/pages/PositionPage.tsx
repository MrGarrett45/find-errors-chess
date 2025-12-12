import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Chessboard,
  type PieceDropHandlerArgs,
  type SquareHandlerArgs,
} from 'react-chessboard'
import { Chess } from 'chess.js'
import { StockfishPanel } from '../components/StockfishPanel'
import { PositionGamesList } from '../components/PositionGamesList'
import type { ErrorPosition, ErrorsResponse } from '../types'

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
  const username = decoded?.username ?? ''
  const initialFen = decoded?.fen ?? ''
  const normalizedInitialFen = useMemo(() => normalizeFen(initialFen), [initialFen])
  const [game, setGame] = useState<Chess | null>(null)
  const rawId = params.id ?? 'board'
  const safeId = `position-${rawId.replace(/[^a-zA-Z0-9_-]/g, '_')}`

  // square selected for click-to-move
  const [selectedSquare, setSelectedSquare] = useState<string | null>(null)
  const [errorsData, setErrorsData] = useState<ErrorsResponse | null>(null)
  const [errorsLoading, setErrorsLoading] = useState(false)
  const [errorsError, setErrorsError] = useState<string | null>(null)

  // Initialize chess.js from the FEN in the URL
  useEffect(() => {
    if (!decoded?.fen) return

    const g = new Chess()
    try {
      g.load(decoded.fen) // starting position is your error FEN
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setGame(g)
      setSelectedSquare(null)
    } catch (e) {
      console.error('Invalid FEN:', e)
      setGame(null)
      setSelectedSquare(null)
    }
  }, [decoded?.fen])

  const API_BASE =
    import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') || 'http://localhost:8080'

  // Shared move helper used by both drag and click
  const makeMove = (from: string, to: string): boolean => {
    if (!game) return false

    const gameCopy = new Chess(game.fen())

    const move = gameCopy.move({
      from,
      to,
      promotion: 'q', // always promote to queen for simplicity
    })

    if (move === null) {
      return false
    }

    setGame(gameCopy)
    return true
  }

  // Drag-to-move
  const handlePieceDrop = ({ sourceSquare, targetSquare }: PieceDropHandlerArgs): boolean => {
    if (!targetSquare) return false
    const moved = makeMove(sourceSquare, targetSquare)
    if (moved) {
      setSelectedSquare(null)
    }
    return moved
  }

  // Click-to-move
  // Be defensive: runtime may pass a string ('e4') or an object ({ square: 'e4' })
  const handleSquareClick = (arg: SquareHandlerArgs | string) => {
    if (!game) return

    const square = typeof arg === 'string' ? arg : arg.square
    if (!square) return

    // No selection yet → select this square
    if (!selectedSquare) {
      setSelectedSquare(square)
      return
    }

    // Click same square again → deselect
    if (selectedSquare === square) {
      setSelectedSquare(null)
      return
    }

    // Try move selectedSquare -> clicked square
    const moved = makeMove(selectedSquare, square)
    if (moved) {
      setSelectedSquare(null)
      return
    }

    // If move illegal, treat clicked square as new selection
    setSelectedSquare(square)
  }

  const currentFen = game ? game.fen() : initialFen

  useEffect(() => {
    const fetchErrors = async () => {
      if (!username) return
      setErrorsLoading(true)
      setErrorsError(null)
      try {
        const res = await fetch(`${API_BASE}/errors/${encodeURIComponent(username)}`)
        if (!res.ok) {
          throw new Error(`Failed to load errors (${res.status})`)
        }
        const body = (await res.json()) as ErrorsResponse
        setErrorsData(body)
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to load errors'
        setErrorsError(message)
      } finally {
        setErrorsLoading(false)
      }
    }

    fetchErrors()
  }, [API_BASE, username])

  const matchedPosition: ErrorPosition | null = useMemo(() => {
    if (!errorsData?.positions?.length || !normalizedInitialFen) return null
    return (
      errorsData.positions.find(
        (p) => normalizeFen(p.BadFen.NormalizedFenBefore) === normalizedInitialFen,
      ) ?? null
    )
  }, [errorsData?.positions, normalizedInitialFen])

  // Optional: highlight selected square
  const squareStyles: Record<string, React.CSSProperties> = {}
  if (selectedSquare) {
    squareStyles[selectedSquare] = {
      boxShadow: 'inset 0 0 0 3px rgba(255, 215, 0, 0.9)',
    }
  }

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

  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Position View</div>
        </div>
        <div className="hero-actions">
          <button className="button" type="button" onClick={() => navigate(-1)}>
            Back
          </button>
        </div>
      </section>

      <section className="results position-results" style={{ margin: 'auto', width: '100%' }}>
        <div className="headline">Error position for {username}</div>
        <div className="board-grid">
          <div className="panel board-panel">
            <Chessboard
              options={{
                id: `position-${safeId}`,
                position: currentFen,
                allowDragging: true,
                showNotation: true,
                boardStyle: {
                  borderRadius: 8,
                  boxShadow: '0 4px 18px rgba(0, 0, 0, 0.12)',
                },
                squareStyles, // highlight selected square
                onPieceDrop: handlePieceDrop,
                onSquareClick: handleSquareClick,
              }}
            />
            <div className="meta board-fen">{currentFen}</div>
          </div>

          <StockfishPanel fen={currentFen} />
        </div>
        <PositionGamesList
          position={matchedPosition}
          isLoading={errorsLoading}
          error={errorsError}
        />
      </section>
    </main>
  )
}

function normalizeFen(fen: string): string {
  const parts = fen.split(' ')
  if (parts.length < 4) return fen
  const [pieces, side, castling, ep] = parts
  return `${pieces} ${side} ${castling || '-'} ${ep || '-'}`
}
