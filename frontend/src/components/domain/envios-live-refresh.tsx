// EnviosLiveRefresh — client component que mostra RealtimeBadge
// e dispara router.refresh() quando recebe evento SSE de envio.

'use client'

import { useRouter } from 'next/navigation'
import { useEventStream } from '@/lib/use-event-stream'
import { RealtimeBadge } from './realtime-badge'

const ENVIOS_KINDS = [
  'sta.submit',
  'sta.confirmed',
  'envio.approved',
  'envio.rejected',
]

interface Props {
  className?: string
}

export function EnviosLiveRefresh({ className }: Props) {
  const router = useRouter()
  const { status, eventCount, lastError } = useEventStream({
    kinds: ENVIOS_KINDS,
    onEvent: () => router.refresh(),
  })

  return (
    <RealtimeBadge
      status={status}
      eventCount={eventCount}
      lastError={lastError}
      className={className}
    />
  )
}