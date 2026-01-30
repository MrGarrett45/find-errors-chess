export function PrivacyPage() {
  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Legal</div>
          <div className="headline">Privacy Policy</div>
          <p className="summary">
            This policy explains how Theory Gap collects and uses data.
          </p>
        </div>
      </section>

      <section className="panel" style={{ marginTop: 20 }}>
        <h2 className="engine-panel__title">1. Data We Collect</h2>
        <p className="summary">
          We collect account details from Auth0 (such as email and name), usage
          metrics, and chess analysis inputs needed to provide the service.
        </p>

        <h2 className="engine-panel__title">2. How We Use Data</h2>
        <p className="summary">
          We use data to operate, maintain, and improve Theory Gap, provide
          customer support, and enforce plan limits.
        </p>

        <h2 className="engine-panel__title">3. Payments</h2>
        <p className="summary">
          Payments are processed by Stripe. We do not store full payment card
          details on our servers.
        </p>

        <h2 className="engine-panel__title">4. Sharing</h2>
        <p className="summary">
          We do not sell your personal data. We share data only with trusted
          providers required to deliver the service (such as hosting and payment
          processing) or when required by law.
        </p>

        <h2 className="engine-panel__title">5. Data Retention</h2>
        <p className="summary">
          We retain data as long as needed to provide the service and comply with
          legal obligations. You may request deletion where applicable.
        </p>

        <h2 className="engine-panel__title">6. Security</h2>
        <p className="summary">
          We use industry-standard safeguards to protect data, including access
          controls and encryption in transit.
        </p>

        <h2 className="engine-panel__title">7. Your Rights</h2>
        <p className="summary">
          Depending on your jurisdiction, you may have rights to access, correct,
          or delete your data. Contact support for requests.
        </p>

        <h2 className="engine-panel__title">8. Changes</h2>
        <p className="summary">
          We may update this policy from time to time. Changes will be posted on
          this page.
        </p>
      </section>
    </main>
  )
}
