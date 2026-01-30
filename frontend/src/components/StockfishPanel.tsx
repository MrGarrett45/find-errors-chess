import { useEffect, useState } from 'react'
import type { EngineLine, EngineScore } from '../types'
import { useStockfish } from '../hooks/useStockfish'

type StockfishPanelProps = {
  fen: string
  depth?: number
  multipv?: number
  minDepth?: number
  maxDepth?: number
  onDepthChange?: (depth: number) => void
}

const formatScore = (score: EngineScore | null): string => {
  if (!score) return '…'
  if (score.type === 'mate') return `M${score.value}`
  const pawns = score.value / 100
  const withSign = pawns > 0 ? `+${pawns.toFixed(2)}` : pawns.toFixed(2)
  return withSign
}

const formatLine = (line: EngineLine): string =>
  (line.san.length ? line.san : line.pv).slice(0, 14).join(' ')

export function StockfishPanel({
  fen,
  depth = 14,
  multipv = 3,
  minDepth = 8,
  maxDepth = 20,
  onDepthChange,
}: StockfishPanelProps) {
  const [depthInput, setDepthInput] = useState(String(depth))

  useEffect(() => {
    setDepthInput(String(depth))
  }, [depth])

  const { lines, isReady, isAnalyzing, error, restart } = useStockfish(fen, {
    depth,
    multipv,
  })

  const statusText = error
    ? 'Engine error'
    : !isReady
      ? 'Warming up Stockfish…'
      : isAnalyzing
        ? 'Analyzing current position…'
        : 'Ready'

  return (
    <div className="panel engine-panel" aria-live="polite">
      <div className="engine-panel__header">
        <div>
          <div className="badge">Stockfish</div>
          <div className="engine-panel__title">Top moves</div>
        </div>
        <div className="controls">
          <div className="input-group stockfish-depth">
            <label className="label" htmlFor="stockfish-depth">
              Depth
            </label>
            <input
              id="stockfish-depth"
              className="input input--compact"
              type="text"
              inputMode="numeric"
              min={minDepth}
              max={maxDepth}
              value={depthInput}
              onChange={(e) => {
                const next = e.target.value
                if (!/^\d*$/.test(next)) return
                setDepthInput(next)
              }}
              onBlur={() => {
                if (!onDepthChange) {
                  setDepthInput(String(depth))
                  return
                }
                const parsed = Number(depthInput)
                if (Number.isNaN(parsed)) {
                  setDepthInput(String(depth))
                  return
                }
                const clamped = Math.min(maxDepth, Math.max(minDepth, parsed))
                setDepthInput(String(clamped))
                onDepthChange(clamped)
              }}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  ;(e.currentTarget as HTMLInputElement).blur()
                }
              }}
            />
          </div>
          <button
            className="button"
            type="button"
            onClick={restart}
            disabled={!isReady || isAnalyzing}
            style={{ display: 'none' }}
            aria-hidden="true"
          >
            Refresh
          </button>
        </div>
      </div>

      <div className="meta">{statusText}</div>

      {error && (
        <div className="status error" role="status">
          {error}
        </div>
      )}

      {!error && lines.length === 0 && isReady && !isAnalyzing && (
        <div className="empty">No engine lines yet.</div>
      )}

      {!error && lines.length === 0 && isAnalyzing && (
        <div className="empty">Crunching moves…</div>
      )}

      {!error && lines.length > 0 && (
        <div className="engine-lines">
          {lines.map((line) => (
            <div key={line.multipv} className="engine-line">
              <div className="engine-line__meta">
                <span className="pill">#{line.multipv}</span>
                <span className="engine-line__score">{formatScore(line.score)}</span>
                <span className="engine-line__depth">d{line.depth}</span>
              </div>
              <div className="engine-line__pv" title={formatLine(line)}>
                {formatLine(line)}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
