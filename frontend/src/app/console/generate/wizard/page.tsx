/**
 * /console/generate/wizard — Sprint 60: Wizard UI de Geração de CADOCs.
 *
 * Server Component — chama getServerSession (server-only) e renderiza
 * AppShell com WizardClient (client component).
 */
import { Suspense } from 'react'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { WizardClient } from './wizard-client'

export const dynamic = 'force-dynamic'

export default async function WizardPage() {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-12 text-center">
        <p className="text-ink-muted">Sessão expirada.</p>
      </div>
    )
  }

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Wizard de Geração',
        subtitle: 'CSV/XLSX → CanonicalDocument → XML',
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Console' },
          { label: 'Generator' },
          { label: 'Wizard' },
        ],
      }}
    >
      <Suspense fallback={<div className="p-8 text-ink-muted text-sm">Carregando wizard…</div>}>
        <WizardClient />
      </Suspense>
    </AppShell>
  )
}
