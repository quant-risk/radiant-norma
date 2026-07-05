// useEventStream — hook React que conecta ao backend SSE.
//
// Usar pra páginas que devem reagir a eventos real-time:
//   /auditoria → refresh ao receber evento audit
//   /radar     → refresh ao receber radar.detected
//   /envios    → refresh ao receber sta.submit
//
// Features:
//   - Auto-reconnect com backoff exponencial (1s, 2s, 4s, max 30s)
//   - Cleanup automático no unmount
//   - Filtro client-side opcional por kind
//   - Status: connecting | open | reconnecting | failed
//
// Backend expõe /v1/events/stream via proxy Next.js edge:
//   /v1-api/events/stream (lê cookie httpOnly rn_jwt server-side)
//
// Edge runtime é requirement pra streaming real — Node runtime buffers
// chunks de 4KB e adiciona latência perceptível em SSE.

'use client'

import { useEffect, useRef, useState } from 'react'

export type StreamStatus = 'idle' | 'connecting' | 'open' | 'reconnecting' | 'failed'

export interface StreamEvent<T = Record<string, unknown>> {
  kind: string
  if_id?: string
  payload: T
  timestamp: string
}

export interface UseEventStreamOptions {
  /** Filter client-side: só eventos com kind matching são chamados no onEvent. Default: all. */
  kinds?: string[]
  /** Callback chamado em cada evento. */
  onEvent?: (evt: StreamEvent) => void
  /** Callback em mudanças de status (pra UI badge). */
  onStatusChange?: (status: StreamStatus) => void
  /** Se false, não conecta (útil pra desabilitar em dev). Default: true. */
  enabled?: boolean
}

export interface UseEventStreamResult {
  status: StreamStatus
  /** Quantidade de eventos recebidos desde o mount. */
  eventCount: number
  /** Último erro (parse ou network). */
  lastError: string | null
  /** Reconecta manualmente (fecha + abre). */
  reconnect: () => void
}

const BACKOFF_MS = [1000, 2000, 4000, 8000, 16000, 30000] // max 30s
const HEARTBEAT_TIMEOUT_MS = 60_000 // backend envia a cada 30s

export function useEventStream(opts: UseEventStreamOptions = {}): UseEventStreamResult {
  const { kinds, onEvent, onStatusChange, enabled = true } = opts
  const [status, setStatus] = useState<StreamStatus>('idle')
  const [eventCount, setEventCount] = useState(0)
  const [lastError, setLastError] = useState<string | null>(null)

  // Refs pra closures estáveis (evita re-run do effect)
  const onEventRef = useRef(onEvent)
  const onStatusRef = useRef(onStatusChange)
  const kindsRef = useRef(kinds)
  const abortRef = useRef<AbortController | null>(null)
  const attemptRef = useRef(0)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    onEventRef.current = onEvent
    onStatusRef.current = onStatusChange
    kindsRef.current = kinds
  }, [onEvent, onStatusChange, kinds])

  const updateStatus = (s: StreamStatus) => {
    setStatus(s)
    onStatusRef.current?.(s)
  }

  const connect = () => {
    if (!enabled) return
    if (abortRef.current) abortRef.current.abort()

    const ctrl = new AbortController()
    abortRef.current = ctrl

    updateStatus(attemptRef.current === 0 ? 'connecting' : 'reconnecting')

    fetch('/v1-api/events/stream', {
      signal: ctrl.signal,
      headers: { Accept: 'text/event-stream' },
    })
      .then(async (resp) => {
        if (!resp.ok || !resp.body) {
          throw new Error(`HTTP ${resp.status}`)
        }

        attemptRef.current = 0 // reset backoff após sucesso
        updateStatus('open')
        setLastError(null)

        const reader = resp.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        let lastEventAt = Date.now()

        const watchdog = setInterval(() => {
          // Se não recebemos nada (nem heartbeat) em 60s, reconecta.
          if (Date.now() - lastEventAt > HEARTBEAT_TIMEOUT_MS) {
            setLastError('heartbeat timeout')
            ctrl.abort()
          }
        }, 15_000)

        try {
          while (true) {
            const { done, value } = await reader.read()
            if (done) break

            lastEventAt = Date.now()
            buffer += decoder.decode(value, { stream: true })

            // SSE events são separados por \n\n. Processa eventos completos.
            let boundary: number
            // eslint-disable-next-line no-cond-assign
            while ((boundary = buffer.indexOf('\n\n')) !== -1) {
              const raw = buffer.slice(0, boundary)
              buffer = buffer.slice(boundary + 2)
              const parsed = parseSSEBlock(raw)
              if (!parsed) continue

              // Skip non-data events (heartbeat, comment)
              if (parsed.event === 'heartbeat' || parsed.event === 'connected') {
                continue
              }

              let evt: StreamEvent
              try {
                evt = JSON.parse(parsed.data) as StreamEvent
              } catch (e) {
                setLastError(`parse: ${(e as Error).message}`)
                continue
              }

              // Filter por kind (se configurado)
              if (kindsRef.current && kindsRef.current.length > 0) {
                if (!kindsRef.current.includes(evt.kind)) continue
              }

              setEventCount((c) => c + 1)
              onEventRef.current?.(evt)
            }
          }
        } finally {
          clearInterval(watchdog)
        }
      })
      .catch((err: Error) => {
        if (err.name === 'AbortError') return // intencional
        setLastError(err.message)
      })
      .finally(() => {
        // Se ainda não fechamos intencionalmente, agenda reconnect
        if (abortRef.current === ctrl) {
          const delay = BACKOFF_MS[Math.min(attemptRef.current, BACKOFF_MS.length - 1)]
          attemptRef.current++
          updateStatus('reconnecting')
          reconnectTimerRef.current = setTimeout(connect, delay)
        }
      })
  }

  const reconnect = () => {
    attemptRef.current = 0
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
    if (abortRef.current) {
      abortRef.current.abort()
      abortRef.current = null
    }
    connect()
  }

  useEffect(() => {
    if (!enabled) {
      updateStatus('idle')
      return
    }
    connect()
    return () => {
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
      if (abortRef.current) {
        abortRef.current.abort()
        abortRef.current = null
      }
      updateStatus('idle')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled])

  return { status, eventCount, lastError, reconnect }
}

interface SSEBlock {
  event: string
  data: string
}

function parseSSEBlock(raw: string): SSEBlock | null {
  if (!raw.trim()) return null
  const lines = raw.split('\n')
  let event = 'message'
  let data = ''
  for (const line of lines) {
    if (line.startsWith(':')) continue // comment / heartbeat
    if (line.startsWith('event: ')) {
      event = line.slice(7).trim()
    } else if (line.startsWith('data: ')) {
      data += (data ? '\n' : '') + line.slice(6)
    }
  }
  if (!data && event === 'message') return null
  return { event, data }
}