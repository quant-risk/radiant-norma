'use client'

/**
 * Topbar — header sticky com 3 zonas:
 *   - Esquerda: breadcrumbs + page title
 *   - Centro: command palette trigger (⌘K)
 *   - Direita: theme toggle + notifications + user menu
 */
import * as React from 'react'
import { Search, Moon, Sun, Bell } from 'lucide-react'
import { useTheme } from '@/components/theme-provider'
import { Kbd } from '@/components/ui/kbd'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export interface TopbarProps {
  title: string
  subtitle?: string
  breadcrumbs?: Array<{ label: string; href?: string }>
  actions?: React.ReactNode
  onCommandPalette?: () => void
}

export function Topbar({
  title,
  subtitle,
  breadcrumbs,
  actions,
  onCommandPalette,
}: TopbarProps) {
  const { theme, toggle } = useTheme()

  return (
    <header
      className={cn(
        'sticky top-0 z-20 h-16',
        'bg-surface-raised/80 backdrop-blur-md',
        'border-b border-border',
        'flex items-center gap-4 px-6',
      )}
    >
      {/* Esquerda: breadcrumbs + title */}
      <div className="flex-1 min-w-0">
        {breadcrumbs && breadcrumbs.length > 0 && (
          <nav className="flex items-center gap-1.5 text-2xs text-ink-muted mb-0.5">
            {breadcrumbs.map((b, i) => (
              <span key={i} className="flex items-center gap-1.5">
                {i > 0 && <span className="text-ink-subtle">/</span>}
                <span
                  className={cn(
                    i === breadcrumbs.length - 1 && 'text-ink font-medium',
                  )}
                >
                  {b.label}
                </span>
              </span>
            ))}
          </nav>
        )}
        <div className="flex items-baseline gap-2">
          <h1 className="text-lg font-semibold text-ink leading-tight truncate">
            {title}
          </h1>
          {subtitle && (
            <span className="text-sm text-ink-muted truncate">
              {subtitle}
            </span>
          )}
        </div>
      </div>

      {/* Centro: command palette trigger */}
      {onCommandPalette && (
        <button
          onClick={onCommandPalette}
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
      )}

      {/* Direita: actions + theme + notifications + user */}
      <div className="flex items-center gap-1.5">
        {actions}
        <Button
          variant="ghost"
          size="sm"
          onClick={toggle}
          aria-label={
            theme === 'dark' ? 'Mudar para tema claro' : 'Mudar para tema escuro'
          }
        >
          {theme === 'dark' ? (
            <Sun className="size-4" />
          ) : (
            <Moon className="size-4" />
          )}
        </Button>
        <Button variant="ghost" size="sm" aria-label="Notificações">
          <Bell className="size-4" />
        </Button>
        <div className="ml-1 size-8 rounded-full bg-gradient-to-br from-accent-500 to-accent-700 flex items-center justify-center text-white text-xs font-semibold">
          HN
        </div>
      </div>
    </header>
  )
}