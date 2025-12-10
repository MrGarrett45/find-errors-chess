import { Chessboard } from 'react-chessboard'
import type { ErrorPosition } from '../types'

type ErrorPositionCardProps = {
  position: ErrorPosition
}

export function ErrorPositionCard({ position }: ErrorPositionCardProps) {
  return (
    <article className="game-card">
      <div style={{ maxWidth: 320, width: '100%' }}>
        <Chessboard
          options={{
            id: `board-${position.BadFen.NormalizedFenBefore}`,
            position: position.BadFen.NormalizedFenBefore,
            allowDragging: false,
            showNotation: false,
            boardStyle: { borderRadius: 8, boxShadow: '0 4px 18px rgba(0,0,0,0.12)' },
            boardOrientation: position.BadFen.SideToMove === 'b' ? 'black' : 'white',
          }}
        />
      </div>
      <div>
        <strong>Position</strong>
        <div className="meta">{position.BadFen.NormalizedFenBefore}</div>
      </div>
      <div>
        <strong>Seen</strong>
        <div className="meta">{position.BadFen.TimesSeen} times</div>
      </div>
      <div>
        <strong>Error rate</strong>
        <div className="meta">{(position.BadFen.ErrorRate * 100).toFixed(0)}%</div>
      </div>
      {position.Moves.slice(0, 2).map((mv, idx) => (
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
  )
}
