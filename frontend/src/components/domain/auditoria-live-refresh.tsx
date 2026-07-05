// AuditoriaLiveRefresh — client component que mostra RealtimeBadge
// e dispara router.refresh() quando recebe evento SSE relevante.
//
// Sprint 10: /auditoria agora atualiza ao vivo sem F5. Cada audit
// event (cadoc.validated, sta.submit, envio.approved, etc) recebida
// via SSE Hub dispara refresh do server component que re-faz
// apiFetch('/v1/audit_log').
//
// Filtra por kinds comuns de auditoria pra evitar refresh por
// eventos irrelevantes.

'use client'

import { useRouter } from 'next/navigation'
import { useEventStream } from '@/lib/use-event-stream'
import { RealtimeBadge } from './realtime-badge'

const AUDIT_KINDS = [
  'cadoc.validated',
  'sta.submit',
  'envio.approved',
  'envio.rejected',
  'radar.detected',
  'radar.resolved',
  'rule.disabled',
  'rule.enabled',
  'auth.login',
]

interface Props {
  className?: string
}

export function AuditoriaLiveRefresh({ className }: Props) {
  const router = useRouter()
  const { status, eventCount, lastError } = useEventStream({
    kinds: AUDIT_KINDS,
    onEvent: () => {
      // Re-fetch server component data sem full reload.
      router.refresh()
    },
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