'use client'

/**
 * EmptyState — versão editorial.
 *
 * Símbolo Fraunces + caption mono. Whitespace generoso. Sempre oferece
 * CTA (empty state sem CTA = mau UX).
 */
import { cn } from '@/lib/utils'

export interface EmptyStateProps {
  icon?: React.ReactNode
  symbol?: string
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
}

export function EmptyState({
  icon,
  symbol,
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center',
        'py-20 px-8 rounded-xl border border-dashed border-border',
        'bg-surface-sunken/30',
        'animate-fade-in',
        className,
      )}
    >
      {symbol && (
        <div
          className="font-serif text-6xl text-ink-subtle mb-4 leading-none tracking-tight"
          aria-hidden
        >
          {symbol}
        </div>
      )}
      {icon && (
        <div
          className={cn(
            'size-12 rounded-full flex items-center justify-center mb-5',
            'bg-surface-raised border border-border text-ink-muted',
            '[&_svg]:size-5',
          )}
          aria-hidden
        >
          {icon}
        </div>
      )}
      <h3 className="font-serif text-lg font-medium text-ink mb-1.5 tracking-tight">
        {title}
      </h3>
      {description && (
        <p className="text-sm text-ink-muted max-w-sm mb-5 leading-relaxed">
          {description}
        </p>
      )}
      {action}
    </div>
  )
}