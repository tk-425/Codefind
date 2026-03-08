import { SignIn, SignUp, useAuth } from '@clerk/react'
import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { getPostAuthPath } from '../lib/auth'
import './page.css'

export function AcceptInvitationPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { isLoaded, isSignedIn, orgId } = useAuth()

  const ticket = searchParams.get('__clerk_ticket')
  const accountStatus = searchParams.get('__clerk_status')

  useEffect(() => {
    if (!isLoaded || !isSignedIn) {
      return
    }
    navigate(getPostAuthPath(orgId), { replace: true })
  }, [isLoaded, isSignedIn, navigate, orgId])

  if (!ticket) {
    return (
      <section className="page-grid">
        <Card className="hero-card border-white/60 bg-white/75 shadow-2xl shadow-sky-950/10 backdrop-blur">
          <CardHeader>
          <p className="eyebrow">Invitation required</p>
            <CardTitle className="text-3xl font-semibold tracking-tight">
              This route only works from a Clerk invitation link.
            </CardTitle>
            <CardDescription className="lede text-base text-slate-600">
            Access is invite-only. Open the invitation URL sent to your email to continue.
            </CardDescription>
          </CardHeader>
        </Card>
      </section>
    )
  }

  return (
    <section className="page-grid">
      <Card className="hero-card border-white/60 bg-white/75 shadow-2xl shadow-sky-950/10 backdrop-blur">
        <CardHeader>
        <p className="eyebrow">Accept invitation</p>
          <CardTitle className="text-3xl font-semibold tracking-tight">
            Join your assigned Code-Find organization.
          </CardTitle>
          <CardDescription className="lede text-base text-slate-600">
          Clerk owns the invitation ticket in the URL. This route completes the correct prebuilt
          sign-in or sign-up flow and then returns you to the approved app state.
          </CardDescription>
        </CardHeader>
      </Card>
      <Card className="panel-card auth-panel border-white/70 bg-white/88 shadow-2xl shadow-sky-950/8 backdrop-blur">
        <CardHeader>
          <CardTitle>{accountStatus === 'sign_in' ? 'Invitation sign-in' : 'Finish invited sign-up'}</CardTitle>
          <CardDescription>
            Clerk continues the invitation flow using the ticket already attached to this URL.
          </CardDescription>
        </CardHeader>
        <CardContent>
        {accountStatus === 'sign_in' ? <SignIn /> : <SignUp />}
        </CardContent>
      </Card>
    </section>
  )
}
