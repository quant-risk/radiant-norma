/**
 * /insights — inteligência operacional.
 *
 * Onda 2 feature. Combina:
 *   1. Heatmap de falhas por CADOC × dia (últimos 14d)
 *   2. Top regras falhando (ranking priorizado)
 *   3. Recomendações acionáveis
 *   4. Comparação temporal (mês atual vs anterior)
 *
 * Mock data por enquanto — Sprint 8c vai puxar /v1/insights/*.
 */

import {
  Sparkles,
  TrendingUp,
  TrendingDown,
  AlertTriangle,
  Lightbulb,
  Calendar,
} from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { StatCard } from '@/components/domain/stat-card'
import { InsightCard } from '@/components/domain/insight-card'
import { Heatmap } from '@/components/domain/heatmap'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'

export const dynamic = 'force-dynamic'

export default async function InsightsPage() {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  // Mock heatmap data: CADOCs × dias (últimos 14d)
  const heatmapRows = ['3040', '3050', '3060', '3070', '4020', '4030']
  const heatmapCols = Array.from({ length: 14 }, (_, i) => {
    const d = new Date(Date.now() - (13 - i) * 24 * 3600_000)
    return d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
  })
  const heatmapData = heatmapRows.flatMap((row) =>
    heatmapCols.map((col) => ({
      row,
      col,
      value: Math.floor(Math.random() * 12),
    })),
  )

  // Top regras falhando
  const topRules = [
    { code: 'F23', desc: 'Formato CNPJ inválido', count: 47, severity: 'E', trend: '+12' },
    { code: 'B12', desc: 'Campo obrigatório ausente', count: 32, severity: 'E', trend: '+8' },
    { code: 'S05', desc: 'Total divergente (cross-doc)', count: 19, severity: 'A', trend: '-3' },
    { code: 'C04', desc: 'Data inválida (AAAAMMDD)', count: 15, severity: 'E', trend: '+2' },
    { code: 'F08', desc: 'Máscara de valor monetário', count: 9, severity: 'A', trend: '-1' },
  ]

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
            Últimos 14 dias
          </Button>
        ),
      }}
    >
      <div className="space-y-8 max-w-7xl">
        {/* KPIs comparativos */}
        <section className="space-y-4">
          <h2 className="text-md font-semibold text-ink">
            Comparativo temporal
          </h2>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              label="Taxa de aprovação"
              value="98.2%"
              delta={{ value: 4.1, direction: 'up', period: 'vs mês anterior' }}
              tone="success"
              icon={<TrendingUp className="size-4" />}
            />
            <StatCard
              label="Falhas detectadas"
              value="142"
              delta={{ value: -18.3, direction: 'down', period: 'vs mês anterior' }}
              tone="success"
              icon={<TrendingDown className="size-4" />}
            />
            <StatCard
              label="Regras acionadas"
              value="60"
              delta={{ value: 0, direction: 'flat', period: 'cobertura 100%' }}
              tone="neutral"
            />
            <StatCard
              label="Tempo médio validação"
              value="2.4s"
              delta={{ value: -22, direction: 'down', period: 'vs mês anterior' }}
              tone="success"
              icon={<TrendingDown className="size-4" />}
            />
          </div>
        </section>

        <Separator />

        {/* Heatmap */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-md font-semibold text-ink">
                Mapa de calor — falhas por CADOC × dia
              </h2>
              <p className="text-xs text-ink-muted">
                Concentração de falhas nos últimos 14 dias
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
            <Heatmap
              data={heatmapData}
              rows={heatmapRows}
              cols={heatmapCols}
              max={12}
            />
          </Card>
        </section>

        {/* Top regras + Recomendações */}
        <section className="grid lg:grid-cols-2 gap-6">
          <div className="space-y-3">
            <div>
              <h2 className="text-md font-semibold text-ink">
                Top regras falhando
              </h2>
              <p className="text-xs text-ink-muted">
                Ranking por impacto · últimos 30 dias
              </p>
            </div>
            <Card padding="none">
              <div className="divide-y divide-border-subtle">
                {topRules.map((rule, i) => {
                  const sevTone =
                    rule.severity === 'E' ? 'critical' : 'warning'
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
                        <p className="text-sm text-ink-muted truncate">
                          {rule.desc}
                        </p>
                      </div>
                      <div className="text-right shrink-0">
                        <div className="text-lg font-semibold text-ink nums">
                          {rule.count}
                        </div>
                        <div
                          className={`text-2xs font-medium ${
                            rule.trend.startsWith('+')
                              ? 'text-critical-600 dark:text-critical-400'
                              : 'text-success-600 dark:text-success-400'
                          }`}
                        >
                          {rule.trend}
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            </Card>
          </div>

          <div className="space-y-3">
            <div>
              <h2 className="text-md font-semibold text-ink">Recomendações</h2>
              <p className="text-xs text-ink-muted">
                Ações priorizadas por impacto estimado
              </p>
            </div>

            <InsightCard
              id="rec1"
              kind="recommendation"
              headline="Habilitar regra F23 reduziria 67% das rejeições"
              narrative="F23 (formato CNPJ) é responsável por 47 das 142 falhas nos últimos 30 dias. Captura automática eliminaria quase 1/3 do retrabalho manual."
              confidence={87}
              impact="high"
              cta={{ label: 'Revisar F23' }}
            />

            <InsightCard
              id="rec2"
              kind="opportunity"
              headline="Regras B12 + F23 juntas cobrem 55% das falhas"
              narrative="Atacar essas 2 regras em paralelo reduziria a fila de envios pendentes em ~3.5h/dia, segundo análise dos últimos 30 envios."
              confidence={78}
              impact="high"
              cta={{ label: 'Ver playbook' }}
            />

            <InsightCard
              id="rec3"
              kind="warning"
              headline="CADOC 3050 acumula 22 falhas em 3 dias"
              narrative="Anomalia detectada — base 05/2026 do CADOC 3050 está com padrão de erro incomum. Investigar se há mudança no schema não detectada pelo radar."
              confidence={91}
              impact="medium"
              cta={{ label: 'Investigar 3050' }}
            />
          </div>
        </section>
      </div>
    </AppShell>
  )
}