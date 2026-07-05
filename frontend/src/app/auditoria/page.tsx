/**
 * /auditoria — audit log tamper-evident (LGPD/SOC 2 compliance).
 *
 * Sprint 8d: filtros URL-driven + ExportMenu (CSV/JSON).
 */

import {
  Shield,
  Hash,
  Lock,
  Activity as ActivityIcon,
} from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { apiFetch } from '@/lib/api-fetch'
import { AppShell } from '@/components/layout/app-shell'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { StatCard } from '@/components/domain/stat-card'
import { ActivityFeed, type ActivityItem } from '@/components/domain/activity-feed'
import { ExportMenu } from '@/components/domain/export-menu'
import { AuditoriaLiveRefresh } from '@/components/domain/auditoria-live-refresh'
import { AuditFilterBar } from './filter-bar'

export const dynamic = 'force-dynamic'

interface AuditEvent {
  id: number
  if_id: string
  actor: string
  action: string
  target: string
  description: string
  payload?: Record<string, unknown>
  created_at: string
}

interface AuditResponse {
  events: AuditEvent[]
  total: number
  chain_valid: boolean
}

interface PageProps {
  searchParams: { [key: string]: string | string[] | undefined }
}

function str(v: string | string[] | undefined): string | undefined {
  if (Array.isArray(v)) return v[0]
  return v || undefined
}

function normalizeAction(action: string): ActivityItem['kind'] {
  switch (action) {
    case 'envio.approved':
    case 'sta.submit':
      return 'envio.approved'
    case 'envio.rejected':
      return 'envio.rejected'
    case 'radar.detected':
      return 'radar.detected'
    case 'radar.resolved':
      return 'radar.detected'
    case 'rule.disabled':
      return 'rule.disabled'
    case 'rule.enabled':
      return 'rule.enabled'
    case 'schema.synced':
      return 'schema.synced'
    case 'auth.login':
      return 'auth.login'
    case 'auth.dev_token':
      return 'auth.dev_token'
    default:
      return 'envio.approved'
  }
}

export default async function AuditoriaPage({ searchParams }: PageProps) {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  const filters = {
    action: str(searchParams.action),
    if_id: str(searchParams.if_id),
  }

  const query = `/v1/audit_log?limit=100${
    filters.action ? `&action=${filters.action}` : ''
  }${filters.if_id ? `&if_id=${filters.if_id}` : ''}`

  const res = await Promise.allSettled([
    apiFetch<AuditResponse>(query, {}, session.token),
  ])
  const auditData: AuditResponse | null =
    res[0].status === 'fulfilled' ? res[0].value : null

  const events: AuditEvent[] = auditData?.events ?? []
  const chainValid = auditData?.chain_valid ?? false
  const activity: ActivityItem[] = events.map((e) => ({
    id: `audit-${e.id}`,
    kind: normalizeAction(e.action),
    timestamp: e.created_at,
    actor: e.actor,
    description: e.description,
    payload: e.payload,
  }))

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
          <div className="flex items-center gap-2">
            <AuditoriaLiveRefresh />
            <ExportMenu
              endpoint="/v1/audit_log"
              filters={filters}
              label="Exportar"
            />
          </div>
        ),
      }}
    >
      <div className="space-y-6 max-w-6xl">
        {/* Chain integrity stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <StatCard
            label="Eventos"
            value={events.length}
            tone="neutral"
            icon={<ActivityIcon className="size-4" />}
            helpText="Eventos retornados pelo /v1/audit_log"
          />
          <StatCard
            label="Integridade da chain"
            value={chainValid ? 'OK' : 'QUEBRADA'}
            tone={chainValid ? 'success' : 'critical'}
            icon={<Shield className="size-4" />}
            helpText="SHA-256 hash chain verificada pelo backend"
          />
          <StatCard
            label="Última verificação"
            value="agora"
            tone="neutral"
            icon={<Hash className="size-4" />}
            helpText="Cada request valida o chain completo"
          />
        </div>

        {/* Filter bar (Sprint 8d) */}
        <AuditFilterBar currentFilters={filters} />

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
                <Badge tone={chainValid ? 'success' : 'warning'} variant="soft" dot>
                  {chainValid ? 'verificado' : 'aguardando'}
                </Badge>
              </div>
              {activity.length === 0 ? (
                <EmptyState
                  icon={<ActivityIcon className="size-6" />}
                  title="Sem eventos no período"
                  description={
                    Object.values(filters).some(Boolean)
                      ? 'Nenhum evento com esses filtros.'
                      : 'Nenhum evento de auditoria registrado para esta IF ainda.'
                  }
                />
              ) : (
                <ActivityFeed items={activity} />
              )}
            </Card>
          </div>

          {/* Como funciona */}
          <div className="space-y-3">
            <Card padding="md">
              <div className="flex items-center gap-2 mb-3">
                <Lock className="size-4 text-accent-600 dark:text-accent-400" />
                <h3 className="text-md font-semibold text-ink">Como funciona</h3>
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
              <h3 className="text-md font-semibold text-ink mb-2">Compliance</h3>
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