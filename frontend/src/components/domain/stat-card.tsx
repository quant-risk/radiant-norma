'use client'

/**
 * StatCard — KPI editorial.
 *
 * Hierarquia:
 *   1. Eyebrow (eyebrow mono)
 *   2. Valor primário (mono, tabular nums, weight 500)
 *   3. Delta (pill com arrow + sparkline opcional)
 *   4. Help text (footnote serif italic)
 *
 * Padrão: 1 número, 1 delta, 1 sparkline. Nunca os 3 visuais juntos.
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
      <div className="rounded-lg border border-border bg-surface-raised p-6 shadow-xs">
        <div className="skeleton h-3 w-24 mb-4 rounded" />
        <div className="skeleton h-10 w-32 mb-3 rounded" />
        <div className="skeleton h-3 w-20 rounded" />
      </div>
    )
  }

  const valueColor: Record<NonNullable<StatCardProps['tone']>, string> = {
    neutral: 'text-ink',
    accent: 'text-gradient-accent',
    success: 'text-success-600 dark:text-success-400',
    warning: 'text-warning-600 dark:text-warning-400',
    critical: 'text-critical-600 dark:text-critical-400',
  }

  return (
    <div
      className={cn(
        'group relative rounded-lg border border-border bg-surface-raised p-6 shadow-xs',
        'transition-all duration-240 ease-out-expo hover:shadow-md hover:border-border-strong',
      )}
    >
      <div className="flex items-start justify-between gap-3 mb-5">
        <span className="text-2xs uppercase tracking-[0.14em] font-mono font-medium text-ink-subtle">
          {label}
        </span>
        {icon && (
          <span
            className={cn(
              'size-8 rounded-md flex items-center justify-center shrink-0',
              'bg-surface-sunken text-ink-muted ring-1 ring-inset ring-border-subtle',
              '[&_svg]:size-4',
            )}
          >
            {icon}
          </span>
        )}
      </div>

      <div className="flex items-baseline gap-1.5 mb-5">
        <span
          className={cn(
            'text-[2rem] leading-none font-medium nums tracking-tight font-serif',
            valueColor[tone],
          )}
        >
          {value}
        </span>
        {unit && (
          <span className="text-sm text-ink-muted font-mono">{unit}</span>
        )}
      </div>

      <div className="flex items-end justify-between gap-3 min-h-6">
        {delta ? (
          <div className="flex items-center gap-1.5 text-xs">
            {delta.direction === 'up' && (
              <ArrowUpRight
                className={cn(
                  'size-3.5',
                  delta.value >= 0
                    ? 'text-success-600 dark:text-success-400'
                    : 'text-critical-600 dark:text-critical-400',
                )}
                strokeWidth={2.25}
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
                strokeWidth={2.25}
              />
            )}
            {delta.direction === 'flat' && (
              <Minus className="size-3.5 text-ink-muted" strokeWidth={2.25} />
            )}
            <span
              className={cn(
                'font-medium nums tracking-tight',
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
              <span className="text-ink-subtle ml-0.5">{delta.period}</span>
            )}
          </div>
        ) : (
          <span />
        )}
        {sparkline && sparkline.length > 0 && (
          <Sparkline values={sparkline} tone={tone} className="w-20 h-7" />
        )}
      </div>

      {helpText && (
        <p className="text-xs text-ink-subtle mt-4 pt-4 border-t border-border-subtle font-mono">
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
  // useId: SSR-safe e único por instância — sem hydration mismatch nem
  // colisão quando múltiplos sparklines dividem a mesma página.
  const reactId = React.useId()
  const gradientId = `sparkline-${reactId.replace(/:/g, '')}`

  if (!values.length) return null

  const min = Math.min(...values)
  const max = Math.max(...values)
  const range = max - min || 1

  const w = 80
  const h = 28
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
          ? 'rgb(225 29 72)'
          : tone === 'accent'
            ? `url(#${gradientId})`
            : 'rgb(140 131 117)'

  return (
    <svg
      viewBox={`0 0 ${w} ${h}`}
      preserveAspectRatio="none"
      className={cn('overflow-visible', className)}
    >
      {tone === 'accent' && (
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stopColor="#7C3AED" />
            <stop offset="100%" stopColor="#D946EF" />
          </linearGradient>
        </defs>
      )}
      {/* Area fill muito sutil */}
      <polygon
        points={`0,${h} ${points} ${w},${h}`}
        fill={tone === 'accent' ? `url(#${gradientId})` : stroke}
        opacity="0.08"
      />
      <polyline
        points={points}
        fill="none"
        stroke={stroke}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* End dot */}
      {values.length > 0 && (
        <circle
          cx={(values.length - 1) * step}
          cy={h - ((values[values.length - 1] - min) / range) * h}
          r="2"
          fill={stroke}
        />
      )}
    </svg>
  )
}