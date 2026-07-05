// Server-side session helpers for App Router (Next.js 14+).
//
// Backend emite JWT. Frontend armazena em cookie httpOnly `rn_jwt`.
// No server-side, verificamos token com chave pública via jose.
//
// Em dev: cookie value é `dev:<if_id>:<role>` (string).
//          Mock verification: split e retornar Session.
// Em prod: cookie value é JWT RS256. Verify via jose.
//
// Sprint 13 — v3.5.2 [S13.7 / C-FE-3]:
// CRITICAL — `dev:` sintético NUNCA é aceito em NODE_ENV=production.
// Backend tem gate no cmd/api/main.go (fail-closed se RADIANT_DEV_TOKEN=1 +
// RADIANT_ENV=production). Aqui é mirror: mesmo se cookie for injetado
// via path antigo, debug, ou XSS, getServerSession devolve null.
// Edge middleware (src/middleware.ts) também checa — defesa em
// profundidade.

import 'server-only'
import { cookies } from 'next/headers'
import { verifyJwtServer } from './auth-server'
import type { Session } from './auth'

const isProd = process.env.NODE_ENV === 'production'

export async function getServerSession(): Promise<Session | null> {
  const cookieStore = cookies()
  const token = cookieStore.get('rn_jwt')?.value
  if (!token) return null

  if (token.startsWith('dev:')) {
    if (isProd) {
      // Bloqueia dev cookie em produção (fail-closed).
      // Log estruturado para forense — não inclui token cru.
      console.warn('[session] dev cookie blocked in prod', {
        node_env: process.env.NODE_ENV,
      })
      return null
    }
    const [, if_id, role] = token.split(':')
    return {
      token,
      if_id,
      role: role as Session['role'],
      sub: if_id,
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    }
  }

  return await verifyJwtServer(token)
}
