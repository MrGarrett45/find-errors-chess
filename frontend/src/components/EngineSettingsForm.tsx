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
  const handleDepth = (e: ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value
    if (val === '') {
      onEngineDepthChange('')
      return
    }
    const num = Number(val)
    onEngineDepthChange(Number.isFinite(num) ? num : '')
  }

  const handleMoveTime = (e: ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value
    if (val === '') {
      onEngineMoveTimeChange('')
      return
    }
    const num = Number(val)
    onEngineMoveTimeChange(Number.isFinite(num) ? num : '')
  }

  return (
    <div className="controls" aria-label="Engine settings" style={{ gap: 10 }}>
      <label className="label" htmlFor="engine-depth">
        Engine depth (1-25)
      </label>
      <input
        id="engine-depth"
        className="input"
        type="number"
        min={1}
        max={25}
        value={engineDepth}
        onChange={handleDepth}
        style={{ width: '120px' }}
        required
        disabled={isDisabled}
      />
      <label className="label" htmlFor="engine-movetime">
        Movetime (ms 0-1000)
      </label>
      <input
        id="engine-movetime"
        className="input"
        type="number"
        min={0}
        max={1000}
        value={engineMoveTime}
        onChange={handleMoveTime}
        style={{ width: '140px' }}
        required
        disabled={isDisabled}
      />
      <label className="label" htmlFor="engine-use-depth">
        Use depth?
      </label>
      <input
        id="engine-use-depth"
        type="checkbox"
        checked={engineUseDepth}
        onChange={(e) => onEngineUseDepthChange(e.target.checked)}
        style={{ width: 20, height: 20 }}
        disabled={isDisabled}
      />
    </div>
  )
}
