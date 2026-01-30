export function TermsPage() {
  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Legal</div>
          <div className="headline">Terms of Service</div>
          <p className="summary">
            These Terms govern your use of Theory Gap and related services.
          </p>
        </div>
      </section>

      <section className="panel" style={{ marginTop: 20 }}>
        <h2 className="engine-panel__title">1. Overview</h2>
        <p className="summary">
          Theory Gap provides chess analysis tools, including opening analysis and
          position review. By using the service you agree to these Terms.
        </p>

        <h2 className="engine-panel__title">2. Eligibility and Accounts</h2>
        <p className="summary">
          You must have a valid Auth0-authenticated account to access authenticated
          features. You are responsible for safeguarding your account and activity.
        </p>

        <h2 className="engine-panel__title">3. Subscriptions and Billing</h2>
        <p className="summary">
          Pro subscriptions are billed monthly via Stripe. You can manage or cancel
          your subscription in the customer portal. Cancellations take effect at
          the end of the current billing period.
        </p>

        <h2 className="engine-panel__title">4. Acceptable Use</h2>
        <p className="summary">
          You agree not to abuse the service, interfere with its operation, or use
          automated tooling to exceed plan limits. We may suspend accounts that
          violate these Terms.
        </p>

        <h2 className="engine-panel__title">5. Content and Data</h2>
        <p className="summary">
          You retain ownership of your data. You grant us a limited license to
          process your data to provide the service.
        </p>

        <h2 className="engine-panel__title">6. Availability and Changes</h2>
        <p className="summary">
          We aim to keep the service reliable but do not guarantee uninterrupted
          availability. We may update features or pricing with reasonable notice.
        </p>

        <h2 className="engine-panel__title">7. Disclaimers</h2>
        <p className="summary">
          The service is provided "as is" without warranties. We are not liable for
          indirect damages or losses arising from your use of the service.
        </p>

        <h2 className="engine-panel__title">8. Contact</h2>
        <p className="summary">
          Questions about these Terms? Contact support via garrettmclaughlin1980@gmail.com
        </p>
      </section>
    </main>
  )
}
