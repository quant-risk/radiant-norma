// Dashboard / — visão geral para IF logado.
//
// Server component: lê session via cookies, faz queries
// paralelas para backend Go via fetch direto (sem axios para streaming
// fetch). Dados: stats de envios, alertas radar abertas, top regras.

import { cookies } from 'next/headers'
import { apiFetch } from '@/lib/api-fetch'
import { getServerSession } from '@/lib/session'

export const dynamic = 'force-dynamic'

interface Stats {
  envios_total: number
  envios_aprovados_24h: number
  alertas_ativas: number
  regras_ativas: number
}

async function getDashboardData(): Promise<Stats | null> {
  const session = await getServerSession()
  if (!session) return null

  const [envios, alertas] = await Promise.all([
    apiFetch<{ total: number }>('/v1/dashboard/envios', {}, session.token),
    apiFetch<{ alerts: unknown[] }>('/v1/radar/alerts?unresolved=true', {}, session.token),
  ])

  return {
    envios_total: envios.total ?? 0,
    envios_aprovados_24h: 0, // TODO Sprint 8
    alertas_ativas: alertas.alerts?.length ?? 0,
    regras_ativas: 60, // 5 raw + 55 tipadas (Sprint 7b)
  }
}

export default async function DashboardPage() {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-8">
        <h1 className="text-2xl font-bold">Não autenticado</h1>
        <p className="mt-4">
          <a href="/login" className="text-primary-600 underline">Faça login</a>
        </p>
      </div>
    )
  }

  const stats = await getDashboardData()

  return (
    <main className="p-8 max-w-6xl mx-auto">
      <h1 className="text-3xl font-bold mb-2">Dashboard</h1>
      <p className="text-slate-600 mb-8">
        IF: <span className="font-mono text-sm">{session.if_id}</span> ·
        Role: <span className="font-mono text-sm">{session.role}</span>
      </p>

      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <StatCard label="Envios totais" value={stats.envios_total} />
          <StatCard
            label="Aprovados (24h)"
            value={stats.envios_aprovados_24h}
          />
          <StatCard label="Alertas radar ativas" value={stats.alertas_ativas} />
          <StatCard label="Regras ativas" value={stats.regras_ativas} />
        </div>
      )}

      <nav className="mt-12 flex gap-4 text-sm">
        <a href="/envios" className="text-primary-600 hover:underline">
          Envios →
        </a>
        <a href="/radar" className="text-primary-600 hover:underline">
          Radar →
        </a>
        <a href="/regras" className="text-primary-600 hover:underline">
          Regras →
        </a>
        <a href="/auditoria" className="text-primary-600 hover:underline">
          Auditoria →
        </a>
      </nav>
    </main>
  )
}

function StatCard({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="bg-white p-6 rounded-lg shadow-sm border border-slate-200">
      <div className="text-xs uppercase text-slate-500 font-medium">
        {label}
      </div>
      <div className="text-3xl font-bold text-slate-900 mt-2">
        {typeof value === 'number' ? value.toLocaleString('pt-BR') : value}
      </div>
    </div>
  )
}
