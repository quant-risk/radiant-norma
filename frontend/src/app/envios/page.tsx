// /envios — lista envios STA do IF logado.
//
// Server component: lê envios via /v1 endpoints (TO DO no backend).
// Até v1.6.0, /v1/envios não existia — Sprint 8 adiciona.
// Por agora: placeholder com lista estática explicando requisitos.

import { getServerSession } from '@/lib/session'

export const dynamic = 'force-dynamic'

export default async function EnviosPage() {
  const session = await getServerSession()
  if (!session) return <p>Não autenticado. <a href="/login">Login</a></p>

  return (
    <main className="p-8 max-w-6xl mx-auto">
      <h1 className="text-3xl font-bold mb-2">Envios STA</h1>
      <p className="text-slate-600 mb-8">
        Histórico de envios para IF <code>{session.if_id}</code>.
      </p>

      <div className="bg-amber-50 border border-amber-200 rounded-md p-4 text-sm">
        <strong>TODO Sprint 8:</strong> backend endpoint <code>/v1/envios?if_id=X</code>
        listar envios do IF. Filtrar por status, data, CADOC. Ver
        <code>internal/worker/worker.go:claimEnvio()</code> + tabela
        <code>envios</code> para backend implementation.
      </div>
    </main>
  )
}
