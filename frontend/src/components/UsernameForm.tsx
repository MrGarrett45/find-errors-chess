import { useState, type FormEvent } from 'react'
import { EngineSettingsForm } from './EngineSettingsForm'

type UsernameFormProps = {
  username: string
  onUsernameChange: (value: string) => void
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

  return (
    <form className="panel input-group" onSubmit={onSubmit} aria-label="Start analysis">
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
        onChange={(e) => onUsernameChange(e.target.value)}
        autoComplete="username"
        required
      />

      <div className="controls-with-popover">
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
