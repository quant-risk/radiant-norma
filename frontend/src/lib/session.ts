// Server-side session helpers for App Router (Next.js 14+).
//
// Backend emite JWT. Frontend armazena em cookie httpOnly `rn_jwt`.
// No server-side, verificamos token com chave pública via jose.
//
// Em dev: cookie value é `dev:<if_id>:<role>` (string).
//          Mock verification: split e retornar Session.
// Em prod: cookie value é JWT RS256. Verify via jose.

import { cookies } from 'next/headers'
import { verifyJwtServer, type Session } from './auth'

export async function getServerSession(): Promise<Session | null> {
  const cookieStore = cookies()
  const token = cookieStore.get('rn_jwt')?.value
  if (!token) return null

  if (token.startsWith('dev:')) {
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
