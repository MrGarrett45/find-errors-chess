import { useEffect, useMemo, useState } from 'react'
import { useAuth0 } from '@auth0/auth0-react'
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
import { authFetch } from '../utils/api'

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
  const { getAccessTokenSilently } = useAuth0()
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
  const [fenHistory, setFenHistory] = useState<string[]>([])
  const [historyIndex, setHistoryIndex] = useState(0)

  // Initialize chess.js from the FEN in the URL
  useEffect(() => {
    if (!decoded?.fen) return

    const g = new Chess()
    try {
      g.load(decoded.fen) // starting position is your error FEN
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setGame(g)
      setFenHistory([g.fen()])
      setHistoryIndex(0)
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
    // push into history, truncating any "future" states
    setFenHistory((prev) => {
      const base = prev.slice(0, historyIndex + 1)
      const next = [...base, gameCopy.fen()]
      setHistoryIndex(next.length - 1)
      return next
    })
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
  const canStepBack = historyIndex > 0
  const canStepForward = historyIndex < fenHistory.length - 1
  const canReset = historyIndex !== 0 && fenHistory.length > 0

  const stepBack = () => {
    if (!canStepBack) return
    setHistoryIndex((idx) => {
      const next = Math.max(0, idx - 1)
      const fen = fenHistory[next]
      if (fen) {
        const g = new Chess(fen)
        setGame(g)
        setSelectedSquare(null)
      }
      return next
    })
  }

  const stepForward = () => {
    if (!canStepForward) return
    setHistoryIndex((idx) => {
      const next = Math.min(fenHistory.length - 1, idx + 1)
      const fen = fenHistory[next]
      if (fen) {
        const g = new Chess(fen)
        setGame(g)
        setSelectedSquare(null)
      }
      return next
    })
  }

  const resetPosition = () => {
    if (!canReset || !fenHistory[0]) return
    const g = new Chess(fenHistory[0])
    setGame(g)
    setSelectedSquare(null)
    setHistoryIndex(0)
  }

  // Keyboard navigation: left/right undo-redo, up resets to original FEN
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null
      if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) {
        return
      }
      if (!fenHistory.length) return

      if (e.key === 'ArrowLeft') {
        e.preventDefault()
        stepBack()
      } else if (e.key === 'ArrowRight') {
        e.preventDefault()
        stepForward()
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        resetPosition()
      }
    }

    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [fenHistory])

  useEffect(() => {
    const fetchErrors = async () => {
      if (!username) return
      setErrorsLoading(true)
      setErrorsError(null)
      try {
        const res = await authFetch(
          `${API_BASE}/errors/${encodeURIComponent(username)}`,
          undefined,
          getAccessTokenSilently,
        )
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
                boardOrientation: matchedPosition?.BadFen.SideToMove === 'b' ? 'black' : 'white',
                onPieceDrop: handlePieceDrop,
                onSquareClick: handleSquareClick,
              }}
            />
            <div className="meta board-fen">{currentFen}</div>
            <div className="button-row" style={{ display: 'flex', gap: 8, marginTop: 8 }}>
              <button className="button" type="button" onClick={stepBack} disabled={!canStepBack}>
                ⬅ Previous
              </button>
              <button className="button" type="button" onClick={stepForward} disabled={!canStepForward}>
                Next ➡
              </button>
              <button className="button" type="button" onClick={resetPosition} disabled={!canReset}>
                Reset ↩
              </button>
            </div>
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
