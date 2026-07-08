'use client'

/**
 * Heatmap — matriz de intensidade editorial.
 *
 * Escala refinada: gradient violet→magenta→rose (sequential) ou
 * violet/magenta vs info (divergent). Theme-aware via useTheme().
 */
import * as React from 'react'
import { cn } from '@/lib/utils'
import { useTheme } from '@/components/theme-provider'

export interface HeatmapCell {
  row: string
  col: string
  value: number
}

export interface HeatmapProps {
  data: HeatmapCell[]
  rows: string[]
  cols: string[]
  max?: number
  variant?: 'sequential' | 'divergent'
  className?: string
}

function getIntensity(value: number, max: number, variant: 'sequential' | 'divergent'): number {
  if (max === 0) return 0
  if (variant === 'sequential') return Math.min(value / max, 1)
  return Math.min(Math.abs(value) / max, 1)
}

export function Heatmap({
  data,
  rows,
  cols,
  max,
  variant = 'sequential',
  className,
}: HeatmapProps) {
  const { theme } = useTheme()
  const isDark = theme === 'dark'

  const cellMap = React.useMemo(() => {
    const m = new Map<string, number>()
    data.forEach((c) => m.set(`${c.row}::${c.col}`, c.value))
    return m
  }, [data])

  const computedMax = max ?? Math.max(...data.map((d) => Math.abs(d.value)), 1)

  // Paletas por tema — em dark a célula 0 usa surface-sunken (noir),
  // e os estágios do gradient começam mais baixos para contraste.
  const palette = React.useMemo(() => {
    if (variant === 'sequential') {
      return isDark
        ? {
            zero: 'rgb(22 21 24)',         // noir border-subtle
            steps: [
              'rgb(46 35 78)',             // accent-950 dim
              'rgb(76 29 149)',            // accent-900
              'rgb(124 58 237)',           // accent-600
              'rgb(167 139 250)',          // accent-400
              'rgb(217 70 239)',           // magenta-500
            ],
            textOnLight: 'rgb(242 239 232)', // paper
            textOnDark: 'rgb(15 14 12)',
          }
        : {
            zero: 'rgb(240 237 230)',     // surface-sunken warm
            steps: [
              'rgb(237 233 254)',         // accent-100
              'rgb(221 214 254)',         // accent-200
              'rgb(196 181 253)',         // accent-300
              'rgb(167 139 250)',         // accent-400
              'rgb(124 58 237)',          // accent-600
            ],
            textOnLight: 'rgb(15 14 12)',
            textOnDark: 'rgb(255 255 255)',
          }
    }
    // divergent
    return isDark
      ? {
          zero: 'rgb(22 21 24)',
          posSteps: [
            'rgb(46 35 78)',
            'rgb(124 58 237)',
            'rgb(217 70 239)',
            'rgb(157 23 77)',
          ],
          negSteps: [
            'rgb(12 74 110)',           // info-900
            'rgb(14 165 233)',          // info-500
            'rgb(56 189 248)',          // info-400
          ],
          textOnLight: 'rgb(242 239 232)',
          textOnDark: 'rgb(15 14 12)',
        }
      : {
          zero: 'rgb(240 237 230)',
          posSteps: [
            'rgb(245 243 255)',
            'rgb(196 181 253)',
            'rgb(217 70 239)',
            'rgb(157 23 77)',
          ],
          negSteps: [
            'rgb(224 242 254)',
            'rgb(125 211 252)',
            'rgb(2 132 199)',
          ],
          textOnLight: 'rgb(15 14 12)',
          textOnDark: 'rgb(255 255 255)',
        }
  }, [variant, isDark])

  function cellColor(value: number): string {
    if (value === 0) return palette.zero
    const intensity = getIntensity(value, computedMax, variant)
    if (variant === 'sequential') {
      const steps = palette.steps ?? []
      if (intensity < 0.2) return steps[0]
      if (intensity < 0.4) return steps[1]
      if (intensity < 0.6) return steps[2]
      if (intensity < 0.8) return steps[3]
      return steps[4]
    }
    // divergent
    if (value > 0) {
      const steps = palette.posSteps ?? []
      if (intensity < 0.3) return steps[0]
      if (intensity < 0.6) return steps[1]
      if (intensity < 0.85) return steps[2]
      return steps[3]
    }
    const steps = palette.negSteps ?? []
    if (intensity < 0.3) return steps[0]
    if (intensity < 0.6) return steps[1]
    return steps[2]
  }

  return (
    <div className={cn('overflow-x-auto', className)}>
      <table className="border-separate" style={{ borderSpacing: '3px' }}>
        <thead>
          <tr>
            <th className="sticky left-0 bg-surface-raised z-10"></th>
            {cols.map((c) => (
              <th
                key={c}
                className="text-2xs font-mono font-medium text-ink-subtle uppercase tracking-wider px-2 py-1.5 text-left"
              >
                {c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r}>
              <th
                scope="row"
                className="sticky left-0 bg-surface-raised z-10 text-2xs font-mono font-medium text-ink-muted uppercase tracking-wider pr-3 py-1.5 text-right whitespace-nowrap"
              >
                {r}
              </th>
              {cols.map((c) => {
                const value = cellMap.get(`${r}::${c}`) ?? 0
                return (
                  <td key={c} className="p-0">
                    <div
                      className="size-9 rounded flex items-center justify-center text-2xs font-mono font-medium cursor-default hover:ring-2 hover:ring-accent-400/60 transition-all duration-180"
                      style={{
                        backgroundColor: cellColor(value),
                        color:
                          Math.abs(value) / computedMax > 0.55
                            ? palette.textOnDark
                            : palette.textOnLight,
                      }}
                      title={`${r} × ${c}: ${value}`}
                    >
                      {value !== 0 ? value : ''}
                    </div>
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}