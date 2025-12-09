import type { AnalysisStatusType } from '../types'

type ProgressBarProps = {
  progress: number
  status: AnalysisStatusType
}

export function ProgressBar({ progress, status }: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, progress))
  const isActive = status === 'running' || status === 'starting'
  const fillWidth = clamped === 0 && isActive ? 5 : clamped

  return (
    <div
      style={{
        width: '100%',
        background: '#eef1f8',
        borderRadius: 12,
        overflow: 'hidden',
        border: '1px solid #e5e8f0',
      }}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={clamped}
      role="progressbar"
    >
      <div
        style={{
          width: `${fillWidth}%`,
          height: 16,
          background: isActive
            ? 'linear-gradient(120deg, #404cff, #5a6dff)'
            : 'linear-gradient(120deg, #20bf55, #01baef)',
          transition: 'width 0.35s ease',
        }}
      />
    </div>
  )
}
