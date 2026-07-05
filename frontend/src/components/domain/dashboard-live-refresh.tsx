// DashboardLiveRefresh — client component que mostra RealtimeBadge
// e dispara router.refresh() em eventos críticos no dashboard.

'use client'

import { useRouter } from 'next/navigation'
import { useEventStream } from '@/lib/use-event-stream'
import { RealtimeBadge } from './realtime-badge'

const DASHBOARD_KINDS = [
  'sta.submit',
  'envio.approved',
  'envio.rejected',
  'radar.detected',
  'cadoc.validated',
]

interface Props {
  className?: string
}

export function DashboardLiveRefresh({ className }: Props) {
  const router = useRouter()
  const { status, eventCount, lastError } = useEventStream({
    kinds: DASHBOARD_KINDS,
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