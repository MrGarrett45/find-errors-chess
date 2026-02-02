import { useEffect, useState, type ChangeEvent } from 'react'

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
  const [depthInput, setDepthInput] = useState(engineDepth === '' ? '' : String(engineDepth))
  const [moveTimeInput, setMoveTimeInput] = useState(
    engineMoveTime === '' ? '' : String(engineMoveTime),
  )

  useEffect(() => {
    const next = engineDepth === '' ? '' : String(engineDepth)
    if (next !== depthInput) {
      setDepthInput(next)
    }
  }, [engineDepth, depthInput])

  useEffect(() => {
    const next = engineMoveTime === '' ? '' : String(engineMoveTime)
    if (next !== moveTimeInput) {
      setMoveTimeInput(next)
    }
  }, [engineMoveTime, moveTimeInput])

  const setMode = (useDepth: boolean) => {
    onEngineUseDepthChange(useDepth)
    if (useDepth) {
      onEngineDepthChange(14)
    } else {
      onEngineMoveTimeChange(50)
    }
  }

  const handleDepth = (e: ChangeEvent<HTMLInputElement>) => {
    const next = e.target.value
    if (!/^\d*$/.test(next)) return
    setDepthInput(next)
  }

  const handleMoveTime = (e: ChangeEvent<HTMLInputElement>) => {
    const next = e.target.value
    if (!/^\d*$/.test(next)) return
    setMoveTimeInput(next)
  }

  const commitDepth = () => {
    if (depthInput === '') {
      onEngineDepthChange('')
      return
    }
    const num = Number(depthInput)
    if (Number.isNaN(num)) {
      setDepthInput(engineDepth === '' ? '' : String(engineDepth))
      return
    }
    const clamped = Math.min(20, Math.max(8, num))
    setDepthInput(String(clamped))
    onEngineDepthChange(clamped)
  }

  const commitMoveTime = () => {
    if (moveTimeInput === '') {
      onEngineMoveTimeChange('')
      return
    }
    const num = Number(moveTimeInput)
    if (Number.isNaN(num)) {
      setMoveTimeInput(engineMoveTime === '' ? '' : String(engineMoveTime))
      return
    }
    const clamped = Math.min(1000, Math.max(25, num))
    setMoveTimeInput(String(clamped))
    onEngineMoveTimeChange(clamped)
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
            type="text"
            inputMode="numeric"
            min={25}
            max={1000}
            value={moveTimeInput}
            onChange={handleMoveTime}
            onBlur={commitMoveTime}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                ;(e.currentTarget as HTMLInputElement).blur()
              }
            }}
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
            type="text"
            inputMode="numeric"
            min={8}
            max={20}
            value={depthInput}
            onChange={handleDepth}
            onBlur={commitDepth}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                ;(e.currentTarget as HTMLInputElement).blur()
              }
            }}
            style={{ width: '120px' }}
            required
            disabled={isDisabled}
          />
        </>
      )}
    </div>
  )
}
