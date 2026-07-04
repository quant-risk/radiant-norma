'use client'

/**
 * StatCard — KPI card com 1 número primário + delta vs período anterior +
 * sparkline trend opcional.
 *
 * Padrão Linear/Stripe: 1 número grande, 1 delta contextual, 1 visual.
 * Nunca 3 visuais no mesmo card.
 */
import * as React from 'react'
import { ArrowUpRight, ArrowDownRight, Minus } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface StatCardProps {
  label: string
  value: string | number
  unit?: string
  delta?: {
    value: number
    direction: 'up' | 'down' | 'flat'
    period?: string
  }
  sparkline?: number[]
  tone?: 'neutral' | 'accent' | 'success' | 'warning' | 'critical'
  icon?: React.ReactNode
  loading?: boolean
  helpText?: string
}

export function StatCard({
  label,
  value,
  unit,
  delta,
  sparkline,
  tone = 'neutral',
  icon,
  loading,
  helpText,
}: StatCardProps) {
  if (loading) {
    return (
      <div className="rounded-lg border border-border bg-surface-raised p-5 shadow-xs">
        <div className="skeleton h-3 w-20 mb-3 rounded" />
        <div className="skeleton h-8 w-24 mb-2 rounded" />
        <div className="skeleton h-3 w-16 rounded" />
      </div>
    )
  }

  const toneClasses: Record<NonNullable<StatCardProps['tone']>, string> = {
    neutral: 'text-ink',
    accent: 'text-accent-600 dark:text-accent-400',
    success: 'text-success-600 dark:text-success-400',
    warning: 'text-warning-600 dark:text-warning-400',
    critical: 'text-critical-600 dark:text-critical-400',
  }

  return (
    <div
      className={cn(
        'group relative rounded-lg border border-border bg-surface-raised p-5 shadow-xs',
        'transition-all duration-150 hover:shadow-md hover:border-border-strong',
      )}
    >
      <div className="flex items-start justify-between gap-3 mb-3">
        <span className="text-xs font-medium text-ink-muted uppercase tracking-wider">
          {label}
        </span>
        {icon && (
          <span
            className={cn(
              'size-7 rounded-md flex items-center justify-center',
              tone === 'neutral'
                ? 'bg-surface-sunken text-ink-muted'
                : tone === 'accent'
                  ? 'bg-accent-50 text-accent-600 dark:bg-accent-950 dark:text-accent-400'
                  : tone === 'success'
                    ? 'bg-success-50 text-success-600 dark:bg-success-950 dark:text-success-400'
                    : tone === 'warning'
                      ? 'bg-warning-50 text-warning-600 dark:bg-warning-950 dark:text-warning-400'
                      : 'bg-critical-50 text-critical-600 dark:bg-critical-950 dark:text-critical-400',
              '[&_svg]:size-3.5',
            )}
          >
            {icon}
          </span>
        )}
      </div>

      <div className="flex items-baseline gap-1.5 mb-2">
        <span className={cn('text-3xl font-semibold nums tracking-tight', toneClasses[tone])}>
          {value}
        </span>
        {unit && <span className="text-sm text-ink-muted">{unit}</span>}
      </div>

      <div className="flex items-center justify-between gap-3 min-h-5">
        {delta ? (
          <div className="flex items-center gap-1 text-xs">
            {delta.direction === 'up' && (
              <ArrowUpRight
                className={cn(
                  'size-3.5',
                  delta.value >= 0
                    ? 'text-success-600 dark:text-success-400'
                    : 'text-critical-600 dark:text-critical-400',
                )}
              />
            )}
            {delta.direction === 'down' && (
              <ArrowDownRight
                className={cn(
                  'size-3.5',
                  delta.value >= 0
                    ? 'text-success-600 dark:text-success-400'
                    : 'text-critical-600 dark:text-critical-400',
                )}
              />
            )}
            {delta.direction === 'flat' && (
              <Minus className="size-3.5 text-ink-muted" />
            )}
            <span
              className={cn(
                'font-medium nums',
                delta.direction === 'flat'
                  ? 'text-ink-muted'
                  : delta.value >= 0
                    ? 'text-success-700 dark:text-success-300'
                    : 'text-critical-700 dark:text-critical-300',
              )}
            >
              {delta.value >= 0 ? '+' : ''}
              {delta.value.toFixed(1)}%
            </span>
            {delta.period && (
              <span className="text-ink-subtle">{delta.period}</span>
            )}
          </div>
        ) : (
          <span />
        )}
        {sparkline && sparkline.length > 0 && (
          <Sparkline values={sparkline} tone={tone} className="w-20 h-6" />
        )}
      </div>

      {helpText && (
        <p className="text-xs text-ink-subtle mt-3 pt-3 border-t border-border-subtle">
          {helpText}
        </p>
      )}
    </div>
  )
}

/* ───────── Sparkline ───────── */

function Sparkline({
  values,
  tone = 'neutral',
  className,
}: {
  values: number[]
  tone?: StatCardProps['tone']
  className?: string
}) {
  if (!values.length) return null

  const min = Math.min(...values)
  const max = Math.max(...values)
  const range = max - min || 1

  const w = 80
  const h = 24
  const step = w / (values.length - 1 || 1)

  const points = values
    .map((v, i) => {
      const x = i * step
      const y = h - ((v - min) / range) * h
      return `${x},${y}`
    })
    .join(' ')

  const stroke =
    tone === 'success'
      ? 'rgb(16 185 129)'
      : tone === 'warning'
        ? 'rgb(245 158 11)'
        : tone === 'critical'
          ? 'rgb(244 63 94)'
          : tone === 'accent'
            ? 'rgb(124 58 237)'
            : 'rgb(100 116 139)'

  return (
    <svg
      viewBox={`0 0 ${w} ${h}`}
      preserveAspectRatio="none"
      className={cn('overflow-visible', className)}
    >
      <polyline
        points={points}
        fill="none"
        stroke={stroke}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}