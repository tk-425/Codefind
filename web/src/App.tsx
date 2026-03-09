import { Show, UserButton, useAuth } from '@clerk/react'
import { Link, Navigate, Route, Routes } from 'react-router-dom'
import './App.css'
import { Button } from '@/components/ui/button'
import { SearchPage } from './pages/Search'
import { AcceptInvitationPage } from './pages/AcceptInvitation'
import { NoAccessPage } from './pages/NoAccess'
import { SignInPage } from './pages/SignIn'

function ApprovedRoute({ children }: { children: React.ReactNode }) {
  const { isLoaded, isSignedIn, orgId } = useAuth()

  if (!isLoaded) {
    return <main className="app-shell loading-shell">Loading authentication...</main>
  }

  if (!isSignedIn) {
    return <Navigate replace to="/signin" />
  }

  if (!orgId) {
    return <Navigate replace to="/no-access" />
  }

  return <>{children}</>
}

function ShellHeader() {
  return (
    <header className="shell-header">
      <div>
        <p className="shell-kicker">Code-Find v2</p>
        <h1 className="shell-title">Invite-only multi-tenant auth flow</h1>
      </div>
      <div className="shell-actions">
        <Show when="signed-out">
          <Button asChild type="button" variant="outline">
            <Link to="/signin">Sign in</Link>
          </Button>
        </Show>
        <Show when="signed-in">
          <UserButton />
        </Show>
      </div>
    </header>
  )
}

function App() {
  return (
    <main className="app-shell">
      <ShellHeader />
      <Routes>
        <Route path="/" element={<Navigate replace to="/signin" />} />
        <Route path="/signin" element={<SignInPage />} />
        <Route path="/accept-invitation" element={<AcceptInvitationPage />} />
        <Route path="/no-access" element={<NoAccessPage />} />
        <Route
          path="/search"
          element={
            <ApprovedRoute>
              <SearchPage />
            </ApprovedRoute>
          }
        />
      </Routes>
    </main>
  )
}

export default App
