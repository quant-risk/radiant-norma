'use client'

/**
 * useFocusTrap — hook para prender foco dentro de um container (modal/dialog).
 *
 * Comportamento:
 *   - Salva `document.activeElement` ao montar
 *   - Foca o primeiro elemento focável dentro do container
 *   - Tab/Shift+Tab ciclam entre primeiro e último (não escapam)
 *   - Restaura foco ao elemento original ao desmontar
 *   - Esc não é capturado aqui (modal decide se fecha ou não)
 *
 * Uso:
 *   const ref = useFocusTrap<HTMLDivElement>(open)
 *   <div ref={ref} role="dialog" ...>
 */
import * as React from 'react'

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'textarea:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

export function useFocusTrap<T extends HTMLElement>(active: boolean) {
  const ref = React.useRef<T>(null)
  const previouslyFocused = React.useRef<HTMLElement | null>(null)

  React.useEffect(() => {
    if (!active) return
    // Salva elemento focado antes do modal abrir (geralmente o trigger)
    previouslyFocused.current = document.activeElement as HTMLElement | null

    const node = ref.current
    if (!node) return

    // Foca o primeiro focável após microtask (deixa render montar)
    const t = setTimeout(() => {
      const focusables = node.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)
      const first = focusables[0]
      if (first) {
        first.focus()
      } else {
        // Sem focáveis internos — foca o container como fallback
        node.setAttribute('tabindex', '-1')
        node.focus()
      }
    }, 0)

    return () => {
      clearTimeout(t)
      // Restaura foco ao elemento original
      previouslyFocused.current?.focus?.()
    }
  }, [active])

  function handleKeyDown(e: React.KeyboardEvent<T>) {
    if (e.key !== 'Tab' || !active) return
    const node = ref.current
    if (!node) return
    const focusables = Array.from(
      node.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR),
    ).filter((el) => !el.hasAttribute('disabled') && el.offsetParent !== null)
    if (focusables.length === 0) {
      e.preventDefault()
      return
    }
    const first = focusables[0]
    const last = focusables[focusables.length - 1]
    const activeEl = document.activeElement as HTMLElement

    if (e.shiftKey) {
      // Shift+Tab do primeiro → vai pro último
      if (activeEl === first || !node.contains(activeEl)) {
        e.preventDefault()
        last.focus()
      }
    } else {
      // Tab do último → volta pro primeiro
      if (activeEl === last || !node.contains(activeEl)) {
        e.preventDefault()
        first.focus()
      }
    }
  }

  return { ref, onKeyDown: handleKeyDown }
}