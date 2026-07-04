// Server-side fetch wrapper que injeta JWT do cookie de sessão.
//
// Difere de api.ts (client): usa next/headers para ler cookie
// durante SSR e adiciona Authorization header automaticamente.

import { cookies } from 'next/headers'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

export async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
  token?: string,
): Promise<T> {
  const url = `${API_URL}${path}`
  const tokenToUse = token ?? (await getCookieToken())
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init.headers as Record<string, string> || {}),
  }
  if (tokenToUse) {
    headers.Authorization = `Bearer ${tokenToUse}`
  }

  const response = await fetch(url, { ...init, headers })
  if (!response.ok) {
    const body = await response.text().catch(() => '')
    throw new Error(
      `API ${init.method || 'GET'} ${path} → ${response.status}: ${body || response.statusText}`,
    )
  }
  return response.json() as Promise<T>
}

// Lê token do cookie via next/headers (server side).
async function getCookieToken(): Promise<string | undefined> {
  // Validação 27 (F27.13): antes era `await import('next/headers')` —
  // dynamic import anti-pattern (Next 14 ESM). Top-level estático.
  return (await cookies()).get('rn_jwt')?.value
}
