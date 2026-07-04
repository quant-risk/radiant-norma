'use client'

/**
 * ExportMenu — botão dropdown com ações de export.
 *
 * Sprint 8d: power users reproduzem views. Ações:
 *   - CSV: download direto do CSV via fetch → blob
 *   - JSON: download direto do JSON
 *   - Copiar URL: copia URL atual (com filtros aplicados)
 *
 * Requer endpoint que aceite ?format=csv|json (Sprint 8d backend).
 */

import { useState } from 'react'
import { Download, FileJson, FileSpreadsheet, Link2, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export interface ExportMenuProps {
  /** Endpoint base (ex: '/v1/envios') */
  endpoint: string
  /** Filtros aplicados (vão como query params) */
  filters?: Record<string, string | undefined>
  /** Label do botão trigger */
  label?: string
  /** Variant do botão */
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost'
  /** Size do botão */
  size?: 'sm' | 'md' | 'lg'
}

export function ExportMenu({
  endpoint,
  filters = {},
  label = 'Exportar',
  variant = 'outline',
  size = 'sm',
}: ExportMenuProps) {
  const [open, setOpen] = useState(false)
  const [copying, setCopying] = useState(false)
  const [copied, setCopied] = useState(false)
  const [loading, setLoading] = useState<'csv' | 'json' | null>(null)

  function buildQuery(format: string): string {
    const params = new URLSearchParams({ format, ...stripUndefined(filters) })
    return `${endpoint}?${params.toString()}`
  }

  async function downloadFile(format: 'csv' | 'json') {
    setLoading(format)
    try {
      const res = await fetch(buildQuery(format), {
        credentials: 'include', // envia cookie rn_jwt
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const blob = await res.blob()

      // Extrai filename do Content-Disposition ou gera fallback
      const cd = res.headers.get('Content-Disposition') ?? ''
      const match = cd.match(/filename="?([^";]+)"?/)
      const filename = match?.[1] ?? `${endpoint.replace(/\//g, '-')}-${Date.now()}.${format}`

      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      setOpen(false)
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('export failed', e)
    } finally {
      setLoading(null)
    }
  }

  async function copyURL() {
    setCopying(true)
    try {
      const url = new URL(window.location.href)
      // Remove params internos do Next (ex: _rsc)
      Array.from(url.searchParams.keys())
        .filter((k) => k.startsWith('_'))
        .forEach((k) => url.searchParams.delete(k))
      await navigator.clipboard.writeText(url.toString())
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('copy failed', e)
    } finally {
      setCopying(false)
    }
  }

  return (
    <div className="relative">
      <Button
        variant={variant}
        size={size}
        onClick={() => setOpen((o) => !o)}
        leftIcon={<Download className="size-3.5" />}
      >
        {label}
      </Button>

      {open && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setOpen(false)}
            aria-hidden
          />

          {/* Dropdown */}
          <div
            className={cn(
              'absolute right-0 top-full mt-1 z-50 w-56',
              'bg-surface-raised border border-border rounded-lg shadow-lg',
              'overflow-hidden animate-fade-in-fast',
            )}
          >
            <div className="px-3 py-2 text-2xs uppercase tracking-wider font-semibold text-ink-subtle border-b border-border-subtle">
              Exportar
            </div>
            <button
              onClick={() => downloadFile('csv')}
              disabled={loading === 'csv'}
              className="w-full flex items-center gap-3 px-3 py-2 text-sm text-ink hover:bg-surface-sunken transition-colors disabled:opacity-50"
            >
              <FileSpreadsheet className="size-3.5 text-ink-muted" />
              <span className="flex-1 text-left">
                {loading === 'csv' ? 'Baixando…' : 'CSV (Excel/Sheets)'}
              </span>
            </button>
            <button
              onClick={() => downloadFile('json')}
              disabled={loading === 'json'}
              className="w-full flex items-center gap-3 px-3 py-2 text-sm text-ink hover:bg-surface-sunken transition-colors disabled:opacity-50"
            >
              <FileJson className="size-3.5 text-ink-muted" />
              <span className="flex-1 text-left">
                {loading === 'json' ? 'Baixando…' : 'JSON (raw)'}
              </span>
            </button>
            <div className="border-t border-border-subtle" />
            <button
              onClick={copyURL}
              disabled={copying}
              className="w-full flex items-center gap-3 px-3 py-2 text-sm text-ink hover:bg-surface-sunken transition-colors disabled:opacity-50"
            >
              {copied ? (
                <Check className="size-3.5 text-success-600" />
              ) : (
                <Link2 className="size-3.5 text-ink-muted" />
              )}
              <span className="flex-1 text-left">
                {copied ? 'URL copiada!' : copying ? 'Copiando…' : 'Copiar URL'}
              </span>
            </button>
          </div>
        </>
      )}
    </div>
  )
}

function stripUndefined(
  obj: Record<string, string | undefined>,
): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(obj)) {
    if (v !== undefined && v !== '') out[k] = v
  }
  return out
}