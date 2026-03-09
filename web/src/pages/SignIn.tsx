import { SignIn, useAuth } from '@clerk/react'
import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  clearStoredCliRedirectUri,
  getCliRedirectUri,
  getPostAuthPath,
  loadStoredCliRedirectUri,
  postCliToken,
  storeCliRedirectUri,
} from '../lib/auth'
import './page.css'

export function SignInPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { getToken, isLoaded, isSignedIn, orgId } = useAuth()
  const [cliStatus, setCliStatus] = useState<'idle' | 'posting' | 'done' | 'error'>('idle')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const queryRedirectUri = getCliRedirectUri(searchParams)

  useEffect(() => {
    if (queryRedirectUri) {
      storeCliRedirectUri(queryRedirectUri)
    }
  }, [queryRedirectUri])

  useEffect(() => {
    if (!isLoaded || !isSignedIn) {
      return
    }

    const redirectUri = queryRedirectUri ?? loadStoredCliRedirectUri()
    if (!redirectUri) {
      navigate(getPostAuthPath(orgId), { replace: true })
      return
    }

    let cancelled = false

    async function handoffToken() {
      setCliStatus('posting')
      setErrorMessage(null)
      try {
        if (!redirectUri) {
          throw new Error('The CLI callback URL is missing.')
        }
        let token: string | null
        try {
          token = await getToken({ template: 'codefind_cli' })
        } catch (error) {
          const message =
            error instanceof Error ? error.message : 'Clerk failed to mint the codefind_cli token.'
          throw new Error(`Template token request failed: ${message}`)
        }
        if (!token) {
          throw new Error('No Clerk session token was available for the CLI callback.')
        }
        await postCliToken(redirectUri, token)
        clearStoredCliRedirectUri()
        if (!cancelled) {
          setCliStatus('done')
        }
      } catch (error) {
        if (!cancelled) {
          setCliStatus('error')
          setErrorMessage(error instanceof Error ? error.message : 'CLI token handoff failed.')
        }
      }
    }

    void handoffToken()
    return () => {
      cancelled = true
    }
  }, [getToken, isLoaded, isSignedIn, navigate, orgId, queryRedirectUri])

  const redirectUri = queryRedirectUri ?? loadStoredCliRedirectUri()
  const isCliFlow = Boolean(redirectUri)

  return (
    <section className="page-grid">
      <Card className="hero-card border-white/60 bg-white/75 shadow-2xl shadow-sky-950/10 backdrop-blur">
        <CardHeader>
        <p className="eyebrow">Invite-only access</p>
          <CardTitle className="text-3xl font-semibold tracking-tight">
            Sign in with your approved Code-Find account.
          </CardTitle>
          <CardDescription className="lede text-base text-slate-600">
            Browser users continue into the app once organization access is active. CLI users
            return a session token to their local callback listener by POST only.
          </CardDescription>
        </CardHeader>
        <CardContent>
        {isCliFlow ? (
          <p className="status-chip">CLI callback detected: {redirectUri}</p>
        ) : null}
        </CardContent>
      </Card>
      <Card className="panel-card auth-panel border-white/70 bg-white/88 shadow-2xl shadow-sky-950/8 backdrop-blur">
        <CardHeader>
          <CardTitle>{cliStatus === 'done' ? 'CLI login complete' : 'Sign in'}</CardTitle>
          <CardDescription>
            {cliStatus === 'done'
              ? 'You can return to the terminal. The token handoff succeeded.'
              : 'Use the Clerk sign-in flow tied to your approved invitation or existing account.'}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
        {cliStatus === 'done' ? (
          <p className="text-sm text-slate-600">The CLI callback accepted the posted token.</p>
        ) : cliStatus === 'error' ? (
          <p className="text-sm text-slate-600">
            The CLI token handoff failed. Review the error below before retrying this login flow.
          </p>
        ) : isSignedIn ? (
          <p className="text-sm text-slate-600">Finishing CLI sign-in and returning your token…</p>
        ) : (
          <SignIn signUpUrl="/no-access" />
        )}
        {cliStatus === 'posting' ? <p>Posting your token back to the CLI callback...</p> : null}
        {errorMessage ? <p className="error-text">{errorMessage}</p> : null}
        </CardContent>
      </Card>
    </section>
  )
}
