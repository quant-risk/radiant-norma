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

import { NextRequest, NextResponse } from 'next/server'

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
  const body: LoginRequest = await req.json()
  if (!body.if_id || body.if_id.length > 64) {
    return NextResponse.json(
      { error: 'if_id required (max 64 chars)' },
      { status: 400 },
    )
  }

  const role = body.role ?? 'if'

  // Sprint 8a: bridge to backend /v1/auth/dev-token.
  //
  // Em prod: IdP externo emite token (Sprint 9+). Por enquanto, frontend
  // chama backend para gerar JWT RS256 in-process.
  const apiUrl = process.env.RADIANT_API_URL || 'http://localhost:8080'

  let token: string
  let expiresAt: string

  try {
    const r = await fetch(`${apiUrl}/v1/auth/dev-token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        if_id: body.if_id,
        role,
        ttl_seconds: 7 * 24 * 60 * 60, // 7 dias
      }),
    })

    if (!r.ok) {
      const err = await r.json().catch(() => ({ error: `dev-token failed: ${r.status}` }))
      const status = r.status === 404 ? 503 : 400
      return NextResponse.json(
        {
          error: err.error || 'dev-token failed',
          hint: r.status === 404
            ? 'Backend dev-token endpoint disabled. Set RADIANT_DEV_TOKEN=1 e RADIANT_DEV_JWT_PRIVATE_KEY no backend.'
            : undefined,
        },
        { status },
      )
    }

    const data: DevTokenResponse = await r.json()
    token = data.token
    expiresAt = data.expires_at
  } catch (e) {
    // Backend unreachable — fail loud com hint claro.
    return NextResponse.json(
      {
        error: 'backend unreachable',
        detail: e instanceof Error ? e.message : String(e),
        hint: 'Verifique se backend Go está rodando em RADIANT_API_URL.',
      },
      { status: 502 },
    )
  }

  // Set httpOnly cookie com JWT real.
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
    // Validação 27 (F27.16): secure flag condicional ao NODE_ENV.
    secure: process.env.NODE_ENV === 'production',
  })

  return response
}
