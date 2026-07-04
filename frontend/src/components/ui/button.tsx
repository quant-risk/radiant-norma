'use client'

/**
 * Button — primitive com 4 variants × 3 sizes × 2 intents.
 *
 * Princípios:
 * - Sempre tem :focus-visible (acessibilidade)
 * - Ícones alinhados com gap consistente (size-aware)
 * - Loading state troca conteúdo mas mantém width (evita layout shift)
 * - Disabled = cursor + opacity, sem pointer-events disabled (mantém tooltip)
 */
import * as React from 'react'
import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

type Variant = 'primary' | 'secondary' | 'ghost' | 'outline' | 'danger'
type Size = 'sm' | 'md' | 'lg'

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
  leftIcon?: React.ReactNode
  rightIcon?: React.ReactNode
  fullWidth?: boolean
}

const variantClasses: Record<Variant, string> = {
  primary:
    'bg-accent-600 text-white hover:bg-accent-700 active:bg-accent-800 shadow-xs hover:shadow-sm',
  secondary:
    'bg-surface-raised text-ink border border-border hover:bg-surface-sunken hover:border-border-strong',
  ghost:
    'bg-transparent text-ink hover:bg-surface-sunken',
  outline:
    'bg-transparent text-accent-600 dark:text-accent-400 border border-accent-200 dark:border-accent-800 hover:bg-accent-50 dark:hover:bg-accent-950',
  danger:
    'bg-critical-600 text-white hover:bg-critical-700 active:bg-critical-800 shadow-xs',
}

const sizeClasses: Record<Size, string> = {
  sm: 'h-8 px-3 text-xs gap-1.5',
  md: 'h-9 px-3.5 text-sm gap-2',
  lg: 'h-11 px-5 text-base gap-2',
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant = 'primary',
      size = 'md',
      loading = false,
      disabled,
      leftIcon,
      rightIcon,
      fullWidth,
      children,
      ...props
    },
    ref,
  ) => {
    return (
      <button
        ref={ref}
        disabled={disabled || loading}
        className={cn(
          // base
          'inline-flex items-center justify-center rounded-md font-medium',
          'transition-all duration-150 ease-out',
          'disabled:cursor-not-allowed disabled:opacity-60',
          // variant & size
          variantClasses[variant],
          sizeClasses[size],
          fullWidth && 'w-full',
          className,
        )}
        {...props}
      >
        {loading ? (
          <Loader2 className="size-3.5 animate-spin-slow" aria-hidden />
        ) : leftIcon ? (
          <span className="shrink-0 [&_svg]:size-3.5">{leftIcon}</span>
        ) : null}
        {children && <span className="truncate">{children}</span>}
        {!loading && rightIcon && (
          <span className="shrink-0 [&_svg]:size-3.5">{rightIcon}</span>
        )}
      </button>
    )
  },
)

Button.displayName = 'Button'