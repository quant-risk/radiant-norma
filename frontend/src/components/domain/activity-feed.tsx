'use client'

/**
 * ActivityFeed — timeline vertical de eventos.
 *
 * Cada item: timestamp relativo, ícone do tipo, ator (se houver),
 * descrição, payload colapsável. Usado em /auditoria.
 */
import * as React from 'react'
import {
  Send,
  Radar,
  BookCheck,
  Database,
  UserCheck,
  ShieldCheck,
  AlertTriangle,
} from 'lucide-react'
import { formatRelativeCompact, formatDateTime } from '@/lib/format'
import { cn } from '@/lib/utils'

export type ActivityKind =
  | 'envio.created'
  | 'envio.approved'
  | 'envio.rejected'
  | 'radar.detected'
  | 'radar.resolved'
  | 'rule.disabled'
  | 'rule.enabled'
  | 'schema.synced'
  | 'auth.login'
  | 'auth.dev_token'

const kindMeta: Record<
  ActivityKind,
  {
    icon: React.ComponentType<{ className?: string }>
    tone: 'neutral' | 'accent' | 'success' | 'warning' | 'critical' | 'info'
    label: string
  }
> = {
  'envio.created': { icon: Send, tone: 'info', label: 'Envio criado' },
  'envio.approved': { icon: ShieldCheck, tone: 'success', label: 'Envio aprovado' },
  'envio.rejected': { icon: AlertTriangle, tone: 'critical', label: 'Envio rejeitado' },
  'radar.detected': { icon: Radar, tone: 'warning', label: 'Alerta detectado' },
  'radar.resolved': { icon: Radar, tone: 'success', label: 'Alerta resolvido' },
  'rule.disabled': { icon: BookCheck, tone: 'neutral', label: 'Regra desabilitada' },
  'rule.enabled': { icon: BookCheck, tone: 'accent', label: 'Regra habilitada' },
  'schema.synced': { icon: Database, tone: 'info', label: 'Schema sincronizado' },
  'auth.login': { icon: UserCheck, tone: 'accent', label: 'Login' },
  'auth.dev_token': { icon: UserCheck, tone: 'warning', label: 'Dev token' },
}

export interface ActivityItem {
  id: string | number
  kind: ActivityKind
  timestamp: string
  actor?: string
  description?: string
  payload?: Record<string, unknown>
}

export interface ActivityFeedProps {
  items: ActivityItem[]
  loading?: boolean
  emptyMessage?: string
}

export function ActivityFeed({
  items,
  loading,
  emptyMessage = 'Sem eventos no período.',
}: ActivityFeedProps) {
  if (loading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex gap-3">
            <div className="skeleton size-8 rounded-full shrink-0" />
            <div className="flex-1 space-y-2">
              <div className="skeleton h-3 w-2/3 rounded" />
              <div className="skeleton h-3 w-1/3 rounded" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="py-12 text-center text-sm text-ink-muted">
        {emptyMessage}
      </div>
    )
  }

  return (
    <ol className="relative space-y-4">
      {/* Timeline line */}
      <div
        className="absolute left-4 top-3 bottom-3 w-px bg-border"
        aria-hidden
      />
      {items.map((item) => {
        const meta = kindMeta[item.kind]
        const Icon = meta.icon
        const toneClasses: Record<typeof meta.tone, string> = {
          neutral: 'bg-surface-raised text-ink-muted border-border',
          accent: 'bg-accent-50 text-accent-600 border-accent-200 dark:bg-accent-950 dark:text-accent-400 dark:border-accent-800',
          success: 'bg-success-50 text-success-600 border-success-200 dark:bg-success-950 dark:text-success-400 dark:border-success-800',
          warning: 'bg-warning-50 text-warning-600 border-warning-200 dark:bg-warning-950 dark:text-warning-400 dark:border-warning-800',
          critical: 'bg-critical-50 text-critical-600 border-critical-200 dark:bg-critical-950 dark:text-critical-400 dark:border-critical-800',
          info: 'bg-info-50 text-info-600 border-info-200 dark:bg-info-950 dark:text-info-400 dark:border-info-800',
        }
        return (
          <li
            key={item.id}
            className="relative flex gap-4 pl-0 animate-fade-in"
          >
            <div
              className={cn(
                'relative z-10 size-8 rounded-full border-2 flex items-center justify-center shrink-0',
                '[&_svg]:size-3.5',
                toneClasses[meta.tone],
              )}
            >
              <Icon />
            </div>
            <div className="flex-1 min-w-0 pt-1">
              <div className="flex items-center gap-2 mb-0.5 flex-wrap">
                <span className="text-sm font-medium text-ink">
                  {meta.label}
                </span>
                {item.actor && (
                  <span className="text-xs text-ink-muted">
                    por <span className="font-mono">{item.actor}</span>
                  </span>
                )}
                <span
                  className="text-2xs text-ink-subtle ml-auto"
                  title={formatDateTime(item.timestamp)}
                >
                  {formatRelativeCompact(item.timestamp)}
                </span>
              </div>
              {item.description && (
                <p className="text-sm text-ink-muted leading-snug">
                  {item.description}
                </p>
              )}
              {item.payload && Object.keys(item.payload).length > 0 && (
                <details className="mt-2">
                  <summary className="text-2xs text-ink-subtle cursor-pointer hover:text-ink-muted select-none">
                    ver detalhes
                  </summary>
                  <pre className="mt-2 text-2xs font-mono bg-surface-sunken border border-border rounded p-2 overflow-x-auto">
                    {JSON.stringify(item.payload, null, 2)}
                  </pre>
                </details>
              )}
            </div>
          </li>
        )
      })}
    </ol>
  )
}