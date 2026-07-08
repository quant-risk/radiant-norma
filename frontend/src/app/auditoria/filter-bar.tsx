'use client'

/**
 * AuditFilterBar — filtros URL-driven para /auditoria.
 */

import { useRouter } from 'next/navigation'
import { Filter, X, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import * as React from 'react'

export interface AuditFilterBarProps {
  currentFilters: {
    action?: string
    if_id?: string
  }
}

const ACTION_OPTIONS = [
  { value: 'envio.approved', label: 'Envio aprovado' },
  { value: 'envio.rejected', label: 'Envio rejeitado' },
  { value: 'envio.created', label: 'Envio criado' },
  { value: 'radar.detected', label: 'Alerta detectado' },
  { value: 'radar.resolved', label: 'Alerta resolvido' },
  { value: 'rule.enabled', label: 'Regra habilitada' },
  { value: 'rule.disabled', label: 'Regra desabilitada' },
  { value: 'schema.synced', label: 'Schema sincronizado' },
  { value: 'auth.login', label: 'Login' },
  { value: 'sta.submit', label: 'STA Submit' },
]

export function AuditFilterBar({ currentFilters }: AuditFilterBarProps) {
  const router = useRouter()

  function setFilter(key: string, value: string | undefined) {
    const params = new URLSearchParams()
    for (const [k, v] of Object.entries(currentFilters)) {
      if (v && k !== key) params.set(k, v)
    }
    if (value) params.set(key, value)
    const q = params.toString()
    router.push(q ? `/auditoria?${q}` : '/auditoria')
  }

  function clearAll() {
    router.push('/auditoria')
  }

  const hasFilters = Object.values(currentFilters).some(Boolean)

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="flex items-center gap-1.5">
        <Filter className="size-3.5 text-ink-subtle" />
        <span className="text-2xs uppercase tracking-[0.14em] font-mono font-medium text-ink-subtle">
          Filtros
        </span>
      </div>

      <FilterDropdown
        label="Ação"
        value={currentFilters.action}
        options={[
          { value: '', label: 'Todas' },
          ...ACTION_OPTIONS,
        ]}
        onChange={(v) => setFilter('action', v || undefined)}
      />

      {hasFilters && (
        <button
          onClick={clearAll}
          className="inline-flex items-center gap-1 h-8 px-3.5 rounded-full text-xs font-medium bg-surface-raised border border-border text-ink-muted hover:text-ink hover:border-border-strong transition-colors"
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
  options: Array<{ value: string; label: string }>
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
          'inline-flex items-center gap-1.5 h-8 px-3.5 rounded-full text-xs font-medium tracking-tight transition-all duration-180 border',
          value
            ? 'bg-gradient-to-br from-accent-50 to-magenta-500/10 dark:from-accent-950/50 dark:to-magenta-500/10 border-accent-300 dark:border-accent-700 text-accent-700 dark:text-accent-300 shadow-sm'
            : 'bg-surface-raised border-border text-ink-muted hover:border-border-strong hover:text-ink',
        )}
      >
        <span className="text-ink-subtle uppercase tracking-[0.14em] font-mono text-2xs">
          {label}:
        </span>
        <span>{displayLabel}</span>
        <ChevronDown className="size-3" />
      </button>

      {open && (
        <div
          className={cn(
            'absolute left-0 top-full mt-2 z-30 w-60 max-h-80 overflow-y-auto',
            'bg-surface-raised/95 backdrop-blur-xl border border-border rounded-lg shadow-xl overflow-hidden',
            'animate-fade-in-fast py-1',
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
                    ? 'bg-accent-50 dark:bg-accent-950/40 text-accent-700 dark:text-accent-300 font-medium'
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