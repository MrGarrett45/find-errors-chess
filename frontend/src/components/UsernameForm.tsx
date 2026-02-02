import { useEffect, useRef, useState, type FormEvent } from 'react'
import { EngineSettingsForm } from './EngineSettingsForm'

type UsernameFormProps = {
  username: string
  onUsernameChange: (value: string) => void
  provider: 'chesscom' | 'lichess'
  onProviderChange: (value: 'chesscom' | 'lichess') => void
  months: number
  onMonthsChange: (value: number) => void
  limit: number | ''
  onLimitChange: (value: number | '') => void
  engineDepth: number | ''
  onEngineDepthChange: (value: number | '') => void
  engineMoveTime: number | ''
  onEngineMoveTimeChange: (value: number | '') => void
  engineUseDepth: boolean
  onEngineUseDepthChange: (value: boolean) => void
  onSubmit: (evt: FormEvent) => void
  onFetchErrors: () => void
  isSubmitDisabled: boolean
  isFetchDisabled: boolean
}

export function UsernameForm({
  username,
  onUsernameChange,
  provider,
  onProviderChange,
  months,
  onMonthsChange,
  limit,
  onLimitChange,
  engineDepth,
  onEngineDepthChange,
  engineMoveTime,
  onEngineMoveTimeChange,
  engineUseDepth,
  onEngineUseDepthChange,
  onSubmit,
  onFetchErrors,
  isSubmitDisabled,
  isFetchDisabled,
}: UsernameFormProps) {
  const [showEngineSettings, setShowEngineSettings] = useState(false)
  const popoverRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!showEngineSettings) return
    const handleClick = (event: MouseEvent) => {
      const target = event.target as Node | null
      if (!target) return
      if (popoverRef.current && !popoverRef.current.contains(target)) {
        setShowEngineSettings(false)
      }
    }

    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [showEngineSettings])

  return (
    <form className="panel input-group" onSubmit={onSubmit} aria-label="Start analysis">
      <div className="label-row">
        <div className="pill-row">
          <button
            type="button"
            className={`pill pill-button ${provider === 'chesscom' ? 'pill-active' : ''}`}
            onClick={() => onProviderChange('chesscom')}
          >
            Chess.com
          </button>
          <button
            type="button"
            className={`pill pill-button ${provider === 'lichess' ? 'pill-active' : ''}`}
            onClick={() => onProviderChange('lichess')}
          >
            Lichess
          </button>
        </div>
      </div>
      <input
        id="username"
        className="input"
        placeholder="Enter username"
        value={username}
        onChange={(e) => onUsernameChange(e.target.value)}
        autoComplete="username"
        required
      />

      <div className="controls-with-popover" ref={popoverRef}>
        <div className="controls" aria-label="Controls">
          <label className="label" htmlFor="months">
            Months
          </label>
          <select
            id="months"
            className="select"
            value={months}
            onChange={(e) => onMonthsChange(Number(e.target.value))}
          >
            {[1, 3, 6, 12].map((m) => (
              <option key={m} value={m}>
                Last {m} month{m > 1 ? 's' : ''}
              </option>
            ))}
          </select>
          <label className="label" htmlFor="limit">
            Limit (max 500)
          </label>
          <input
            id="limit"
            className="input"
            type="number"
            min={1}
            max={500}
            value={limit}
            onChange={(e) => {
              const val = e.target.value
              if (val === '') {
                onLimitChange('')
              } else {
                const num = Number(val)
                onLimitChange(Number.isFinite(num) ? num : '')
              }
            }}
            style={{ width: '120px' }}
          />
          <button
            type="button"
            className="button ghost"
            onClick={() => setShowEngineSettings((prev) => !prev)}
          >
            {showEngineSettings ? 'Hide engine settings' : 'Engine settings'}
          </button>
          <button
            className="button"
            type="submit"
            disabled={isSubmitDisabled || !username.trim()}
          >
            {isSubmitDisabled && username.trim() ? 'Starting...' : 'Start analysis'}
          </button>
          <button
            className="button"
            type="button"
            disabled={!username.trim() || isFetchDisabled}
            onClick={onFetchErrors}
            style={{ display: 'none' }}
            aria-hidden="true"
          >
            Fetch errors only
          </button>
        </div>

        <div
          className={`engine-settings-popover ${showEngineSettings ? 'open' : ''}`}
          role="dialog"
          aria-label="Engine settings"
          aria-hidden={!showEngineSettings}
        >
          <div className="engine-settings-surface">
            <EngineSettingsForm
              engineDepth={engineDepth}
              onEngineDepthChange={onEngineDepthChange}
              engineMoveTime={engineMoveTime}
              onEngineMoveTimeChange={onEngineMoveTimeChange}
              engineUseDepth={engineUseDepth}
              onEngineUseDepthChange={onEngineUseDepthChange}
              isDisabled={isSubmitDisabled}
            />
          </div>
        </div>
      </div>
    </form>
  )
}
