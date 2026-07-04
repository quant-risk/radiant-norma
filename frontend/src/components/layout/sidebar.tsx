'use client'

/**
 * Sidebar — navegação primária.
 *
 * Validação 29:
 *   - H1 fix: collapsed state persiste em localStorage
 *   - H2 fix: mobile drawer (<1024px) — abre/fecha via botão hamburger
 *     no Topbar (controlado via prop)
 *   - H6 fix: badge 'count' removido (dead code — só 'live' tinha render)
 *   - C8-adjacent fix: active state match mais estrito
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
    icon: React.ComponentType<{ className?: string }>
    badge?: 'live'
  }>
}> = [
  {
    label: 'Operação',
    items: [
      { href: '/', label: 'Dashboard', icon: LayoutDashboard },
      { href: '/envios', label: 'Envios', icon: Send },
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
  // Strict prefix match: /radar/foo mas NÃO /radar-extras
  if (itemHref === '/') return false // Dashboard só match exato
  return pathname.startsWith(`${itemHref}/`)
}

export function Sidebar({ session, mobileOpen, onMobileClose }: SidebarProps) {
  const pathname = usePathname()
  // H1 fix: collapsed state hidratado de localStorage no mount
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

  // H2 fix: fecha drawer mobile ao navegar
  React.useEffect(() => {
    onMobileClose?.()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname])

  // SSR-safe: evita render diferente do client no primeiro paint
  const widthClass = hydrated ? (collapsed ? 'w-16' : 'w-64') : 'w-64'

  // Desktop sidebar
  const desktopSidebar = (
    <aside
      className={cn(
        'hidden lg:flex flex-col h-screen sticky top-0 z-30',
        'bg-surface-raised border-r border-border',
        'transition-[width] duration-200 ease-out',
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

  // Mobile drawer (H2)
  const mobileDrawer = mobileOpen ? (
    <div className="lg:hidden fixed inset-0 z-40">
      <div
        className="absolute inset-0 bg-slate-950/40 backdrop-blur-sm animate-fade-in-fast"
        onClick={onMobileClose}
        aria-hidden
      />
      <aside
        className={cn(
          'absolute left-0 top-0 bottom-0 w-64',
          'bg-surface-raised border-r border-border shadow-xl',
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
      {/* Logo + IF */}
      <div
        className={cn(
          'flex items-center gap-3 px-4 h-16 border-b border-border',
          collapsed && 'justify-center px-0',
        )}
      >
        <div
          className="size-9 shrink-0 rounded-lg bg-gradient-to-br from-accent-500 to-accent-700 flex items-center justify-center text-white shadow-sm"
          aria-hidden
        >
          <svg viewBox="0 0 24 24" className="size-5" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M12 2L2 7l10 5 10-5-10-5z" />
            <path d="M2 17l10 5 10-5" />
            <path d="M2 12l10 5 10-5" />
          </svg>
        </div>
        {!collapsed && (
          <div className="flex flex-col min-w-0">
            <span className="text-sm font-semibold text-ink leading-tight">
              Radiant Norma
            </span>
            <span className="text-2xs text-ink-muted font-mono truncate">
              {session.if_id}
            </span>
          </div>
        )}
      </div>

      {/* Nav groups */}
      <nav className="flex-1 overflow-y-auto py-4 px-2">
        {NAV_GROUPS.map((group) => (
          <div key={group.label} className="mb-6">
            {!collapsed && (
              <div className="text-2xs uppercase tracking-wider font-semibold text-ink-subtle px-3 mb-2">
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
                      'flex items-center gap-3 px-3 h-9 rounded-md',
                      'text-sm font-medium transition-all duration-150',
                      active
                        ? 'bg-accent-50 text-accent-700 dark:bg-accent-950 dark:text-accent-300'
                        : 'text-ink-muted hover:bg-surface-sunken hover:text-ink',
                      collapsed && 'justify-center px-0',
                    )}
                  >
                    <Icon
                      className={cn(
                        'size-4 shrink-0',
                        active && 'text-accent-600 dark:text-accent-400',
                      )}
                    />
                    {!collapsed && (
                      <span className="flex-1 truncate">{item.label}</span>
                    )}
                    {!collapsed && item.badge === 'live' && (
                      <span className="flex items-center gap-1 text-2xs text-ink-subtle">
                        <span className="size-1.5 rounded-full bg-success-500 animate-pulse-soft" />
                        live
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

      {/* Footer: settings + collapse */}
      <div
        className={cn(
          'border-t border-border p-2 flex flex-col gap-0.5',
          collapsed && 'items-center',
        )}
      >
        <button
          onClick={onToggleCollapse}
          className={cn(
            'flex items-center gap-3 px-3 h-9 rounded-md w-full',
            'text-sm font-medium text-ink-muted hover:bg-surface-sunken hover:text-ink transition-colors',
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
          <div className="mt-2 px-3 py-2 rounded-md bg-surface-sunken flex items-center gap-2">
            <span
              className={cn(
                'size-2 rounded-full',
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
          </div>
        )}
      </div>
    </>
  )
}