'use client'

/**
 * Card — primitive base com variants.
 *
 * Filosofia: cards devem ser neutros por default (raised surface).
 * Hover/focus state sutil quando interativo (cursor pointer + ring).
 */
import * as React from 'react'
import { cn } from '@/lib/utils'

type CardVariant = 'default' | 'raised' | 'ghost' | 'outlined'
type CardPadding = 'none' | 'sm' | 'md' | 'lg'

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: CardVariant
  padding?: CardPadding
  interactive?: boolean
}

const variantClasses: Record<CardVariant, string> = {
  default: 'bg-surface-raised border border-border shadow-xs',
  raised: 'bg-surface-raised border border-border shadow-sm',
  ghost: 'bg-transparent border border-transparent',
  outlined: 'bg-transparent border-2 border-border-strong',
}

const paddingClasses: Record<CardPadding, string> = {
  none: '',
  sm: 'p-4',
  md: 'p-6',
  lg: 'p-8',
}

export const Card = React.forwardRef<HTMLDivElement, CardProps>(
  (
    {
      className,
      variant = 'default',
      padding = 'md',
      interactive,
      ...props
    },
    ref,
  ) => {
    return (
      <div
        ref={ref}
        className={cn(
          'rounded-lg transition-all duration-150',
          variantClasses[variant],
          paddingClasses[padding],
          interactive &&
            'cursor-pointer hover:border-border-strong hover:shadow-md hover:-translate-y-px active:translate-y-0',
          className,
        )}
        {...props}
      />
    )
  },
)

Card.displayName = 'Card'

export const CardHeader = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) => (
  <div
    className={cn('flex items-start justify-between gap-3 mb-3', className)}
    {...props}
  />
)

export const CardTitle = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) => (
  <h3
    className={cn(
      'text-md font-semibold text-ink leading-tight tracking-tight',
      className,
    )}
    {...props}
  />
)

export const CardDescription = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) => (
  <p
    className={cn('text-sm text-ink-muted leading-relaxed', className)}
    {...props}
  />
)