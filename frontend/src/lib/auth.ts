// Auth/session helpers para o frontend.
//
// Backend emite JWT RS256. Frontend armazena no cookie httpOnly
// `rn_jwt` setado pelo /api/login endpoint.
//
// `useSession()` hook: retorna current user, role, e refresh
// callback. Usa Zustand para client-side state + auto-refresh.

'use client'

import { create } from 'zustand'
import { api } from './api'

export interface Session {
  token: string
  if_id: string
  role: 'if' | 'admin' | 'readonly'
  sub: string
  expires_at: string // RFC3339
}

interface SessionState {
  session: Session | null
  setSession: (s: Session | null) => void
  loginAsIf: (if_id: string, role?: Session['role']) => Promise<void>
  logout: () => void
}

export const useSession = create<SessionState>((set) => ({
  session: null,
  setSession: (s) => set({ session: s }),
  loginAsIf: async (if_id, role = 'if') => {
    // Em dev: backend aceita X-IF-ID via RADIANT_DEV_AUTH=1.
    // Aqui chamamos /api/login do frontend que retorna JWT.
    const r = await fetch('/api/login', {
      method: 'POST',
      body: JSON.stringify({ if_id, role }),
      headers: { 'Content-Type': 'application/json' },
    })
    if (!r.ok) throw new Error('login failed')
    const data = await r.json()
    set({ session: data })
  },
  logout: () => set({ session: null }),
}))

// Server-side: verificar JWT em server components / route handlers.
// Usa chave pública do env NEXT_PUBLIC_RADIANT_API_JWT_PUBKEY.
import { jwtVerify, importSPKI } from 'jose'

export async function verifyJwtServer(token: string): Promise<Session | null> {
  const pubKeyPem = process.env.NEXT_PUBLIC_RADIANT_API_JWT_PUBKEY
  if (!pubKeyPem) return null
  try {
    const pubKey = await importSPKI(pubKeyPem, 'RS256')
    const { payload } = await jwtVerify(token, pubKey, {
      issuer: process.env.NEXT_PUBLIC_RADIANT_API_JWT_ISSUER,
    })
    return {
      token,
      if_id: String(payload['if_id']),
      role: payload['role'] as Session['role'],
      sub: String(payload.sub),
      expires_at: new Date((payload.exp as number) * 1000).toISOString(),
    }
  } catch {
    return null
  }
}

// Re-export api for convenience.
export { api }
