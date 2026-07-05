// useRulePreferences — hook que sincroniza regras desabilitadas com backend.
//
// Sprint 11: antes usava localStorage. Agora backend persiste por IF, emite
// audit event, frontend sincroniza via API. Optimistic concurrency via
// expected_state no body (409 se estado mudou entre fetch e toggle).
//
// Pattern: source of truth é backend. Local state é cache que reflete backend.
// Em caso de 409, refetch + mostra toast/aviso.

'use client'

import { useCallback, useEffect, useState } from 'react'

export interface UseRulePreferencesResult {
  /** Set de rule_codes desabilitadas (do backend). */
  disabled: Set<string>
  /** Se está carregando initial state. */
  loading: boolean
  /** Toggle uma regra. Retorna new_state ou error. */
  toggle: (code: string) => Promise<{ newState: string } | { error: string }>
  /** Refresh manual. */
  refresh: () => Promise<void>
}

export function useRulePreferences(): UseRulePreferencesResult {
  const [disabled, setDisabled] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const resp = await fetch('/api/rules/disabled')
      if (!resp.ok) {
        if (resp.status === 401) {
          // sem auth — não é erro fatal, apenas estado vazio
          setDisabled(new Set())
          return
        }
        throw new Error(`HTTP ${resp.status}`)
      }
      const data = (await resp.json()) as { codes: string[] }
      setDisabled(new Set(data.codes ?? []))
    } catch (e) {
      console.error('[useRulePreferences] refresh failed:', e)
      // Manter estado anterior em caso de erro de rede
    } finally {
      setLoading(false)
    }
  }, [])

  // Initial load
  useEffect(() => {
    refresh()
  }, [refresh])

  const toggle = useCallback(
    async (code: string) => {
      // Compute expected_state antes do POST
      const expectedState = disabled.has(code) ? 'disabled' : 'enabled'

      try {
        const resp = await fetch(`/api/rules/${encodeURIComponent(code)}/toggle`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ expected_state: expectedState }),
        })

        if (resp.status === 409) {
          // State mismatch — refetch e retorna erro
          await refresh()
          return { error: 'state_changed' }
        }

        if (!resp.ok) {
          return { error: `HTTP ${resp.status}` }
        }

        const data = (await resp.json()) as { new_state: string }
        // Update local state com new_state
        setDisabled((prev) => {
          const next = new Set(prev)
          if (data.new_state === 'disabled') {
            next.add(code)
          } else {
            next.delete(code)
          }
          return next
        })
        return { newState: data.new_state }
      } catch (e) {
        return { error: (e as Error).message }
      }
    },
    [disabled, refresh],
  )

  return { disabled, loading, toggle, refresh }
}