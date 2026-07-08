// NextAuth.js-like route para dev/demo login.
//
// Sprint 8a (v2.1.0): bridge JWT real frontend↔backend.
//
// Fluxo:
//   1. User → /login → POST /api/login com if_id + role
//   2. Frontend chama backend POST /v1/auth/dev-token (que retorna JWT RS256)
//   3. Cookie rn_jwt httpOnly setado com JWT
//   4. SSR pages leem via next/headers cookies()
//   5. Server proxy /v1-api/proxy/[...path] injeta Authorization via JWT
//   6. Backend JWT verifier (mesma chave pública em keyring) aceita token
//
// Em produção: tokens emitidos por IdP externo. /v1/auth/dev-token retorna
// 404. Frontend redireciona para IdP OAuth flow.
//
// Sprint 13 — v3.5.2 [S13.7 / C-FE-2]:
// CRITICAL — em NODE_ENV=production, este endpoint retorna 404 IMEDIATO
// antes de chamar backend. Defense-in-depth: backend já bloqueia
// RADIANT_DEV_TOKEN=1 quando RADIANT_ENV=production (fail-closed em
// cmd/api/main.go). Aqui é mirror: se deploy esquecer de unsetar
// ambas as flags, ainda assim frontend não emite cookie dev.

import { NextRequest, NextResponse } from 'next/server'

const isProd = process.env.NODE_ENV === 'production'

interface LoginRequest {
  if_id: string
  role?: 'if' | 'admin' | 'readonly'
}

interface DevTokenResponse {
  token: string
  if_id: string
  role: string
  expires_at: string
  ttl_seconds: number
}

export async function POST(req: NextRequest) {
  // Sprint 13 [S13.7 / C-FE-2]: gate de produção.
  // Em prod, /api/login não existe — IdP externo deve emitir tokens.
  if (isProd) {
    return NextResponse.json(
      {
        error: 'login endpoint disabled',
        hint: 'Production deploys devem usar IdP externo (Keycloak/Okta/OIDC).'
      },
      { status: 404 },
    )
  }

  const body: LoginRequest = await req.json()
  if (!body.if_id || body.if_id.length > 64) {
    return NextResponse.json(
      { error: 'if_id required (max 64 chars)' },
      { status: 400 },
    )
  }

  const role = body.role ?? 'if'

  // Dev mode: usa cookie simples no formato dev:<if_id>:<role>
  // que é aceito pelo middleware sem verificação JWT.
  // Isso evita problemas com chaves RSA em dev.
  const token = `dev:${body.if_id}:${role}`
  const expiresAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString()

  const response = NextResponse.json({
    if_id: body.if_id,
    role,
    expires_at: expiresAt,
  })
  response.cookies.set({
    name: 'rn_jwt',
    value: token,
    httpOnly: true,
    sameSite: 'lax',
    path: '/',
    maxAge: 7 * 24 * 60 * 60,
    secure: process.env.NODE_ENV === 'production',
  })

  return response
}
