import { type FormEvent, useEffect, useMemo, useRef, useState } from 'react'
import { useAuth0 } from '@auth0/auth0-react'
import { UsernameForm } from '../components/UsernameForm'
import { AnalysisStatus } from '../components/AnalysisStatus'
import { ErrorsList } from '../components/ErrorsList'
import type { AnalysisStatusType, ErrorsResponse, JobStatus, MeResponse } from '../types'
import { authFetch } from '../utils/api'

const API_BASE =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') || 'http://localhost:8080'

export function AnalyzePage() {
  const { getAccessTokenSilently } = useAuth0()
  const [username, setUsername] = useState('')
  const [me, setMe] = useState<MeResponse | null>(null)
  const [meError, setMeError] = useState<string | null>(null)
  const [jobId, setJobId] = useState<string | null>(null)
  const [status, setStatus] = useState<AnalysisStatusType>('idle')
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [totalBatches, setTotalBatches] = useState<number | null>(null)
  const [months, setMonths] = useState(3)
  const [limit, setLimit] = useState<number | ''>(100)
  const [engineDepth, setEngineDepth] = useState<number | ''>(14)
  const [engineMoveTime, setEngineMoveTime] = useState<number | ''>(50)
  const [engineUseDepth, setEngineUseDepth] = useState(false)
  const [errorsData, setErrorsData] = useState<ErrorsResponse | null>(null)
  const [errorsLoading, setErrorsLoading] = useState(false)
  const [errorsError, setErrorsError] = useState<string | null>(null)
  const [showExistingModal, setShowExistingModal] = useState(false)
  const [existingCount, setExistingCount] = useState(0)
  const [pendingUsername, setPendingUsername] = useState<string | null>(null)
  const jobRef = useRef<JobStatus | null>(null)
  const targetProgressRef = useRef(0) // snap-to value from backend
  const rafId = useRef<number | null>(null)
  const pollId = useRef<number | null>(null)

  const ERROR_CACHE_KEY = 'errorsCache'
  const depthVal = typeof engineDepth === 'number' ? engineDepth : NaN
  const moveTimeVal = typeof engineMoveTime === 'number' ? engineMoveTime : NaN
  const depthValid = depthVal >= 8 && depthVal <= 20
  const moveTimeValid = moveTimeVal >= 25 && moveTimeVal <= 1000
  const quotaBlocked = me?.plan === 'FREE' && (me.remaining ?? 0) <= 0
  const submitDisabled =
    status === 'starting' ||
    status === 'running' ||
    !username.trim() ||
    !depthValid ||
    !moveTimeValid ||
    quotaBlocked

  useEffect(() => {
    const loadMe = async () => {
      setMeError(null)
      try {
        const res = await authFetch(`${API_BASE}/me`, undefined, getAccessTokenSilently)
        if (!res.ok) {
          throw new Error(`Failed to load usage (${res.status})`)
        }
        const body = (await res.json()) as MeResponse
        setMe(body)
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to load usage'
        setMeError(message)
      }
    }

    loadMe()
  }, [getAccessTokenSilently])

  // Restore cached errors (e.g., when returning from position page)
  useEffect(() => {
    if (errorsData) return
    const cached = sessionStorage.getItem(ERROR_CACHE_KEY)
    if (!cached) return
    try {
      const parsed = JSON.parse(cached) as { username: string; data: ErrorsResponse }
      if (parsed?.data && parsed?.username) {
        setErrorsData(parsed.data)
        if (!username) {
          setUsername(parsed.username)
        }
      }
    } catch {
      // ignore malformed cache
    }
  }, [errorsData, username])

  const introCopy = useMemo(
    () => ({
      title: 'Analyze your chess openings',
      desc: 'Kick off an analysis of your openings. Expect about 15 seconds of analysis for every 100 games',
    }),
    [],
  )

  const startAnalysis = async (evt: FormEvent, overrideUser?: string) => {
    evt.preventDefault()
    const user = (overrideUser ?? username).trim()
    if (!user) return

    if (!depthValid || !moveTimeValid) {
      setError('Please enter a depth (8-20) or movetime (25-1000ms).')
      return
    }

    setStatus('starting')
    setError(null)
    setProgress(0)
    setJobId(null)
    setTotalBatches(null)
    jobRef.current = null
    targetProgressRef.current = 0
    setErrorsData(null)
    setErrorsError(null)
    sessionStorage.removeItem(ERROR_CACHE_KEY)

    try {
      if (!overrideUser) {
        const countRes = await authFetch(
          `${API_BASE}/games/count/${encodeURIComponent(user)}`,
          undefined,
          getAccessTokenSilently,
        )
        if (countRes.ok) {
          const countBody = await countRes.json()
          const countVal = Number(countBody?.count ?? 0)
          if (countVal > 0) {
            setExistingCount(countVal)
            setPendingUsername(user)
            setShowExistingModal(true)
            setStatus('idle')
            return
          }
        }
      }

      const cappedLimit =
        typeof limit === 'number' ? Math.max(1, Math.min(limit, 500)) : 200
      const params = new URLSearchParams({
        months: String(months),
        limit: String(cappedLimit),
        engine_depth: String(depthVal),
        engine_move_time: String(moveTimeVal),
        engine_depth_or_time: String(engineUseDepth),
      })
      const res = await authFetch(
        `${API_BASE}/chessgames/${encodeURIComponent(user)}?${params.toString()}`,
        undefined,
        getAccessTokenSilently,
      )
      if (!res.ok) {
        const body = await res.json().catch(() => null)
        if (res.status === 402 || res.status === 429) {
          setError(
            body?.message ||
              "You've used your 100 free games this week. Upgrade to PRO for unlimited analysis.",
          )
          setStatus('failed')
          return
        }
        throw new Error(`Failed to start analysis (status ${res.status})`)
      }
      const body = await res.json()
      const newJobId: string | undefined = body?.job_id
      const batches: number | undefined = body?.batches
      const analyzedCount: number = Number(body?.count ?? 0)

      if (me?.plan === 'FREE') {
        setMe((prev) =>
          prev
            ? {
                ...prev,
                analysesUsed: prev.analysesUsed + analyzedCount,
                remaining:
                  prev.remaining != null
                    ? Math.max(0, prev.remaining - analyzedCount)
                    : prev.remaining,
              }
            : prev,
        )
      }

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
        const res = await authFetch(
          `${API_BASE}/jobs/${jobId}`,
          undefined,
          getAccessTokenSilently,
        )
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
          fetchErrors(username)
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

  useEffect(() => {
    if (!showExistingModal) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        handleCloseModal()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [showExistingModal])

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

  const fetchErrors = async (user: string) => {
    setErrorsLoading(true)
    setErrorsError(null)
    try {
      const res = await authFetch(
        `${API_BASE}/errors/${encodeURIComponent(user)}`,
        undefined,
        getAccessTokenSilently,
      )
      if (!res.ok) {
        throw new Error(`Errors request failed (${res.status})`)
      }
      const body = (await res.json()) as ErrorsResponse
      setErrorsData(body)
      sessionStorage.setItem(
        ERROR_CACHE_KEY,
        JSON.stringify({ username: user, data: body }),
      )
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load errors'
      setErrorsError(message)
    } finally {
      setErrorsLoading(false)
    }
  }

  const handleFetchExisting = async () => {
    if (!pendingUsername) return
    setShowExistingModal(false)
    await fetchErrors(pendingUsername)
    setStatus('completed')
    setProgress(100)
  }

  const handleRunNew = async () => {
    if (!pendingUsername) return
    setShowExistingModal(false)
    await startAnalysis(new Event('submit') as unknown as FormEvent, pendingUsername)
  }

  const handleCloseModal = () => {
    setShowExistingModal(false)
    setPendingUsername(null)
  }

  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Theory Gap</div>
          <div className="headline">{introCopy.title}</div>
          <p className="summary">{introCopy.desc}</p>
          {me ? (
            me.plan === 'PRO' ? (
              <p className="summary">Pro plan: unlimited analyses each week.</p>
            ) : (
              <p className="summary">
                Free plan: {me.remaining ?? 0} analyses left this week (limit{' '}
                {me.weeklyLimit ?? 100}).
              </p>
            )
          ) : meError ? (
            <p className="summary">{meError}</p>
          ) : null}
        </div>

        <UsernameForm
          username={username}
          onUsernameChange={setUsername}
          months={months}
          onMonthsChange={setMonths}
          limit={limit}
          onLimitChange={setLimit}
          engineDepth={engineDepth}
          onEngineDepthChange={setEngineDepth}
          engineMoveTime={engineMoveTime}
          onEngineMoveTimeChange={setEngineMoveTime}
          engineUseDepth={engineUseDepth}
          onEngineUseDepthChange={setEngineUseDepth}
          onSubmit={startAnalysis}
          onFetchErrors={() => fetchErrors(username)}
          isSubmitDisabled={submitDisabled}
          isFetchDisabled={status === 'starting' || status === 'running'}
        />
      </section>

      <AnalysisStatus status={status} progress={progress} error={error} />
      {(status === 'completed' || errorsLoading || errorsError || errorsData) && (
        <ErrorsList data={errorsData} isLoading={errorsLoading} error={errorsError} />
      )}

      {showExistingModal && (
        <div
          className="modal-overlay"
          role="presentation"
          onClick={handleCloseModal}
        >
          <div
            className="modal panel"
            role="dialog"
            aria-modal="true"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="headline" style={{ fontSize: 22 }}>
              Existing analysis found
            </div>
            <p className="summary">
              {existingCount} games are already ingested for {pendingUsername}. Would you like to
              fetch the existing analysis or run a new analysis?
            </p>
            <div className="controls modal-actions" style={{ justifyContent: 'flex-end' }}>
              <button className="button button--ghost" type="button" onClick={handleFetchExisting}>
                Fetch existing analysis
              </button>
              <button className="button" type="button" onClick={handleRunNew}>
                Run new analysis
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  )
}
