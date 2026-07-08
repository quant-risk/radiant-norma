// RealtimeBadge — indicador visual minimalista de conexão SSE.
//
// Versão premium: pill minimalista com dot animado (sem fundo colorido
// dominante). Cor restrita ao dot, texto em ink-muted.

'use client'

import { Activity, Wifi, WifiOff, RefreshCw } from 'lucide-react'
import { type StreamStatus } from '@/lib/use-event-stream'
import { cn } from '@/lib/utils'

interface RealtimeBadgeProps {
  status: StreamStatus
  eventCount: number
  lastError?: string | null
  className?: string
}

const STATUS_LABEL: Record<StreamStatus, string> = {
  idle: 'Desconectado',
  connecting: 'Conectando…',
  open: 'Ao vivo',
  reconnecting: 'Reconectando…',
  failed: 'Falhou',
}

export function RealtimeBadge({
  status,
  eventCount,
  lastError,
  className,
}: RealtimeBadgeProps) {
  const isLive = status === 'open'
  const isConnecting = status === 'connecting' || status === 'reconnecting'
  const isFailed = status === 'failed'

  const dotColor = isLive
    ? 'bg-success-500'
    : isConnecting
      ? 'bg-warning-500'
      : 'bg-ink-subtle'

  const Icon = isLive ? Activity : isConnecting ? RefreshCw : WifiOff

  const tooltip = lastError
    ? `${STATUS_LABEL[status]} — erro: ${lastError}`
    : STATUS_LABEL[status]

  return (
    <div
      className={cn(
        'inline-flex items-center gap-2 rounded-full px-2.5 py-1',
        'bg-surface-raised border border-border-subtle',
        'text-xs font-medium text-ink-muted',
        'transition-colors hover:border-border',
        className,
      )}
      title={tooltip}
      aria-live="polite"
    >
      {isLive ? (
        <span className="relative flex size-1.5">
          <span
            className={cn(
              'absolute inset-0 rounded-full animate-ping opacity-60',
              dotColor,
            )}
          />
          <span className={cn('relative size-1.5 rounded-full', dotColor)} />
        </span>
      ) : (
        <Icon
          className={cn(
            'size-3',
            isConnecting && 'animate-spin-slow',
            isFailed && 'text-critical-500',
          )}
        />
      )}
      <span className="tracking-tight">{STATUS_LABEL[status]}</span>
      {isLive && eventCount > 0 && (
        <span className="ml-1 text-2xs text-ink-subtle font-mono">
          · {eventCount}
        </span>
      )}
      {isFailed && <Wifi className="size-3 opacity-40" />}
    </div>
  )
}