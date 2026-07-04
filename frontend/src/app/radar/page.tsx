// /radar — lista alertas regulatórios abertos.
//
// Server component: chama /v1/radar/alerts?unresolved=true, renderiza
// cards com severity-based color. Permite resolver (POST).

import { apiFetch } from '@/lib/api-fetch'
import { getServerSession } from '@/lib/session'
import { ResolveButton } from '@/components/resolve-alert-button'

export const dynamic = 'force-dynamic'

interface RadarAlert {
  id: number
  cadoc_code: string
  severity: 'info' | 'warn' | 'critical'
  title: string
  description: string
  source_url: string
  detected_at: string
  resolved: boolean
}

async function getAlerts(token: string): Promise<RadarAlert[]> {
  // Backend retorna { alerts: [], total: N } — strip wrapper.
  const r = await apiFetch<{ alerts: RadarAlert[]; total: number }>(
    '/v1/radar/alerts?unresolved=true',
    {},
    token,
  )
  return r.alerts ?? []
}

export default async function RadarPage() {
  const session = await getServerSession()
  if (!session) return <p>Não autenticado. <a href="/login">Login</a></p>

  const alerts = await getAlerts(session.token)

  return (
    <main className="p-8 max-w-6xl mx-auto">
      <h1 className="text-3xl font-bold mb-2">Radar Regulatório</h1>
      <p className="text-slate-600 mb-8">
        {alerts.length} alerta(s) não resolvidos
      </p>

      {alerts.length === 0 && (
        <div className="bg-green-50 border border-green-200 rounded-md p-6 text-green-700">
          ✨ Nenhum alerta ativo. URLs BACEN estão estáveis.
        </div>
      )}

      <div className="space-y-3">
        {alerts.map((a) => (
          <article
            key={a.id}
            className={`bg-white p-4 rounded-md border-l-4 ${
              a.severity === 'critical'
                ? 'border-red-500'
                : a.severity === 'warn'
                ? 'border-amber-500'
                : 'border-sky-500'
            } shadow-sm`}
          >
            <header className="flex items-baseline gap-3 mb-1">
              <span className="text-xs uppercase font-bold text-slate-500">
                {a.cadoc_code}
              </span>
              <h2 className="text-lg font-medium">{a.title}</h2>
              <span className="ml-auto text-xs text-slate-400">
                {new Date(a.detected_at).toLocaleString('pt-BR')}
              </span>
            </header>
            <p className="text-sm text-slate-700 mb-2">{a.description}</p>
            <a
              href={a.source_url}
              target="_blank"
              rel="noreferrer"
              className="text-xs text-primary-600 underline"
            >
              {a.source_url}
            </a>
            <ResolveButton id={a.id} />
          </article>
        ))}
      </div>
    </main>
  )
}
