/**
 * /radar — lista de alertas regulatórios BACEN.
 */

import { Radar as RadarIcon } from 'lucide-react'
import { apiFetch } from '@/lib/api-fetch'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { AlertCard } from '@/components/domain/alert-card'
import { RadarLiveRefresh } from '@/components/domain/radar-live-refresh'
import { Card, CardTitle, CardEyebrow } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { SectionHeader } from '@/components/ui/section-header'
import { Divider } from '@/components/ui/divider'
import { cn } from '@/lib/utils'

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

const severityPanel = {
  critical: {
    label: 'Críticos',
    tone: 'critical' as const,
    accentClass: 'from-critical-500 to-critical-600',
    bgClass: 'bg-critical-50/60 dark:bg-critical-950/30 border-critical-200/60 dark:border-critical-800/40',
    action: 'ação imediata',
  },
  warn: {
    label: 'Atenção',
    tone: 'warning' as const,
    accentClass: 'from-warning-500 to-warning-600',
    bgClass: 'bg-warning-50/60 dark:bg-warning-950/30 border-warning-200/60 dark:border-warning-800/40',
    action: 'monitorar',
  },
  info: {
    label: 'Info',
    tone: 'info' as const,
    accentClass: 'from-info-500 to-info-600',
    bgClass: 'bg-info-50/60 dark:bg-info-950/30 border-info-200/60 dark:border-info-800/40',
    action: 'apenas nota',
  },
}

export default async function RadarPage() {
  const data = await getData()
  if (!data) {
    return (
      <div className="p-12 text-center">
        <p className="text-ink-muted">Sessão expirada.</p>
      </div>
    )
  }

  const { session, alerts, schemas } = data

  const counts = {
    critical: alerts.filter((a) => a.severity === 'critical').length,
    warn: alerts.filter((a) => a.severity === 'warn').length,
    info: alerts.filter((a) => a.severity === 'info').length,
  }

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
        actions: <RadarLiveRefresh />,
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
      <div className="space-y-10 max-w-6xl">
        {/* Severidade summary */}
        <section>
          <SectionHeader
            eyebrow="Visão por severidade"
            title="Fila de alertas"
            description="Detecção automática a cada 6h em URLs oficiais do BACEN."
          />
          <div className="mt-6 grid grid-cols-1 md:grid-cols-3 gap-4">
            {(Object.keys(severityPanel) as Array<keyof typeof severityPanel>).map((key) => {
              const panel = severityPanel[key]
              const count = counts[key]
              return (
                <Card
                  key={key}
                  padding="md"
                  className={cn('relative overflow-hidden', panel.bgClass)}
                >
                  <div
                    className={cn(
                      'absolute top-0 left-0 right-0 h-[3px] bg-gradient-to-r',
                      panel.accentClass,
                    )}
                    aria-hidden
                  />
                  <div className="flex items-center justify-between mb-4">
                    <span className="text-xs font-medium text-ink-muted uppercase tracking-[0.14em] font-mono">
                      {panel.label}
                    </span>
                    <Badge tone={panel.tone} variant="soft" dot size="sm">
                      {panel.action}
                    </Badge>
                  </div>
                  <div className="flex items-baseline gap-2">
                    <span
                      className={cn(
                        'text-4xl font-medium nums tracking-tight font-serif',
                        panel.tone === 'critical'
                          ? 'text-critical-700 dark:text-critical-300'
                          : panel.tone === 'warning'
                            ? 'text-warning-700 dark:text-warning-300'
                            : 'text-info-700 dark:text-info-300',
                      )}
                    >
                      {count}
                    </span>
                    <span className="text-xs text-ink-muted">
                      {count === 1 ? 'alerta' : 'alertas'}
                    </span>
                  </div>
                </Card>
              )
            })}
          </div>
        </section>

        <Divider />

        {/* Lista de alertas */}
        <section>
          {alerts.length === 0 ? (
            <EmptyState
              symbol="∅"
              icon={<RadarIcon className="size-5" strokeWidth={1.75} />}
              title="Nenhum alerta ativo"
              description="URLs BACEN estão estáveis. Próxima varredura em ~6h."
              action={
                <Button variant="secondary" size="md">
                  Forçar varredura agora
                </Button>
              }
            />
          ) : (
            <div className="space-y-10">
              {Object.entries(byCadoc).map(([cadoc, cadocAlerts]) => (
                <section key={cadoc} className="space-y-3">
                  <div className="flex items-baseline gap-3">
                    <span className="font-mono text-sm font-medium text-accent-600 dark:text-accent-400">
                      CADOC {cadoc}
                    </span>
                    <span className="text-xs text-ink-subtle font-mono">
                      {cadocAlerts.length} alerta{cadocAlerts.length !== 1 ? 's' : ''}
                    </span>
                    <span className="flex-1 h-px bg-border-subtle ml-1" />
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
        </section>
      </div>
    </AppShell>
  )
}