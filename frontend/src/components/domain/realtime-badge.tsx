// RealtimeBadge — indicador visual de conexão SSE.
//
// Mostra:
//   🟢 Live — conectado, recebendo eventos
//   🟡 Connecting / Reconnecting — em progresso (com dot pulsante)
//   🔴 Failed / Idle — desconectado (sem retry)
//
// Props:
//   status: StreamStatus do useEventStream
//   eventCount: número de eventos recebidos
//   lastError: string de erro (opcional, mostra em tooltip)
//
// Usa ícones lucide-react (consistente com o resto do design system).

'use client'

import { Activity, Wifi, WifiOff, RefreshCw } from 'lucide-react'
import { type StreamStatus } from '@/lib/use-event-stream'

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

  const colorClass = isLive
    ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border-emerald-500/30'
    : isConnecting
      ? 'bg-amber-500/15 text-amber-700 dark:text-amber-300 border-amber-500/30'
      : 'bg-zinc-500/10 text-zinc-600 dark:text-zinc-400 border-zinc-500/20'

  const Icon = isLive ? Activity : isConnecting ? RefreshCw : WifiOff
  const tooltip = lastError
    ? `${STATUS_LABEL[status]} — erro: ${lastError}`
    : STATUS_LABEL[status]

  return (
    <div
      className={
        'inline-flex items-center gap-2 rounded-full border px-2.5 py-1 text-xs font-medium ' +
        colorClass +
        (className ? ` ${className}` : '')
      }
      title={tooltip}
      aria-live="polite"
    >
      {isLive ? (
        <span className="relative flex h-1.5 w-1.5">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
          <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500" />
        </span>
      ) : (
        <Icon className={isConnecting ? 'h-3 w-3 animate-spin' : 'h-3 w-3'} />
      )}
      <span>{STATUS_LABEL[status]}</span>
      {isLive && eventCount > 0 && (
        <span className="ml-1 tabular-nums text-[10px] opacity-70">
          · {eventCount} {eventCount === 1 ? 'evento' : 'eventos'}
        </span>
      )}
      {isFailed && <Wifi className="h-3 w-3 opacity-50" />}
    </div>
  )
}