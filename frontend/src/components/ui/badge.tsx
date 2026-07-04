'use client'

/**
 * Badge — primitive semântico.
 *
 * 5 tons: neutral, accent, success, warning, critical, info.
 * 2 estilos: solid (alto contraste) e soft (background tinted).
 *
 * Regra: sempre par (ícone + texto) em badges críticos pra acessibilidade
 * (cor não pode ser o único sinal — WCAG 1.4.1).
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
}

const solidClasses: Record<Tone, string> = {
  neutral: 'bg-slate-700 text-white',
  accent: 'bg-accent-600 text-white',
  success: 'bg-success-600 text-white',
  warning: 'bg-warning-600 text-white',
  critical: 'bg-critical-600 text-white',
  info: 'bg-info-600 text-white',
}

const softClasses: Record<Tone, string> = {
  neutral: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300',
  accent: 'bg-accent-50 text-accent-700 dark:bg-accent-950 dark:text-accent-300',
  success: 'bg-success-50 text-success-700 dark:bg-success-950 dark:text-success-300',
  warning: 'bg-warning-50 text-warning-700 dark:bg-warning-950 dark:text-warning-300',
  critical: 'bg-critical-50 text-critical-700 dark:bg-critical-950 dark:text-critical-300',
  info: 'bg-info-50 text-info-700 dark:bg-info-950 dark:text-info-300',
}

const outlineClasses: Record<Tone, string> = {
  neutral: 'border border-slate-300 text-slate-700 dark:border-slate-700 dark:text-slate-300',
  accent: 'border border-accent-300 text-accent-700 dark:border-accent-700 dark:text-accent-300',
  success: 'border border-success-300 text-success-700 dark:border-success-700 dark:text-success-300',
  warning: 'border border-warning-300 text-warning-700 dark:border-warning-700 dark:text-warning-300',
  critical: 'border border-critical-300 text-critical-700 dark:border-critical-700 dark:text-critical-300',
  info: 'border border-info-300 text-info-700 dark:border-info-700 dark:text-info-300',
}

const dotClasses: Record<Tone, string> = {
  neutral: 'bg-slate-500',
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
  className,
  children,
  ...props
}: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-2xs font-medium uppercase tracking-wide whitespace-nowrap',
        variant === 'solid' && solidClasses[tone],
        variant === 'soft' && softClasses[tone],
        variant === 'outline' && outlineClasses[tone],
        className,
      )}
      {...props}
    >
      {dot && (
        <span
          className={cn('size-1.5 rounded-full', dotClasses[tone])}
          aria-hidden
        />
      )}
      {icon && <span className="[&_svg]:size-3">{icon}</span>}
      {children}
    </span>
  )
}