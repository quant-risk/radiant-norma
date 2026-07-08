'use client'

/**
 * ActivityFeed — timeline editorial.
 *
 * Cada item: glyph + label serif + actor mono + timestamp relative.
 * Linha de timeline com gradiente sutil.
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
    icon: React.ComponentType<{ className?: string; strokeWidth?: number | string }>
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

// Módulo-level: classe por tone. Recriado 1x por import do módulo,
// não por item renderizado (perf).
const TONE_CLASSES: Record<ActivityKind extends never ? never : 'neutral' | 'accent' | 'success' | 'warning' | 'critical' | 'info', string> = {
  neutral: 'bg-surface-raised text-ink-muted ring-border',
  accent: 'bg-accent-50 text-accent-600 ring-accent-200/60 dark:bg-accent-950/50 dark:text-accent-300 dark:ring-accent-800/40',
  success: 'bg-success-50 text-success-600 ring-success-200/60 dark:bg-success-950/50 dark:text-success-300 dark:ring-success-800/40',
  warning: 'bg-warning-50 text-warning-600 ring-warning-200/60 dark:bg-warning-950/50 dark:text-warning-300 dark:ring-warning-800/40',
  critical: 'bg-critical-50 text-critical-600 ring-critical-200/60 dark:bg-critical-950/50 dark:text-critical-300 dark:ring-critical-800/40',
  info: 'bg-info-50 text-info-600 ring-info-200/60 dark:bg-info-950/50 dark:text-info-300 dark:ring-info-800/40',
}

export function ActivityFeed({
  items,
  loading,
  emptyMessage = 'Sem eventos no período.',
}: ActivityFeedProps) {
  if (loading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex gap-4">
            <div className="skeleton size-8 rounded-md shrink-0" />
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
    <ol className="relative">
      {/* Timeline line — gradient sutil */}
      <div
        className="absolute left-4 top-3 bottom-3 w-px bg-gradient-to-b from-border via-border to-transparent"
        aria-hidden
      />
      {items.map((item, i) => {
        const meta = kindMeta[item.kind]
        const Icon = meta.icon
        return (
          <li
            key={item.id}
            className={cn(
              'relative flex gap-4 pb-5 last:pb-0 animate-fade-in',
              i > 0 && 'pt-5',
            )}
          >
            <div
              className={cn(
                'relative z-10 size-8 rounded-md ring-1 ring-inset flex items-center justify-center shrink-0',
                '[&_svg]:size-3.5',
                TONE_CLASSES[meta.tone],
              )}
            >
              <Icon strokeWidth={2.25} />
            </div>
            <div className="flex-1 min-w-0 pt-0.5">
              <div className="flex items-center gap-2 mb-1 flex-wrap">
                <span className="font-serif text-sm font-medium text-ink tracking-tight">
                  {meta.label}
                </span>
                {item.actor && (
                  <span className="text-xs text-ink-muted">
                    por <span className="font-mono text-ink">{item.actor}</span>
                  </span>
                )}
                <span
                  className="text-2xs text-ink-subtle ml-auto font-mono"
                  title={formatDateTime(item.timestamp)}
                >
                  {formatRelativeCompact(item.timestamp)}
                </span>
              </div>
              {item.description && (
                <p className="text-sm text-ink-muted leading-relaxed">
                  {item.description}
                </p>
              )}
              {item.payload && Object.keys(item.payload).length > 0 && (
                <details className="mt-2 group">
                  <summary className="text-2xs text-ink-subtle cursor-pointer hover:text-ink-muted select-none font-mono uppercase tracking-wider">
                    ver detalhes
                  </summary>
                  <pre className="mt-2 text-2xs font-mono bg-surface-sunken border border-border-subtle rounded-md p-3 overflow-x-auto text-ink-muted">
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