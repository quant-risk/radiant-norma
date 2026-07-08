'use client'

/**
 * Badge — primitive semântico refinado.
 *
 * 5 tons × 3 estilos (solid, soft, outline).
 * Solid usa gradiente accent em vez de cor pura (mais premium).
 *
 * Regra: sempre par (ícone + texto OU dot + texto) em badges críticos
 * pra acessibilidade (cor não pode ser o único sinal — WCAG 1.4.1).
 */
import * as React from 'react'
import { cn } from '@/lib/utils'

type Tone = 'neutral' | 'accent' | 'success' | 'warning' | 'critical' | 'info'
type Style = 'solid' | 'soft' | 'outline'

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  tone?: Tone
  variant?: Style
  icon?: React.ReactNode
  dot?: boolean
  size?: 'sm' | 'md'
}

const solidClasses: Record<Tone, string> = {
  neutral: 'bg-ink text-ink-inverse',
  accent: 'bg-gradient-to-br from-accent-600 to-accent-700 text-white shadow-sm',
  success: 'bg-success-600 text-white',
  warning: 'bg-warning-600 text-white',
  critical: 'bg-critical-600 text-white',
  info: 'bg-info-600 text-white',
}

const softClasses: Record<Tone, string> = {
  neutral: 'bg-surface-sunken text-ink-muted ring-1 ring-inset ring-border',
  accent: 'bg-accent-50 text-accent-700 dark:bg-accent-950/40 dark:text-accent-300 ring-1 ring-inset ring-accent-200/60 dark:ring-accent-800/40',
  success: 'bg-success-50 text-success-700 dark:bg-success-950/40 dark:text-success-300 ring-1 ring-inset ring-success-200/60 dark:ring-success-800/40',
  warning: 'bg-warning-50 text-warning-700 dark:bg-warning-950/40 dark:text-warning-300 ring-1 ring-inset ring-warning-200/60 dark:ring-warning-800/40',
  critical: 'bg-critical-50 text-critical-700 dark:bg-critical-950/40 dark:text-critical-300 ring-1 ring-inset ring-critical-200/60 dark:ring-critical-800/40',
  info: 'bg-info-50 text-info-700 dark:bg-info-950/40 dark:text-info-300 ring-1 ring-inset ring-info-200/60 dark:ring-info-800/40',
}

const outlineClasses: Record<Tone, string> = {
  neutral: 'border border-border-strong text-ink-muted',
  accent: 'border border-accent-300 dark:border-accent-700 text-accent-700 dark:text-accent-300',
  success: 'border border-success-300 dark:border-success-700 text-success-700 dark:text-success-300',
  warning: 'border border-warning-300 dark:border-warning-700 text-warning-700 dark:text-warning-300',
  critical: 'border border-critical-300 dark:border-critical-700 text-critical-700 dark:text-critical-300',
  info: 'border border-info-300 dark:border-info-700 text-info-700 dark:text-info-300',
}

const dotClasses: Record<Tone, string> = {
  neutral: 'bg-ink-muted',
  accent: 'bg-accent-500',
  success: 'bg-success-500',
  warning: 'bg-warning-500',
  critical: 'bg-critical-500',
  info: 'bg-info-500',
}

export function Badge({
  tone = 'neutral',
  variant = 'soft',
  icon,
  dot,
  size = 'sm',
  className,
  children,
  ...props
}: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full font-medium tracking-wide whitespace-nowrap',
        size === 'sm'
          ? 'px-2 py-0.5 text-2xs uppercase tracking-[0.08em]'
          : 'px-2.5 py-1 text-xs',
        variant === 'solid' && solidClasses[tone],
        variant === 'soft' && softClasses[tone],
        variant === 'outline' && outlineClasses[tone],
        className,
      )}
      {...props}
    >
      {dot && (
        <span
          className={cn(
            'size-1.5 rounded-full shrink-0',
            dotClasses[tone],
          )}
          aria-hidden
        />
      )}
      {icon && <span className="shrink-0 [&_svg]:size-3">{icon}</span>}
      {children}
    </span>
  )
}