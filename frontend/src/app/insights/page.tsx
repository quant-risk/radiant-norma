/**
 * /insights — inteligência operacional.
 *
 * Sprint 8c: dados reais de /v1/insights/* (kpis, heatmap, top-failing,
 * recommendations). Heurística no backend gera recommendations baseadas
 * nos dados reais do IF logado.
 */

import { Calendar, Sparkles, AlertTriangle, TrendingUp } from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { apiFetch } from '@/lib/api-fetch'
import { AppShell } from '@/components/layout/app-shell'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { StatCard } from '@/components/domain/stat-card'
import { Heatmap } from '@/components/domain/heatmap'
import { InsightCard } from '@/components/domain/insight-card'

export const dynamic = 'force-dynamic'

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

interface HeatmapCell {
  row: string
  col: string
  value: number
}

interface HeatmapData {
  data: HeatmapCell[]
  rows: string[]
  cols: string[]
  days: number
}

interface TopFailingRule {
  code: string
  severity: string
  count: number
  delta_pct: number
  trend_direction: 'up' | 'down' | 'flat'
}

interface Recommendation {
  id: string
  kind: 'recommendation' | 'warning' | 'opportunity'
  headline: string
  narrative: string
  impact: 'low' | 'medium' | 'high'
  confidence: number
  cta: { label: string; href: string }
}

async function getInsightsData() {
  const session = await getServerSession()
  if (!session) return null

  const [kpisRes, heatmapRes, topFailingRes, recsRes] = await Promise.allSettled([
    apiFetch<InsightsKPIs>('/v1/insights/kpis', {}, session.token),
    apiFetch<HeatmapData>(
      '/v1/insights/heatmap?days=14',
      {},
      session.token,
    ),
    apiFetch<{ rules: TopFailingRule[] }>(
      '/v1/insights/rules/top-failing?limit=10',
      {},
      session.token,
    ),
    apiFetch<{ recommendations: Recommendation[] }>(
      '/v1/insights/recommendations',
      {},
      session.token,
    ),
  ])

  return {
    session,
    kpis: kpisRes.status === 'fulfilled' ? kpisRes.value : null,
    heatmap: heatmapRes.status === 'fulfilled' ? heatmapRes.value : null,
    topFailing:
      topFailingRes.status === 'fulfilled'
        ? topFailingRes.value.rules ?? []
        : [],
    recommendations:
      recsRes.status === 'fulfilled'
        ? recsRes.value.recommendations ?? []
        : [],
  }
}

export default async function InsightsPage() {
  const data = await getInsightsData()
  if (!data) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  const { session, kpis, heatmap, topFailing, recommendations } = data

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Insights',
        subtitle: 'Inteligência operacional · anomalia, tendência, recomendação',
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Insights' },
        ],
        actions: (
          <Button variant="outline" size="sm" leftIcon={<Calendar className="size-3.5" />}>
            Últimos 30 dias
          </Button>
        ),
      }}
    >
      <div className="space-y-8 max-w-7xl">
        {/* KPIs comparativos */}
        <section className="space-y-4">
          <h2 className="text-md font-semibold text-ink">Comparativo temporal</h2>
          {kpis ? (
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <StatCard
                label="Taxa de aprovação"
                value={`${kpis.current.approval_rate.toFixed(1)}%`}
                delta={
                  kpis.previous.approval_rate > 0
                    ? {
                        value: kpis.delta.approval_rate_pct,
                        direction: kpis.delta.approval_rate_pct >= 0 ? 'up' : 'down',
                        period: 'vs 30d anteriores',
                      }
                    : undefined
                }
                tone={
                  kpis.current.approval_rate >= 90
                    ? 'success'
                    : kpis.current.approval_rate >= 70
                      ? 'warning'
                      : 'critical'
                }
                icon={<TrendingUp className="size-4" />}
                helpText={`${kpis.current.accepted}/${kpis.current.sent_total} envios`}
              />
              <StatCard
                label="Falhas detectadas"
                value={kpis.current.failures_total}
                delta={
                  kpis.previous.sent_total > 0
                    ? {
                        value: kpis.delta.failures_total_pct,
                        direction: kpis.delta.failures_total_pct <= 0 ? 'down' : 'up',
                        period: 'vs 30d anteriores',
                      }
                    : undefined
                }
                tone="warning"
                icon={<AlertTriangle className="size-4" />}
              />
              <StatCard
                label="Regras acionadas"
                value="—"
                tone="neutral"
                helpText="Sprint 8d: virá do /v1/insights/rules"
              />
              <StatCard
                label="Tempo médio validação"
                value={`${(kpis.current.avg_duration_ms / 1000).toFixed(1)}s`}
                delta={
                  kpis.previous.avg_duration_ms > 0
                    ? {
                        value: kpis.delta.avg_duration_ms_pct,
                        direction: kpis.delta.avg_duration_ms_pct <= 0 ? 'down' : 'up',
                        period: 'vs 30d anteriores',
                      }
                    : undefined
                }
                tone="neutral"
              />
            </div>
          ) : (
            <Card padding="lg">
              <p className="text-sm text-ink-muted text-center py-6">
                Sem dados de KPIs disponíveis
              </p>
            </Card>
          )}
        </section>

        {/* Heatmap */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-md font-semibold text-ink">
                Mapa de calor — falhas por CADOC × dia
              </h2>
              <p className="text-xs text-ink-muted">
                Concentração de falhas nos últimos {heatmap?.days ?? 14} dias
              </p>
            </div>
            <div className="flex items-center gap-3 text-2xs text-ink-muted">
              <span>Menos</span>
              <div className="flex gap-0.5">
                {[100, 300, 500, 700, 900].map((g) => (
                  <div
                    key={g}
                    className="size-3 rounded-sm"
                    style={{
                      background:
                        g === 100
                          ? 'rgb(220 252 231)'
                          : g === 300
                            ? 'rgb(187 247 208)'
                            : g === 500
                              ? 'rgb(134 239 172)'
                              : g === 700
                                ? 'rgb(74 222 128)'
                                : 'rgb(22 163 74)',
                    }}
                  />
                ))}
              </div>
              <span>Mais</span>
            </div>
          </div>

          <Card padding="md">
            {heatmap && heatmap.data.length > 0 ? (
              <Heatmap
                data={heatmap.data}
                rows={heatmap.rows}
                cols={heatmap.cols}
                max={Math.max(...heatmap.data.map((c) => c.value), 1)}
              />
            ) : (
              <p className="text-sm text-ink-muted text-center py-8">
                Sem falhas detectadas nos últimos 14 dias · tudo limpo
              </p>
            )}
          </Card>
        </section>

        {/* Top regras + Recomendações */}
        <section className="grid lg:grid-cols-2 gap-6">
          <div className="space-y-3">
            <div>
              <h2 className="text-md font-semibold text-ink">Top regras falhando</h2>
              <p className="text-xs text-ink-muted">
                Ranking por impacto · últimos 30 dias
              </p>
            </div>
            <Card padding="none">
              {topFailing.length === 0 ? (
                <div className="py-12 text-center text-sm text-ink-muted">
                  Nenhuma falha detectada nos últimos 30 dias
                </div>
              ) : (
                <div className="divide-y divide-border-subtle">
                  {topFailing.map((rule, i) => {
                    const sevTone =
                      rule.severity === 'E'
                        ? 'critical'
                        : rule.severity === 'A'
                          ? 'warning'
                          : 'info'
                    return (
                      <div
                        key={rule.code}
                        className="px-5 py-4 flex items-center gap-4 hover:bg-surface-sunken/50 transition-colors"
                      >
                        <span className="size-7 rounded-full bg-surface-sunken text-ink-muted flex items-center justify-center text-2xs font-semibold shrink-0">
                          {i + 1}
                        </span>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-0.5">
                            <span className="font-mono text-xs font-semibold text-accent-600 dark:text-accent-400">
                              {rule.code}
                            </span>
                            <Badge tone={sevTone} variant="soft">
                              {rule.severity}
                            </Badge>
                          </div>
                          <p className="text-xs text-ink-subtle font-mono">
                            {rule.severity === 'E'
                              ? 'Erro — bloqueia envio'
                              : rule.severity === 'A'
                                ? 'Alerta — revisar antes'
                                : 'Info — informativa'}
                          </p>
                        </div>
                        <div className="text-right shrink-0">
                          <div className="text-lg font-semibold text-ink nums">
                            {rule.count}
                          </div>
                          {rule.delta_pct !== 0 && (
                            <div
                              className={`text-2xs font-medium ${
                                rule.delta_pct > 0
                                  ? 'text-critical-600 dark:text-critical-400'
                                  : 'text-success-600 dark:text-success-400'
                              }`}
                            >
                              {rule.delta_pct > 0 ? '+' : ''}
                              {rule.delta_pct}%
                            </div>
                          )}
                        </div>
                      </div>
                    )
                  })}
                </div>
              )}
            </Card>
          </div>

          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Sparkles className="size-4 text-accent-600 dark:text-accent-400" />
              <h2 className="text-md font-semibold text-ink">Recomendações</h2>
            </div>
            {recommendations.length === 0 ? (
              <Card padding="lg" className="text-center">
                <div className="size-10 mx-auto mb-3 rounded-full bg-success-50 dark:bg-success-950 text-success-600 dark:text-success-400 flex items-center justify-center">
                  <Sparkles className="size-5" />
                </div>
                <h4 className="text-sm font-semibold text-ink mb-1">
                  Tudo otimizado
                </h4>
                <p className="text-xs text-ink-muted">
                  Nenhuma recomendação prioritária no momento.
                </p>
              </Card>
            ) : (
              <div className="space-y-3">
                {recommendations.map((r) => (
                  <InsightCard
                    key={r.id || r.headline}
                    id={r.id || r.headline}
                    kind={r.kind}
                    headline={r.headline}
                    narrative={r.narrative}
                    impact={r.impact}
                    confidence={r.confidence}
                    cta={{ label: r.cta.label, href: r.cta.href }}
                  />
                ))}
              </div>
            )}
          </div>
        </section>
      </div>
    </AppShell>
  )
}