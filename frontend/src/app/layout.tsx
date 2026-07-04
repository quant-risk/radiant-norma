// App Router root layout para o frontend Radiant Norma.
//
// Stack: Next.js 14 App Router + React 18 + TailwindCSS 3 +
// TanStack Query (server state via Server Components).
//
// Server components first, client components only when needed.

import type { Metadata } from 'next'
import { ReactQueryProvider } from './react-query-provider'
import './globals.css'

export const metadata: Metadata = {
  title: 'Radiant Norma — Console',
  description: 'Dashboard IF para validação CADOCs, alertas radar, regras e auditoria.',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="pt-BR">
      <body className="bg-slate-50 text-slate-900">
        <ReactQueryProvider>{children}</ReactQueryProvider>
      </body>
    </html>
  )
}
