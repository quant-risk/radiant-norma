'use client'

/**
 * AppShell — wrapper client-side que junta Sidebar + Topbar + CommandPalette.
 *
 * Validação 29:
 *   - H2 fix: orquestra mobile drawer state
 *   - H3 fix: calcula user initials de session.if_id
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
  topbar: Omit<TopbarProps, 'onMobileMenuClick' | 'userInitials'>
  commandData?: Omit<CommandPaletteProps, never>
  children: React.ReactNode
}

function initialsFromIfId(ifId: string): string {
  if (!ifId) return '··'
  const cleaned = ifId.replace(/^demo[-_]?/i, '')
  const alpha = cleaned.match(/[a-zA-Z]/g)
  if (alpha && alpha.length >= 2) {
    return (alpha[0] + alpha[1]).toUpperCase()
  }
  return ifId.slice(0, 2).toUpperCase()
}

export function AppShell({
  session,
  topbar,
  commandData,
  children,
}: AppShellProps) {
  const [mobileOpen, setMobileOpen] = React.useState(false)

  return (
    <div className="flex min-h-screen bg-surface">
      <Sidebar
        session={session}
        mobileOpen={mobileOpen}
        onMobileClose={() => setMobileOpen(false)}
      />
      <div className="flex-1 flex flex-col min-w-0">
        <Topbar
          {...topbar}
          userInitials={initialsFromIfId(session.if_id)}
          onMobileMenuClick={() => setMobileOpen(true)}
        />
        <main className="flex-1 px-6 py-6 lg:px-8 lg:py-8">{children}</main>
      </div>
      {commandData && <CommandPalette {...commandData} />}
    </div>
  )
}