'use client'

/**
 * Card — primitive base refinado.
 *
 * 4 variants: default (raised + hairline border), raised (sombra sutil),
 * ghost (sem bg), outlined (border 2px). Padding consistente.
 *
 * Hierarquia visual:
 *   default = uso geral (90% dos casos)
 *   raised = destaque (cards de destaque em listas)
 *   outlined = ênfase sem peso visual (estados selecionados)
 *   ghost = sobreposição em cards existentes (sem peso)
 */
import * as React from 'react'
import { cn } from '@/lib/utils'

type CardVariant = 'default' | 'raised' | 'ghost' | 'outlined' | 'glass'
type CardPadding = 'none' | 'sm' | 'md' | 'lg' | 'xl'

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: CardVariant
  padding?: CardPadding
  interactive?: boolean
}

const variantClasses: Record<CardVariant, string> = {
  default:
    'bg-surface-raised border border-border shadow-xs',
  raised:
    'bg-surface-raised border border-border shadow-sm',
  ghost:
    'bg-transparent border border-transparent',
  outlined:
    'bg-transparent border-2 border-border-strong',
  glass:
    'glass border border-border-subtle shadow-sm',
}

const paddingClasses: Record<CardPadding, string> = {
  none: '',
  sm: 'p-4',
  md: 'p-5 md:p-6',
  lg: 'p-6 md:p-8',
  xl: 'p-8 md:p-10',
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
          'rounded-lg transition-all duration-180 ease-out-expo',
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
    className={cn('flex items-start justify-between gap-3 mb-4', className)}
    {...props}
  />
)

export const CardTitle = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) => (
  <h3
    className={cn(
      'font-serif text-xl font-medium text-ink leading-tight tracking-tight',
      className,
    )}
    {...props}
  />
)

export const CardEyebrow = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) => (
  <p
    className={cn(
      'eyebrow font-mono text-accent-600 dark:text-accent-400',
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
    className={cn('text-sm text-ink-muted leading-relaxed mt-1.5', className)}
    {...props}
  />
)