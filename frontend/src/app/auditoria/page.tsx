/**
 * /auditoria — audit log tamper-evident (LGPD/SOC 2).
 *
 * Sprint 8: vai puxar /v1/audit_log (admin only). Por enquanto: activity
 * feed mock + explicabilidade do mecanismo de hash chain.
 */

import {
  History,
  Shield,
  Hash,
  Lock,
  Download,
  Activity as ActivityIcon,
} from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import {
  ActivityFeed,
  type ActivityItem,
} from '@/components/domain/activity-feed'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { StatCard } from '@/components/domain/stat-card'

export const dynamic = 'force-dynamic'

// Mock audit log (vai vir de /v1/audit_log)
const mockAuditLog: ActivityItem[] = [
  {
    id: 'a1',
    kind: 'envio.approved',
    timestamp: new Date(Date.now() - 12 * 60_000).toISOString(),
    actor: '9999901',
    description: 'CADOC 3040 base 05/2026 aprovado · 58 regras passaram',
    payload: { envio_id: 'ENV-2026-00184', rules_passed: 58 },
  },
  {
    id: 'a2',
    kind: 'radar.detected',
    timestamp: new Date(Date.now() - 47 * 60_000).toISOString(),
    description: 'URL BACEN 3040 alterada · severidade warn',
    payload: { cadoc: '3040', old_url: 'https://...', new_url: 'https://...' },
  },
  {
    id: 'a3',
    kind: 'rule.enabled',
    timestamp: new Date(Date.now() - 2 * 3600_000).toISOString(),
    actor: '9999901',
    description: 'Regra B12 habilitada — campos obrigatórios',
    payload: { rule: 'B12', previous: 'disabled' },
  },
  {
    id: 'a4',
    kind: 'schema.synced',
    timestamp: new Date(Date.now() - 5 * 3600_000).toISOString(),
    description: 'Schema 3040 v8.2.1 sincronizado do portal BACEN',
    payload: { schema: '3040', from: '8.2.0', to: '8.2.1' },
  },
  {
    id: 'a5',
    kind: 'auth.login',
    timestamp: new Date(Date.now() - 8 * 3600_000).toISOString(),
    actor: '9999901',
    payload: { ip: '10.0.x.x', user_agent: 'Chrome/macOS' },
  },
  {
    id: 'a6',
    kind: 'envio.rejected',
    timestamp: new Date(Date.now() - 26 * 3600_000).toISOString(),
    description: 'CADOC 3040 base 04/2026 rejeitado · 10 regras falharam',
    payload: { envio_id: 'ENV-2026-00181', rules_failed: 10 },
  },
  {
    id: 'a7',
    kind: 'rule.disabled',
    timestamp: new Date(Date.now() - 32 * 3600_000).toISOString(),
    actor: '9999901',
    description: 'Regra S05 desabilitada — sem impacto',
    payload: { rule: 'S05', reason: 'false_positive' },
  },
  {
    id: 'a8',
    kind: 'auth.dev_token',
    timestamp: new Date(Date.now() - 48 * 3600_000).toISOString(),
    actor: '9999901',
    description: 'Token dev mintado (apenas dev mode)',
  },
]

export default async function AuditoriaPage() {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  // Verificação da chain (mock — em prod: backend faz auditlog.Verify())
  const chainValid = true
  const lastHash = 'a1b2c3d4e5f67890'

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
          <Button variant="outline" size="sm" leftIcon={<Download className="size-3.5" />}>
            Exportar
          </Button>
        ),
      }}
    >
      <div className="space-y-6 max-w-6xl">
        {/* Integridade da chain */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <StatCard
            label="Eventos (30d)"
            value={mockAuditLog.length * 142}
            tone="neutral"
            icon={<ActivityIcon className="size-4" />}
            helpText="Eventos verificados · sem falhas"
          />
          <StatCard
            label="Integridade da chain"
            value={chainValid ? 'OK' : 'QUEBRADA'}
            tone={chainValid ? 'success' : 'critical'}
            icon={<Shield className="size-4" />}
            helpText="SHA-256 hash chain verificada"
          />
          <StatCard
            label="Último hash"
            value={lastHash}
            tone="neutral"
            icon={<Hash className="size-4" />}
            helpText="8 primeiros caracteres"
          />
        </div>

        <div className="grid lg:grid-cols-3 gap-6">
          {/* Activity feed */}
          <div className="lg:col-span-2">
            <Card padding="md">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <CardTitle>Eventos recentes</CardTitle>
                  <CardDescription>
                    Audit log imutável com SHA-256 hash chain
                  </CardDescription>
                </div>
                <Badge tone="success" variant="soft" dot>
                  verificado
                </Badge>
              </div>
              <ActivityFeed items={mockAuditLog} />
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