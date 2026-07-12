// App Router root layout para o frontend Radiant Norma.
//
// Tipografia tripla via CSS custom properties (system fonts):
//   - --font-sans: system-ui (corpo)
//   - --font-serif: Georgia/'Times New Roman' (display/editorial)
//   - --font-mono: 'SF Mono', Menlo (dados críticos)
//
// Motivo: next/font/google faz fetch de fonts.googleapis.com no build,
// que não funciona neste ambiente. Sprint 34 [S34.1] — migrar para
// fonts locais quando infraestrutura de CDN estiver disponível.
//
// CSP nonce: middleware gera nonce por request e expõe via header
// `x-nonce`. O themeScript inline recebe esse nonce pra passar no
// CSP `script-src 'self' 'nonce-{nonce}'` em produção.

import type { Metadata } from 'next'
import { headers } from 'next/headers'
import { ReactQueryProvider } from './react-query-provider'
import { ThemeProvider, themeScript } from '@/components/theme-provider'
import './globals.css'

export const metadata: Metadata = {
  title: 'Radiant Norma · Console Regulatório',
  description:
    'Plataforma de validação CADOC e monitoramento regulatório para Instituições Financeiras brasileiras.',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  // Lê nonce do header injetado pelo middleware (Edge runtime).
  // Em dev pode ser undefined → script fica sem nonce (CSP relaxado).
  const nonce = headers().get('x-nonce') ?? undefined

  return (
    <html lang="pt-BR" suppressHydrationWarning>
      <head>
        {/* Theme bootstrap — evita FOUC em dark mode. nonce pra CSP prod. */}
        <script
          nonce={nonce}
          dangerouslySetInnerHTML={{ __html: themeScript }}
        />
      </head>
      <body className="bg-surface text-ink antialiased">
        <ThemeProvider>
          <ReactQueryProvider>{children}</ReactQueryProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}