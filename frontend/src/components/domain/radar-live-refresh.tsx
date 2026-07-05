// RadarLiveRefresh — client component que mostra RealtimeBadge
// e dispara router.refresh() quando recebe evento SSE de radar.

'use client'

import { useRouter } from 'next/navigation'
import { useEventStream } from '@/lib/use-event-stream'
import { RealtimeBadge } from './realtime-badge'

const RADAR_KINDS = [
  'radar.detected',
  'radar.resolved',
  'radar.scan_completed',
]

interface Props {
  className?: string
}

export function RadarLiveRefresh({ className }: Props) {
  const router = useRouter()
  const { status, eventCount, lastError } = useEventStream({
    kinds: RADAR_KINDS,
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