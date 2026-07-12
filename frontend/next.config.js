/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  typedRoutes: true,
  env: {
    RADIANT_API_URL: process.env.RADIANT_API_URL || 'http://localhost:8080',
  },
  // Sprint 13 — v3.5.2 [S13.5 / C-FE-5] security headers.
  //
  // Headers estáticos (não-CSP) ficam aqui. CSP é aplicado dinamicamente
  // pelo middleware (Edge runtime) com nonce por request — necessário pra
  // permitir o <script> inline anti-FOUC do themeScript em produção.
  //
  // Em dev: middleware relaxa CSP (unsafe-eval/unsafe-inline para HMR).
  // Em prod: strict com nonce-{nonce} — script inline passa só com nonce.
  async headers() {
    const isProd = process.env.NODE_ENV === 'production'
    const apiUrl = process.env.RADIANT_API_URL || 'http://localhost:8080'

    // CSP fallback (caso middleware não rode — ex: build estático).
    // Em prod, o middleware sobrescreve este header com o nonce correto.
    const csp = [
      "default-src 'self'",
      isProd
        ? "script-src 'self'"
        : "script-src 'self' 'unsafe-eval' 'unsafe-inline'",
      "style-src 'self' 'unsafe-inline'", // Tailwind + shadcn usam inline
      "img-src 'self' data: blob:",
      "font-src 'self' data:",
      `connect-src 'self' ${apiUrl} ${apiUrl.replace('http', 'ws')} ${apiUrl.replace('http', 'wss')}`,
      "frame-ancestors 'none'",
      "base-uri 'self'",
      "form-action 'self'",
      "object-src 'none'",
      "manifest-src 'self'",
    ].join('; ')

    return [
      {
        source: '/(.*)',
        // skipMiddlewareHeaderInjection: true impede o Next de injetar
        // headers automaticamente. CSP é responsabilidade do middleware.
        // (você não quer CSP estático + dinâmico conflitando.)
        headers: [
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'X-Frame-Options', value: 'DENY' },
          { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
          {
            key: 'Permissions-Policy',
            value: 'camera=(), microphone=(), geolocation=(), interest-cohort=()',
          },
          { key: 'Cross-Origin-Opener-Policy', value: 'same-origin-allow-popups' },
          {
            key: 'Strict-Transport-Security',
            value: isProd
              ? 'max-age=31536000; includeSubDomains; preload'
              : 'max-age=0',
          },
        ],
      },
    ]
  },
}

module.exports = nextConfig
