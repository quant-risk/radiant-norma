'use client'

/**
 * Heatmap — matriz de intensidade (CADOCs × período).
 *
 * Cores: escala sequencial (success-50 → success-700) ou divergent
 * (success-50 ↔ critical-500) com ponto neutro em info-100.
 * Tooltip on hover.
 */
import * as React from 'react'
import { cn } from '@/lib/utils'

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
  // divergent: assume value/max em [-1, 1]
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
  const cellMap = React.useMemo(() => {
    const m = new Map<string, number>()
    data.forEach((c) => m.set(`${c.row}::${c.col}`, c.value))
    return m
  }, [data])

  const computedMax = max ?? Math.max(...data.map((d) => Math.abs(d.value)), 1)

  function cellColor(value: number): string {
    if (value === 0) return 'rgb(241 245 249)' // slate-100
    const intensity = getIntensity(value, computedMax, variant)
    if (variant === 'sequential') {
      if (intensity < 0.2) return 'rgb(220 252 231)' // success-100
      if (intensity < 0.4) return 'rgb(187 247 208)' // success-200
      if (intensity < 0.6) return 'rgb(134 239 172)' // success-300
      if (intensity < 0.8) return 'rgb(74 222 128)' // success-400
      return 'rgb(22 163 74)' // success-600
    }
    // divergent
    if (value > 0) {
      if (intensity < 0.3) return 'rgb(254 215 170)' // warning-200
      if (intensity < 0.6) return 'rgb(251 146 60)' // warning-400
      return 'rgb(220 38 38)' // critical-600
    } else {
      if (intensity < 0.3) return 'rgb(186 230 253)' // info-200
      if (intensity < 0.6) return 'rgb(56 189 248)' // info-400
      return 'rgb(2 132 199)' // info-600
    }
  }

  return (
    <div className={cn('overflow-x-auto', className)}>
      <table className="border-separate border-spacing-1">
        <thead>
          <tr>
            <th className="sticky left-0 bg-surface-raised z-10"></th>
            {cols.map((c) => (
              <th
                key={c}
                className="text-2xs font-medium text-ink-muted uppercase tracking-wider px-2 py-1 text-left"
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
                className="sticky left-0 bg-surface-raised z-10 text-2xs font-medium text-ink-muted uppercase tracking-wider pr-3 py-1 text-right whitespace-nowrap"
              >
                {r}
              </th>
              {cols.map((c) => {
                const value = cellMap.get(`${r}::${c}`) ?? 0
                return (
                  <td key={c} className="p-0">
                    <div
                      className="size-9 rounded flex items-center justify-center text-2xs font-mono font-medium cursor-default hover:ring-2 hover:ring-accent-400 transition-shadow"
                      style={{
                        backgroundColor: cellColor(value),
                        color:
                          Math.abs(value) / computedMax > 0.5
                            ? 'white'
                            : 'rgb(15 23 42)',
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