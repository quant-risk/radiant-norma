/**
 * /radar — lista de alertas regulatórios BACEN.
 *
 * Server component. Carrega alertas não resolvidos, agrupa por CADOC,
 * mostra contador por severidade, permite resolver inline (server action
 * via API).
 */

import { Radar as RadarIcon } from 'lucide-react'
import { apiFetch } from '@/lib/api-fetch'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { AlertCard } from '@/components/domain/alert-card'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { Separator } from '@/components/ui/separator'

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

interface Schema {
  cadoc: string
  description: string
}

async function getData() {
  const session = await getServerSession()
  if (!session) return null

  const [alertsRes, schemasRes] = await Promise.allSettled([
    apiFetch<{ alerts: Alert[]; total: number } | Alert[]>(
      '/v1/radar/alerts?unresolved=true',
      {},
      session.token,
    ),
    apiFetch<{ schemas: Schema[] } | Schema[]>(
      '/v1/schemas',
      {},
      session.token,
    ),
  ])

  const alerts =
    alertsRes.status === 'fulfilled'
      ? Array.isArray(alertsRes.value)
        ? alertsRes.value
        : alertsRes.value.alerts ?? []
      : []
  const schemas =
    schemasRes.status === 'fulfilled'
      ? Array.isArray(schemasRes.value)
        ? schemasRes.value
        : schemasRes.value.schemas ?? []
      : []

  return { session, alerts, schemas }
}

export default async function RadarPage() {
  const data = await getData()
  if (!data) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  const { session, alerts, schemas } = data

  const counts = {
    critical: alerts.filter((a) => a.severity === 'critical').length,
    warn: alerts.filter((a) => a.severity === 'warn').length,
    info: alerts.filter((a) => a.severity === 'info').length,
  }

  // Agrupa por CADOC pra hierarquia visual
  const byCadoc = alerts.reduce<Record<string, Alert[]>>((acc, alert) => {
    acc[alert.cadoc_code] = acc[alert.cadoc_code] ?? []
    acc[alert.cadoc_code].push(alert)
    return acc
  }, {})

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Radar Regulatório',
        subtitle: `${alerts.length} alerta${alerts.length !== 1 ? 's' : ''} não resolvido${alerts.length !== 1 ? 's' : ''}`,
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Radar' },
        ],
      }}
      commandData={{
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
      <div className="space-y-6 max-w-6xl">
        {/* Severidade summary */}
        <div className="grid grid-cols-3 gap-4">
          <Card padding="md" className="border-critical-200 dark:border-critical-900">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-ink-muted uppercase tracking-wider">
                Críticos
              </span>
              <span className="size-2 rounded-full bg-critical-500 animate-pulse-soft" />
            </div>
            <div className="flex items-baseline gap-2">
              <span className="text-3xl font-semibold text-critical-600 dark:text-critical-400 nums">
                {counts.critical}
              </span>
              <span className="text-xs text-ink-muted">
                ação imediata
              </span>
            </div>
          </Card>

          <Card padding="md" className="border-warning-200 dark:border-warning-900">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-ink-muted uppercase tracking-wider">
                Atenção
              </span>
              <span className="size-2 rounded-full bg-warning-500" />
            </div>
            <div className="flex items-baseline gap-2">
              <span className="text-3xl font-semibold text-warning-600 dark:text-warning-400 nums">
                {counts.warn}
              </span>
              <span className="text-xs text-ink-muted">monitorar</span>
            </div>
          </Card>

          <Card padding="md" className="border-info-200 dark:border-info-900">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-ink-muted uppercase tracking-wider">
                Info
              </span>
              <span className="size-2 rounded-full bg-info-500" />
            </div>
            <div className="flex items-baseline gap-2">
              <span className="text-3xl font-semibold text-info-600 dark:text-info-400 nums">
                {counts.info}
              </span>
              <span className="text-xs text-ink-muted">apenas nota</span>
            </div>
          </Card>
        </div>

        {/* Lista de alertas */}
        {alerts.length === 0 ? (
          <EmptyState
            icon={<RadarIcon className="size-6" />}
            title="Nenhum alerta ativo"
            description="URLs BACEN estão estáveis. Próxima varredura em ~6h."
            action={
              <Button variant="outline" size="sm">
                Forçar varredura agora
              </Button>
            }
          />
        ) : (
          <div className="space-y-8">
            {Object.entries(byCadoc).map(([cadoc, cadocAlerts]) => (
              <section key={cadoc}>
                <div className="flex items-baseline gap-3 mb-3">
                  <h3 className="text-md font-semibold text-ink font-mono">
                    {cadoc}
                  </h3>
                  <span className="text-sm text-ink-muted">
                    {cadocAlerts.length} alerta
                    {cadocAlerts.length !== 1 ? 's' : ''}
                  </span>
                  <Separator className="flex-1" />
                </div>
                <div className="grid gap-3">
                  {cadocAlerts.map((alert) => (
                    <AlertCard key={alert.id} {...alert} />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </div>
    </AppShell>
  )
}