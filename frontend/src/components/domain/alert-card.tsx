'use client'

/**
 * AlertCard — visualização de alerta radar.
 *
 * Hierarquia visual:
 *   1. Severity (cor + ícone) — comunicação primária
 *   2. CADOC code (mono) — referência técnica
 *   3. Title (título do alerta)
 *   4. Description (snippet)
 *   5. Metadata (timestamp relativo, source URL)
 *   6. Action (resolver via server action inline)
 *
 * Resolve action usa Next.js server action via API direta (fetch),
 * NÃO prop function — server actions inline como prop em client
 * component é anti-pattern (validação 29 / C1).
 */
import * as React from 'react'
import {
  AlertTriangle,
  AlertCircle,
  Info,
  ExternalLink,
  CheckCircle2,
} from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { formatRelativeCompact } from '@/lib/format'
import { cn } from '@/lib/utils'

type Severity = 'info' | 'warn' | 'critical'

export interface AlertCardProps {
  id: number
  cadoc_code: string
  severity: Severity
  title: string
  description: string
  source_url: string
  detected_at: string
  resolved?: boolean
}

const severityConfig: Record<
  Severity,
  {
    icon: React.ComponentType<{ className?: string }>
    label: string
    tone: 'info' | 'warning' | 'critical'
    borderAccent: string
    iconColor: string
  }
> = {
  info: {
    icon: Info,
    label: 'Info',
    tone: 'info',
    borderAccent: 'border-l-info-500',
    iconColor: 'text-info-600 dark:text-info-400',
  },
  warn: {
    icon: AlertCircle,
    label: 'Atenção',
    tone: 'warning',
    borderAccent: 'border-l-warning-500',
    iconColor: 'text-warning-600 dark:text-warning-400',
  },
  critical: {
    icon: AlertTriangle,
    label: 'Crítico',
    tone: 'critical',
    borderAccent: 'border-l-critical-500',
    iconColor: 'text-critical-600 dark:text-critical-400',
  },
}

export function AlertCard({
  id,
  cadoc_code,
  severity,
  title,
  description,
  source_url,
  detected_at,
  resolved,
}: AlertCardProps) {
  const cfg = severityConfig[severity]
  const Icon = cfg.icon
  const [resolving, setResolving] = React.useState(false)

  async function handleResolve() {
    setResolving(true)
    try {
      // Validação 29 (C1 fix): chamada direta via fetch em vez de
      // server action inline como prop. Backend já é o source of truth
      // (POST /v1/radar/alerts/:id/resolve).
      //
      // Cookie httpOnly `rn_jwt` é enviado automaticamente (same-origin).
      const r = await fetch(`/api/radar/alerts/${id}/resolve`, {
        method: 'POST',
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      // Full page refresh para o server component re-renderizar.
      // Alternativa futura: router.refresh() + useTransition.
      window.location.reload()
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('resolve alert failed', e)
      setResolving(false)
    }
  }

  return (
    <Card
      padding="md"
      className={cn(
        'border-l-4 transition-all duration-150',
        cfg.borderAccent,
        resolved && 'opacity-60',
      )}
    >
      <div className="flex items-start gap-4">
        {/* Severity icon */}
        <div
          className={cn(
            'shrink-0 size-9 rounded-lg flex items-center justify-center',
            severity === 'critical'
              ? 'bg-critical-50 dark:bg-critical-950'
              : severity === 'warn'
                ? 'bg-warning-50 dark:bg-warning-950'
                : 'bg-info-50 dark:bg-info-950',
          )}
        >
          <Icon className={cn('size-4', cfg.iconColor)} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1.5 flex-wrap">
            <Badge tone={cfg.tone} variant="soft" dot>
              {cfg.label}
            </Badge>
            <span className="font-mono text-xs text-ink-muted">
              {cadoc_code}
            </span>
            <span className="text-2xs text-ink-subtle ml-auto">
              {formatRelativeCompact(detected_at)}
            </span>
          </div>

          <h3
            className={cn(
              'text-sm font-semibold text-ink leading-snug mb-1',
              resolved && 'line-through',
            )}
          >
            {title}
          </h3>

          <p className="text-sm text-ink-muted line-clamp-2 mb-3 leading-relaxed">
            {description}
          </p>

          <div className="flex items-center gap-3">
            <a
              href={source_url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-xs text-accent-600 dark:text-accent-400 hover:underline"
            >
              <ExternalLink className="size-3" />
              Fonte BACEN
            </a>
            {!resolved && (
              <Button
                size="sm"
                variant="ghost"
                loading={resolving}
                onClick={handleResolve}
                leftIcon={<CheckCircle2 className="size-3.5" />}
              >
                Resolver
              </Button>
            )}
            {resolved && (
              <Badge tone="success" variant="soft" icon={<CheckCircle2 className="size-3" />}>
                Resolvido
              </Badge>
            )}
          </div>
        </div>
      </div>
    </Card>
  )
}