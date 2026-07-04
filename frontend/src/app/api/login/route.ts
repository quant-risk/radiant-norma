// NextAuth.js-like route para dev/demo login.
//
// Em prod: integra com backend JWT real (Sprint 7a v1.6.0).
// Aqui: dev helper que pega `if_id` via body, chama backend
// `cmd/jwt-mint` (em dev) ou backend `/v1/auth/dev-token`
// (TODO Sprint 8+).
//
// Resposta: 200 com JWT string (frontend armazena em cookie httpOnly).

import { NextRequest, NextResponse } from 'next/server'

interface LoginRequest {
  if_id: string
  role?: 'if' | 'admin' | 'readonly'
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

  // Dev mode: gerar JWT via dev endpoint. Em prod, vem do IdP.
  const apiUrl = process.env.RADIANT_API_URL || 'http://localhost:8080'
  let token: string

  // Opção 1: backend dev endpoint (TODO Sprint 8).
  // Opção 2: server-side gera JWT usando private key from env
  //          (NÃO recom — usar cmd/jwt-mint local em dev).
  //
  // Para simplicidade: usamos o dev_endpoint se existir, senão
  // fallback para X-IF-ID sem JWT (RADIANT_DEV_AUTH=1 no backend).
  const devMode = process.env.NEXT_PUBLIC_RADIANT_DEV_MODE === '1'

  if (devMode) {
    // Dev: reusa X-IF-ID via header. Frontend passa via cookie.
    // Backend aceita X-IF-ID quando RADIANT_DEV_AUTH=1.
    // Aqui só retornamos um "fake" token para que o axios
    // interceptor set o Authorization via cookie separado.
    token = `dev:${body.if_id}:${role}`
  } else {
    // Production sem IdP configurado: 503.
    return NextResponse.json(
      { error: 'JWT issuance not configured. Set NEXT_PUBLIC_RADIANT_DEV_MODE=1 (dev only)' },
      { status: 503 },
    )
  }

  // Set httpOnly cookie for security
  const response = NextResponse.json({
    if_id: body.if_id,
    role,
    expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
  })
  response.cookies.set({
    name: 'rn_jwt',
    value: token,
    httpOnly: true,
    sameSite: 'lax',
    path: '/',
    maxAge: 7 * 24 * 60 * 60,
  })

  return response
}
