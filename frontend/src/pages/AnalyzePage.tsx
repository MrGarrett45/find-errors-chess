import { type FormEvent, useEffect, useMemo, useRef, useState } from 'react'
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
  const [months, setMonths] = useState(3)
  const [limit, setLimit] = useState<number | ''>(200)
  const jobRef = useRef<JobStatus | null>(null)
  const targetProgressRef = useRef(0) // snap-to value from backend
  const rafId = useRef<number | null>(null)
  const pollId = useRef<number | null>(null)

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
    jobRef.current = null
    targetProgressRef.current = 0

    try {
      const cappedLimit =
        typeof limit === 'number' ? Math.max(1, Math.min(limit, 500)) : 200
      const res = await fetch(
        `${API_BASE}/chessgames/${encodeURIComponent(user)}?months=${months}&limit=${cappedLimit}`,
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

      const nextJob: JobStatus = {
        id: newJobId,
        total_batches: batches,
        completed_batches: 0,
        status: 'running',
      }
      jobRef.current = nextJob
      setJobId(newJobId)
      setTotalBatches(batches)
      setStatus('running')
      setProgress(0)
      startSmoothing()
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
        jobRef.current = job
        const completed = job.completed_batches ?? 0
        const total = job.total_batches ?? totalBatches ?? 0
        const snap = total > 0 ? Math.min(100, Math.round((completed / total) * 100)) : 0
        targetProgressRef.current = snap
        setProgress((prev) => Math.max(prev, snap))

        if (job.status === 'completed' || completed >= total) {
          setStatus('completed')
          setProgress(100)
          stopSmoothing()
        } else if (job.status === 'failed') {
          setStatus('failed')
          setError('Analysis failed')
          stopSmoothing()
        } else {
          setStatus('running')
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Error polling status'
        setError(message)
        setStatus('failed')
        stopSmoothing()
      }
    }

    pollId.current = window.setInterval(poll, 1500)
    startSmoothing()
    return () => {
      if (pollId.current !== null) {
        window.clearInterval(pollId.current)
        pollId.current = null
      }
      stopSmoothing()
    }
    // we intentionally omit startSmoothing/stopSmoothing from deps to avoid restarting raf
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [jobId, status, totalBatches])

  const startSmoothing = () => {
    stopSmoothing()
    const tick = () => {
      setProgress((prev) => {
        const job = jobRef.current
        if (!job || job.total_batches <= 0) return prev

        const perBatch = 100 / job.total_batches
        const completed = job.completed_batches ?? 0
        const lower = completed * perBatch
        // cap the last batch at <100% until completion is confirmed
        const isLastBatch = completed + 1 >= job.total_batches
        const batchCap = Math.min(100, lower + perBatch)
        const upper = isLastBatch ? Math.min(99, batchCap) : batchCap

        let next = prev
        const target = targetProgressRef.current

        // Never lag behind the backend snap value
        if (next < target) {
          next = target
        }

        // Ease toward the current batch cap, but never cross it.
        if (next < upper) {
          const distance = upper - next
          // slow easing (half again)
          const increment = Math.max(0.02, distance * 0.00375)
          next = Math.min(upper, next + increment)
        }

        return next
      })
      rafId.current = requestAnimationFrame(tick)
    }
    rafId.current = requestAnimationFrame(tick)
  }

  const stopSmoothing = () => {
    if (rafId.current !== null) {
      cancelAnimationFrame(rafId.current)
      rafId.current = null
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

        <UsernameForm
          username={username}
          onUsernameChange={setUsername}
          months={months}
          onMonthsChange={setMonths}
          limit={limit}
          onLimitChange={setLimit}
          onSubmit={startAnalysis}
          isDisabled={status === 'starting' || status === 'running'}
        />
      </section>

      <AnalysisStatus status={status} progress={progress} error={error} />
    </main>
  )
}
