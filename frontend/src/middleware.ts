// Edge middleware for Next.js — runs at the boundary BEFORE any
// page route or API proxy route is invoked.
//
// Sprint 13 — v3.5.2 [S13.6 / C-FE-1]:
// CRITICAL — authz do frontend estava só nos page server components
// (cada page.tsx chama getServerSession() com fallback client-side).
// VETOR: usuário deslogado navegava para /auditoria → SSR renderiza
// "Sessão expirada" mas o backend Go inteiro era alcançável via
// /api/login, /v1-api/proxy/* porque NÃO existia middleware na borda.
// Atacar fetch(curl) `GET /v1-api/proxy/v1/audit_log` retornaria
// JSON cru porque proxy usa cookie httpOnly.
//
// Estratégia: este middleware valida `rn_jwt` em todas as rotas
// EXCETO as explicitamente públicas. É leve (não parseia JWT —
// só checa presença e formato mínimo), o parse real é feito em
// server components via verifyJwtServer().
//
// Routes públicas:
//   /login                 — precisa estar deslogado pra ver
//   /api/login             — POST endpoint pra mintar token
//   /api/logout            — POST endpoint pra limpar cookie
//   /_next/*               — assets/build do Next
//   /favicon.ico           — favicon
//
// API routes internas (no mesmo host) que proxyam para backend:
//   /v1-api/proxy/*        — REQUER auth (handler interno valida)
//   /v1-api/events/stream  — SSE; auth checada no handler (auth middleware
//                            do backend também roda).
//
// NOTA: este middleware roda no Edge Runtime. Não consegue usar
// crypto/jose aqui (limites). Só checa shape do cookie.
//
// Sprint 13 [S13.7]: cookie `dev:<if_id>:<role>` é aceito aqui
// APENAS em NODE_ENV != production. Em prod, qualquer cookie que
// comece com `dev:` é treated como missing (força redirect /login).
//
// CSP nonce (Sprint 55 follow-up): o themeScript inline do layout.tsx
// (anti-FOUC dark mode) precisa de nonce em prod pra passar no CSP
// `script-src 'self'`. Geramos um nonce por request, devolvemos via
// header `x-nonce` e o layout.tsx aplica no <script>.

import { NextRequest, NextResponse } from 'next/server'

const PROTECTED_PATHS = /^\/(?!login|api\/login|api\/logout|_next|favicon\.ico).*/
// API routes que precisam de auth (proxy interno + SSE)
const PROTECTED_API = /^\/(v1-api|v1)\//

const isProd = process.env.NODE_ENV === 'production'

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl

  // CSP nonce — gerado por request (Edge Runtime: crypto.randomUUID())
  const nonce = req.headers.get('x-nonce') ?? generateNonce()

  // CSP dinâmico por request: nonce aplicado em script-src. O header
  // CSP aqui SUBSTITUI o do next.config.js (não há conflito porque
  // removemos CSP do next.config acima).
  const apiUrl = process.env.RADIANT_API_URL || 'http://localhost:8080'
  const csp = [
    "default-src 'self'",
    isProd
      ? `script-src 'self' 'nonce-${nonce}'`
      : "script-src 'self' 'unsafe-eval' 'unsafe-inline'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: blob:",
    "font-src 'self' data:",
    `connect-src 'self' ${apiUrl} ${apiUrl.replace('http', 'ws')} ${apiUrl.replace('http', 'wss')}`,
    "frame-ancestors 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "object-src 'none'",
    "manifest-src 'self'",
  ].join('; ')

  // Helper: aplica nonce + CSP e retorna response
  const passThrough = () => {
    const res = NextResponse.next({
      request: { headers: addNonceHeader(req, nonce) },
    })
    res.headers.set('x-nonce', nonce)
    res.headers.set('Content-Security-Policy', csp)
    return res
  }

  // Páginas públicas: /login (e seus assets) passam direto.
  if (!PROTECTED_PATHS.test(pathname) && !PROTECTED_API.test(pathname)) {
    return passThrough()
  }

  // Pega cookie rn_jwt
  const token = req.cookies.get('rn_jwt')?.value
  if (!token) {
    return redirectToLogin(req, 'missing token', nonce, csp)
  }

  // Sprint 13 [S13.7 / C-FE-2 / C-FE-3]:
  // Em produção, NUNCA aceitar cookie dev:* sintético. Em dev, é o
  // caminho oficial (backend emite via /v1/auth/dev-token).
  if (token.startsWith('dev:')) {
    if (isProd) {
      return redirectToLogin(req, 'dev cookie in prod', nonce, csp)
    }
    return passThrough()
  }

  // Cookie presente e não-dev: validar shape mínimo.
  if (token.split('.').length !== 3) {
    return redirectToLogin(req, 'malformed token', nonce, csp)
  }

  return passThrough()
}

function redirectToLogin(req: NextRequest, reason: string, nonce: string, csp: string) {
  const url = req.nextUrl.clone()
  url.pathname = '/login'
  url.searchParams.set('next', req.nextUrl.pathname + req.nextUrl.search)
  console.warn('[middleware] redirect', {
    path: req.nextUrl.pathname,
    reason,
    node_env: process.env.NODE_ENV,
  })
  const res = NextResponse.redirect(url)
  res.headers.set('x-nonce', nonce)
  res.headers.set('Content-Security-Policy', csp)
  return res
}

function generateNonce(): string {
  // Edge Runtime tem crypto.randomUUID(). Fallback manual se não houver.
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID().replace(/-/g, '')
  }
  return Array.from({ length: 32 }, () =>
    Math.floor(Math.random() * 16).toString(16),
  ).join('')
}

function addNonceHeader(req: NextRequest, nonce: string): Headers {
  const headers = new Headers(req.headers)
  headers.set('x-nonce', nonce)
  return headers
}

// Config: rodar middleware em todas as rotas exceto static assets.
export const config = {
  matcher: [
    '/((?!_next/static|_next/image|favicon\\.ico).*)',
  ],
}
