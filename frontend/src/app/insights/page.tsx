/**
 * /insights — inteligência operacional.
 */

import { Calendar, Sparkles, AlertTriangle, TrendingUp } from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { apiFetch } from '@/lib/api-fetch'
import { AppShell } from '@/components/layout/app-shell'
import { Card, CardTitle, CardEyebrow, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { StatCard } from '@/components/domain/stat-card'
import { Heatmap } from '@/components/domain/heatmap'
import { InsightCard } from '@/components/domain/insight-card'
import { SectionHeader } from '@/components/ui/section-header'
import { Divider } from '@/components/ui/divider'

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
        <p className="text-ink-muted">Sessão expirada.</p>
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
          <Button variant="secondary" size="sm" leftIcon={<Calendar className="size-3.5" strokeWidth={2.25} />}>
            Últimos 30 dias
          </Button>
        ),
      }}
    >
      <div className="space-y-10 max-w-7xl">
        {/* KPIs comparativos */}
        <section>
          <SectionHeader
            eyebrow="Comparativo temporal"
            title="Indicadores-chave"
            description="Análise comparativa dos últimos 30 dias vs. período anterior."
          />
          {kpis ? (
            <div className="mt-6 grid grid-cols-2 lg:grid-cols-4 gap-4">
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
                icon={<TrendingUp className="size-4" strokeWidth={2.25} />}
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
                icon={<AlertTriangle className="size-4" strokeWidth={2.25} />}
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
            <Card padding="lg" className="mt-6 text-center">
              <p className="text-sm text-ink-muted">
                Sem dados de KPIs disponíveis
              </p>
            </Card>
          )}
        </section>

        <Divider />

        {/* Heatmap */}
        <section className="space-y-5">
          <SectionHeader
            eyebrow="Mapa de calor"
            title="Falhas por CADOC × dia"
            description={`Concentração de falhas nos últimos ${heatmap?.days ?? 14} dias`}
            actions={
              <div className="flex items-center gap-2.5 text-2xs text-ink-subtle font-mono">
                <span>Menos</span>
                <div className="flex gap-0.5">
                  {['#ede9fe', '#c4b5fd', '#a78bfa', '#7c3aed', '#5b21b6'].map((c) => (
                    <div
                      key={c}
                      className="size-3.5 rounded-sm"
                      style={{ backgroundColor: c }}
                    />
                  ))}
                </div>
                <span>Mais</span>
              </div>
            }
          />

          <Card padding="md">
            {heatmap && heatmap.data.length > 0 ? (
              <Heatmap
                data={heatmap.data}
                rows={heatmap.rows}
                cols={heatmap.cols}
                max={Math.max(...heatmap.data.map((c) => c.value), 1)}
              />
            ) : (
              <div className="py-12 text-center">
                <p className="font-serif text-base text-ink mb-1">
                  Tudo limpo
                </p>
                <p className="text-xs text-ink-muted">
                  Sem falhas detectadas nos últimos 14 dias
                </p>
              </div>
            )}
          </Card>
        </section>

        {/* Top regras + Recomendações */}
        <section className="grid lg:grid-cols-2 gap-6">
          <div className="space-y-5">
            <SectionHeader
              eyebrow="Ranking"
              title="Top regras falhando"
              description="Por impacto · últimos 30 dias"
            />
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
                        className="px-6 py-4 flex items-center gap-4 hover:bg-surface-sunken/40 transition-colors"
                      >
                        <span className="size-8 rounded-md bg-surface-sunken text-ink-muted flex items-center justify-center text-xs font-mono font-medium shrink-0">
                          {String(i + 1).padStart(2, '0')}
                        </span>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-0.5">
                            <span className="font-mono text-xs font-medium text-accent-600 dark:text-accent-400">
                              {rule.code}
                            </span>
                            <Badge tone={sevTone} variant="soft" size="sm">
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
                          <div className="text-lg font-medium text-ink nums font-serif tracking-tight">
                            {rule.count}
                          </div>
                          {rule.delta_pct !== 0 && (
                            <div
                              className={`text-2xs font-mono font-medium ${
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

          <div className="space-y-5">
            <SectionHeader
              eyebrow={
                <span className="flex items-center gap-1.5">
                  <Sparkles className="size-3" strokeWidth={2.25} />
                  Motor de IA
                </span>
              }
              title="Recomendações"
              description="Geradas pela heurística do backend sobre seus dados reais."
            />
            {recommendations.length === 0 ? (
              <Card padding="lg" className="text-center">
                <div className="size-12 mx-auto mb-4 rounded-full bg-success-50 dark:bg-success-950/30 text-success-600 dark:text-success-300 flex items-center justify-center ring-1 ring-inset ring-success-200/60 dark:ring-success-800/40">
                  <Sparkles className="size-5" strokeWidth={2.25} />
                </div>
                <h4 className="font-serif text-base font-medium text-ink mb-1 tracking-tight">
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