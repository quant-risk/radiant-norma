// App Router root layout para o frontend Radiant Norma.
//
// Stack: Next.js 14 App Router + React 18 + TailwindCSS 3 +
// TanStack Query (server state via Server Components) + ThemeProvider
// (dark mode via classe .dark em <html>).
//
// Server components first, client components only when needed.

import type { Metadata } from 'next'
import { Inter, JetBrains_Mono } from 'next/font/google'
import { ReactQueryProvider } from './react-query-provider'
import { ThemeProvider, themeScript } from '@/components/theme-provider'
import './globals.css'

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-sans',
  display: 'swap',
})

const jetbrains = JetBrains_Mono({
  subsets: ['latin'],
  variable: '--font-mono',
  display: 'swap',
})

export const metadata: Metadata = {
  title: 'Radiant Norma — Console',
  description:
    'Plataforma de validação CADOC e monitoramento regulatório para Instituições Financeiras brasileiras.',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html
      lang="pt-BR"
      className={`${inter.variable} ${jetbrains.variable}`}
      suppressHydrationWarning
    >
      <head>
        {/* Theme bootstrap — evita FOUC em dark mode */}
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="bg-surface text-ink antialiased">
        <ThemeProvider>
          <ReactQueryProvider>{children}</ReactQueryProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}