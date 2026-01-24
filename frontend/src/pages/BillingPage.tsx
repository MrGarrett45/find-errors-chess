import { useEffect, useState } from 'react'
import { useAuth0 } from '@auth0/auth0-react'
import type { MeResponse } from '../types'
import { authFetch, createBillingPortalSession, createCheckoutSession } from '../utils/api'

const API_BASE =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') || 'http://localhost:8080'

export function BillingPage() {
  const { getAccessTokenSilently } = useAuth0()
  const [me, setMe] = useState<MeResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [checkoutError, setCheckoutError] = useState<string | null>(null)
  const [checkoutLoading, setCheckoutLoading] = useState(false)
  const [portalError, setPortalError] = useState<string | null>(null)
  const [portalLoading, setPortalLoading] = useState(false)

  useEffect(() => {
    const loadMe = async () => {
      setLoading(true)
      setError(null)
      try {
        const res = await authFetch(`${API_BASE}/me`, undefined, getAccessTokenSilently)
        if (!res.ok) {
          throw new Error(`Failed to load usage (${res.status})`)
        }
        const body = (await res.json()) as MeResponse
        setMe(body)
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to load plan'
        setError(message)
      } finally {
        setLoading(false)
      }
    }

    loadMe()
  }, [getAccessTokenSilently])

  const handleUpgradeToPro = async () => {
    setCheckoutError(null)
    setCheckoutLoading(true)
    try {
      const token = await getAccessTokenSilently()
      const { url } = await createCheckoutSession(token)
      window.location.href = url
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Failed to start checkout'
      setCheckoutError(message)
      setCheckoutLoading(false)
    }
  }

  const handleManageBilling = async () => {
    setPortalError(null)
    setPortalLoading(true)
    try {
      const token = await getAccessTokenSilently()
      const { url } = await createBillingPortalSession(token)
      window.location.href = url
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Failed to open billing portal'
      setPortalError(message)
      setPortalLoading(false)
    }
  }

  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Billing</div>
          <div className="headline">Manage your plan</div>
          {me?.plan === 'PRO' ? (
            <p className="summary">Manage your subscription details below.</p>
          ) : (
            <p className="summary">
              Upgrade to Pro for unlimited weekly analysis.
            </p>
          )}
        </div>
        <div className="panel">
          {loading ? (
            <div className="status loading">Loading your plan...</div>
          ) : error ? (
            <div className="status error">{error}</div>
          ) : me?.plan === 'PRO' ? (
            <div>
              <div className="status">You are on the Pro plan ($5/month).</div>
              <div className="controls">
                <button
                  className="button"
                  onClick={handleManageBilling}
                  disabled={portalLoading}
                >
                  Manage billing
                </button>
              </div>
              {portalError ? <p className="summary">{portalError}</p> : null}
            </div>
          ) : (
            <div>
              <button
                className="button"
                onClick={handleUpgradeToPro}
                disabled={checkoutLoading}
              >
                Upgrade to Pro $5/month
              </button>
              {checkoutError ? (
                <p className="summary">{checkoutError}</p>
              ) : null}
            </div>
          )}
        </div>
      </section>
    </main>
  )
}
