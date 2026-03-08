import { useAuth } from '@clerk/react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import './page.css'

export function SearchPage() {
  const { orgId, userId } = useAuth()

  return (
    <section className="page-grid">
      <Card className="hero-card border-white/60 bg-white/75 shadow-2xl shadow-sky-950/10 backdrop-blur">
        <CardHeader>
        <p className="eyebrow">Authenticated</p>
          <CardTitle className="text-3xl font-semibold tracking-tight">
            Search is unlocked for approved organization members.
          </CardTitle>
          <CardDescription className="lede text-base text-slate-600">
          Phase 5 focuses on auth only. Query/search functionality arrives in later phases.
          </CardDescription>
        </CardHeader>
      </Card>
      <Card className="panel-card auth-panel border-white/70 bg-white/88 shadow-2xl shadow-sky-950/8 backdrop-blur">
        <CardHeader>
          <CardTitle>Session context</CardTitle>
          <CardDescription>Authenticated Clerk context is available for later API calls.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-slate-700">
          <p>Org: {orgId ?? 'missing'}</p>
          <p>User: {userId ?? 'missing'}</p>
        </CardContent>
      </Card>
    </section>
  )
}
