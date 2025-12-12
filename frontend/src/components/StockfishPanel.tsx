import type { EngineLine, EngineScore } from '../types'
import { useStockfish } from '../hooks/useStockfish'

type StockfishPanelProps = {
  fen: string
  depth?: number
  multipv?: number
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

export function StockfishPanel({ fen, depth = 14, multipv = 3 }: StockfishPanelProps) {
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
        <button
          className="button"
          type="button"
          onClick={restart}
          disabled={!isReady || isAnalyzing}
        >
          Refresh
        </button>
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
