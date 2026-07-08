'use client'

/**
 * Divider — hairline editorial separator.
 * Inspirado em Stripe Docs: linha fina com label opcional centralizada.
 */
import * as React from 'react'
import { cn } from '@/lib/utils'

export interface DividerProps extends React.HTMLAttributes<HTMLDivElement> {
  label?: string
  orientation?: 'horizontal' | 'vertical'
}

export function Divider({
  label,
  orientation = 'horizontal',
  className,
  ...props
}: DividerProps) {
  if (orientation === 'vertical') {
    return (
      <div
        role="separator"
        aria-orientation="vertical"
        className={cn('w-px h-full bg-border', className)}
        {...props}
      />
    )
  }

  if (!label) {
    return (
      <div
        role="separator"
        className={cn('h-px w-full bg-border', className)}
        {...props}
      />
    )
  }

  return (
    <div
      role="separator"
      className={cn('flex items-center gap-3', className)}
      {...props}
    >
      <span className="h-px flex-1 bg-border" />
      <span className="text-2xs uppercase tracking-[0.18em] font-medium text-ink-subtle">
        {label}
      </span>
      <span className="h-px flex-1 bg-border" />
    </div>
  )
}