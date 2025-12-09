import type { FormEvent } from 'react'

type UsernameFormProps = {
  username: string
  onUsernameChange: (value: string) => void
  onSubmit: (evt: FormEvent) => void
  isDisabled: boolean
}

export function UsernameForm({ username, onUsernameChange, onSubmit, isDisabled }: UsernameFormProps) {
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
        <button className="button" type="submit" disabled={isDisabled || !username.trim()}>
          {isDisabled ? 'Starting…' : 'Start analysis'}
        </button>
      </div>
    </form>
  )
}
