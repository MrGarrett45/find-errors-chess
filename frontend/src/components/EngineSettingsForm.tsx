import type { ChangeEvent } from 'react'

type EngineSettingsFormProps = {
  engineDepth: number | ''
  onEngineDepthChange: (value: number | '') => void
  engineMoveTime: number | ''
  onEngineMoveTimeChange: (value: number | '') => void
  engineUseDepth: boolean
  onEngineUseDepthChange: (value: boolean) => void
  isDisabled: boolean
}

export function EngineSettingsForm({
  engineDepth,
  onEngineDepthChange,
  engineMoveTime,
  onEngineMoveTimeChange,
  engineUseDepth,
  onEngineUseDepthChange,
  isDisabled,
}: EngineSettingsFormProps) {
  const setMode = (useDepth: boolean) => {
    onEngineUseDepthChange(useDepth)
    if (useDepth) {
      onEngineDepthChange(14)
    } else {
      onEngineMoveTimeChange(50)
    }
  }

  const handleDepth = (e: ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value
    if (val === '') {
      onEngineDepthChange('')
      return
    }
    const num = Number(val)
    const clamped = Math.min(20, Math.max(8, num))
    onEngineDepthChange(Number.isFinite(clamped) ? clamped : '')
  }

  const handleMoveTime = (e: ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value
    if (val === '') {
      onEngineMoveTimeChange('')
      return
    }
    const num = Number(val)
    const clamped = Math.min(1000, Math.max(25, num))
    onEngineMoveTimeChange(Number.isFinite(clamped) ? clamped : '')
  }

  return (
    <div className="input-group" aria-label="Engine settings" style={{ gap: 10 }}>
      <div className="pill-row">
        <button
          type="button"
          className={`pill pill-button ${!engineUseDepth ? 'pill-active' : ''}`}
          onClick={() => setMode(false)}
          disabled={isDisabled}
        >
          Movetime
        </button>
        <button
          type="button"
          className={`pill pill-button ${engineUseDepth ? 'pill-active' : ''}`}
          onClick={() => setMode(true)}
          disabled={isDisabled}
        >
          Depth
        </button>
      </div>

      {!engineUseDepth ? (
        <>
          <label className="label" htmlFor="engine-movetime">
            Movetime (ms 25-1000)
          </label>
          <input
            id="engine-movetime"
            className="input"
            type="number"
            min={25}
            max={1000}
            value={engineMoveTime}
            onChange={handleMoveTime}
            style={{ width: '140px' }}
            required
            disabled={isDisabled}
          />
        </>
      ) : (
        <>
          <label className="label" htmlFor="engine-depth">
            Engine depth (8-20)
          </label>
          <input
            id="engine-depth"
            className="input"
            type="number"
            min={8}
            max={20}
            value={engineDepth}
            onChange={handleDepth}
            style={{ width: '120px' }}
            required
            disabled={isDisabled}
          />
        </>
      )}
    </div>
  )
}
