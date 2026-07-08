'use client'

/**
 * CommandPalette — ⌘K global refinado.
 *
 * Visual: glass panel, grupos com eyebrow label serif, item ativo com
 * hairline gradient accent à esquerda.
 *
 * Lógica preservada: fuzzy score, agrupamento, keyboard nav, dynamic items.
 */

import * as React from 'react'
import { useRouter } from 'next/navigation'
import {
  Search,
  LayoutDashboard,
  Send,
  Radar,
  BookCheck,
  History,
  Sparkles,
  Sun,
  Moon,
  ArrowRight,
  Hash,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Kbd } from '@/components/ui/kbd'
import { useTheme } from '@/components/theme-provider'
import { useFocusTrap } from '@/lib/use-focus-trap'

interface CommandItem {
  id: string
  label: string
  description?: string
  icon: React.ComponentType<{ className?: string; strokeWidth?: number | string }>
  group: 'Navegação' | 'Tema' | 'Regras' | 'Alertas' | 'CADOCs'
  shortcut?: string
  action: () => void
  keywords?: string[]
}

export interface CommandPaletteProps {
  rules?: Array<{ code: string; description: string; severity: string }>
  alerts?: Array<{ id: number; title: string; severity: string; cadoc_code: string }>
  schemas?: Array<{ cadoc: string; description: string }>
}

export function CommandPalette({
  rules = [],
  alerts = [],
  schemas = [],
}: CommandPaletteProps) {
  const router = useRouter()
  const { theme, toggle } = useTheme()
  const [open, setOpen] = React.useState(false)
  const [query, setQuery] = React.useState('')
  const [activeIdx, setActiveIdx] = React.useState(0)
  const inputRef = React.useRef<HTMLInputElement>(null)

  const baseItems: CommandItem[] = React.useMemo(
    () => [
      {
        id: 'nav-dashboard',
        label: 'Dashboard',
        description: 'Visão geral de envios, alertas e regras',
        icon: LayoutDashboard,
        group: 'Navegação',
        action: () => router.push('/'),
      },
      {
        id: 'nav-envios',
        label: 'Envios STA',
        description: 'Lista de envios CADOC',
        icon: Send,
        group: 'Navegação',
        action: () => router.push('/envios'),
      },
      {
        id: 'nav-radar',
        label: 'Radar Regulatório',
        description: 'Alertas BACEN abertos',
        icon: Radar,
        group: 'Navegação',
        action: () => router.push('/radar'),
      },
      {
        id: 'nav-regras',
        label: 'Catálogo de Regras',
        description: '60 regras 3040 tipadas',
        icon: BookCheck,
        group: 'Navegação',
        action: () => router.push('/regras'),
      },
      {
        id: 'nav-insights',
        label: 'Insights',
        description: 'Anomalias, priorização, recomendações',
        icon: Sparkles,
        group: 'Navegação',
        action: () => router.push('/insights'),
      },
      {
        id: 'nav-auditoria',
        label: 'Auditoria',
        description: 'Histórico de eventos',
        icon: History,
        group: 'Navegação',
        action: () => router.push('/auditoria'),
      },
      {
        id: 'action-theme',
        label: theme === 'dark' ? 'Mudar para tema claro' : 'Mudar para tema escuro',
        icon: theme === 'dark' ? Sun : Moon,
        group: 'Tema',
        shortcut: '⌘⇧L',
        action: () => toggle(),
      },
    ],
    [router, theme, toggle],
  )

  const dynamicItems: CommandItem[] = React.useMemo(() => {
    const items: CommandItem[] = []
    rules.forEach((r) => {
      items.push({
        id: `rule-${r.code}`,
        label: r.code,
        description: r.description,
        icon: Hash,
        group: 'Regras',
        action: () => router.push(`/regras?focus=${r.code}`),
        keywords: [r.code, r.severity, r.description],
      })
    })
    alerts.forEach((a) => {
      items.push({
        id: `alert-${a.id}`,
        label: a.title,
        description: `Alerta ${a.severity} · ${a.cadoc_code}`,
        icon: Radar,
        group: 'Alertas',
        action: () => router.push(`/radar?focus=${a.id}`),
        keywords: [a.title, a.cadoc_code, a.severity],
      })
    })
    schemas.forEach((s) => {
      items.push({
        id: `schema-${s.cadoc}`,
        label: `CADOC ${s.cadoc}`,
        description: s.description,
        icon: BookCheck,
        group: 'CADOCs',
        action: () => router.push(`/regras?cadoc=${s.cadoc}`),
        keywords: [s.cadoc, s.description],
      })
    })
    return items
  }, [rules, alerts, schemas, router])

  const filtered = React.useMemo(() => {
    const all = [...baseItems, ...dynamicItems]
    if (!query.trim()) return all.slice(0, 8)
    const q = query.toLowerCase()
    return all
      .map((item) => {
        const haystack = [
          item.label,
          item.description ?? '',
          ...(item.keywords ?? []),
        ]
          .join(' ')
          .toLowerCase()
        let score = 0
        if (haystack.startsWith(q)) score += 100
        if (haystack.includes(q)) score += 50
        let qi = 0
        for (const ch of haystack) {
          if (ch === q[qi]) qi++
          if (qi === q.length) break
        }
        if (qi === q.length) score += 25
        return { item, score }
      })
      .filter((x) => x.score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, 12)
      .map((x) => x.item)
  }, [baseItems, dynamicItems, query])

  const grouped = React.useMemo(() => {
    const groups: Record<string, CommandItem[]> = {}
    filtered.forEach((item) => {
      groups[item.group] = groups[item.group] ?? []
      groups[item.group].push(item)
    })
    return groups
  }, [filtered])

  React.useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setOpen((o) => !o)
      }
      if (e.key === 'Escape' && open) {
        setOpen(false)
      }
    }
    function onOpenEvent() {
      setOpen(true)
    }
    window.addEventListener('keydown', onKey)
    window.addEventListener('rn:open-command-palette', onOpenEvent)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('rn:open-command-palette', onOpenEvent)
    }
  }, [open])

  React.useEffect(() => {
    if (open) {
      setQuery('')
      setActiveIdx(0)
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [open])

  React.useEffect(() => {
    setActiveIdx(0)
  }, [query])

  const flatItems = filtered

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIdx((i) => Math.min(i + 1, flatItems.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIdx((i) => Math.max(0, i - 1))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      const item = flatItems[activeIdx]
      if (item) {
        item.action()
        setOpen(false)
      }
    }
  }

  const { ref: trapRef, onKeyDown: trapKeyDown } = useFocusTrap<HTMLDivElement>(open)

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] px-4"
      role="dialog"
      aria-modal="true"
    >
      <div
        className="absolute inset-0 bg-ink/40 backdrop-blur-sm animate-fade-in-fast"
        onClick={() => setOpen(false)}
        aria-hidden
      />

      <div
        ref={trapRef}
        className={cn(
          'relative w-full max-w-xl',
          'bg-surface-raised/95 backdrop-blur-xl',
          'border border-border rounded-xl shadow-2xl',
          'overflow-hidden animate-scale-in',
        )}
        onKeyDown={(e) => {
          handleKeyDown(e)
          trapKeyDown(e)
        }}
      >
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 h-14 border-b border-border-subtle">
          <Search className="size-4 text-ink-subtle shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Buscar regra, alerta, CADOC, página…"
            className="flex-1 bg-transparent outline-none text-sm placeholder:text-ink-subtle"
          />
          <Kbd>esc</Kbd>
        </div>

        {/* Results */}
        <div className="max-h-96 overflow-y-auto py-2">
          {filtered.length === 0 ? (
            <div className="px-6 py-12 text-center">
              <p className="font-serif text-base text-ink mb-1">
                Nada encontrado
              </p>
              <p className="text-xs text-ink-subtle font-mono">
                para &ldquo;{query}&rdquo; — tente B01, 3040 ou &ldquo;rejeitados&rdquo;
              </p>
            </div>
          ) : (
            (['Navegação', 'Tema', 'Regras', 'Alertas', 'CADOCs'] as const).map(
              (groupName) => {
                const items = grouped[groupName]
                if (!items || items.length === 0) return null
                return (
                  <div key={groupName} className="px-2 py-1.5">
                    <div className="px-3 py-1.5 text-2xs uppercase tracking-[0.18em] font-mono font-medium text-ink-subtle">
                      {groupName}
                    </div>
                    {items.map((item) => {
                      const Icon = item.icon
                      const idx = flatItems.indexOf(item)
                      const isActive = idx === activeIdx
                      return (
                        <button
                          key={item.id}
                          onClick={() => {
                            item.action()
                            setOpen(false)
                          }}
                          onMouseEnter={() => setActiveIdx(idx)}
                          className={cn(
                            'group relative w-full flex items-center gap-3 px-3 py-2 rounded-md',
                            'text-left transition-colors duration-180',
                            isActive
                              ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/50 dark:text-accent-300'
                              : 'text-ink hover:bg-surface-sunken',
                          )}
                        >
                          {isActive && (
                            <span
                              className="absolute left-0 top-2 bottom-2 w-[2px] rounded-r-full bg-gradient-to-b from-accent-500 to-magenta-500"
                              aria-hidden
                            />
                          )}
                          <Icon
                            className={cn(
                              'size-4 shrink-0',
                              isActive
                                ? 'text-accent-600 dark:text-accent-400'
                                : 'text-ink-muted',
                            )}
                          />
                          <div className="flex-1 min-w-0">
                            <div className="text-sm font-medium truncate tracking-tight">
                              {item.label}
                            </div>
                            {item.description && (
                              <div className="text-xs text-ink-muted truncate mt-0.5">
                                {item.description}
                              </div>
                            )}
                          </div>
                          {item.shortcut && <Kbd>{item.shortcut}</Kbd>}
                          {isActive && (
                            <ArrowRight className="size-3.5 text-accent-600 dark:text-accent-400" />
                          )}
                        </button>
                      )
                    })}
                  </div>
                )
              },
            )
          )}
        </div>

        {/* Footer hints */}
        <div className="flex items-center gap-4 px-4 h-10 border-t border-border-subtle text-2xs text-ink-muted bg-surface-sunken/60">
          <span className="flex items-center gap-1.5">
            <Kbd>↑</Kbd>
            <Kbd>↓</Kbd>
            navegar
          </span>
          <span className="flex items-center gap-1.5">
            <Kbd>↵</Kbd>
            abrir
          </span>
          <span className="flex items-center gap-1.5">
            <Kbd>esc</Kbd>
            fechar
          </span>
          <span className="ml-auto text-ink-subtle font-mono">
            {flatItems.length} resultado{flatItems.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>
    </div>
  )
}