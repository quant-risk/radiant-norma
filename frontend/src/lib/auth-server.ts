// Server-side JWT verification.
//
// Separado de auth.ts porque auth.ts é client-only (`'use client'`
// directive no topo). Importar server-only code de um módulo client
// quebra no Next App Router: webpack descarta a função no client
// bundle, e server-side imports recebem `undefined`.
//
// Em prod: tokens emitidos por IdP externo (Keycloak/Okta/etc).
// Em dev: backend emite JWT RS256 via /v1/auth/dev-token (Sprint 8a).

import { jwtVerify, importSPKI } from 'jose'
import type { Session } from './auth'

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