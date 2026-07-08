'use client'

/**
 * AlertCard — visualização de alerta radar, versão "press release".
 *
 * Hierarquia:
 *   1. Hairline accent (severity) à esquerda — comunicação primária
 *   2. Eyebrow (severity · CADOC · timestamp)
 *   3. Título (serif)
 *   4. Descrição
 *   5. Ação (resolver / ver fonte)
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
    icon: React.ComponentType<{ className?: string; strokeWidth?: number | string }>
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
    borderAccent: 'before:bg-info-500',
    iconColor: 'text-info-600 dark:text-info-400',
  },
  warn: {
    icon: AlertCircle,
    label: 'Atenção',
    tone: 'warning',
    borderAccent: 'before:bg-warning-500',
    iconColor: 'text-warning-600 dark:text-warning-400',
  },
  critical: {
    icon: AlertTriangle,
    label: 'Crítico',
    tone: 'critical',
    borderAccent: 'before:bg-critical-500',
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
      const r = await fetch(`/api/radar/alerts/${id}/resolve`, {
        method: 'POST',
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      window.location.reload()
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('resolve alert failed', e)
      setResolving(false)
    }
  }

  return (
    <Card
      padding="none"
      className={cn(
        'relative overflow-hidden transition-all duration-240 ease-out-expo',
        'before:absolute before:left-0 before:top-0 before:bottom-0 before:w-[3px]',
        cfg.borderAccent,
        resolved && 'opacity-60',
      )}
    >
      <div className="flex items-start gap-5 p-5 md:p-6 pl-7">
        {/* Severity icon */}
        <div
          className={cn(
            'shrink-0 size-10 rounded-lg flex items-center justify-center ring-1 ring-inset',
            severity === 'critical'
              ? 'bg-critical-50 dark:bg-critical-950/40 ring-critical-200/60 dark:ring-critical-800/40'
              : severity === 'warn'
                ? 'bg-warning-50 dark:bg-warning-950/40 ring-warning-200/60 dark:ring-warning-800/40'
                : 'bg-info-50 dark:bg-info-950/40 ring-info-200/60 dark:ring-info-800/40',
          )}
        >
          <Icon className={cn('size-4', cfg.iconColor)} strokeWidth={2.25} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2.5 mb-2 flex-wrap">
            <Badge tone={cfg.tone} variant="soft" dot size="sm">
              {cfg.label}
            </Badge>
            <span className="text-xs text-ink-muted font-mono tracking-tight">
              CADOC {cadoc_code}
            </span>
            <span className="text-2xs text-ink-subtle ml-auto font-mono">
              {formatRelativeCompact(detected_at)}
            </span>
          </div>

          <h3
            className={cn(
              'font-serif text-base font-medium text-ink leading-snug mb-1.5 tracking-tight',
              resolved && 'line-through opacity-70',
            )}
          >
            {title}
          </h3>

          <p className="text-sm text-ink-muted line-clamp-2 leading-relaxed">
            {description}
          </p>

          <div className="flex items-center gap-3 mt-4 pt-4 border-t border-border-subtle">
            <a
              href={source_url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1.5 text-xs font-medium text-ink-muted hover:text-accent-600 dark:hover:text-accent-400 transition-colors"
            >
              <ExternalLink className="size-3" strokeWidth={2} />
              Fonte BACEN
            </a>
            {!resolved && (
              <Button
                size="sm"
                variant="secondary"
                loading={resolving}
                onClick={handleResolve}
                leftIcon={<CheckCircle2 className="size-3.5" />}
                className="ml-auto"
              >
                Resolver
              </Button>
            )}
            {resolved && (
              <Badge tone="success" variant="soft" icon={<CheckCircle2 className="size-3" />} size="sm">
                Resolvido
              </Badge>
            )}
          </div>
        </div>
      </div>
    </Card>
  )
}