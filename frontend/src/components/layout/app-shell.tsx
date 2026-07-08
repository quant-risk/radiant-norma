'use client'

/**
 * AppShell — wrapper client-side refinado.
 *
 * Composição: Sidebar (left rail) + Topbar (sticky) + main content.
 * CommandPalette é montada uma única vez no root e escuta atalhos globais.
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
  commandData?: CommandPaletteProps
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
      <div className="flex-1 flex flex-col min-w-0 relative">
        <Topbar
          {...topbar}
          userInitials={initialsFromIfId(session.if_id)}
          onMobileMenuClick={() => setMobileOpen(true)}
        />
        <main className="flex-1 px-6 py-6 lg:px-10 lg:py-10 max-w-[1400px] w-full mx-auto">
          {children}
        </main>
      </div>
      {commandData && <CommandPalette {...commandData} />}
    </div>
  )
}