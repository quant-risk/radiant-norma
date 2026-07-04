'use client'

/**
 * Kbd — keyboard shortcut visual. Usado em command palette hints e
 * docs de atalhos. Segue convenção macOS (⌘) com fallback Windows.
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
        'inline-flex items-center justify-center min-w-5 h-5 px-1.5',
        'rounded border border-border bg-surface-sunken',
        'text-2xs font-mono font-medium text-ink-muted',
        'shadow-[0_1px_0_1px_rgb(0_0_0/0.06)]',
        className,
      )}
    >
      {children}
    </kbd>
  )
}