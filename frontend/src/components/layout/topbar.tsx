'use client'

/**
 * Topbar — header sticky refinado.
 *
 * Estrutura: hamburger (mobile) + breadcrumb/title à esquerda · search
 * trigger central · actions + theme + avatar à direita.
 *
 * Visual: glass header (frosted) com hairline border-bottom.
 */

import * as React from 'react'
import Link from 'next/link'
import type { Route } from 'next'
import { Search, Moon, Sun, Bell, Menu, ChevronDown } from 'lucide-react'
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

  React.useEffect(() => {
    setMounted(true)
  }, [])

  return (
    <header
      className={cn(
        'sticky top-0 z-20 h-[68px]',
        'glass border-b border-border',
        'flex items-center gap-3 px-4 md:px-6 lg:px-8',
      )}
    >
      {/* Hamburger (mobile) */}
      <button
        onClick={onMobileMenuClick}
        className="lg:hidden size-9 rounded-md flex items-center justify-center text-ink-muted hover:bg-surface-sunken hover:text-ink transition-colors"
        aria-label="Abrir menu"
      >
        <Menu className="size-5" />
      </button>

      {/* Esquerda: breadcrumbs + title */}
      <div className="flex-1 min-w-0">
        {breadcrumbs && breadcrumbs.length > 0 && (
          <nav
            aria-label="Breadcrumb"
            className="flex items-center gap-1.5 text-2xs text-ink-subtle mb-1 font-mono"
          >
            {breadcrumbs.map((b, i) => {
              const isLast = i === breadcrumbs.length - 1
              const inner = (
                <span
                  className={cn(
                    'truncate max-w-[160px]',
                    isLast && 'text-ink-muted',
                    b.href && !isLast && 'hover:text-ink-muted',
                  )}
                >
                  {b.label}
                </span>
              )
              return (
                <span key={i} className="flex items-center gap-1.5">
                  {i > 0 && <span className="text-ink-subtle/60">/</span>}
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
        <div className="flex items-baseline gap-2.5 min-w-0">
          <h1 className="font-serif text-xl font-medium text-ink leading-tight truncate tracking-tight">
            {title}
          </h1>
          {subtitle && (
            <span className="text-xs text-ink-subtle truncate hidden sm:inline font-mono">
              · {subtitle}
            </span>
          )}
        </div>
      </div>

      {/* Centro: command palette trigger */}
      <button
        onClick={() => {
          window.dispatchEvent(new CustomEvent('rn:open-command-palette'))
        }}
        className={cn(
          'hidden md:flex items-center gap-2.5 h-9 px-3 rounded-md',
          'bg-surface-sunken/80 border border-border-subtle',
          'text-sm text-ink-subtle hover:border-border-strong hover:text-ink-muted',
          'transition-all duration-180 min-w-[280px]',
          'group',
        )}
        aria-label="Abrir paleta de comandos"
      >
        <Search className="size-3.5 shrink-0 text-ink-subtle group-hover:text-ink-muted" />
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
          className="size-9 px-0"
          aria-label={
            mounted && theme === 'dark'
              ? 'Mudar para tema claro'
              : 'Mudar para tema escuro'
          }
        >
          {!mounted ? (
            <span className="size-4 block" aria-hidden />
          ) : theme === 'dark' ? (
            <Sun className="size-4" />
          ) : (
            <Moon className="size-4" />
          )}
        </Button>

        <Button
          variant="ghost"
          size="sm"
          className="size-9 px-0 relative"
          aria-label="Notificações"
        >
          <Bell className="size-4" />
          <span className="absolute top-2 right-2 size-1.5 rounded-full bg-accent-500" />
        </Button>

        <button
          type="button"
          className="ml-1 size-9 rounded-md bg-gradient-to-br from-accent-600 to-magenta-500 flex items-center justify-center text-white text-xs font-medium font-serif shadow-glow-accent-sm hover:scale-105 active:scale-100 transition-transform focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-400 focus-visible:ring-offset-2 focus-visible:ring-offset-surface"
          aria-label={`Conta de ${userInitials || 'usuário'}`}
        >
          {userInitials || '··'}
        </button>
      </div>
    </header>
  )
}