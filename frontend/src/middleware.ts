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

import { NextRequest, NextResponse } from 'next/server'

const PROTECTED_PATHS = /^\/(?!login|api\/login|api\/logout|_next|favicon\.ico).*/
// API routes que precisam de auth (proxy interno + SSE)
const PROTECTED_API = /^\/(v1-api|v1)\//

const isProd = process.env.NODE_ENV === 'production'

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl

  // Páginas públicas: /login (e seus assets) passam direto.
  if (!PROTECTED_PATHS.test(pathname) && !PROTECTED_API.test(pathname)) {
    return NextResponse.next()
  }

  // Pega cookie rn_jwt
  const token = req.cookies.get('rn_jwt')?.value
  if (!token) {
    return redirectToLogin(req, 'missing token')
  }

  // Sprint 13 [S13.7 / C-FE-2 / C-FE-3]:
  // Em produção, NUNCA aceitar cookie dev:* sintético. Em dev, é o
  // caminho oficial (backend emite via /v1/auth/dev-token).
  if (token.startsWith('dev:')) {
    if (isProd) {
      return redirectToLogin(req, 'dev cookie in prod')
    }
    // Em dev: passa direto (validação completa em getServerSession)
    return NextResponse.next()
  }

  // Cookie presente e não-dev: validar shape mínimo.
  // JWT tem 3 segmentos base64 separados por '.'.
  if (token.split('.').length !== 3) {
    return redirectToLogin(req, 'malformed token')
  }

  return NextResponse.next()
}

function redirectToLogin(req: NextRequest, reason: string) {
  const url = req.nextUrl.clone()
  url.pathname = '/login'
  url.searchParams.set('next', req.nextUrl.pathname + req.nextUrl.search)
  // Log estruturado no servidor (Edge runtime tem console disponível)
  console.warn('[middleware] redirect', {
    path: req.nextUrl.pathname,
    reason,
    node_env: process.env.NODE_ENV,
  })
  return NextResponse.redirect(url)
}

// Config: rodar middleware em todas as rotas exceto static assets.
export const config = {
  matcher: [
    // Match everything EXCEPT:
    //   - _next/static (CSS, JS bundles)
    //   - _next/image (image optimization)
    //   - favicon
    '/((?!_next/static|_next/image|favicon\\.ico).*)',
  ],
}
