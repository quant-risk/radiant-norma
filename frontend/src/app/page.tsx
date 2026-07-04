/**
 * / — Dashboard executivo.
 *
 * Server component. Lê dados via Promise.allSettled (resiliência a
 * endpoints 404). Estrutura:
 *   1. Hero strip: 1 número primário + 4 KPIs com sparkline
 *   2. "O que precisa de atenção" — top 3 alertas priorizados
 *   3. Activity feed lateral
 *   4. Insights pre-computados (mock por enquanto; virá do backend Sprint 8c)
 */

import Link from 'next/link'
import {
  Send,
  AlertTriangle,
  BookCheck,
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
import { ActivityFeed, type ActivityItem } from '@/components/domain/activity-feed'
import { InsightCard } from '@/components/domain/insight-card'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

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

async function getDashboardData() {
  const session = await getServerSession()
  if (!session) return null

  const [alertsRes, rulesRes, schemasRes] = await Promise.allSettled([
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
  ])

  const alerts: Alert[] =
    alertsRes.status === 'fulfilled'
      ? Array.isArray(alertsRes.value)
        ? alertsRes.value
        : alertsRes.value.alerts ?? []
      : []
  const rules =
    rulesRes.status === 'fulfilled'
      ? Array.isArray(rulesRes.value)
        ? rulesRes.value
        : rulesRes.value.rules ?? []
      : []
  const schemas =
    schemasRes.status === 'fulfilled'
      ? Array.isArray(schemasRes.value)
        ? schemasRes.value
        : schemasRes.value.schemas ?? []
      : []

  return { session, alerts, rules, schemas }
}

export default async function DashboardPage() {
  const data = await getDashboardData()
  if (!data) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Card padding="lg" className="max-w-md text-center">
          <h2 className="text-lg font-semibold mb-2">
            Sessão expirada
          </h2>
          <p className="text-sm text-ink-muted mb-4">
            Faça login para acessar o console.
          </p>
          <Link href="/login">
            <Button variant="primary" fullWidth>
              Ir para login
            </Button>
          </Link>
        </Card>
      </div>
    )
  }

  const { session, alerts, rules, schemas } = data

  // Mock trend data (Sprint 8c vai puxar do backend)
  const trendEnvios = [12, 18, 14, 22, 19, 26, 24]
  const trendAlertas = [3, 5, 4, 6, 4, 3, 2]
  const trendAprovacao = [94, 92, 95, 93, 96, 97, 98]

  // Top 3 alertas priorizados: critical > warn > info; + recentes primeiro
  const topAlerts = [...alerts]
    .sort((a, b) => {
      const sev: Record<Alert['severity'], number> = {
        critical: 0,
        warn: 1,
        info: 2,
      }
      if (sev[a.severity] !== sev[b.severity])
        return sev[a.severity] - sev[b.severity]
      return new Date(b.detected_at).getTime() - new Date(a.detected_at).getTime()
    })
    .slice(0, 3)

  // Mock activity feed (Sprint 8c: backend /v1/audit/recent)
  const mockActivity: ActivityItem[] = [
    {
      id: 1,
      kind: 'envio.approved',
      timestamp: new Date(Date.now() - 12 * 60_000).toISOString(),
      actor: session.if_id,
      description: 'CADOC 3040 base 05/2026 aprovado',
      payload: { cadoc: '3040', periodo: '05/2026', rules_passed: 58 },
    },
    {
      id: 2,
      kind: 'radar.detected',
      timestamp: new Date(Date.now() - 47 * 60_000).toISOString(),
      description: 'URL do layout 3040 alterada no portal BACEN',
      payload: { cadoc: '3040', severity: 'warn' },
    },
    {
      id: 3,
      kind: 'rule.enabled',
      timestamp: new Date(Date.now() - 2 * 3600_000).toISOString(),
      actor: session.if_id,
      description: 'Regra B12 habilitada — campos obrigatórios',
      payload: { rule: 'B12' },
    },
    {
      id: 4,
      kind: 'schema.synced',
      timestamp: new Date(Date.now() - 5 * 3600_000).toISOString(),
      description: 'Schema 3040 v8.2.1 sincronizado',
      payload: { schema: '3040', version: '8.2.1' },
    },
    {
      id: 5,
      kind: 'auth.login',
      timestamp: new Date(Date.now() - 8 * 3600_000).toISOString(),
      actor: session.if_id,
    },
  ]

  const rulesCount = rules.length || 60
  const schemasCount = schemas.length || 10
  const criticalAlerts = alerts.filter((a) => a.severity === 'critical').length

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Dashboard',
        subtitle: 'Visão geral da operação regulatória',
        breadcrumbs: [{ label: 'Radiant Norma' }, { label: 'Dashboard' }],
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
      <div className="space-y-8 max-w-7xl">
        {/* Hero strip — 1 número primário + 4 KPIs */}
        <section className="space-y-4">
          <div className="flex items-end justify-between gap-4">
            <div>
              <p className="text-2xs uppercase tracking-wider text-ink-subtle font-semibold mb-1">
                Status operacional
              </p>
              <h2 className="text-2xl font-semibold text-ink tracking-tight">
                Tudo em ordem, com 1 ponto de atenção
              </h2>
            </div>
            <Badge tone="success" variant="soft" dot className="text-sm py-1">
              <span className="font-medium">
                {trendAprovacao[trendAprovacao.length - 1]}%
              </span>
              <span className="text-ink-muted ml-1">aprovação</span>
            </Badge>
          </div>

          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              label="Envios (7d)"
              value={trendEnvios.reduce((a, b) => a + b, 0)}
              delta={{
                value: 12.4,
                direction: 'up',
                period: 'vs semana anterior',
              }}
              sparkline={trendEnvios}
              tone="accent"
              icon={<Send className="size-4" />}
            />
            <StatCard
              label="Alertas ativos"
              value={alerts.length}
              delta={{
                value: criticalAlerts > 0 ? -100 * (1 - trendAlertas[6] / Math.max(trendAlertas[5], 1)) : 0,
                direction: trendAlertas[6] < trendAlertas[5] ? 'down' : 'up',
                period: `${trendAlertas[6]} novos hoje`,
              }}
              sparkline={trendAlertas}
              tone={criticalAlerts > 0 ? 'critical' : 'warning'}
              icon={<AlertTriangle className="size-4" />}
            />
            <StatCard
              label="Regras ativas"
              value={rulesCount}
              delta={{ value: 0, direction: 'flat', period: '60 cadastradas' }}
              tone="neutral"
              icon={<BookCheck className="size-4" />}
              helpText="5 raw + 55 tipadas (Sprint 7b)"
            />
            <StatCard
              label="CADOCs monitorados"
              value={schemasCount}
              delta={{ value: 0, direction: 'flat', period: '10 schemas BACEN' }}
              tone="neutral"
              icon={<Database className="size-4" />}
            />
          </div>
        </section>

        {/* O que precisa de atenção */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-md font-semibold text-ink">
                O que precisa de atenção
              </h3>
              <p className="text-xs text-ink-muted">
                Priorizado por severidade e recência
              </p>
            </div>
            <Link href="/radar">
              <Button
                variant="ghost"
                size="sm"
                rightIcon={<ArrowUpRight className="size-3.5" />}
              >
                Ver radar completo
              </Button>
            </Link>
          </div>

          {topAlerts.length === 0 ? (
            <Card padding="lg" className="text-center">
              <div className="size-10 mx-auto mb-3 rounded-full bg-success-50 dark:bg-success-950 text-success-600 dark:text-success-400 flex items-center justify-center">
                <TrendingUp className="size-5" />
              </div>
              <h4 className="text-sm font-semibold text-ink mb-1">
                Nenhum alerta aberto
              </h4>
              <p className="text-xs text-ink-muted">
                URLs BACEN estáveis · última varredura há 12 min
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

        {/* Insights + Activity feed */}
        <section className="grid lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-3">
            <div>
              <h3 className="text-md font-semibold text-ink">Insights</h3>
              <p className="text-xs text-ink-muted">
                Anomalias detectadas nos seus envios
              </p>
            </div>

            <InsightCard
              id="i1"
              kind="recommendation"
              headline="Habilitar regra F23 reduziria 67% das rejeições"
              narrative="Nos últimos 30 envios da base 3040, regra F23 (formato CNPJ) foi responsável por 8 de 12 rejeições. Habilitar captura esse cenário automaticamente."
              confidence={87}
              impact="high"
              cta={{ label: 'Revisar regra F23', href: '/regras?focus=F23' }}
            />

            <InsightCard
              id="i2"
              kind="trend-down"
              headline="Taxa de aprovação subiu 4pp nas últimas 2 semanas"
              narrative="De 94% para 98%. Padrão consistente — sem outliers estatisticamente significativos."
              confidence={92}
              impact="low"
            />

            <InsightCard
              id="i3"
              kind="warning"
              headline="Layout 3040 v8.2.1 publicado — ação recomendada"
              narrative="BACEN publicou atualização de layout hoje. Mudança afeta 3 campos do bloco 'Identificação'. Janela de adaptação: 15 dias."
              confidence={100}
              impact="medium"
              cta={{
                label: 'Ver diff do schema',
                href: '/radar',
              }}
            />
          </div>

          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Activity className="size-4 text-ink-muted" />
              <h3 className="text-md font-semibold text-ink">Atividade recente</h3>
            </div>
            <Card padding="md">
              <ActivityFeed items={mockActivity} />
            </Card>
          </div>
        </section>

        {/* Cobertura por CADOC */}
        <section className="space-y-4">
          <div>
            <h3 className="text-md font-semibold text-ink">
              Cobertura por CADOC
            </h3>
            <p className="text-xs text-ink-muted">
              {schemasCount} schemas monitorados · varredura a cada 6h
            </p>
          </div>

          <Card padding="md">
            <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {(schemas.length > 0 ? schemas : mockSchemas).map((s) => {
                const coverage = Math.floor(60 + Math.random() * 40)
                return (
                  <div key={s.cadoc} className="space-y-2">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-xs font-semibold text-accent-600 dark:text-accent-400">
                          {s.cadoc}
                        </span>
                        <span className="text-xs text-ink-muted truncate">
                          {s.description}
                        </span>
                      </div>
                      <span className="text-2xs font-medium text-ink-muted nums">
                        {coverage}%
                      </span>
                    </div>
                    <div className="h-1.5 rounded-full bg-surface-sunken overflow-hidden">
                      <div
                        className="h-full rounded-full bg-gradient-to-r from-accent-500 to-accent-600 transition-all duration-500"
                        style={{ width: `${coverage}%` }}
                      />
                    </div>
                  </div>
                )
              })}
            </div>
          </Card>
        </section>
      </div>
    </AppShell>
  )
}

const mockSchemas: Schema[] = [
  { cadoc: '3040', description: 'Risco de Crédito', versions: 1, latest_version: '8.2.1' },
  { cadoc: '3050', description: 'Risco de Mercado', versions: 1, latest_version: '6.1.0' },
  { cadoc: '3060', description: 'Risco de Liquidez', versions: 1, latest_version: '4.0.2' },
  { cadoc: '3070', description: 'Capital', versions: 1, latest_version: '5.3.0' },
  { cadoc: '4020', description: 'Operações de Crédito', versions: 1, latest_version: '3.2.1' },
  { cadoc: '4030', description: 'Operações Ativas', versions: 1, latest_version: '2.8.0' },
]