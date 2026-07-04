'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'

export function ResolveButton({ id }: { id: number }) {
  const [loading, setLoading] = useState(false)
  const router = useRouter()

  async function resolve() {
    setLoading(true)
    try {
      const token = document.cookie
        .split('; ')
        .find((c) => c.startsWith('rn_jwt='))
        ?.split('=')[1]
      const r = await fetch(`/v1-api/proxy/radar/alerts/${id}/resolve`, {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!r.ok) {
        alert(`falhou: ${r.status}`)
        return
      }
      router.refresh()
    } finally {
      setLoading(false)
    }
  }

  return (
    <button
      type="button"
      onClick={resolve}
      disabled={loading}
      className="mt-2 text-sm bg-primary-600 text-white px-3 py-1 rounded disabled:opacity-50"
    >
      {loading ? 'resolvendo...' : 'Resolver'}
    </button>
  )
}
