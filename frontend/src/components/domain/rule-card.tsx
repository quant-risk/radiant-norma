'use client'

/**
 * RuleCard — visualização de regra catalogada (3040, etc).
 *
 * Hierarquia:
 *   1. Code (mono, accent)
 *   2. Severity (badge colorido)
 *   3. Description (snippet)
 *   4. Example (mono, muted — referência técnica)
 *   5. Status (enabled/disabled) + toggle
 */
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

type Severity = 'E' | 'A' | 'I' // Erro, Alerta, Info

const severityMeta: Record<
  Severity,
  { label: string; tone: 'critical' | 'warning' | 'info' }
> = {
  E: { label: 'Erro', tone: 'critical' },
  A: { label: 'Alerta', tone: 'warning' },
  I: { label: 'Info', tone: 'info' },
}

export interface RuleCardProps {
  code: string
  severity: Severity
  sheet?: string
  description: string
  example?: string
  enabled?: boolean
  failedCount?: number
  highlight?: boolean
  onClick?: () => void
}

export function RuleCard({
  code,
  severity,
  sheet,
  description,
  example,
  enabled = true,
  failedCount,
  highlight,
  onClick,
}: RuleCardProps) {
  const meta = severityMeta[severity]
  return (
    <Card
      padding="md"
      interactive={!!onClick}
      onClick={onClick}
      className={cn(
        'group transition-all',
        highlight && 'ring-2 ring-accent-400 border-accent-300',
        !enabled && 'opacity-50',
      )}
    >
      <div className="flex items-start justify-between gap-3 mb-2">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="font-mono text-xs font-semibold text-accent-600 dark:text-accent-400">
            {code}
          </span>
          <Badge tone={meta.tone} variant="soft">
            {meta.label}
          </Badge>
          {sheet && (
            <span className="text-2xs text-ink-subtle font-mono">
              {sheet}
            </span>
          )}
        </div>
        {failedCount !== undefined && failedCount > 0 && (
          <span className="flex items-center gap-1 text-2xs font-semibold text-critical-700 dark:text-critical-300 bg-critical-50 dark:bg-critical-950 px-2 py-0.5 rounded-full">
            {failedCount} falha{failedCount !== 1 ? 's' : ''}
          </span>
        )}
      </div>
      <p className="text-sm text-ink leading-snug mb-2 line-clamp-2">
        {description}
      </p>
      {example && (
        <code className="block text-2xs text-ink-subtle font-mono leading-relaxed bg-surface-sunken px-2 py-1.5 rounded border border-border-subtle">
          {example}
        </code>
      )}
    </Card>
  )
}