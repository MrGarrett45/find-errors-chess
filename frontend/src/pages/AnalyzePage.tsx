import { type FormEvent, useEffect, useMemo, useState } from 'react'
import { UsernameForm } from '../components/UsernameForm'
import { AnalysisStatus } from '../components/AnalysisStatus'
import type { AnalysisStatusType, JobStatus } from '../types'

const API_BASE =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') || 'http://localhost:8080'

export function AnalyzePage() {
  const [username, setUsername] = useState('')
  const [jobId, setJobId] = useState<string | null>(null)
  const [status, setStatus] = useState<AnalysisStatusType>('idle')
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [totalBatches, setTotalBatches] = useState<number | null>(null)

  const introCopy = useMemo(
    () => ({
      title: 'Analyze your chess openings',
      desc: 'Kick off an analysis job and track progress as batches complete.',
    }),
    [],
  )

  const startAnalysis = async (evt: FormEvent) => {
    evt.preventDefault()
    const user = username.trim()
    if (!user) return

    setStatus('starting')
    setError(null)
    setProgress(0)
    setJobId(null)
    setTotalBatches(null)

    try {
      const res = await fetch(
        `${API_BASE}/chessgames/${encodeURIComponent(user)}?months=3&limit=200`,
      )
      if (!res.ok) {
        throw new Error(`Failed to start analysis (status ${res.status})`)
      }
      const body = await res.json()
      const newJobId: string | undefined = body?.job_id
      const batches: number | undefined = body?.batches

      if (!newJobId || !batches) {
        setStatus('completed')
        setProgress(100)
        return
      }

      setJobId(newJobId)
      setTotalBatches(batches)
      setStatus('running')
      setProgress(0)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unexpected error'
      setError(message)
      setStatus('failed')
    }
  }

  useEffect(() => {
    if (!jobId || (status !== 'running' && status !== 'starting')) return

    const poll = async () => {
      try {
        const res = await fetch(`${API_BASE}/jobs/${jobId}`)
        if (!res.ok) {
          throw new Error(`Status request failed (${res.status})`)
        }
        const body = await res.json()
        const job: JobStatus | undefined = body?.job
        if (!job) {
          throw new Error('Malformed job status response')
        }
        const completed = job.completed_batches ?? 0
        const total = job.total_batches ?? totalBatches ?? 0
        const pct = total > 0 ? Math.min(100, Math.round((completed / total) * 100)) : 0
        setProgress(pct)

        if (job.status === 'completed' || completed >= total) {
          setStatus('completed')
          setProgress(100)
        } else if (job.status === 'failed') {
          setStatus('failed')
          setError('Analysis failed')
        } else {
          setStatus('running')
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Error polling status'
        setError(message)
        setStatus('failed')
      }
    }

    const id = window.setInterval(poll, 1500)
    return () => window.clearInterval(id)
  }, [jobId, status, totalBatches])

  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Chess Insights</div>
          <div className="headline">{introCopy.title}</div>
          <p className="summary">{introCopy.desc}</p>
        </div>

        <UsernameForm
          username={username}
          onUsernameChange={setUsername}
          onSubmit={startAnalysis}
          isDisabled={status === 'starting' || status === 'running'}
        />
      </section>

      <AnalysisStatus status={status} progress={progress} error={error} />
    </main>
  )
}
