// App Router root layout para o frontend Radiant Norma.
//
// Tipografia tripla via next/font:
//   - Inter (corpo)
//   - Fraunces (display/editorial — serifa moderna)
//   - JetBrains Mono (dados críticos)
//
// CSP nonce: middleware gera nonce por request e expõe via header
// `x-nonce`. O themeScript inline recebe esse nonce pra passar no
// CSP `script-src 'self' 'nonce-{nonce}'` em produção.

import type { Metadata } from 'next'
import { headers } from 'next/headers'
import { Inter, Fraunces, JetBrains_Mono } from 'next/font/google'
import { ReactQueryProvider } from './react-query-provider'
import { ThemeProvider, themeScript } from '@/components/theme-provider'
import './globals.css'

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-sans',
  display: 'swap',
})

const fraunces = Fraunces({
  subsets: ['latin'],
  variable: '--font-serif',
  display: 'swap',
  axes: ['SOFT', 'opsz'],
})

const jetbrainsMono = JetBrains_Mono({
  subsets: ['latin'],
  variable: '--font-mono',
  display: 'swap',
})

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
    <html
      lang="pt-BR"
      className={`${inter.variable} ${fraunces.variable} ${jetbrainsMono.variable}`}
      suppressHydrationWarning
    >
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