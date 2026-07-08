'use client'

/**
 * SectionHeader — header editorial com eyebrow + título + descrição.
 *
 * Padrão: eyebrow (uppercase tracking) → título display (Fraunces)
 * → descrição muted. CTA opcional à direita (alinhamento vertical baseline).
 */
import * as React from 'react'
import { cn } from '@/lib/utils'

export interface SectionHeaderProps {
  eyebrow?: React.ReactNode
  title: React.ReactNode
  description?: React.ReactNode
  actions?: React.ReactNode
  align?: 'start' | 'between'
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

export function SectionHeader({
  eyebrow,
  title,
  description,
  actions,
  align = 'between',
  className,
  size = 'md',
}: SectionHeaderProps) {
  const titleSize =
    size === 'lg'
      ? 'text-3xl md:text-4xl'
      : size === 'sm'
        ? 'text-xl md:text-2xl'
        : 'text-2xl md:text-3xl'

  return (
    <header
      className={cn(
        'flex flex-col gap-3',
        align === 'between' && 'sm:flex-row sm:items-end sm:justify-between',
        className,
      )}
    >
      <div className="space-y-2 max-w-2xl">
        {eyebrow && (
          <div className="eyebrow font-mono text-accent-600 dark:text-accent-400 flex items-center gap-1.5">
            {eyebrow}
          </div>
        )}
        <h2
          className={cn(
            'font-serif font-medium text-ink tracking-tight',
            titleSize,
          )}
        >
          {title}
        </h2>
        {description && (
          <p className="text-sm text-ink-muted leading-relaxed">{description}</p>
        )}
      </div>
      {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
    </header>
  )
}