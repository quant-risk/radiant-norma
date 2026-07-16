/**
 * /status — Status page pública (SLA requirement).
 *
 * Página pública que mostra o estado operacional do Radiant Norma.
 * Não requer autenticação. Atualiza automaticamente a cada 30 segundos.
 * URL: status.radiant.digital
 */

import Link from 'next/link'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

interface Component {
  name: string
  status: string
  latency?: string
  since?: string
}

interface Incident {
  id: string
  title: string
  status: string
  severity: string
  started_at: string
  updated_at?: string
  resolved_at?: string
  description?: string
}

interface StatusResponse {
  status: 'operational' | 'degraded' | 'outage'
  uptime_this_month_pct: number
  components: Component[]
  incidents: Incident[]
  checked_at: string
}

async function getStatus(): Promise<StatusResponse | null> {
  try {
    const res = await fetch(`${API_URL}/status`, {
      next: { revalidate: 0 }, // always fetch fresh
    })
    if (!res.ok) return null
    return res.json()
  } catch {
    return null
  }
}

function StatusDot({ status }: { status: string }) {
  const colors: Record<string, string> = {
    operational: 'bg-success-500',
    degraded: 'bg-warning-500',
    outage: 'bg-critical-500',
  }
  const pulse: Record<string, string> = {
    operational: '',
    degraded: 'animate-pulse-soft',
    outage: 'animate-pulse-ring',
  }
  return (
    <span
      className={`inline-block w-2.5 h-2.5 rounded-full ${colors[status] ?? 'bg-ink-subtle'} ${pulse[status] ?? ''}`}
    />
  )
}

function OverallStatusBanner({ status }: { status: string }) {
  if (status === 'operational') {
    return (
      <div className="flex items-center gap-3 px-5 py-3 rounded-xl bg-success-900/20 border border-success-700/30">
        <span className="inline-block w-3 h-3 rounded-full bg-success-500 animate-pulse-soft" />
        <span className="font-medium text-success-300 text-sm">Todos os sistemas operacionais</span>
      </div>
    )
  }
  if (status === 'degraded') {
    return (
      <div className="flex items-center gap-3 px-5 py-3 rounded-xl bg-warning-900/20 border border-warning-700/30">
        <span className="inline-block w-3 h-3 rounded-full bg-warning-500 animate-pulse-soft" />
        <span className="font-medium text-warning-300 text-sm">Sistema degradado — alguns componentes com performance reduzida</span>
      </div>
    )
  }
  return (
    <div className="flex items-center gap-3 px-5 py-3 rounded-xl bg-critical-900/20 border border-critical-700/30">
      <span className="inline-block w-3 h-3 rounded-full bg-critical-500 animate-pulse-ring" />
      <span className="font-medium text-critical-300 text-sm">Incidente ativo — resposta em andamento</span>
    </div>
  )
}

function ComponentRow({ component }: { component: Component }) {
  const statusLabel: Record<string, string> = {
    operational: 'Operacional',
    degraded: 'Degradado',
    outage: 'Indisponível',
  }
  return (
    <div className="flex items-center justify-between py-3 px-4 rounded-lg hover:bg-surface-raised transition-colors">
      <div className="flex items-center gap-3">
        <StatusDot status={component.status} />
        <span className="text-sm font-medium text-ink">{component.name}</span>
      </div>
      <div className="flex items-center gap-4">
        {component.latency && (
          <span className="text-xs font-mono text-ink-muted">{component.latency}</span>
        )}
        <span className={`text-xs font-medium ${
          component.status === 'operational' ? 'text-success-600 dark:text-success-400' :
          component.status === 'degraded' ? 'text-warning-600 dark:text-warning-400' :
          'text-critical-600 dark:text-critical-400'
        }`}>
          {statusLabel[component.status] ?? component.status}
        </span>
      </div>
    </div>
  )
}

function IncidentCard({ incident }: { incident: Incident }) {
  const severityColor: Record<string, string> = {
    critical: 'text-critical-500',
    high: 'text-critical-400',
    medium: 'text-warning-500',
    low: 'text-info-500',
  }
  const statusLabel: Record<string, string> = {
    investigating: 'Investigando',
    identified: 'Identificado',
    monitoring: 'Monitorando',
    resolved: 'Resolvido',
  }
  return (
    <div className="p-4 rounded-xl border border-border bg-surface-raised">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className={`text-xs font-mono font-medium uppercase tracking-wider ${severityColor[incident.severity] ?? 'text-ink-muted'}`}>
              {incident.severity}
            </span>
            <span className="text-xs text-ink-subtle">·</span>
            <span className="text-xs text-ink-muted">
              {new Date(incident.started_at).toLocaleString('pt-BR', {
                timeZone: 'America/Sao_Paulo',
                hour: '2-digit',
                minute: '2-digit',
                day: '2-digit',
                month: 'short',
              })} BRT
            </span>
          </div>
          <h4 className="text-sm font-medium text-ink">{incident.title}</h4>
          {incident.description && (
            <p className="mt-1 text-xs text-ink-muted">{incident.description}</p>
          )}
        </div>
        <span className="shrink-0 text-xs font-medium text-ink-muted px-2 py-0.5 rounded-md bg-surface text-ink-subtle">
          {statusLabel[incident.status] ?? incident.status}
        </span>
      </div>
    </div>
  )
}

export const dynamic = 'force-dynamic'

export default async function StatusPage() {
  const data = await getStatus()
  const now = new Date()

  return (
    <div className="min-h-screen bg-surface">
      {/* Header */}
      <header className="border-b border-border">
        <div className="max-w-2xl mx-auto px-6 py-5 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-7 h-7 rounded-lg bg-accent-600 flex items-center justify-center">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M7 1L12.5 4V10L7 13L1.5 10V4L7 1Z" stroke="white" strokeWidth="1.5" strokeLinejoin="round"/>
                <path d="M7 5L9.5 6.5V9L7 10.5L4.5 9V6.5L7 5Z" fill="white"/>
              </svg>
            </div>
            <div>
              <h1 className="text-sm font-semibold text-ink">Radiant Norma</h1>
              <p className="text-xs text-ink-muted">Status Operacional</p>
            </div>
          </div>
          <div className="text-right">
            <p className="text-xs font-mono text-ink-muted">
              {now.toLocaleString('pt-BR', {
                timeZone: 'America/Sao_Paulo',
                dateStyle: 'short',
                timeStyle: 'short',
              })} BRT
            </p>
          </div>
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-6 py-10 space-y-8">
        {/* Overall Status */}
        <section>
          <OverallStatusBanner status={data?.status ?? 'outage'} />
        </section>

        {/* Uptime */}
        <section className="grid grid-cols-3 gap-4">
          {[
            { label: 'Uptime (30 dias)', value: data?.uptime_this_month_pct ?? 0, suffix: '%' },
            { label: 'SLA Enterprise', value: 99.9, suffix: '%' },
            { label: 'Verificado às', value: data?.checked_at ? new Date(data.checked_at).toLocaleTimeString('pt-BR', { timeZone: 'America/Sao_Paulo', hour: '2-digit', minute: '2-digit' }) : '--:--', suffix: ' BRT' },
          ].map((stat) => (
            <div key={stat.label} className="p-4 rounded-xl border border-border bg-surface-raised text-center">
              <p className="text-2xl font-mono font-semibold text-ink">{stat.value}{stat.suffix}</p>
              <p className="mt-1 text-xs text-ink-muted">{stat.label}</p>
            </div>
          ))}
        </section>

        {/* Components */}
        <section>
          <h2 className="text-xs font-semibold uppercase tracking-wider text-ink-muted mb-3">
            Componentes
          </h2>
          <div className="rounded-xl border border-border overflow-hidden">
            {data?.components.map((c) => (
              <ComponentRow key={c.name} component={c} />
            )) ?? (
              <div className="py-8 text-center text-sm text-ink-muted">
                Não foi possível conectar ao servidor de status.
              </div>
            )}
          </div>
        </section>

        {/* Incidents */}
        {(data?.incidents?.length ?? 0) > 0 && (
          <section>
            <h2 className="text-xs font-semibold uppercase tracking-wider text-ink-muted mb-3">
              Incidentes ativos
            </h2>
            <div className="space-y-3">
              {data!.incidents.map((inc) => (
                <IncidentCard key={inc.id} incident={inc} />
              ))}
            </div>
          </section>
        )}

        {/* Footer */}
        <footer className="pt-4 border-t border-border space-y-3">
          <p className="text-xs text-ink-muted text-center">
            Para incidentes, contacte{' '}
            <a href="mailto:suporte@radiant.digital" className="text-accent-600 dark:text-accent-400 hover:underline">
              suporte@radiant.digital
            </a>
            {' '}· SLA Enterprise: 99.9% uptime
          </p>
          <div className="flex justify-center gap-6 text-xs text-ink-subtle">
            <Link href="/docs/SLA.md" className="hover:text-ink-muted transition-colors">SLA</Link>
            <Link href="/docs/LGPD_COMPLIANCE.pdf" className="hover:text-ink-muted transition-colors">LGPD</Link>
            <Link href="/docs/SOC2_READINESS.md" className="hover:text-ink-muted transition-colors">SOC 2</Link>
            <Link href="https://radiant.digital" className="hover:text-ink-muted transition-colors">radiant.digital</Link>
          </div>
        </footer>
      </main>
    </div>
  )
}
