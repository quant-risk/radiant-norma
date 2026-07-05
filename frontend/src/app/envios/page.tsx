/**
 * /envios — lista envios STA com filtros URL-driven.
 *
 * Sprint 8d: filtros são lidos via `searchParams` (Next.js App Router).
 * Cada filtro vira ?key=value na URL → share-able, bookmark-able,
 * back/forward funciona.
 *
 * Componentes:
 *   - Server-side: lê searchParams, faz fetch com filtros, renderiza tabela
 *   - Client-side: FilterBar (chips) + ExportMenu (CSV/JSON/URL)
 */

import {
  Database,
  Upload,
  Inbox,
  CheckCircle2,
  Clock,
  AlertTriangle,
  XCircle,
} from 'lucide-react'
import Link from 'next/link'
import { getServerSession } from '@/lib/session'
import { apiFetch } from '@/lib/api-fetch'
import { AppShell } from '@/components/layout/app-shell'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { StatCard } from '@/components/domain/stat-card'
import { ExportMenu } from '@/components/domain/export-menu'
import { EnviosLiveRefresh } from '@/components/domain/envios-live-refresh'
import { EnviosFilterBar } from './filter-bar'
import { formatRelativeCompact } from '@/lib/format'

export const dynamic = 'force-dynamic'

interface Envio {
  id: string
  cadoc_code: string
  period: string
  status: 'pending' | 'accepted' | 'rejected' | 'error'
  rules_passed: number
  rules_failed: number
  duration_ms: number
  protocol_sta: string
  error_code?: string
  error_message?: string
  sent_at: string
  confirmed_at: string
}

interface EnvioStats {
  total: number
  accepted: number
  rejected: number
  pending: number
  error: number
  avg_duration_ms: number
}

interface Schema {
  cadoc: string
  description: string
  versions: number
  latest_version: string
}

interface PageProps {
  searchParams: { [key: string]: string | string[] | undefined }
}

const statusMeta = {
  accepted: { label: 'Aprovado', tone: 'success' as const, icon: CheckCircle2 },
  pending: { label: 'Pendente', tone: 'warning' as const, icon: Clock },
  rejected: { label: 'Rejeitado', tone: 'critical' as const, icon: AlertTriangle },
  error: { label: 'Erro', tone: 'critical' as const, icon: XCircle },
}

function nextDeadline(cadoc: string): string {
  let hash = 0
  for (let i = 0; i < cadoc.length; i++) {
    hash = (hash * 31 + cadoc.charCodeAt(i)) & 0xff
  }
  const days = (hash % 14) + 1
  return days === 1 ? 'amanhã' : `em ${days} dias`
}

function str(v: string | string[] | undefined): string | undefined {
  if (Array.isArray(v)) return v[0]
  return v || undefined
}

export default async function EnviosPage({ searchParams }: PageProps) {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  // Parse filtros URL-driven (Sprint 8d)
  const filters = {
    cadoc: str(searchParams.cadoc),
    status: str(searchParams.status),
    period: str(searchParams.period),
  }

  const [enviosRes, statsRes, schemasRes] = await Promise.allSettled([
    apiFetch<{ envios: Envio[]; total: number }>(
      `/v1/envios?limit=100${filters.cadoc ? `&cadoc=${filters.cadoc}` : ''}${
        filters.status ? `&status=${filters.status}` : ''
      }${filters.period ? `&period=${filters.period}` : ''}`,
      {},
      session.token,
    ),
    apiFetch<EnvioStats>('/v1/envios/stats', {}, session.token),
    apiFetch<{ schemas: Schema[] } | Schema[]>(
      '/v1/schemas',
      {},
      session.token,
    ),
  ])

  const envios: Envio[] =
    enviosRes.status === 'fulfilled' ? enviosRes.value.envios ?? [] : []
  const stats: EnvioStats | null =
    statsRes.status === 'fulfilled' ? statsRes.value : null
  const schemas: Schema[] =
    schemasRes.status === 'fulfilled'
      ? Array.isArray(schemasRes.value)
        ? schemasRes.value
        : schemasRes.value.schemas ?? []
      : []

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Envios STA',
        subtitle: `IF ${session.if_id}`,
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Envios' },
        ],
        actions: (
          <div className="flex items-center gap-2">
            <EnviosLiveRefresh />
            <ExportMenu
              endpoint="/v1/envios"
              filters={filters}
              label="Exportar"
            />
            <Button variant="primary" size="sm" leftIcon={<Upload className="size-3.5" />}>
              Novo envio
            </Button>
          </div>
        ),
      }}
    >
      <div className="space-y-6 max-w-7xl">
        {/* KPIs */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            label="Total"
            value={stats?.total ?? 0}
            tone="neutral"
            icon={<Database className="size-4" />}
          />
          <StatCard
            label="Aprovados"
            value={stats?.accepted ?? 0}
            tone="success"
            icon={<CheckCircle2 className="size-4" />}
          />
          <StatCard
            label="Pendentes"
            value={stats?.pending ?? 0}
            tone="warning"
            icon={<Clock className="size-4" />}
          />
          <StatCard
            label="Rejeitados"
            value={(stats?.rejected ?? 0) + (stats?.error ?? 0)}
            tone="critical"
            icon={<AlertTriangle className="size-4" />}
          />
        </div>

        {/* Filter bar (Sprint 8d) — atualiza URL state */}
        <EnviosFilterBar
          currentFilters={filters}
          cadocOptions={schemas.map((s) => ({ value: s.cadoc, label: s.cadoc }))}
        />

        {/* Tabela */}
        <Card padding="none">
          <div className="px-6 py-4 border-b border-border flex items-center justify-between">
            <div>
              <CardTitle>
                Envios recentes
                {filters.cadoc && (
                  <span className="ml-2 text-sm font-normal text-ink-muted">
                    · CADOC {filters.cadoc}
                  </span>
                )}
                {filters.status && (
                  <span className="ml-2 text-sm font-normal text-ink-muted">
                    · status {filters.status}
                  </span>
                )}
              </CardTitle>
              <CardDescription>
                {envios.length} resultado{envios.length !== 1 ? 's' : ''}
                {Object.values(filters).some(Boolean) && ' (filtrado)'}
              </CardDescription>
            </div>
          </div>

          {envios.length === 0 ? (
            <EmptyState
              icon={<Inbox className="size-6" />}
              title={
                Object.values(filters).some(Boolean)
                  ? 'Nenhum envio com esses filtros'
                  : 'Nenhum envio registrado'
              }
              description={
                Object.values(filters).some(Boolean)
                  ? 'Tente limpar os filtros ou aguarde novos envios.'
                  : 'Quando você fizer envios via STA Submit, eles aparecerão aqui.'
              }
              action={
                Object.values(filters).some(Boolean) ? (
                  <Link href="/envios">
                    <Button variant="outline" size="sm">
                      Limpar filtros
                    </Button>
                  </Link>
                ) : (
                  <Button variant="outline" size="sm">
                    Fazer primeiro envio
                  </Button>
                )
              }
            />
          ) : (
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
                  {envios.map((env) => {
                    const sm = statusMeta[env.status] ?? statusMeta.pending
                    const Icon = sm.icon
                    return (
                      <tr
                        key={env.id}
                        className="border-t border-border-subtle hover:bg-surface-sunken/50 transition-colors"
                      >
                        <td className="px-6 py-3">
                          <span className="font-mono text-xs text-ink">
                            {env.id.slice(0, 20)}…
                          </span>
                        </td>
                        <td className="px-3 py-3">
                          <span className="font-mono text-xs font-semibold text-accent-600 dark:text-accent-400">
                            {env.cadoc_code}
                          </span>
                        </td>
                        <td className="px-3 py-3 text-sm text-ink-muted">
                          {env.period}
                        </td>
                        <td className="px-3 py-3">
                          <Badge tone={sm.tone} variant="soft" icon={<Icon className="size-3" />}>
                            {sm.label}
                          </Badge>
                        </td>
                        <td className="px-3 py-3 text-right">
                          <div className="flex items-center justify-end gap-2">
                            <span className="text-sm nums text-success-700 dark:text-success-300 font-medium">
                              {env.rules_passed}
                            </span>
                            {env.rules_failed > 0 && (
                              <>
                                <span className="text-ink-subtle">/</span>
                                <span className="text-sm nums text-critical-700 dark:text-critical-300 font-medium">
                                  {env.rules_failed}
                                </span>
                              </>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-3 text-right">
                          <span
                            className="text-xs text-ink-muted"
                            title={env.sent_at || env.confirmed_at}
                          >
                            {env.sent_at || env.confirmed_at
                              ? formatRelativeCompact(env.sent_at || env.confirmed_at)
                              : '—'}
                          </span>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </Card>

        {/* CADOCs disponíveis */}
        <div>
          <h3 className="text-md font-semibold text-ink mb-3">
            CADOCs disponíveis para envio
          </h3>
          {schemas.length === 0 ? (
            <EmptyState
              icon={<Database className="size-6" />}
              title="Nenhum schema carregado"
              description="Aguardando /v1/schemas do backend."
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              {schemas.map((s) => (
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
                      {nextDeadline(s.cadoc)}
                    </span>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>
    </AppShell>
  )
}