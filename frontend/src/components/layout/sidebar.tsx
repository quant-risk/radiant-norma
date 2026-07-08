'use client'

/**
 * Sidebar — navegação primária refinado.
 *
 * Identidade visual: wordmark "RN" em Fraunces serif + "Radiant Norma"
 * em Inter. Hairline divider entre header e nav. Active state: rail
 * vertical de 2px em gradient accent à esquerda do item + bg accent-50.
 *
 * Validação 29 (preservada):
 *   - H1 fix: collapsed state persiste em localStorage
 *   - H2 fix: mobile drawer
 *   - H6 fix: badge apenas 'live'
 *   - C8: strict route match
 */

import * as React from 'react'
import Link from 'next/link'
import type { Route } from 'next'
import { usePathname } from 'next/navigation'
import {
  LayoutDashboard,
  Send,
  Radar,
  BookCheck,
  Sparkles,
  History,
  ChevronsLeft,
  ChevronsRight,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Tooltip } from '@/components/ui/tooltip'

interface Session {
  if_id: string
  role: 'admin' | 'if' | 'auditor' | 'readonly'
}

interface SidebarProps {
  session: Session
  mobileOpen?: boolean
  onMobileClose?: () => void
}

const NAV_GROUPS: Array<{
  label: string
  items: Array<{
    href: Route
    label: string
    icon: React.ComponentType<{ className?: string; strokeWidth?: number | string }>
    badge?: 'live'
  }>
}> = [
  {
    label: 'Operação',
    items: [
      { href: '/', label: 'Dashboard', icon: LayoutDashboard },
      { href: '/envios', label: 'Envios STA', icon: Send },
      { href: '/radar', label: 'Radar', icon: Radar, badge: 'live' },
      { href: '/regras', label: 'Regras', icon: BookCheck },
    ],
  },
  {
    label: 'Inteligência',
    items: [
      { href: '/insights', label: 'Insights', icon: Sparkles },
      { href: '/auditoria', label: 'Auditoria', icon: History },
    ],
  },
]

function isActiveRoute(pathname: string | null, itemHref: Route): boolean {
  if (!pathname) return false
  if (pathname === itemHref) return true
  if (itemHref === '/') return false
  return pathname.startsWith(`${itemHref}/`)
}

export function Sidebar({ session, mobileOpen, onMobileClose }: SidebarProps) {
  const pathname = usePathname()
  const [collapsed, setCollapsed] = React.useState(false)
  const [hydrated, setHydrated] = React.useState(false)

  React.useEffect(() => {
    setHydrated(true)
    const stored = localStorage.getItem('rn_sidebar_collapsed')
    if (stored === '1') setCollapsed(true)
  }, [])

  React.useEffect(() => {
    if (hydrated) {
      localStorage.setItem('rn_sidebar_collapsed', collapsed ? '1' : '0')
    }
  }, [collapsed, hydrated])

  React.useEffect(() => {
    onMobileClose?.()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname])

  const widthClass = hydrated ? (collapsed ? 'w-[72px]' : 'w-64') : 'w-64'

  const desktopSidebar = (
    <aside
      className={cn(
        'hidden lg:flex flex-col h-screen sticky top-0 z-30',
        'bg-surface-raised/95 backdrop-blur-xl',
        'border-r border-border',
        'transition-[width] duration-320 ease-out-expo',
        widthClass,
      )}
    >
      <SidebarContent
        session={session}
        collapsed={hydrated ? collapsed : false}
        pathname={pathname}
        onToggleCollapse={() => setCollapsed((c) => !c)}
      />
    </aside>
  )

  const mobileDrawer = mobileOpen ? (
    <div className="lg:hidden fixed inset-0 z-40">
      <div
        className="absolute inset-0 bg-ink/40 backdrop-blur-sm animate-fade-in-fast"
        onClick={onMobileClose}
        aria-hidden
      />
      <aside
        className={cn(
          'absolute left-0 top-0 bottom-0 w-64',
          'bg-surface-raised border-r border-border shadow-2xl',
          'animate-slide-down flex flex-col',
        )}
      >
        <SidebarContent
          session={session}
          collapsed={false}
          pathname={pathname}
          onToggleCollapse={() => {
            onMobileClose?.()
          }}
        />
      </aside>
    </div>
  ) : null

  return (
    <>
      {desktopSidebar}
      {mobileDrawer}
    </>
  )
}

/* ─────── SidebarContent ─────── */

function SidebarContent({
  session,
  collapsed,
  pathname,
  onToggleCollapse,
}: {
  session: Session
  collapsed: boolean
  pathname: string | null
  onToggleCollapse: () => void
}) {
  return (
    <>
      {/* Wordmark */}
      <div
        className={cn(
          'flex items-center h-[68px] px-5 border-b border-border',
          collapsed && 'justify-center px-0',
        )}
      >
        {collapsed ? (
          <div
            className="size-9 shrink-0 rounded-md bg-gradient-to-br from-accent-600 to-magenta-500 flex items-center justify-center text-white font-serif text-base font-medium shadow-glow-accent-sm"
            aria-label="Radiant Norma"
          >
            R
          </div>
        ) : (
          <div className="flex items-center gap-2.5 min-w-0">
            <div
              className="size-9 shrink-0 rounded-md bg-gradient-to-br from-accent-600 to-magenta-500 flex items-center justify-center text-white font-serif text-base font-medium shadow-glow-accent-sm"
              aria-hidden
            >
              R
            </div>
            <div className="flex flex-col min-w-0 leading-none">
              <span className="font-serif text-[15px] font-medium text-ink tracking-tight truncate">
                Radiant Norma
              </span>
              <span className="text-2xs uppercase tracking-[0.18em] text-ink-subtle font-mono mt-0.5 truncate">
                Console
              </span>
            </div>
          </div>
        )}
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-5 px-3">
        {NAV_GROUPS.map((group, gi) => (
          <div key={group.label} className={cn(gi > 0 && 'mt-6')}>
            {!collapsed && (
              <div className="px-3 mb-2.5 text-2xs uppercase tracking-[0.18em] font-mono font-medium text-ink-subtle">
                {group.label}
              </div>
            )}
            <ul className="space-y-0.5">
              {group.items.map((item) => {
                const active = isActiveRoute(pathname, item.href)
                const Icon = item.icon

                const link = (
                  <Link
                    href={item.href}
                    aria-current={active ? 'page' : undefined}
                    className={cn(
                      'group relative flex items-center gap-3 px-3 h-9 rounded-md',
                      'text-sm font-medium tracking-tight',
                      'transition-all duration-180 ease-out-expo',
                      active
                        ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/50 dark:text-accent-300'
                        : 'text-ink-muted hover:bg-surface-sunken hover:text-ink',
                      collapsed && 'justify-center px-0',
                    )}
                  >
                    {active && (
                      <span
                        className="absolute left-0 top-1.5 bottom-1.5 w-[2px] rounded-r-full bg-gradient-to-b from-accent-500 to-magenta-500"
                        aria-hidden
                      />
                    )}
                    <Icon
                      className={cn(
                        'size-4 shrink-0 transition-colors',
                        active && 'text-accent-600 dark:text-accent-400',
                      )}
                      strokeWidth={active ? 2.25 : 1.75}
                    />
                    {!collapsed && (
                      <span className="flex-1 truncate">{item.label}</span>
                    )}
                    {!collapsed && item.badge === 'live' && (
                      <span className="flex items-center gap-1 text-2xs text-ink-subtle">
                        <span className="relative flex size-1.5">
                          <span className="absolute inset-0 rounded-full bg-success-500 animate-ping opacity-60" />
                          <span className="relative rounded-full size-1.5 bg-success-500" />
                        </span>
                      </span>
                    )}
                  </Link>
                )

                if (collapsed) {
                  return (
                    <li key={item.href}>
                      <Tooltip content={item.label} side="right">
                        {link}
                      </Tooltip>
                    </li>
                  )
                }
                return <li key={item.href}>{link}</li>
              })}
            </ul>
          </div>
        ))}
      </nav>

      {/* Footer */}
      <div
        className={cn(
          'border-t border-border p-3 flex flex-col gap-2',
          collapsed && 'items-center',
        )}
      >
        <button
          onClick={onToggleCollapse}
          className={cn(
            'flex items-center gap-2.5 px-3 h-8 rounded-md w-full',
            'text-xs font-medium text-ink-subtle hover:bg-surface-sunken hover:text-ink-muted transition-colors',
            collapsed && 'justify-center px-0',
          )}
          aria-label={collapsed ? 'Expandir sidebar' : 'Recolher sidebar'}
        >
          {collapsed ? (
            <ChevronsRight className="size-4" />
          ) : (
            <>
              <ChevronsLeft className="size-4" />
              <span>Recolher</span>
            </>
          )}
        </button>
        {!collapsed && (
          <div className="px-3 py-2 rounded-md bg-surface-sunken flex items-center gap-2">
            <span
              className={cn(
                'size-1.5 rounded-full',
                session.role === 'admin'
                  ? 'bg-accent-500'
                  : session.role === 'auditor'
                    ? 'bg-info-500'
                    : 'bg-success-500',
              )}
              aria-hidden
            />
            <span className="text-xs text-ink-muted capitalize">
              {session.role}
            </span>
            <span className="text-2xs text-ink-subtle ml-auto font-mono truncate">
              {session.if_id.slice(0, 12)}
            </span>
          </div>
        )}
      </div>
    </>
  )
}