// /auditoria — view-only audit log (LGPD/SOC2 compliance).
//
// Sprint 8+: query via /v1/audit_log com auth.role=admin.
// Sprint 7c (v2.0.0) MVP: page explicando o que audit log contém.

import { getServerSession } from '@/lib/session'

export const dynamic = 'force-dynamic'

export default async function AuditoriaPage() {
  const session = await getServerSession()
  if (!session) return <p>Não autenticado. <a href="/login">Login</a></p>

  return (
    <main className="p-8 max-w-6xl mx-auto">
      <h1 className="text-3xl font-bold mb-2">Auditoria</h1>
      <p className="text-slate-600 mb-8">
        Logs tamper-evident (SHA-256 hash chain) para compliance
        LGPD / SOC 2.
      </p>

      <div className="bg-amber-50 border border-amber-200 rounded-md p-4 text-sm">
        <strong>TODO Sprint 8:</strong> endpoint <code>/v1/audit_log</code>
        com paginação, filtros por IF/action/período. Requer
        <code>auth.role=admin</code>. Apenas admins podem ler.
      </div>

      <section className="mt-8 bg-white p-6 rounded-md shadow-sm">
        <h2 className="text-xl font-bold mb-3">Como funciona</h2>
        <ol className="space-y-2 text-sm text-slate-700 list-decimal pl-4">
          <li>
            Toda mutação na API emite entry no <code>audit_log</code>.
          </li>
          <li>
            Cada entry referencia SHA-256 da entry anterior (chain).
          </li>
          <li>
            Modify qualquer entry → chain quebrada →
            <code>auditlog.Verify()</code> detecta.
          </li>
          <li>
            Storage: SQLite local ou Postgres (driver dual).
          </li>
        </ol>
      </section>
    </main>
  )
}
