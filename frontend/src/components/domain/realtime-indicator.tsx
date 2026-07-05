// RealtimeIndicator — wrapper que combina RealtimeBadge + auto-refresh
// via useEventStream. Componente client-side que dispara callback
// onEvent em cada evento SSE.
//
// Uso típico: passar refresh callback do React Query pra invalidar
// a query ativa quando recebe evento relevante. Isso faz com que a UI
// atualize sem F5.
//
// Exemplo:
//   const queryClient = useQueryClient()
//   <RealtimeIndicator
//     kinds={['audit', 'sta.submit']}
//     onEvent={() => queryClient.invalidateQueries({ queryKey: ['audit'] })}
//   />

'use client'

import { useEventStream } from '@/lib/use-event-stream'
import { RealtimeBadge } from './realtime-badge'

interface RealtimeIndicatorProps {
  kinds?: string[]
  onEvent?: (evt: import('@/lib/use-event-stream').StreamEvent) => void
  className?: string
}

export function RealtimeIndicator({ kinds, onEvent, className }: RealtimeIndicatorProps) {
  const { status, eventCount, lastError } = useEventStream({ kinds, onEvent })

  return (
    <RealtimeBadge
      status={status}
      eventCount={eventCount}
      lastError={lastError}
      className={className}
    />
  )
}