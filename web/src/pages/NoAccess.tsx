import { Show, UserButton, useAuth } from '@clerk/react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import './page.css'

export function NoAccessPage() {
  const { isLoaded, orgId } = useAuth()

  return (
    <section className="page-grid">
      <Card className="hero-card border-white/60 bg-white/75 shadow-2xl shadow-sky-950/10 backdrop-blur">
        <CardHeader>
        <p className="eyebrow">Approval required</p>
          <CardTitle className="text-3xl font-semibold tracking-tight">
            Your account is signed in, but no organization access is active.
          </CardTitle>
          <CardDescription className="lede text-base text-slate-600">
          Code-Find is invite-only. New accounts must come through an administrator-issued
          invitation link. If you were not invited, you cannot create an organization or access
          the product from this page.
          </CardDescription>
        </CardHeader>
      </Card>
      <Card className="panel-card auth-panel border-white/70 bg-white/88 shadow-2xl shadow-sky-950/8 backdrop-blur">
        <CardHeader>
          <CardTitle>Current state</CardTitle>
          <CardDescription>
            {!isLoaded
              ? 'Checking your session...'
              : orgId
                ? 'Organization detected.'
                : 'No active organization.'}
          </CardDescription>
        </CardHeader>
        <CardContent>
        <div className="inline-actions">
          <Show when="signed-out">
            <Button asChild type="button">
              <Link to="/signin">Back to sign in</Link>
            </Button>
          </Show>
          <Show when="signed-in">
            <UserButton />
          </Show>
        </div>
        </CardContent>
      </Card>
    </section>
  )
}
