import './page.css'

export function SignInPage() {
  return (
    <section className="page-grid">
      <div className="hero-card">
        <p className="eyebrow">Code-Find v2</p>
        <h1>Multi-tenant search starts with a clean scaffold.</h1>
        <p className="lede">
          Phase 1 creates the shell for Clerk auth, org setup, and query search
          without implementing the business flow yet.
        </p>
      </div>
      <div className="panel-card">
        <h2>Sign In Placeholder</h2>
        <p>
          This page will host the Clerk sign-in flow in a later phase. The
          scaffold exists now so auth structure does not get bolted on later.
        </p>
      </div>
    </section>
  )
}
