/**
 * /envios — visão consolidada de envios STA.
 *
 * Backend não tem /v1/envios ainda (TODO Sprint 8), então mostramos:
 *   1. Cards por CADOC com status agregado (última sincronização,
 *      regras passando, próximas janelas)
 *   2. Tabela de envios recentes (mock data — virá do backend)
 *   3. CTA "Novo envio" (ainda disabled — virá Sprint 8c)
 *
 * Filosofia: página deve ser ÚTIL mesmo sem dados reais. Mostra o
 * skeleton do que vai existir + status operacional dos schemas.
 */

import {
  Send,
  Calendar,
  CheckCircle2,
  Clock,
  AlertTriangle,
  Upload,
  Database,
} from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { apiFetch } from '@/lib/api-fetch'
import { AppShell } from '@/components/layout/app-shell'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { StatCard } from '@/components/domain/stat-card'
import { EmptyState } from '@/components/ui/empty-state'
import { cn } from '@/lib/utils'
import { formatRelativeCompact } from '@/lib/format'

export const dynamic = 'force-dynamic'

interface Schema {
  cadoc: string
  description: string
  versions: number
  latest_version: string
}

async function getData() {
  const session = await getServerSession()
  if (!session) return null
  const res = await Promise.allSettled([
    apiFetch<{ schemas: Schema[] } | Schema[]>(
      '/v1/schemas',
      {},
      session.token,
    ),
  ])
  const schemas =
    res[0].status === 'fulfilled'
      ? Array.isArray(res[0].value)
        ? res[0].value
        : res[0].value.schemas ?? []
      : []
  return { session, schemas }
}

// Mock de envios recentes (Sprint 8c: substituir por /v1/envios)
const recentEnvios = [
  {
    id: 'ENV-2026-00184',
    cadoc: '3040',
    periodo: '05/2026',
    status: 'approved' as const,
    submittedAt: new Date(Date.now() - 12 * 60_000).toISOString(),
    rulesPassed: 58,
    rulesFailed: 2,
    approver: 'sistema',
  },
  {
    id: 'ENV-2026-00183',
    cadoc: '3050',
    periodo: '05/2026',
    status: 'approved' as const,
    submittedAt: new Date(Date.now() - 5 * 3600_000).toISOString(),
    rulesPassed: 42,
    rulesFailed: 0,
    approver: 'sistema',
  },
  {
    id: 'ENV-2026-00182',
    cadoc: '3040',
    periodo: '04/2026',
    status: 'pending' as const,
    submittedAt: new Date(Date.now() - 18 * 3600_000).toISOString(),
    rulesPassed: 0,
    rulesFailed: 0,
    approver: undefined,
  },
  {
    id: 'ENV-2026-00181',
    cadoc: '3040',
    periodo: '04/2026',
    status: 'rejected' as const,
    submittedAt: new Date(Date.now() - 26 * 3600_000).toISOString(),
    rulesPassed: 50,
    rulesFailed: 10,
    approver: 'sistema',
  },
  {
    id: 'ENV-2026-00180',
    cadoc: '4020',
    periodo: '04/2026',
    status: 'approved' as const,
    submittedAt: new Date(Date.now() - 2 * 24 * 3600_000).toISOString(),
    rulesPassed: 28,
    rulesFailed: 1,
    approver: 'sistema',
  },
]

const statusMeta = {
  approved: {
    label: 'Aprovado',
    tone: 'success' as const,
    icon: CheckCircle2,
  },
  pending: {
    label: 'Em processamento',
    tone: 'warning' as const,
    icon: Clock,
  },
  rejected: {
    label: 'Rejeitado',
    tone: 'critical' as const,
    icon: AlertTriangle,
  },
}

export default async function EnviosPage() {
  const data = await getData()
  if (!data) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  const { session, schemas } = data

  const stats = {
    total: recentEnvios.length,
    approved: recentEnvios.filter((e) => e.status === 'approved').length,
    pending: recentEnvios.filter((e) => e.status === 'pending').length,
    rejected: recentEnvios.filter((e) => e.status === 'rejected').length,
  }

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Envios STA',
        subtitle: `${stats.total} envios · IF ${session.if_id}`,
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Envios' },
        ],
        actions: (
          <Button variant="primary" size="sm" leftIcon={<Upload className="size-3.5" />}>
            Novo envio
          </Button>
        ),
      }}
    >
      <div className="space-y-6 max-w-7xl">
        {/* KPIs */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            label="Total"
            value={stats.total}
            tone="neutral"
            icon={<Send className="size-4" />}
          />
          <StatCard
            label="Aprovados"
            value={stats.approved}
            tone="success"
            icon={<CheckCircle2 className="size-4" />}
          />
          <StatCard
            label="Em processamento"
            value={stats.pending}
            tone="warning"
            icon={<Clock className="size-4" />}
          />
          <StatCard
            label="Rejeitados"
            value={stats.rejected}
            tone="critical"
            icon={<AlertTriangle className="size-4" />}
          />
        </div>

        {/* Tabela de envios recentes */}
        <Card padding="none">
          <div className="px-6 py-4 border-b border-border flex items-center justify-between">
            <div>
              <CardTitle>Envios recentes</CardTitle>
              <CardDescription>
                Últimos 30 dias · ordenado por mais recente
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost" size="sm">
                <Calendar className="size-3.5" />
                Período
              </Button>
              <Button variant="outline" size="sm">
                Exportar CSV
              </Button>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-surface-sunken">
                  <th className="text-left text-2xs uppercase tracking-wider font-semibold text-ink-subtle px-6 py-2.5">
                    ID
                  </th>
                  <th className="text-left text-2xs uppercase tracking-wider font-semibold text-ink-subtle px-3 py-2.5">
                    CADOC
                  </th>
                  <th className="text-left text-2xs uppercase tracking-wider font-semibold text-ink-subtle px-3 py-2.5">
                    Período
                  </th>
                  <th className="text-left text-2xs uppercase tracking-wider font-semibold text-ink-subtle px-3 py-2.5">
                    Status
                  </th>
                  <th className="text-right text-2xs uppercase tracking-wider font-semibold text-ink-subtle px-3 py-2.5">
                    Regras
                  </th>
                  <th className="text-right text-2xs uppercase tracking-wider font-semibold text-ink-subtle px-6 py-2.5">
                    Enviado
                  </th>
                </tr>
              </thead>
              <tbody>
                {recentEnvios.map((env) => {
                  const sm = statusMeta[env.status]
                  const Icon = sm.icon
                  return (
                    <tr
                      key={env.id}
                      className="border-t border-border-subtle hover:bg-surface-sunken/50 transition-colors"
                    >
                      <td className="px-6 py-3">
                        <span className="font-mono text-xs text-ink">
                          {env.id}
                        </span>
                      </td>
                      <td className="px-3 py-3">
                        <span className="font-mono text-xs font-semibold text-accent-600 dark:text-accent-400">
                          {env.cadoc}
                        </span>
                      </td>
                      <td className="px-3 py-3 text-sm text-ink-muted">
                        {env.periodo}
                      </td>
                      <td className="px-3 py-3">
                        <Badge tone={sm.tone} variant="soft" icon={<Icon className="size-3" />}>
                          {sm.label}
                        </Badge>
                      </td>
                      <td className="px-3 py-3 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <span className="text-sm nums text-success-700 dark:text-success-300 font-medium">
                            {env.rulesPassed}
                          </span>
                          {env.rulesFailed > 0 && (
                            <>
                              <span className="text-ink-subtle">/</span>
                              <span className="text-sm nums text-critical-700 dark:text-critical-300 font-medium">
                                {env.rulesFailed}
                              </span>
                            </>
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-3 text-right">
                        <span
                          className="text-xs text-ink-muted"
                          title={new Date(env.submittedAt).toLocaleString('pt-BR')}
                        >
                          {formatRelativeCompact(env.submittedAt)}
                        </span>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </Card>

        {/* CADOCs disponíveis */}
        <div>
          <h3 className="text-md font-semibold text-ink mb-3">
            CADOCs disponíveis
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {(schemas.length > 0 ? schemas : mockSchemas).map((s) => {
              const next = nextDeadline(s.cadoc)
              return (
                <Card key={s.cadoc} padding="md" className="group">
                  <div className="flex items-start justify-between gap-3 mb-3">
                    <div className="flex items-center gap-2">
                      <div className="size-9 rounded-lg bg-accent-50 dark:bg-accent-950 text-accent-600 dark:text-accent-400 flex items-center justify-center">
                        <Database className="size-4" />
                      </div>
                      <div>
                        <div className="font-mono text-sm font-semibold text-ink">
                          {s.cadoc}
                        </div>
                        <div className="text-xs text-ink-muted">
                          v{s.latest_version}
                        </div>
                      </div>
                    </div>
                    <Badge tone="success" variant="soft" dot>
                      ativo
                    </Badge>
                  </div>
                  <p className="text-sm text-ink-muted line-clamp-2 mb-3">
                    {s.description}
                  </p>
                  <div className="flex items-center justify-between pt-3 border-t border-border-subtle">
                    <span className="text-xs text-ink-muted">
                      Próximo deadline
                    </span>
                    <span className="text-xs font-medium text-ink nums">
                      {next}
                    </span>
                  </div>
                </Card>
              )
            })}
          </div>
        </div>
      </div>
    </AppShell>
  )
}

const mockSchemas: Schema[] = [
  { cadoc: '3040', description: 'Risco de Crédito — exposição e provisão', versions: 1, latest_version: '8.2.1' },
  { cadoc: '3050', description: 'Risco de Mercado — exposição em derivativos', versions: 1, latest_version: '6.1.0' },
  { cadoc: '3060', description: 'Risco de Liquidez — LCR e fluxo de caixa', versions: 1, latest_version: '4.0.2' },
  { cadoc: '3070', description: 'Capital — Basiléia III e PR a definir', versions: 1, latest_version: '5.3.0' },
  { cadoc: '4020', description: 'Operações de Crédito — concessões mensais', versions: 1, latest_version: '3.2.1' },
  { cadoc: '4030', description: 'Carteira ativa — operações vigentes', versions: 1, latest_version: '2.8.0' },
]

function nextDeadline(cadoc: string): string {
  // Mock: cálculo simplificado baseado no CADOC
  const days: Record<string, number> = {
    '3040': 5,
    '3050': 8,
    '3060': 12,
    '3070': 15,
    '4020': 3,
    '4030': 3,
  }
  const d = days[cadoc] ?? 7
  return d === 1 ? 'amanhã' : `em ${d} dias`
}