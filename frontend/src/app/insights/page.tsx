/**
 * /insights — inteligência operacional.
 *
 * Validação 29 (C3, C4 fix): ZERO dados fake. Tudo é empty state
 * explícito com CTA claro até o backend expor /v1/insights/* (Sprint 8c).
 *
 * Estrutura planejada (preparada para dados reais):
 *   1. Comparativo temporal (4 KPIs com delta vs período anterior)
 *   2. Heatmap 14d (CADOC × dia)
 *   3. Top regras falhando
 *   4. Recomendações acionáveis
 */

import { Calendar, Sparkles } from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

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
            Período
          </Button>
        ),
      }}
    >
      <div className="space-y-8 max-w-7xl">
        {/* Banner de status */}
        <Card padding="lg" className="text-center bg-accent-50/50 dark:bg-accent-950/20 border-accent-200 dark:border-accent-900">
          <div className="size-12 mx-auto mb-4 rounded-full bg-accent-100 dark:bg-accent-950 text-accent-600 dark:text-accent-400 flex items-center justify-center">
            <Sparkles className="size-6" />
          </div>
          <CardTitle className="mb-2">
            Insights ainda não disponíveis
          </CardTitle>
          <CardDescription className="max-w-md mx-auto mb-4">
            Quando o backend expor <code className="text-2xs font-mono bg-surface-raised px-1 py-0.5 rounded">/v1/insights</code> (Sprint 8c),
            esta página vai mostrar comparativo temporal, heatmap de falhas,
            ranking de regras falhando e recomendações priorizadas — tudo
            derivado dos seus envios reais.
          </CardDescription>
          <Badge tone="info" variant="soft">
            Roadmap · Sprint 8c
          </Badge>
        </Card>

        {/* Seções placeholder — UI pronta, dados aguardando */}
        <section className="space-y-4">
          <h2 className="text-md font-semibold text-ink">
            Comparativo temporal
          </h2>
          <Card padding="lg">
            <p className="text-sm text-ink-muted text-center py-8">
              Aguardando endpoint{' '}
              <code className="text-2xs font-mono bg-surface-sunken px-1 py-0.5 rounded">
                GET /v1/insights/kpis
              </code>{' '}
              — virá com taxa de aprovação, falhas detectadas, regras acionadas e tempo médio de validação.
            </p>
          </Card>
        </section>

        <section className="space-y-4">
          <h2 className="text-md font-semibold text-ink">
            Mapa de calor — falhas por CADOC × dia
          </h2>
          <Card padding="lg">
            <p className="text-sm text-ink-muted text-center py-8">
              Aguardando endpoint{' '}
              <code className="text-2xs font-mono bg-surface-sunken px-1 py-0.5 rounded">
                GET /v1/insights/heatmap?days=14
              </code>{' '}
              — vai mostrar concentração de falhas nos últimos 14 dias.
            </p>
          </Card>
        </section>

        <section className="grid lg:grid-cols-2 gap-6">
          <section className="space-y-4">
            <h2 className="text-md font-semibold text-ink">
              Top regras falhando
            </h2>
            <Card padding="lg">
              <p className="text-sm text-ink-muted text-center py-8">
                Aguardando{' '}
                <code className="text-2xs font-mono bg-surface-sunken px-1 py-0.5 rounded">
                  GET /v1/insights/rules/top-failing
                </code>
                .
              </p>
            </Card>
          </section>

          <section className="space-y-4">
            <h2 className="text-md font-semibold text-ink">
              Recomendações
            </h2>
            <Card padding="lg">
              <p className="text-sm text-ink-muted text-center py-8">
                Aguardando{' '}
                <code className="text-2xs font-mono bg-surface-sunken px-1 py-0.5 rounded">
                  GET /v1/insights/recommendations
                </code>
                .
              </p>
            </Card>
          </section>
        </section>
      </div>
    </AppShell>
  )
}