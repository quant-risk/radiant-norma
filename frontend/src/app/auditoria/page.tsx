/**
 * /auditoria — audit log tamper-evident (LGPD/SOC 2 compliance).
 *
 * Validação 29 (C5 fix): NÃO finge integridade. Até o backend expor
 * /v1/audit_log, mostra empty state honesto com explicabilidade do
 * mecanismo de hash chain.
 */

import {
  Shield,
  Hash,
  Lock,
  Download,
  Activity as ActivityIcon,
} from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'

export const dynamic = 'force-dynamic'

export default async function AuditoriaPage() {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  // Validação 29 (C5 fix): chainValid = true era hardcoded → fingia
  // integridade LGPD/SOC 2. Agora: empty state honesto até ter dados.
  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Auditoria',
        subtitle: 'Logs tamper-evident · LGPD / SOC 2',
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Auditoria' },
        ],
        actions: (
          <Button variant="outline" size="sm" leftIcon={<Download className="size-3.5" />} disabled>
            Exportar
          </Button>
        ),
      }}
    >
      <div className="space-y-6 max-w-6xl">
        {/* Banner de status */}
        <Card padding="lg" className="text-center bg-warning-50/50 dark:bg-warning-950/20 border-warning-200 dark:border-warning-900">
          <div className="size-12 mx-auto mb-4 rounded-full bg-warning-100 dark:bg-warning-950 text-warning-600 dark:text-warning-400 flex items-center justify-center">
            <ActivityIcon className="size-6" />
          </div>
          <CardTitle className="mb-2">
            Audit log ainda não populado
          </CardTitle>
          <CardDescription className="max-w-md mx-auto">
            Quando o backend expor{' '}
            <code className="text-2xs font-mono bg-surface-raised px-1 py-0.5 rounded">
              GET /v1/audit_log
            </code>{' '}
            (Sprint 8c, role admin), esta página vai listar eventos imutáveis
            com SHA-256 hash chain, contadores de integridade, e o último
            hash verificado.
          </CardDescription>
          <Badge tone="warning" variant="soft" className="mt-3">
            aguardando dados
          </Badge>
        </Card>

        <div className="grid lg:grid-cols-3 gap-6">
          {/* Empty state principal */}
          <div className="lg:col-span-2">
            <Card padding="none">
              <div className="px-6 py-4 border-b border-border flex items-center justify-between">
                <div>
                  <CardTitle>Eventos recentes</CardTitle>
                  <CardDescription>
                    Audit log imutável com SHA-256 hash chain
                  </CardDescription>
                </div>
              </div>
              <EmptyState
                icon={<ActivityIcon className="size-6" />}
                title="Sem eventos no período"
                description="Nenhum evento de auditoria registrado para esta IF ainda."
              />
            </Card>
          </div>

          {/* Como funciona */}
          <div className="space-y-3">
            <Card padding="md">
              <div className="flex items-center gap-2 mb-3">
                <Lock className="size-4 text-accent-600 dark:text-accent-400" />
                <h3 className="text-md font-semibold text-ink">
                  Como funciona
                </h3>
              </div>
              <ol className="space-y-3 text-sm text-ink-muted">
                <li className="flex gap-3">
                  <span className="size-5 rounded-full bg-accent-50 dark:bg-accent-950 text-accent-600 dark:text-accent-400 flex items-center justify-center text-2xs font-semibold shrink-0">
                    1
                  </span>
                  <span>
                    Toda mutação na API emite um entry no{' '}
                    <code className="text-2xs font-mono bg-surface-sunken px-1 py-0.5 rounded">
                      audit_log
                    </code>
                  </span>
                </li>
                <li className="flex gap-3">
                  <span className="size-5 rounded-full bg-accent-50 dark:bg-accent-950 text-accent-600 dark:text-accent-400 flex items-center justify-center text-2xs font-semibold shrink-0">
                    2
                  </span>
                  <span>
                    Cada entry referencia SHA-256 da entry anterior (chain)
                  </span>
                </li>
                <li className="flex gap-3">
                  <span className="size-5 rounded-full bg-accent-50 dark:bg-accent-950 text-accent-600 dark:text-accent-400 flex items-center justify-center text-2xs font-semibold shrink-0">
                    3
                  </span>
                  <span>
                    Modify qualquer entry → chain quebrada →{' '}
                    <code className="text-2xs font-mono bg-surface-sunken px-1 py-0.5 rounded">
                      auditlog.Verify()
                    </code>{' '}
                    detecta
                  </span>
                </li>
                <li className="flex gap-3">
                  <span className="size-5 rounded-full bg-accent-50 dark:bg-accent-950 text-accent-600 dark:text-accent-400 flex items-center justify-center text-2xs font-semibold shrink-0">
                    4
                  </span>
                  <span>
                    Storage: SQLite local ou Postgres (driver dual, validação 21)
                  </span>
                </li>
              </ol>
            </Card>

            <Card padding="md">
              <h3 className="text-md font-semibold text-ink mb-2">
                Compliance
              </h3>
              <ul className="space-y-2 text-sm text-ink-muted">
                <li className="flex items-start gap-2">
                  <Badge tone="success" variant="soft" className="shrink-0 mt-0.5">
                    LGPD
                  </Badge>
                  <span>
                    Logs não contêm dados pessoais sensíveis (Art. 5º II)
                  </span>
                </li>
                <li className="flex items-start gap-2">
                  <Badge tone="success" variant="soft" className="shrink-0 mt-0.5">
                    SOC 2
                  </Badge>
                  <span>
                    Tamper-evident + retention configurável (default 5 anos)
                  </span>
                </li>
                <li className="flex items-start gap-2">
                  <Badge tone="success" variant="soft" className="shrink-0 mt-0.5">
                    BACEN
                  </Badge>
                  <span>
                    Compatível com requisitos de auditoria regulatória
                  </span>
                </li>
              </ul>
            </Card>
          </div>
        </div>
      </div>
    </AppShell>
  )
}