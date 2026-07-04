'use client'

/**
 * AppShell — wrapper client-side que junta Sidebar + Topbar + CommandPalette
 * ao redor do conteúdo autenticado.
 *
 * Server components filhos podem passar children direto; este wrapper
 * só renderiza o chrome.
 */
import * as React from 'react'
import { Sidebar } from './sidebar'
import { Topbar, TopbarProps } from './topbar'
import { CommandPalette, CommandPaletteProps } from './command-palette'

interface Session {
  if_id: string
  role: 'admin' | 'if' | 'auditor' | 'readonly'
}

export interface AppShellProps {
  session: Session
  topbar: Omit<TopbarProps, 'onCommandPalette'>
  commandData?: Omit<CommandPaletteProps, never>
  children: React.ReactNode
}

export function AppShell({
  session,
  topbar,
  commandData,
  children,
}: AppShellProps) {
  // command palette open é controlado aqui pra passar pro Topbar e o
  // CommandPalette renderizar. Sem prop drilling — mantém aberto/fechado
  // interno no CommandPalette via ⌘K global.
  return (
    <div className="flex min-h-screen bg-surface">
      <Sidebar session={session} />
      <div className="flex-1 flex flex-col min-w-0">
        <Topbar {...topbar} />
        <main className="flex-1 px-6 py-6 lg:px-8 lg:py-8">
          {children}
        </main>
      </div>
      {commandData && (
        <CommandPalette
          rules={commandData.rules}
          alerts={commandData.alerts}
          schemas={commandData.schemas}
        />
      )}
    </div>
  )
}