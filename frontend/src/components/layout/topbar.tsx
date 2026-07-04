'use client'

/**
 * Topbar — header sticky com 4 zonas:
 *   - Esquerda: hamburger (mobile) + breadcrumbs + title
 *   - Centro: command palette trigger (⌘K)
 *   - Direita: actions + theme toggle + notifications + user avatar
 *
 * Validação 29:
 *   - H3 fix: user initials derivados de session.if_id (não hardcoded)
 *   - H5 fix: command palette é auto-contido (sem prop dead)
 *   - H8 fix: breadcrumbs com href clicável
 */

import * as React from 'react'
import Link from 'next/link'
import type { Route } from 'next'
import { Search, Moon, Sun, Bell, Menu } from 'lucide-react'
import { useTheme } from '@/components/theme-provider'
import { Kbd } from '@/components/ui/kbd'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export interface BreadcrumbItem {
  label: string
  href?: Route
}

export interface TopbarProps {
  title: string
  subtitle?: string
  breadcrumbs?: BreadcrumbItem[]
  actions?: React.ReactNode
  userInitials?: string
  onMobileMenuClick?: () => void
}

export function Topbar({
  title,
  subtitle,
  breadcrumbs,
  actions,
  userInitials,
  onMobileMenuClick,
}: TopbarProps) {
  const { theme, toggle } = useTheme()
  const [mounted, setMounted] = React.useState(false)

  // Validação 29 (H7 fix): evita hydration mismatch no theme toggle.
  // Botão renderiza placeholder durante SSR; após mount, renderiza ícone real.
  React.useEffect(() => {
    setMounted(true)
  }, [])

  return (
    <header
      className={cn(
        'sticky top-0 z-20 h-16',
        'bg-surface-raised/80 backdrop-blur-md',
        'border-b border-border',
        'flex items-center gap-4 px-4 md:px-6',
      )}
    >
      {/* Hamburger (mobile only) */}
      <button
        onClick={onMobileMenuClick}
        className="lg:hidden size-9 rounded-md flex items-center justify-center text-ink-muted hover:bg-surface-sunken hover:text-ink"
        aria-label="Abrir menu"
      >
        <Menu className="size-5" />
      </button>

      {/* Esquerda: breadcrumbs + title */}
      <div className="flex-1 min-w-0">
        {breadcrumbs && breadcrumbs.length > 0 && (
          <nav
            aria-label="Breadcrumb"
            className="flex items-center gap-1.5 text-2xs text-ink-muted mb-0.5"
          >
            {breadcrumbs.map((b, i) => {
              const isLast = i === breadcrumbs.length - 1
              const inner = (
                <span
                  className={cn(
                    'truncate max-w-32',
                    isLast && 'text-ink font-medium',
                    b.href && !isLast && 'hover:text-ink',
                  )}
                >
                  {b.label}
                </span>
              )
              return (
                <span key={i} className="flex items-center gap-1.5">
                  {i > 0 && <span className="text-ink-subtle">/</span>}
                  {b.href && !isLast ? (
                    <Link href={b.href}>{inner}</Link>
                  ) : (
                    inner
                  )}
                </span>
              )
            })}
          </nav>
        )}
        <div className="flex items-baseline gap-2">
          <h1 className="text-lg font-semibold text-ink leading-tight truncate">
            {title}
          </h1>
          {subtitle && (
            <span className="text-sm text-ink-muted truncate hidden sm:inline">
              {subtitle}
            </span>
          )}
        </div>
      </div>

      {/* Centro: command palette trigger */}
      <button
        onClick={() => {
          // H5 fix: dispatch global ⌘K — CommandPalette escuta via window listener.
          window.dispatchEvent(new CustomEvent('rn:open-command-palette'))
        }}
        className={cn(
          'hidden md:flex items-center gap-2 h-9 px-3 rounded-md',
          'bg-surface-sunken border border-border',
          'text-sm text-ink-muted hover:border-border-strong hover:text-ink',
          'transition-all min-w-64',
        )}
        aria-label="Abrir paleta de comandos"
      >
        <Search className="size-3.5 shrink-0" />
        <span className="flex-1 text-left truncate">Buscar regra, alerta, CADOC…</span>
        <Kbd>⌘K</Kbd>
      </button>

      {/* Direita: actions + theme + notifications + user */}
      <div className="flex items-center gap-1.5">
        {actions}
        <Button
          variant="ghost"
          size="sm"
          onClick={toggle}
          aria-label={
            mounted && theme === 'dark'
              ? 'Mudar para tema claro'
              : 'Mudar para tema escuro'
          }
        >
          {/* H7 fix: renderização condicional só após mount */}
          {!mounted ? (
            <span className="size-4 block" aria-hidden />
          ) : theme === 'dark' ? (
            <Sun className="size-4" />
          ) : (
            <Moon className="size-4" />
          )}
        </Button>
        <Button variant="ghost" size="sm" aria-label="Notificações">
          <Bell className="size-4" />
        </Button>
        <div
          className="ml-1 size-8 rounded-full bg-gradient-to-br from-accent-500 to-accent-700 flex items-center justify-center text-white text-2xs font-semibold"
          aria-label="Avatar"
        >
          {userInitials || '··'}
        </div>
      </div>
    </header>
  )
}