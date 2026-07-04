// Server-side fetch wrapper que injeta JWT do cookie de sessão.
//
// Difere de api.ts (client): usa next/headers para ler cookie
// durante SSR e adiciona Authorization header automaticamente.

import { headers } from 'next/headers'

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
  // `cookies()` foi removido em headers() no Next 14 server components.
  // Usar cookies() é mais limpo — mas para evitar dup import, leitura
  // direta:
  const { cookies } = await import('next/headers')
  return (await cookies()).get('rn_jwt')?.value
}
