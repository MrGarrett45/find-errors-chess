import type { FormEvent } from 'react'

type UsernameFormProps = {
  username: string
  onUsernameChange: (value: string) => void
  months: number
  onMonthsChange: (value: number) => void
  limit: number | ''
  onLimitChange: (value: number | '') => void
  onSubmit: (evt: FormEvent) => void
  isDisabled: boolean
}

export function UsernameForm({
  username,
  onUsernameChange,
  months,
  onMonthsChange,
  limit,
  onLimitChange,
  onSubmit,
  isDisabled,
}: UsernameFormProps) {
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
        <button className="button" type="submit" disabled={isDisabled || !username.trim()}>
          {isDisabled ? 'Starting…' : 'Start analysis'}
        </button>
      </div>
    </form>
  )
}
