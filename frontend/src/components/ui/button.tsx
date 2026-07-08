'use client'

/**
 * Button — primitive refinado.
 *
 * Primary = gradient accent (violet→magenta) com sombra glow sutil.
 * Secondary = surface com border, hover sunken.
 * Ghost = transparente, hover sunken.
 * Outline = borda accent, hover fill accent.
 * Danger = critical fill.
 *
 * Touch targets: 36px (sm), 40px (md), 44px (lg).
 * Active state: scale-[0.98] pra dar feedback tátil elegante.
 *
 * Polimorfismo: `asChild` permite usar Button como wrapper de outro
 * elemento (ex: <Button asChild><Link href="...">...</Link></Button>).
 * Isso evita HTML inválido (<a><button></button></a>) e mantém
 * 1 único elemento focável. Implementação minimalista sem @radix-ui.
 */
import * as React from 'react'
import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

type Variant = 'primary' | 'secondary' | 'ghost' | 'outline' | 'danger'
type Size = 'sm' | 'md' | 'lg'

export interface ButtonProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  variant?: Variant
  size?: Size
  loading?: boolean
  leftIcon?: React.ReactNode
  rightIcon?: React.ReactNode
  fullWidth?: boolean
  asChild?: boolean
  children?: React.ReactNode
}

const variantClasses: Record<Variant, string> = {
  primary:
    'bg-gradient-to-br from-accent-600 to-accent-700 text-white shadow-sm hover:shadow-glow-accent-sm hover:from-accent-500 hover:to-magenta-500 active:scale-[0.98]',
  secondary:
    'bg-surface-raised text-ink border border-border hover:bg-surface-sunken hover:border-border-strong active:scale-[0.98]',
  ghost:
    'bg-transparent text-ink-muted hover:bg-surface-sunken hover:text-ink active:scale-[0.98]',
  outline:
    'bg-transparent text-accent-700 dark:text-accent-300 border border-accent-200 dark:border-accent-800 hover:bg-accent-50 dark:hover:bg-accent-950 hover:border-accent-300 dark:hover:border-accent-700 active:scale-[0.98]',
  danger:
    'bg-critical-600 text-white shadow-sm hover:bg-critical-700 hover:shadow-glow-critical active:scale-[0.98]',
}

const sizeClasses: Record<Size, string> = {
  sm: 'h-9 px-3.5 text-xs gap-1.5 rounded-md',
  md: 'h-10 px-4 text-sm gap-2 rounded-md',
  lg: 'h-12 px-6 text-base gap-2.5 rounded-lg',
}

const baseClasses =
  'relative inline-flex items-center justify-center font-medium tracking-tight ' +
  'transition-all duration-180 ease-out-expo ' +
  'disabled:cursor-not-allowed disabled:opacity-60 ' +
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-400 focus-visible:ring-offset-2 focus-visible:ring-offset-surface'

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
      asChild = false,
      children,
      ...props
    },
    ref,
  ) => {
    // Clases finais — aplicadas ao <button> OU injetadas no child via Slot
    const composedClassName = cn(
      baseClasses,
      variantClasses[variant],
      sizeClasses[size],
      fullWidth && 'w-full',
      className,
    )

    // Conteúdo visual (loading/spinner + ícones + texto)
    const inner = (
      <>
        {loading ? (
          <Loader2 className="size-4 animate-spin-slow" aria-hidden />
        ) : leftIcon ? (
          <span className="shrink-0 [&_svg]:size-4">{leftIcon}</span>
        ) : null}
        {children && <span className="truncate">{children}</span>}
        {!loading && rightIcon && (
          <span className="shrink-0 [&_svg]:size-4">{rightIcon}</span>
        )}
      </>
    )

    // asChild: clona o único filho e injeta classe + ref + handlers nele.
    // Preserva a semântica original (Link vira link, button vira button).
    if (asChild && React.isValidElement(children)) {
      const child = children as React.ReactElement<{
        className?: string
        onClick?: React.MouseEventHandler
        tabIndex?: number
      }>
      const childProps = child.props
      return React.cloneElement(child, {
        ...props,
        className: cn(composedClassName, childProps.className),
        onClick: (e: React.MouseEvent) => {
          if (loading || disabled) {
            e.preventDefault()
            return
          }
          childProps.onClick?.(e)
          ;(props as { onClick?: React.MouseEventHandler }).onClick?.(e)
        },
        tabIndex: childProps.tabIndex ?? 0,
      })
    }

    return (
      <button
        ref={ref}
        disabled={disabled || loading}
        className={composedClassName}
        {...props}
      >
        {inner}
      </button>
    )
  },
)

Button.displayName = 'Button'