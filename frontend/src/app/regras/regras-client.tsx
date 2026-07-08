'use client'

/**
 * /regras — catálogo interativo de regras (60 regras 3040).
 *
 * Refinado: cards maiores, typography editorial, modal com hairline header,
 * filtros como pills com estilo premium.
 */

import { useState, useMemo, useEffect, useCallback } from 'react'
import {
  Search,
  Filter,
  X,
  ChevronRight,
  CheckCircle2,
  Circle,
  Hash,
  Loader2,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { useRulePreferences } from '@/lib/use-rule-preferences'
import { useFocusTrap } from '@/lib/use-focus-trap'

type Severity = 'E' | 'A' | 'I'
type Category = 'B' | 'F' | 'C' | 'S'

interface Rule {
  code: string
  severity: Severity
  sheet?: string
  description: string
  example?: string
  category: Category
}

interface RegrasClientProps {
  rules: Rule[]
}

const categoryMeta: Record<Category, { label: string; description: string }> = {
  B: { label: 'Básicas', description: 'Estrutura do arquivo, encoding, layout' },
  F: { label: 'Formato', description: 'Formato de campos, máscaras, tipos' },
  C: { label: 'Campos', description: 'Obrigatoriedade, presença' },
  S: { label: 'Semânticas', description: 'Regras de negócio, coerência' },
}

const severityMeta: Record<
  Severity,
  { label: string; tone: 'critical' | 'warning' | 'info'; dot: string }
> = {
  E: { label: 'Erro', tone: 'critical', dot: 'bg-critical-500' },
  A: { label: 'Alerta', tone: 'warning', dot: 'bg-warning-500' },
  I: { label: 'Info', tone: 'info', dot: 'bg-info-500' },
}

export function RegrasClient({ rules }: RegrasClientProps) {
  const [query, setQuery] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<Category | 'ALL'>('ALL')
  const [severityFilter, setSeverityFilter] = useState<Severity | 'ALL'>('ALL')
  const [enabledOnly, setEnabledOnly] = useState(false)
  const [focused, setFocused] = useState<Rule | null>(null)

  const { disabled, loading: prefsLoading, toggle: toggleRuleBackend } = useRulePreferences()
  const [togglePending, setTogglePending] = useState<Set<string>>(new Set())

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const focusCode = params.get('focus')
    if (focusCode) {
      const rule = rules.find((r) => r.code === focusCode)
      if (rule) setFocused(rule)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return rules.filter((r) => {
      if (categoryFilter !== 'ALL' && r.category !== categoryFilter) return false
      if (severityFilter !== 'ALL' && r.severity !== severityFilter) return false
      if (enabledOnly && disabled.has(r.code)) return false
      if (q) {
        const haystack = `${r.code} ${r.description} ${r.example ?? ''}`.toLowerCase()
        if (!haystack.includes(q)) return false
      }
      return true
    })
  }, [rules, query, categoryFilter, severityFilter, enabledOnly, disabled])

  const counts = useMemo(
    () => ({
      B: rules.filter((r) => r.category === 'B').length,
      F: rules.filter((r) => r.category === 'F').length,
      C: rules.filter((r) => r.category === 'C').length,
      S: rules.filter((r) => r.category === 'S').length,
      E: rules.filter((r) => r.severity === 'E').length,
      A: rules.filter((r) => r.severity === 'A').length,
      I: rules.filter((r) => r.severity === 'I').length,
      enabled: rules.length - disabled.size,
    }),
    [rules, disabled],
  )

  const toggleRule = useCallback(
    async (code: string) => {
      if (togglePending.has(code)) return
      setTogglePending((p) => new Set(p).add(code))
      try {
        const result = await toggleRuleBackend(code)
        if ('error' in result) {
          console.warn(`[regras] toggle ${code}: ${result.error}`)
        }
      } finally {
        setTogglePending((p) => {
          const next = new Set(p)
          next.delete(code)
          return next
        })
      }
    },
    [togglePending, toggleRuleBackend],
  )

  useEffect(() => {
    if (!focused) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.stopPropagation()
        setFocused(null)
      }
    }
    window.addEventListener('keydown', onKey, { capture: true })
    return () => window.removeEventListener('keydown', onKey, { capture: true })
  }, [focused])

  return (
    <>
      {/* Toolbar */}
      <div className="space-y-4 mb-8">
        <div className="flex flex-col md:flex-row gap-3">
          <div className="flex-1 relative">
            <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 size-4 text-ink-subtle pointer-events-none" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Buscar por código ou descrição…"
              className={cn(
                'w-full h-11 pl-11 pr-10 rounded-lg',
                'bg-surface-raised border border-border',
                'text-sm placeholder:text-ink-subtle',
                'focus:outline-none focus:border-accent-400 focus:ring-2 focus:ring-accent-400/20',
                'transition-all duration-180',
              )}
            />
            {query && (
              <button
                onClick={() => setQuery('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 size-6 rounded text-ink-subtle hover:text-ink hover:bg-surface-sunken flex items-center justify-center transition-colors"
                aria-label="Limpar busca"
              >
                <X className="size-3.5" />
              </button>
            )}
          </div>
        </div>

        {/* Filter chips */}
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-1.5 pr-1">
            <Filter className="size-3.5 text-ink-subtle" />
            <span className="text-2xs uppercase tracking-[0.14em] font-mono font-medium text-ink-subtle">
              Categoria
            </span>
          </div>
          <FilterChip
            active={categoryFilter === 'ALL'}
            onClick={() => setCategoryFilter('ALL')}
          >
            Todas ({rules.length})
          </FilterChip>
          {(Object.keys(categoryMeta) as Category[]).map((c) => (
            <FilterChip
              key={c}
              active={categoryFilter === c}
              onClick={() => setCategoryFilter(c)}
            >
              {categoryMeta[c].label} ({counts[c]})
            </FilterChip>
          ))}

          <span className="mx-1 h-5 w-px bg-border" />

          <span className="text-2xs uppercase tracking-[0.14em] font-mono font-medium text-ink-subtle pr-1">
            Severidade
          </span>
          <FilterChip
            active={severityFilter === 'ALL'}
            onClick={() => setSeverityFilter('ALL')}
          >
            Todas
          </FilterChip>
          {(['E', 'A', 'I'] as Severity[]).map((s) => (
            <FilterChip
              key={s}
              active={severityFilter === s}
              onClick={() => setSeverityFilter(s)}
              dot={severityMeta[s].dot}
            >
              {severityMeta[s].label} ({counts[s]})
            </FilterChip>
          ))}

          <span className="mx-1 h-5 w-px bg-border" />

          <FilterChip
            active={enabledOnly}
            onClick={() => setEnabledOnly(!enabledOnly)}
          >
            <CheckCircle2 className="size-3" strokeWidth={2.25} />
            Apenas habilitadas
          </FilterChip>
        </div>
      </div>

      {/* Results */}
      {filtered.length === 0 ? (
        <EmptyState
          icon={<Search className="size-5" strokeWidth={1.75} />}
          title="Nenhuma regra encontrada"
          description={`Sem regras que correspondam aos filtros${query ? ` para "${query}"` : ''}.`}
          action={
            <Button
              variant="secondary"
              size="md"
              onClick={() => {
                setQuery('')
                setCategoryFilter('ALL')
                setSeverityFilter('ALL')
                setEnabledOnly(false)
              }}
            >
              Limpar filtros
            </Button>
          }
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {filtered.map((r) => {
            const sev = severityMeta[r.severity]
            const isDisabled = disabled.has(r.code)
            return (
              <div
                key={r.code}
                role="button"
                tabIndex={0}
                onClick={() => setFocused(r)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    setFocused(r)
                  }
                }}
                className={cn(
                  'text-left rounded-lg border bg-surface-raised p-5 cursor-pointer',
                  'transition-all duration-240 ease-out-expo hover:shadow-md hover:border-border-strong hover:-translate-y-px',
                  'group relative outline-none',
                  'focus-visible:ring-2 focus-visible:ring-accent-400/40',
                  isDisabled
                    ? 'border-border opacity-60'
                    : 'border-border',
                )}
              >
                <div className="flex items-start justify-between gap-2 mb-3">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-mono text-xs font-medium text-accent-600 dark:text-accent-400">
                      {r.code}
                    </span>
                    <Badge tone={sev.tone} variant="soft" dot size="sm">
                      {sev.label}
                    </Badge>
                    {togglePending.has(r.code) && (
                      <Loader2 className="size-3 animate-spin text-ink-subtle" />
                    )}
                  </div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      toggleRule(r.code)
                    }}
                    disabled={togglePending.has(r.code)}
                    className={cn(
                      'shrink-0 p-1 rounded transition-colors',
                      'disabled:opacity-50 disabled:cursor-wait',
                      isDisabled
                        ? 'text-ink-subtle hover:text-ink'
                        : 'text-success-600 dark:text-success-400 hover:bg-success-50 dark:hover:bg-success-950/40',
                    )}
                    aria-label={
                      isDisabled ? 'Habilitar regra' : 'Desabilitar regra'
                    }
                  >
                    {isDisabled ? (
                      <Circle className="size-3.5" />
                    ) : (
                      <CheckCircle2 className="size-3.5" strokeWidth={2.25} />
                    )}
                  </button>
                </div>
                <p className="font-serif text-sm font-medium text-ink leading-snug line-clamp-2 mb-2.5 tracking-tight">
                  {r.description}
                </p>
                {r.example && (
                  <code className="block text-2xs text-ink-subtle font-mono bg-surface-sunken px-2 py-1 rounded border border-border-subtle truncate">
                    {r.example}
                  </code>
                )}
                <div className="mt-3 flex items-center justify-end">
                  <ChevronRight className="size-3.5 text-ink-subtle group-hover:text-accent-600 group-hover:translate-x-0.5 transition-all" strokeWidth={2.25} />
                </div>
              </div>
            )
          })}
        </div>
      )}

      {focused && (
        <RuleDetailModal
          rule={focused}
          isDisabled={disabled.has(focused.code)}
          onClose={() => setFocused(null)}
          onToggle={() => toggleRule(focused.code)}
          prefsLoading={prefsLoading}
        />
      )}
    </>
  )
}

function FilterChip({
  active,
  onClick,
  children,
  dot,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
  dot?: string
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'inline-flex items-center gap-1.5 h-8 px-3.5 rounded-full text-xs font-medium tracking-tight',
        'transition-all duration-180 border',
        active
          ? 'bg-gradient-to-br from-accent-50 to-magenta-500/10 dark:from-accent-950/50 dark:to-magenta-500/10 border-accent-300 dark:border-accent-700 text-accent-700 dark:text-accent-300 shadow-sm'
          : 'bg-surface-raised border-border text-ink-muted hover:border-border-strong hover:text-ink',
      )}
    >
      {dot && <span className={cn('size-1.5 rounded-full', dot)} />}
      {children}
    </button>
  )
}

function RuleDetailModal({
  rule,
  isDisabled,
  onClose,
  onToggle,
  prefsLoading,
}: {
  rule: Rule
  isDisabled: boolean
  onClose: () => void
  onToggle: () => void
  prefsLoading: boolean
}) {
  const sev = severityMeta[rule.severity]
  const cat = categoryMeta[rule.category]
  const { ref: trapRef, onKeyDown: trapKeyDown } = useFocusTrap<HTMLDivElement>(true)
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="rule-modal-title"
    >
      <div
        className="absolute inset-0 bg-ink/40 backdrop-blur-sm animate-fade-in-fast"
        onClick={onClose}
        aria-hidden
      />
      <div
        ref={trapRef}
        onKeyDown={trapKeyDown}
        className="relative w-full max-w-xl bg-surface-raised border border-border rounded-xl shadow-2xl overflow-hidden animate-scale-in"
      >
        <div className="relative px-7 py-6 border-b border-border-subtle">
          <div
            className="absolute top-0 left-0 right-0 h-[3px] bg-gradient-to-r from-accent-500 to-magenta-500"
            aria-hidden
          />
          <div className="flex items-start justify-between gap-4">
            <div>
              <div className="flex items-center gap-2 mb-2 flex-wrap">
                <span className="font-mono text-sm font-medium text-accent-600 dark:text-accent-400">
                  {rule.code}
                </span>
                <Badge tone={sev.tone} variant="soft" dot size="sm">
                  {sev.label}
                </Badge>
                <Badge tone="neutral" variant="soft" size="sm">
                  {cat.label}
                </Badge>
                {rule.sheet && (
                  <span className="text-2xs text-ink-subtle font-mono">
                    {rule.sheet}
                  </span>
                )}
              </div>
              <h2 id="rule-modal-title" className="font-serif text-xl font-medium text-ink tracking-tight">
                {cat.description}
              </h2>
            </div>
            <button
              onClick={onClose}
              className="size-8 rounded-md text-ink-muted hover:bg-surface-sunken hover:text-ink flex items-center justify-center transition-colors"
              aria-label="Fechar modal (ESC)"
            >
              <X className="size-4" />
            </button>
          </div>
        </div>

        <div className="px-7 py-6 space-y-5">
          <div>
            <div className="text-2xs uppercase tracking-[0.14em] font-mono font-medium text-ink-subtle mb-2">
              Descrição
            </div>
            <p className="text-sm text-ink leading-relaxed">
              {rule.description}
            </p>
          </div>

          {rule.example && (
            <div>
              <div className="text-2xs uppercase tracking-[0.14em] font-mono font-medium text-ink-subtle mb-2 flex items-center gap-1.5">
                <Hash className="size-3" strokeWidth={2.25} />
                Exemplo
              </div>
              <code className="block text-xs text-ink font-mono bg-surface-sunken border border-border-subtle rounded-md p-3.5 overflow-x-auto">
                {rule.example}
              </code>
            </div>
          )}
        </div>

        <div className="px-7 py-4 border-t border-border-subtle bg-surface-sunken/60 flex items-center justify-between gap-3">
          <div className="text-xs text-ink-muted">
            {isDisabled ? 'Regra desabilitada' : 'Regra habilitada'} — afeta{' '}
            <span className="font-mono font-medium">todos os CADOCs 3040</span>
            {prefsLoading && (
              <span className="ml-2 text-ink-subtle font-mono">· sincronizando…</span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" onClick={onClose}>
              Fechar
            </Button>
            <Button
              variant={isDisabled ? 'primary' : 'secondary'}
              size="sm"
              onClick={onToggle}
            >
              {isDisabled ? 'Habilitar regra' : 'Desabilitar regra'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}