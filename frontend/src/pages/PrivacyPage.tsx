export function PrivacyPage() {
  return (
    <main className="page">
      <section className="hero">
        <div>
          <div className="badge">Legal</div>
          <h1 className="headline">Privacy Policy</h1>
          <p className="summary">Last updated: January 30, 2026</p>
          <p className="summary">
            Theory Gap is a chess analysis application. This policy explains how
            personal data is collected and used.
          </p>
        </div>
      </section>

      <section className="panel" style={{ marginTop: 20 }}>
        <h2 className="engine-panel__title">1. Introduction</h2>
        <p className="summary">
          This policy applies to the Theory Gap website and application.
        </p>

        <h2 className="engine-panel__title">2. Information We Collect</h2>
        <p className="summary">
          Account information: email address, name, and profile image (if
          provided via Google Sign-In).
        </p>
        <p className="summary">
          Authentication data: unique user ID provided by Google OAuth.
        </p>
        <p className="summary">
          Chess data: games, PGNs, analysis results, and user preferences.
        </p>
        <p className="summary">
          Usage data: pages visited, interactions, and basic analytics events.
        </p>
        <p className="summary">
          Technical data: IP address, browser/device info, cookies or local
          storage.
        </p>

        <h2 className="engine-panel__title">3. How We Use Information</h2>
        <p className="summary">
          We use data to create and manage user accounts, authenticate users via
          Google Sign-In, provide chess analysis features, and improve
          reliability, performance, and security.
        </p>

        <h2 className="engine-panel__title">4. Google OAuth and Google User Data</h2>
        <p className="summary">
          Theory Gap uses Google OAuth solely for authentication. We only request
          basic profile scopes (email and profile). Google user data is not used
          for advertising and is not sold. We do not share Google user data with
          third parties except as required to operate the service. We do not
          request access to other Google services or data beyond sign-in.
        </p>

        <h2 className="engine-panel__title">5. Data Sharing</h2>
        <p className="summary">
          Personal data may be shared only with infrastructure and service
          providers (such as hosting or monitoring) acting on our behalf, or with
          legal authorities if required by law. Personal data is not sold.
        </p>

        <h2 className="engine-panel__title">6. Data Retention</h2>
        <p className="summary">
          Account data is retained while the account is active. Upon account
          deletion, personal data is deleted or anonymized within a reasonable
          time (about 30 days), except for backups or legal obligations.
        </p>

        <h2 className="engine-panel__title">7. Data Deletion and Account Removal</h2>
        <p className="summary">
          You may request account and data deletion by emailing
          support@theorygap.com. Deletion removes account information and
          associated chess data.
        </p>

        <h2 className="engine-panel__title">8. Cookies and Local Storage</h2>
        <p className="summary">
          We use cookies or local storage for authentication, session management,
          and preferences. You can control cookies through your browser settings.
        </p>

        <h2 className="engine-panel__title">9. Security</h2>
        <p className="summary">
          We use reasonable technical and organizational safeguards to protect
          data. No system can be 100% secure.
        </p>

        <h2 className="engine-panel__title">10. Children</h2>
        <p className="summary">
          The service is not intended for users under 13. Theory Gap does not
          knowingly collect data from children.
        </p>

        <h2 className="engine-panel__title">11. Changes to This Policy</h2>
        <p className="summary">
          We may update this policy periodically. The "Last updated" date above
          reflects changes.
        </p>

        <h2 className="engine-panel__title">12. Contact</h2>
        <p className="summary">Contact: garrettmclaughlin1980@gmail.com</p>
      </section>
    </main>
  )
}
