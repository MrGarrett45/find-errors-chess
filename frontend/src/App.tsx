import { type FormEvent, useMemo, useState } from 'react'
import './App.css'

type GameLite = {
  url: string
  when_unix: number
  color: string
  opponent: string
  opponent_rating: number
  result: string
  rated: boolean
  time_class: string
  time_control: string
  pgn?: string
  eco?: string
}

type GamesResponse = {
  username: string
  count: number
  games: GameLite[]
  job_id?: string
  batches?: number
}

type ErrorPosition = {
  BadFen: {
    NormalizedFenBefore: string
    TimesSeen: number
    SuboptimalCount: number
    InaccuracyCount: number
    MistakeCount: number
    ErrorCount: number
    ErrorRate: number
  }
  Moves: {
    Move: string
    PlayedBy: string
    MoveNumber: number
    Ply: number
    Color: string
    Opponent: string
    URL: string
    ECO?: string
  }[]
}

type ErrorsResponse = {
  username: string
  count: number
  positions: ErrorPosition[]
}

const API_BASE =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') || 'http://localhost:8080'

function formatTimestamp(unixSeconds: number): string {
  const d = new Date(unixSeconds * 1000)
  return d.toLocaleDateString() + ' · ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function App() {
  const [username, setUsername] = useState('')
  const [months, setMonths] = useState(3)
  const [data, setData] = useState<GamesResponse | null>(null)
  const [status, setStatus] = useState<'idle' | 'loading' | 'error'>('idle')
  const [error, setError] = useState<string>('')
  const [limit, setLimit] = useState<number | ''>(200)
  const [errorsData, setErrorsData] = useState<ErrorsResponse | null>(null)
  const [errorsStatus, setErrorsStatus] = useState<'idle' | 'loading' | 'error'>('idle')
  const [errorsMessage, setErrorsMessage] = useState<string>('')

  const disabled = !username.trim() || status === 'loading'

  const introCopy = useMemo(
    () => ({
      title: 'Analyze your chess openings',
      desc: 'Fetch your recent Chess.com games, capture opening info, and queue deeper engine analysis.'
    }),
    []
  )

  const fetchGames = async (evt: FormEvent) => {
    evt.preventDefault()
    const user = username.trim()
    if (!user) return

    setStatus('loading')
    setError('')
    setData(null)
    setErrorsData(null)

    try {
      const cappedLimit = typeof limit === 'number' ? Math.max(1, Math.min(limit, 1000)) : 200
      const res = await fetch(
        `${API_BASE}/chessgames/${encodeURIComponent(user)}?months=${months}&limit=${cappedLimit}`
      )
      if (!res.ok) {
        throw new Error(`Request failed with status ${res.status}`)
      }
      const body = (await res.json()) as GamesResponse
      setData(body)
      setStatus('idle')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unexpected error'
      setError(message)
      setStatus('error')
    }
  }

  const fetchErrors = async () => {
    const user = username.trim()
    if (!user) return
    setErrorsStatus('loading')
    setErrorsMessage('')
    setErrorsData(null)

    try {
      const res = await fetch(`${API_BASE}/errors/${encodeURIComponent(user)}`)
      if (!res.ok) {
        throw new Error(`Request failed with status ${res.status}`)
      }
      const body = (await res.json()) as ErrorsResponse
      setErrorsData(body)
      setErrorsStatus('idle')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unexpected error'
      setErrorsMessage(message)
      setErrorsStatus('error')
    }
  }

  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Chess Insights</div>
          <div className="headline">{introCopy.title}</div>
          <p className="summary">{introCopy.desc}</p>
        </div>

        <form className="panel input-group" onSubmit={fetchGames} aria-label="Fetch chess games">
          <div className="label-row">
            <label htmlFor="username" className="label">
              Chess.com username
            </label>
            <span className="meta">e.g., hikaru</span>
          </div>
          <input
            id="username"
            className="input"
            placeholder="Enter username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            required
          />

          <div className="controls" aria-label="Filters">
            <label className="label" htmlFor="months">
              Months:
            </label>
            <select
              id="months"
              className="select"
              value={months}
              onChange={(e) => setMonths(Number(e.target.value))}
            >
              {[1, 3, 6, 12].map((m) => (
                <option key={m} value={m}>
                  Last {m} month{m > 1 ? 's' : ''}
                </option>
              ))}
            </select>
            <label className="label" htmlFor="limit">
              Limit (max 1000):
            </label>
            <input
              id="limit"
              className="input"
              type="number"
              min={1}
              max={1000}
              value={limit}
              onChange={(e) => {
                const val = e.target.value
                if (val === '') {
                  setLimit('')
                } else {
                  const num = Number(val)
                  setLimit(Number.isFinite(num) ? num : '')
                }
              }}
              style={{ width: '120px' }}
            />
            <button className="button" type="submit" disabled={disabled}>
              {status === 'loading' ? 'Fetching…' : 'Fetch games'}
            </button>
            <button
              className="button"
              type="button"
              disabled={!username.trim() || errorsStatus === 'loading'}
              onClick={fetchErrors}
            >
              {errorsStatus === 'loading' ? 'Finding…' : 'Find errors'}
            </button>
          </div>

          {status === 'loading' && <span className="status loading">Loading games…</span>}
          {status === 'error' && <span className="status error">Error: {error}</span>}
          {errorsStatus === 'error' && <span className="status error">Errors fetch failed: {errorsMessage}</span>}
        </form>
      </section>

      {errorsData && (
        <section className="results" aria-live="polite">
          <div className="pill-row">
            <div className="pill">
              Errors found: <strong>{errorsData.count}</strong>
            </div>
            <div className="pill">
              User: <strong>{errorsData.username}</strong>
            </div>
          </div>
          {errorsData.positions.length === 0 ? (
            <div className="empty">No error positions found.</div>
          ) : (
            errorsData.positions.map((pos) => (
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
      )}

      <section className="results" aria-live="polite">
        {data && (
          <>
            <div className="pill-row">
              <div className="pill">
                User: <strong>{data.username}</strong>
              </div>
              <div className="pill">
                Games fetched: <strong>{data.count}</strong>
              </div>
              {data.job_id && (
                <div className="pill">
                  Job ID: <strong>{data.job_id}</strong>
                </div>
              )}
            </div>

            {data.games.length === 0 ? (
              <div className="empty">No games found for this user.</div>
            ) : (
              data.games.map((game) => (
                <article key={game.url} className="game-card">
                  <div>
                    <strong>Opponent</strong>
                    <div className="meta">
                      {game.opponent} · {game.opponent_rating || '—'} Elo
                    </div>
                  </div>
                  <div>
                    <strong>Color</strong>
                    <div className="meta">{game.color}</div>
                  </div>
                  <div>
                    <strong>Result</strong>
                    <div className="meta">{game.result || '—'}</div>
                  </div>
                  <div>
                    <strong>Time</strong>
                    <div className="meta">{formatTimestamp(game.when_unix)}</div>
                  </div>
                  <div>
                    <strong>Mode</strong>
                    <div className="meta">
                      {game.time_class} · {game.time_control}
                    </div>
                  </div>
                  {game.eco && (
                    <div>
                      <strong>Opening</strong>
                      <div className="meta">{game.eco}</div>
                    </div>
                  )}
                  <div>
                    <a href={game.url} target="_blank" rel="noreferrer" className="button" style={{ padding: '10px 12px' }}>
                      View game
                    </a>
                  </div>
                </article>
              ))
            )}
          </>
        )}
      </section>
    </main>
  )
}

export default App
