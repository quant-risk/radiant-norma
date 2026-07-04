'use client'

/**
 * EmptyState — usado quando lista/conteúdo está vazio.
 *
 * Sempre oferece 3 elementos: ícone visual, copy clara, ação primária
 * (quando aplicável). Sem exceção: empty state sem CTA = mau UX.
 */
import { cn } from '@/lib/utils'

export interface EmptyStateProps {
  icon: React.ReactNode
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
}

export function EmptyState({
  icon,
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center',
        'py-16 px-6 rounded-lg border border-dashed border-border',
        'bg-surface-sunken/40',
        className,
      )}
    >
      <div
        className={cn(
          'size-12 rounded-full flex items-center justify-center mb-4',
          'bg-surface-raised border border-border text-ink-muted',
          '[&_svg]:size-6',
        )}
        aria-hidden
      >
        {icon}
      </div>
      <h3 className="text-md font-semibold text-ink mb-1">{title}</h3>
      {description && (
        <p className="text-sm text-ink-muted max-w-sm mb-4 leading-relaxed">
          {description}
        </p>
      )}
      {action}
    </div>
  )
}