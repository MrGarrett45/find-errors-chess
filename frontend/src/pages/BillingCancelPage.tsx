import { Link } from 'react-router-dom'

export function BillingCancelPage() {
  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Billing</div>
          <div className="headline">Payment canceled</div>
          <p className="summary">
            Your payment was canceled. You are still on the Free plan.
          </p>
        </div>
        <div className="panel">
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
