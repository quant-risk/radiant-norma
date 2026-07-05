// Server-side JWT verification.
//
// Separado de auth.ts porque auth.ts é client-only (`'use client'`
// directive no topo). Importar server-only code de um módulo client
// quebra no Next App Router: webpack descarta a função no client
// bundle, e server-side imports recebem `undefined`.
//
// Em prod: tokens emitidos por IdP externo (Keycloak/Okta/etc).
// Em dev: backend emite JWT RS256 via /v1/auth/dev-token (Sprint 8a).
//
// Sprint 13 — v3.5.2 [S13.5 / C-FE-4]:
// CRITICAL — variáveis JWT SEM prefixo `NEXT_PUBLIC_`. Variáveis
// `NEXT_PUBLIC_*` são inline no bundle do browser (mesmo server-only
// files que importam-nas). Mesmo sendo public key, expor issuer +
// pubkey no client facilita reconhecimento + força bruta de kid.
// Renomeado para `RADIANT_API_JWT_PUBKEY` / `RADIANT_API_JWT_ISSUER`.

import 'server-only' // Sprint 13: garante fail-loud se client importar
import { jwtVerify, importSPKI } from 'jose'
import type { Session } from './auth'

export async function verifyJwtServer(token: string): Promise<Session | null> {
  const pubKeyPem = process.env.RADIANT_API_JWT_PUBKEY
  if (!pubKeyPem) return null
  try {
    const pubKey = await importSPKI(pubKeyPem, 'RS256')
    const { payload } = await jwtVerify(token, pubKey, {
      issuer: process.env.RADIANT_API_JWT_ISSUER,
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