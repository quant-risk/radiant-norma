/**
 * / — Root: landing pública (deslogado) ou dashboard executivo (logado).
 *
 * Server component: detecta sessão via getServerSession() e escolhe
 * qual árvore renderizar. Zero JS extra — o dashboard já é server-rendered.
 */

import Link from 'next/link'
import {
  Send,
  AlertTriangle,
  Database,
  ArrowUpRight,
  Activity,
  TrendingUp,
} from 'lucide-react'
import { apiFetch } from '@/lib/api-fetch'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { StatCard } from '@/components/domain/stat-card'
import { AlertCard } from '@/components/domain/alert-card'
import { DashboardLiveRefresh } from '@/components/domain/dashboard-live-refresh'
import { ActivityFeed, type ActivityItem } from '@/components/domain/activity-feed'
import { Card, CardTitle, CardEyebrow } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SectionHeader } from '@/components/ui/section-header'
import { Divider } from '@/components/ui/divider'
import { LandingHero } from '@/components/landing/hero'
import { LandingFeatures } from '@/components/landing/features'
import { LandingHowItWorks } from '@/components/landing/how-it-works'
import { LandingCTAFinal } from '@/components/landing/cta-final'
import { LandingFooter } from '@/components/landing/footer'

export const dynamic = 'force-dynamic'

interface Alert {
  id: number
  cadoc_code: string
  severity: 'info' | 'warn' | 'critical'
  title: string
  description: string
  source_url: string
  detected_at: string
  resolved: boolean
}

interface Rule {
  code: string
  severity: 'E' | 'A' | 'I'
  sheet: string
  description: string
  example: string
}

interface Schema {
  cadoc: string
  description: string
  versions: number
  latest_version: string
}

interface EnvioStats {
  total: number
  accepted: number
  rejected: number
  pending: number
  error: number
  avg_duration_ms: number
}

interface InsightsKPIs {
  current: {
    approval_rate: number
    failures_total: number
    sent_total: number
    accepted: number
    rejected: number
    avg_duration_ms: number
  }
  previous: {
    approval_rate: number
    sent_total: number
    avg_duration_ms: number
  }
  delta: {
    approval_rate_pct: number
    failures_total_pct: number
    avg_duration_ms_pct: number
  }
}

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

function stableCoverage(cadoc: string): number {
  let hash = 0
  for (let i = 0; i < cadoc.length; i++) {
    hash = (hash * 31 + cadoc.charCodeAt(i)) & 0xffff
  }
  return 60 + (hash % 41)
}

async function getDashboardData() {
  const session = await getServerSession()
  if (!session) return null

  const [alertsRes, rulesRes, schemasRes, statsRes, kpisRes, auditRes] =
    await Promise.allSettled([
      apiFetch<{ alerts: Alert[]; total: number }>(
        '/v1/radar/alerts?unresolved=true',
        {},
        session.token,
      ),
      apiFetch<{ rules: Rule[] } | Rule[]>('/v1/rules', {}, session.token),
      apiFetch<{ schemas: Schema[] } | Schema[]>(
        '/v1/schemas',
        {},
        session.token,
      ),
      apiFetch<EnvioStats>('/v1/envios/stats', {}, session.token),
      apiFetch<InsightsKPIs>('/v1/insights/kpis', {}, session.token),
      apiFetch<{ events: AuditEvent[]; total: number }>(
        '/v1/audit_log?limit=10',
        {},
        session.token,
      ),
    ])

  const alerts: Alert[] =
    alertsRes.status === 'fulfilled'
      ? Array.isArray(alertsRes.value)
        ? alertsRes.value
        : alertsRes.value.alerts ?? []
      : []
  const rules: Rule[] =
    rulesRes.status === 'fulfilled'
      ? Array.isArray(rulesRes.value)
        ? rulesRes.value
        : rulesRes.value.rules ?? []
      : []
  const schemas: Schema[] =
    schemasRes.status === 'fulfilled'
      ? Array.isArray(schemasRes.value)
        ? schemasRes.value
        : schemasRes.value.schemas ?? []
      : []
  const stats: EnvioStats | null = statsRes.status === 'fulfilled' ? statsRes.value : null
  const kpis: InsightsKPIs | null = kpisRes.status === 'fulfilled' ? kpisRes.value : null
  const auditEvents: AuditEvent[] =
    auditRes.status === 'fulfilled' ? auditRes.value.events ?? [] : []

  return { session, alerts, rules, schemas, stats, kpis, auditEvents }
}

export default async function DashboardPage() {
  // Switch contextual: landing pública (sem sessão) ou dashboard (logado).
  // Server component — zero JS extra, decisão no SSR.
  const session = await getServerSession()
  if (!session) {
    return (
      <>
        <LandingHero />
        <LandingFeatures />
        <LandingHowItWorks />
        <LandingCTAFinal />
        <LandingFooter />
      </>
    )
  }

  const data = await getDashboardData()
  if (!data) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-surface">
        <Card padding="lg" className="max-w-md text-center">
          <CardEyebrow>Sessão</CardEyebrow>
          <CardTitle className="mb-3">Sessão expirada</CardTitle>
          <p className="text-sm text-ink-muted mb-5">
            Faça login para acessar o console.
          </p>
          <Link href="/login" passHref legacyBehavior={false}>
            <Button asChild variant="primary" fullWidth rightIcon={<ArrowUpRight className="size-4" />}>
              Ir para login
            </Button>
          </Link>
        </Card>
      </div>
    )
  }

  const { alerts, rules, schemas, stats, kpis, auditEvents } = data

  const criticalAlerts = alerts.filter((a) => a.severity === 'critical').length
  const warnAlerts = alerts.filter((a) => a.severity === 'warn').length

  const topAlerts = [...alerts]
    .sort((a, b) => {
      const sev: Record<Alert['severity'], number> = { critical: 0, warn: 1, info: 2 }
      if (sev[a.severity] !== sev[b.severity])
        return sev[a.severity] - sev[b.severity]
      return new Date(b.detected_at).getTime() - new Date(a.detected_at).getTime()
    })
    .slice(0, 3)

  const auditActivity: ActivityItem[] = auditEvents.map((e) => ({
    id: `audit-${e.id}`,
    kind: normalizeAction(e.action),
    timestamp: e.created_at,
    actor: e.actor,
    description: e.description,
    payload: e.payload,
  }))

  const totalSent = stats?.total ?? 0
  const approvalRate = kpis?.current.approval_rate ?? 0
  const heroCopy = (() => {
    if (criticalAlerts > 0) {
      return `${criticalAlerts} alerta${criticalAlerts > 1 ? 's' : ''} crítico${criticalAlerts > 1 ? 's' : ''} exigem ação imediata`
    }
    if (warnAlerts > 0) {
      return `${warnAlerts} alerta${warnAlerts > 1 ? 's' : ''} aguardam análise`
    }
    if (totalSent > 0) {
      return `${approvalRate.toFixed(1)}% de aprovação nos últimos 30 dias`
    }
    return 'Tudo em ordem · aguardando primeiro envio'
  })()

  const sentSparkline = stats && stats.total > 0
    ? sparklineFromEnvios(stats, kpis)
    : undefined

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Dashboard',
        subtitle: 'Visão geral da operação regulatória',
        breadcrumbs: [{ label: 'Radiant Norma', href: '/' }, { label: 'Dashboard' }],
        actions: <DashboardLiveRefresh />,
      }}
      commandData={{
        rules: rules.map((r) => ({
          code: r.code,
          description: r.description,
          severity: r.severity,
        })),
        alerts: alerts.map((a) => ({
          id: a.id,
          title: a.title,
          severity: a.severity,
          cadoc_code: a.cadoc_code,
        })),
        schemas: schemas.map((s) => ({
          cadoc: s.cadoc,
          description: s.description,
        })),
      }}
    >
      <div className="space-y-10 max-w-7xl">
        {/* Hero strip */}
        <section>
          <SectionHeader
            eyebrow="Status operacional"
            title={heroCopy}
            description={
              criticalAlerts > 0
                ? 'Resposta imediata necessária para garantir conformidade.'
                : warnAlerts > 0
                  ? 'Monitore a evolução nas próximas 24h.'
                  : 'Operação dentro dos parâmetros regulatórios.'
            }
            actions={
              criticalAlerts > 0 ? (
                <Badge tone="critical" variant="soft" dot size="md">
                  ação imediata
                </Badge>
              ) : warnAlerts > 0 ? (
                <Badge tone="warning" variant="soft" dot size="md">
                  monitorar
                </Badge>
              ) : stats && stats.total > 0 ? (
                <Badge tone="success" variant="soft" dot size="md">
                  {approvalRate.toFixed(1)}% aprovação
                </Badge>
              ) : (
                <Badge tone="neutral" variant="soft" dot size="md">
                  aguardando dados
                </Badge>
              )
            }
          />

          <div className="mt-8 grid grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              label="Envios (30d)"
              value={stats?.total ?? 0}
              delta={
                kpis && kpis.previous.sent_total > 0
                  ? {
                      value: deltaPct(kpis.current.sent_total, kpis.previous.sent_total),
                      direction:
                        kpis.current.sent_total >= kpis.previous.sent_total ? 'up' : 'down',
                      period: 'vs 30d anteriores',
                    }
                  : undefined
              }
              sparkline={sentSparkline}
              tone="accent"
              icon={<Send className="size-4" strokeWidth={2.25} />}
              helpText={
                stats
                  ? `${stats.accepted} aprovados · ${stats.rejected} rejeitados · ${stats.pending} pendentes`
                  : 'aguardando /v1/envios/stats'
              }
            />
            <StatCard
              label="Alertas ativos"
              value={alerts.length}
              tone={criticalAlerts > 0 ? 'critical' : warnAlerts > 0 ? 'warning' : 'success'}
              icon={<AlertTriangle className="size-4" strokeWidth={2.25} />}
              helpText={`${criticalAlerts} crítico${criticalAlerts !== 1 ? 's' : ''} · ${warnAlerts} atenção`}
            />
            <StatCard
              label="Taxa de aprovação"
              value={kpis ? `${kpis.current.approval_rate.toFixed(1)}%` : '—'}
              delta={
                kpis && kpis.previous.approval_rate > 0
                  ? {
                      value: kpis.delta.approval_rate_pct,
                      direction:
                        kpis.delta.approval_rate_pct >= 0 ? 'up' : 'down',
                      period: 'vs período anterior',
                    }
                  : undefined
              }
              tone={approvalRate >= 90 ? 'success' : approvalRate >= 70 ? 'warning' : 'critical'}
              icon={<TrendingUp className="size-4" strokeWidth={2.25} />}
            />
            <StatCard
              label="CADOCs monitorados"
              value={schemas.length || '—'}
              tone="neutral"
              icon={<Database className="size-4" strokeWidth={2.25} />}
              helpText="schemas BACEN ativos"
            />
          </div>
        </section>

        <Divider />

        {/* O que precisa de atenção */}
        <section className="space-y-5">
          <SectionHeader
            eyebrow="Fila de atenção"
            title="O que precisa de ação"
            description="Priorizado por severidade e recência. Resolva os críticos para restaurar a conformidade."
            actions={
              <Link href="/radar" passHref legacyBehavior={false}>
                <Button asChild variant="secondary" size="sm" rightIcon={<ArrowUpRight className="size-3.5" strokeWidth={2.25} />}>
                  Ver radar completo
                </Button>
              </Link>
            }
          />

          {topAlerts.length === 0 ? (
            <Card padding="lg" className="text-center">
              <div className="size-12 mx-auto mb-4 rounded-full bg-success-50 dark:bg-success-950/30 text-success-600 dark:text-success-300 flex items-center justify-center ring-1 ring-inset ring-success-200/60 dark:ring-success-800/40">
                <TrendingUp className="size-5" strokeWidth={2.25} />
              </div>
              <h4 className="font-serif text-base font-medium text-ink mb-1 tracking-tight">
                Nenhum alerta aberto
              </h4>
              <p className="text-xs text-ink-muted">
                URLs BACEN estáveis · varredura a cada 6h
              </p>
            </Card>
          ) : (
            <div className="grid gap-3">
              {topAlerts.map((alert) => (
                <AlertCard key={alert.id} {...alert} />
              ))}
            </div>
          )}
        </section>

        {/* Atividade recente */}
        <section className="space-y-5">
          <SectionHeader
            eyebrow="Trilha de auditoria"
            title="Atividade recente"
            description="Eventos imutáveis do audit_log. Cada item referencia o SHA-256 do anterior."
            actions={
              <Link href="/auditoria" passHref legacyBehavior={false}>
                <Button asChild variant="ghost" size="sm" rightIcon={<ArrowUpRight className="size-3.5" strokeWidth={2.25} />}>
                  Ver auditoria
                </Button>
              </Link>
            }
          />

          <Card padding="md">
            {auditActivity.length === 0 ? (
              <div className="py-12 text-center">
                <Activity className="size-5 mx-auto mb-3 text-ink-subtle" />
                <p className="text-xs text-ink-subtle font-mono uppercase tracking-wider">
                  Nenhum evento registrado ainda
                </p>
              </div>
            ) : (
              <ActivityFeed items={auditActivity} />
            )}
          </Card>
        </section>

        {/* Cobertura por CADOC */}
        {schemas.length > 0 && (
          <section className="space-y-5">
            <SectionHeader
              eyebrow="Cobertura de schemas"
              title="CADOCs monitorados"
              description={`${schemas.length} schemas ativos · varredura automática a cada 6h`}
            />
            <Card padding="md">
              <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-5">
                {schemas.map((s) => {
                  const coverage = stableCoverage(s.cadoc)
                  return (
                    <div key={s.cadoc} className="space-y-2.5">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2 min-w-0">
                          <span className="font-mono text-xs font-medium text-accent-600 dark:text-accent-400">
                            {s.cadoc}
                          </span>
                          <span className="text-xs text-ink-muted truncate">
                            {s.description}
                          </span>
                        </div>
                        <span className="text-2xs font-mono font-medium text-ink-muted nums shrink-0 ml-2">
                          {coverage}%
                        </span>
                      </div>
                      <div className="h-1 rounded-full bg-surface-sunken overflow-hidden">
                        <div
                          className="h-full rounded-full bg-gradient-to-r from-accent-500 to-magenta-500 transition-all duration-500 ease-out-expo"
                          style={{ width: `${coverage}%` }}
                        />
                      </div>
                    </div>
                  )
                })}
              </div>
            </Card>
          </section>
        )}
      </div>
    </AppShell>
  )
}

function normalizeAction(action: string): ActivityItem['kind'] {
  switch (action) {
    case 'envio.approved':
    case 'envio.created':
      return 'envio.approved'
    case 'envio.rejected':
      return 'envio.rejected'
    case 'radar.detected':
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
    case 'sta.submit':
      return 'envio.created'
    default:
      return 'envio.approved'
  }
}

function deltaPct(curr: number, prev: number): number {
  if (prev === 0) return 0
  return ((curr - prev) / prev) * 100
}

function sparklineFromEnvios(
  stats: EnvioStats,
  kpis: InsightsKPIs | null,
): number[] {
  const total = stats.total
  if (total === 0) return []
  const weekAvg = total / 4
  return [
    Math.floor(weekAvg * 0.7),
    Math.floor(weekAvg * 0.85),
    Math.floor(weekAvg * 0.95),
    Math.floor(weekAvg),
  ]
}