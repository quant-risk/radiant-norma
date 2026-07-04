'use client'

/**
 * EnviosFilterBar — filtros URL-driven para /envios.
 *
 * Sprint 8d: cada filtro atualiza a URL via `router.push(?key=value)`.
 * URL é a source of truth — copy/share/back-button funcionam naturalmente.
 *
 * Design: chips horizontais com X pra remover individual + botão "Limpar tudo".
 */

import * as React from 'react'
import { useRouter } from 'next/navigation'
import { Filter, X, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface EnviosFilterBarProps {
  currentFilters: {
    cadoc?: string
    status?: string
    period?: string
  }
  cadocOptions: Array<{ value: string; label: string }>
}

const STATUS_OPTIONS = [
  { value: 'accepted', label: 'Aprovados', tone: 'success' as const },
  { value: 'rejected', label: 'Rejeitados', tone: 'critical' as const },
  { value: 'pending', label: 'Pendentes', tone: 'warning' as const },
  { value: 'error', label: 'Erro', tone: 'critical' as const },
]

export function EnviosFilterBar({
  currentFilters,
  cadocOptions,
}: EnviosFilterBarProps) {
  const router = useRouter()

  function setFilter(key: string, value: string | undefined) {
    const params = new URLSearchParams()
    for (const [k, v] of Object.entries(currentFilters)) {
      if (v && k !== key) params.set(k, v)
    }
    if (value) params.set(key, value)
    const q = params.toString()
    router.push(q ? `/envios?${q}` : '/envios')
  }

  function clearAll() {
    router.push('/envios')
  }

  const hasFilters = Object.values(currentFilters).some(Boolean)

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="flex items-center gap-1.5">
        <Filter className="size-3.5 text-ink-subtle" />
        <span className="text-2xs uppercase tracking-wider font-semibold text-ink-subtle">
          Filtros
        </span>
      </div>

      {/* CADOC filter */}
      <FilterDropdown
        label="CADOC"
        value={currentFilters.cadoc}
        options={[
          { value: '', label: 'Todos' },
          ...cadocOptions.map((o) => ({ value: o.value, label: o.label })),
        ]}
        onChange={(v) => setFilter('cadoc', v || undefined)}
      />

      {/* Status filter */}
      <FilterDropdown
        label="Status"
        value={currentFilters.status}
        options={[
          { value: '', label: 'Todos' },
          ...STATUS_OPTIONS.map((o) => ({
            value: o.value,
            label: o.label,
            tone: o.tone,
          })),
        ]}
        onChange={(v) => setFilter('status', v || undefined)}
      />

      {/* Period filter */}
      <FilterDropdown
        label="Período"
        value={currentFilters.period}
        options={[
          { value: '', label: 'Todos' },
          ...generateRecentPeriods(12).map((p) => ({ value: p, label: p })),
        ]}
        onChange={(v) => setFilter('period', v || undefined)}
      />

      {hasFilters && (
        <button
          onClick={clearAll}
          className="inline-flex items-center gap-1 h-7 px-2.5 rounded-full text-2xs font-medium bg-surface-raised border border-border text-ink-muted hover:text-ink hover:border-border-strong transition-colors"
        >
          <X className="size-3" />
          Limpar tudo
        </button>
      )}
    </div>
  )
}

interface FilterDropdownProps {
  label: string
  value: string | undefined
  options: Array<{ value: string; label: string; tone?: 'success' | 'warning' | 'critical' }>
  onChange: (value: string) => void
}

function FilterDropdown({ label, value, options, onChange }: FilterDropdownProps) {
  const [open, setOpen] = React.useState(false)
  const ref = React.useRef<HTMLDivElement>(null)

  React.useEffect(() => {
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [open])

  const current = options.find((o) => o.value === value)
  const displayLabel = current?.label ?? 'Todos'

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        className={cn(
          'inline-flex items-center gap-1.5 h-7 px-3 rounded-full text-2xs font-medium transition-colors border',
          value
            ? 'bg-accent-50 dark:bg-accent-950 border-accent-300 dark:border-accent-700 text-accent-700 dark:text-accent-300'
            : 'bg-surface-raised border-border text-ink-muted hover:text-ink hover:border-border-strong',
        )}
      >
        <span className="text-ink-subtle uppercase tracking-wider text-2xs">
          {label}:
        </span>
        <span>{displayLabel}</span>
        <ChevronDown className="size-3" />
      </button>

      {open && (
        <div
          className={cn(
            'absolute left-0 top-full mt-1 z-30 w-48',
            'bg-surface-raised border border-border rounded-lg shadow-lg overflow-hidden',
            'animate-fade-in-fast',
          )}
        >
          {options.map((opt) => {
            const isActive = opt.value === (value ?? '')
            return (
              <button
                key={opt.value || 'all'}
                onClick={() => {
                  onChange(opt.value)
                  setOpen(false)
                }}
                className={cn(
                  'w-full text-left px-3 py-1.5 text-xs transition-colors',
                  isActive
                    ? 'bg-accent-50 dark:bg-accent-950 text-accent-700 dark:text-accent-300 font-medium'
                    : 'text-ink hover:bg-surface-sunken',
                )}
              >
                {opt.label}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

function generateRecentPeriods(n: number): string[] {
  const periods: string[] = []
  const now = new Date()
  for (let i = 0; i < n; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    periods.push(`${mm}/${d.getFullYear()}`)
  }
  return periods
}