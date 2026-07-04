'use client'

/**
 * Separator — horizontal/vertical com 3 intensidades.
 */
import { cn } from '@/lib/utils'

export function Separator({
  className,
  orientation = 'horizontal',
}: {
  className?: string
  orientation?: 'horizontal' | 'vertical'
}) {
  return (
    <div
      role="separator"
      aria-orientation={orientation}
      className={cn(
        'bg-border',
        orientation === 'horizontal' ? 'h-px w-full' : 'w-px h-full',
        className,
      )}
    />
  )
}