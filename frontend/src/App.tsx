import './App.css'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { useAuth0 } from '@auth0/auth0-react'
import { AnalyzePage } from './pages/AnalyzePage'
import { PositionPage } from './pages/PositionPage'
import { LoginButton } from './components/LoginButton'
import { LogoutButton } from './components/LogoutButton'
import { Profile } from './components/Profile'

export default function App() {
  const { isAuthenticated, isLoading, error } = useAuth0()

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
            <div className="status loading">Checking authentication…</div>
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
    return (
      <main className="page">
        <section className="hero auth-hero">
          <div>
            <img src="/theory-gap-logo.png" alt="Theory Gap logo" className="auth-logo" />
            <div className="headline">Theory Gap</div>
            <p className="summary">
              Analyze your openings, spot recurring mistakes, and sharpen your prep.
            </p>
          </div>
          <div className="panel auth-panel">
            <div className="auth-title">Sign in to continue</div>
            <div className="auth-copy">
              Connect your account to unlock personalized analysis and saved positions.
            </div>
            <LoginButton />
          </div>
        </section>
      </main>
    )
  }

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="app-brand">
          <img src="/theory-gap-logo.png" alt="Theory Gap logo" className="app-logo" />
          <div>
            <div className="app-title">Theory Gap</div>
            <div className="app-subtitle">Opening analysis workspace</div>
          </div>
        </div>
        <div className="app-actions">
          <Profile />
          <LogoutButton />
        </div>
      </header>

      <BrowserRouter>
        <Routes>
          <Route path="/" element={<AnalyzePage />} />
          <Route path="/position/:id" element={<PositionPage />} />
        </Routes>
      </BrowserRouter>
    </div>
  )
}
