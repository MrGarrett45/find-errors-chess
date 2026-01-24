import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth0 } from '@auth0/auth0-react'
import type { MeResponse } from '../types'
import { authFetch, updatePlan } from '../utils/api'

const API_BASE =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') || 'http://localhost:8080'

export function BillingSuccessPage() {
  const { getAccessTokenSilently } = useAuth0()
  const [me, setMe] = useState<MeResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const loadMe = async () => {
      setLoading(true)
      setError(null)
      try {
        const token = await getAccessTokenSilently()
        await updatePlan(token, 'PRO')
        const res = await authFetch(`${API_BASE}/me`, undefined, getAccessTokenSilently)
        if (!res.ok) {
          throw new Error(`Failed to load usage (${res.status})`)
        }
        const body = (await res.json()) as MeResponse
        setMe(body)
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to refresh plan'
        setError(message)
      } finally {
        setLoading(false)
      }
    }

    loadMe()
  }, [getAccessTokenSilently])

  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Billing</div>
          <div className="headline">Thanks for upgrading to Pro!</div>
          <p className="summary">
            Your subscription is now active. Enjoy unlimited weekly analyses.
          </p>
        </div>
        <div className="panel">
          {loading ? (
            <div className="status loading">Refreshing your plan...</div>
          ) : error ? (
            <div className="status error">{error}</div>
          ) : me ? (
            <div className="status">Current plan: {me.plan}</div>
          ) : null}
          <div className="controls">
            <Link className="button button--ghost" to="/billing">
              Back to billing
            </Link>
            <Link className="button" to="/">
              Go to analysis
            </Link>
          </div>
        </div>
      </section>
    </main>
  )
}
