// /login — escolhe IF para login dev.
//
// Sprint 7c (v2.0.0) MVP: lista de IFs demo + role selector.
// Em produção, integra com IdP (Keycloak/Okta etc).

'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'

const DEMO_IFS = [
  { id: 'demo', role: 'if' as const, label: 'Demo IF (SCD)' },
  { id: 'demo-banco', role: 'if' as const, label: 'Demo Banco (BC)' },
  { id: 'demo-admin', role: 'admin' as const, label: 'Demo Admin' },
]

export default function LoginPage() {
  const [selected, setSelected] = useState(DEMO_IFS[0].id)
  const [loading, setLoading] = useState(false)
  const router = useRouter()

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    try {
      const r = await fetch('/api/login', {
        method: 'POST',
        body: JSON.stringify({ if_id: selected }),
        headers: { 'Content-Type': 'application/json' },
      })
      if (!r.ok) {
        const err = await r.json().catch(() => ({ error: 'login failed' }))
        alert(`login failed: ${err.error}`)
        return
      }
      router.push('/')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="min-h-screen flex items-center justify-center bg-slate-50 p-4">
      <form
        onSubmit={handleLogin}
        className="bg-white p-8 rounded-lg shadow-md w-full max-w-md"
      >
        <h1 className="text-2xl font-bold mb-2">Radiant Norma</h1>
        <p className="text-slate-500 text-sm mb-6">Console · Sprint 7c (v2.0.0)</p>

        <label className="block mb-4">
          <span className="text-sm font-medium text-slate-700">IF</span>
          <select
            value={selected}
            onChange={(e) => setSelected(e.target.value)}
            className="w-full mt-1 px-3 py-2 border border-slate-300 rounded-md"
          >
            {DEMO_IFS.map((d) => (
              <option key={d.id} value={d.id}>
                {d.label}
              </option>
            ))}
          </select>
        </label>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-primary-600 text-white py-2 px-4 rounded-md font-medium hover:bg-primary-700 disabled:opacity-50"
        >
          {loading ? 'autenticando...' : 'Entrar'}
        </button>

        <p className="mt-4 text-xs text-slate-500">
          Dev mode: backend aceita X-IF-ID via RADIANT_DEV_AUTH=1.
        </p>
      </form>
    </main>
  )
}
