'use client'

/**
 * Sidebar — navegação primária.
 *
 * Estrutura:
 *   - Header com logo + IF badge (mostra IF ativo)
 *   - 1 seção "Operação" (Dashboard, Envios, Radar, Regras)
 *   - 1 seção "Inteligência" (Insights, Auditoria)
 *   - Footer com settings + user menu
 *
 * Persistente em ≥1024px (256px); em <1024px vira drawer (será implementado
 * via estado no Topbar).
 *
 * Estado persistido: item ativo é derivado do pathname (não estado local).
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
}

const NAV_GROUPS: Array<{
  label: string
  items: Array<{
    href: Route
    label: string
    icon: React.ComponentType<{ className?: string }>
    badge?: 'count' | 'live'
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
      { href: '/insights', label: 'Insights', icon: Sparkles, badge: 'count' },
      { href: '/auditoria', label: 'Auditoria', icon: History },
    ],
  },
]

export function Sidebar({ session }: SidebarProps) {
  const pathname = usePathname()
  const [collapsed, setCollapsed] = React.useState(false)

  return (
    <aside
      className={cn(
        'hidden lg:flex flex-col h-screen sticky top-0 z-30',
        'bg-surface-raised border-r border-border',
        'transition-[width] duration-200 ease-out',
        collapsed ? 'w-16' : 'w-64',
      )}
    >
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
                const active =
                  pathname === item.href ||
                  (item.href !== '/' && pathname?.startsWith(item.href))
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
          onClick={() => setCollapsed((c) => !c)}
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
    </aside>
  )
}