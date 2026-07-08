'use client'

/**
 * Kbd — keyboard shortcut visual refinado.
 */
import { cn } from '@/lib/utils'

export function Kbd({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <kbd
      className={cn(
        'inline-flex items-center justify-center min-w-[20px] h-5 px-1.5',
        'rounded border border-border bg-surface-raised',
        'text-2xs font-mono font-medium text-ink-muted',
        'shadow-[inset_0_-1px_0_rgb(0_0_0/0.06),0_1px_0_rgb(255_255_255/0.5)]',
        'dark:shadow-[inset_0_-1px_0_rgb(255_255_255/0.06),0_1px_0_rgb(0_0_0/0.3)]',
        className,
      )}
    >
      {children}
    </kbd>
  )
}