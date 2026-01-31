import './App.css'
import { BrowserRouter, Link, Route, Routes, useLocation } from 'react-router-dom'
import { useAuth0 } from '@auth0/auth0-react'
import { useEffect, useState } from 'react'
import { AnalyzePage } from './pages/AnalyzePage'
import { PositionPage } from './pages/PositionPage'
import { BillingPage } from './pages/BillingPage'
import { BillingSuccessPage } from './pages/BillingSuccessPage'
import { BillingCancelPage } from './pages/BillingCancelPage'
import { TermsPage } from './pages/TermsPage'
import { PrivacyPage } from './pages/PrivacyPage'
import { LoginButton } from './components/LoginButton'
import { LogoutButton } from './components/LogoutButton'
import { Profile } from './components/Profile'
import type { MeResponse } from './types'
import { authFetch } from './utils/api'

const API_BASE =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') || 'http://localhost:8080'

function AppRoutes() {
  const { isAuthenticated, isLoading, error, getAccessTokenSilently } = useAuth0()
  const [me, setMe] = useState<MeResponse | null>(null)
  const [meLoading, setMeLoading] = useState(false)
  const location = useLocation()

  useEffect(() => {
    if (!isAuthenticated) return
    const loadMe = async () => {
      setMeLoading(true)
      try {
        const res = await authFetch(`${API_BASE}/me`, undefined, getAccessTokenSilently)
        if (!res.ok) {
          throw new Error(`Failed to load usage (${res.status})`)
        }
        const body = (await res.json()) as MeResponse
        setMe(body)
      } catch {
        setMe(null)
      } finally {
        setMeLoading(false)
      }
    }

    loadMe()
  }, [getAccessTokenSilently, isAuthenticated])

  if (isLoading) {
    return (
      <main className="page">
        <section className="hero auth-hero">
          <div>
            <img src="/theory-gap-logo.png" alt="Theory Gap logo" className="auth-logo" />
            <div className="headline">Loading your session</div>
            <p className="summary">Syncing your workspace and preferences.</p>
          </div>
          <div className="panel auth-panel">
            <div className="status loading">Checking authentication...</div>
          </div>
        </section>
      </main>
    )
  }

  if (error) {
    return (
      <main className="page">
        <section className="hero auth-hero">
          <div>
            <img src="/theory-gap-logo.png" alt="Theory Gap logo" className="auth-logo" />
            <div className="headline">We hit a snag</div>
            <p className="summary">Please try signing in again.</p>
          </div>
          <div className="panel auth-panel">
            <div className="status error">{error.message}</div>
            <LoginButton />
          </div>
        </section>
      </main>
    )
  }

  if (!isAuthenticated) {
    if (location.pathname === '/terms' || location.pathname === '/privacy') {
      return (
        <Routes>
          <Route path="/terms" element={<TermsPage />} />
          <Route path="/privacy" element={<PrivacyPage />} />
          <Route path="*" element={<TermsPage />} />
        </Routes>
      )
    }
    return (
      <main className="page">
        <section className="hero auth-hero">
          <div>
            <img src="/theory-gap-logo.png" alt="Theory Gap logo" className="auth-logo" />
            <div className="headline">Theory Gap</div>
            <p className="summary">
              Analyze your chess openings move by move, spot recurring mistakes, and sharpen your prep.
            </p>
          </div>
          <div className="panel auth-panel">
            <div className="auth-title">Sign in to continue</div>
            <div className="auth-copy">
              Connect your account to unlock personalized analysis and saved positions.
            </div>
            <LoginButton />
          </div>
          <div className="auth-links">
            <a href="/terms">Terms</a>
            <a href="/privacy">Privacy</a>
          </div>
        </section>
      </main>
    )
  }

  return (
    <div className="app-shell">
      <header className="app-header">
        <Link className="app-brand-link" to="/">
          <div className="app-brand">
            <img src="/theory-gap-logo.png" alt="Theory Gap logo" className="app-logo" />
            <div>
              <div className="app-title">Theory Gap</div>
              <div className="app-subtitle">Opening analysis workspace</div>
            </div>
          </div>
        </Link>
        <div className="app-actions">
          <Profile />
          {!meLoading && (
            <Link className="button button--ghost" to="/billing">
              {me?.plan === 'PRO' ? 'Billing' : 'Go Pro!'}
            </Link>
          )}
          <LogoutButton />
        </div>
      </header>

      <Routes>
        <Route path="/" element={<AnalyzePage />} />
        <Route path="/position/:id" element={<PositionPage />} />
        <Route path="/billing" element={<BillingPage />} />
        <Route path="/settings/billing" element={<BillingPage />} />
        <Route path="/billing/success" element={<BillingSuccessPage />} />
        <Route path="/billing/cancel" element={<BillingCancelPage />} />
        <Route path="/terms" element={<TermsPage />} />
        <Route path="/privacy" element={<PrivacyPage />} />
      </Routes>
    </div>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  )
}
