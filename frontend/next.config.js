/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  experimental: {
    typedRoutes: true,
  },
  env: {
    RADIANT_API_URL: process.env.RADIANT_API_URL || 'http://localhost:8080',
  },
  // Sprint 13 — v3.5.2 [S13.5 / C-FE-5] security headers.
  //
  // Aplicação regulatória BACEN sem CSP é vetor trivial de XSS via
  // qualquer dependência npm comprometida. Headers:
  //   * CSP: default-src 'self' (+ connect-src para API + SSE stream)
  //   * HSTS: 1 ano com includeSubDomains (HTTPS only)
  //   * X-Content-Type-Options: nosniff (anti MIME-sniff)
  //   * X-Frame-Options: DENY (anti clickjacking; redundante com frame-ancestors)
  //   * Referrer-Policy: strict-origin-when-cross-origin
  //   * Permissions-Policy: deny mic/camera/geolocation/etc (não usados)
  //   * Cross-Origin-Opener-Policy/Embedder-Policy: same-origin-allow-popups
  //
  // Em dev: CSP fica relaxada (script-src unsafe-eval/unsafe-inline para
  // Next.js HMR e styled-jsx). Em prod: strict.
  async headers() {
    const isProd = process.env.NODE_ENV === 'production'
    const apiUrl = process.env.RADIANT_API_URL || 'http://localhost:8080'

    // CSP base — conecta no API backend + WS para SSE
    const csp = [
      "default-src 'self'",
      isProd
        ? "script-src 'self'" // nonces aplicados via middleware
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
        headers: [
          { key: 'Content-Security-Policy', value: csp },
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
              : 'max-age=0', // dev: não cachear HSTS
          },
        ],
      },
    ]
  },
}

module.exports = nextConfig
