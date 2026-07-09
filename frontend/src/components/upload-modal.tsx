'use client'

import { useState } from 'react'
import { Upload, X, FileText, AlertCircle, CheckCircle2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface UploadModalProps {
  isOpen: boolean
  onClose: () => void
  cadocs: string[]
  onSuccess: () => void
  token: string
}

export function UploadModal({ isOpen, onClose, cadocs, onSuccess, token }: UploadModalProps) {
  const [cadoc, setCadoc] = useState('')
  const [period, setPeriod] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  if (!isOpen) return null

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!cadoc || !period || !file) {
      setError('Preencha todos os campos')
      return
    }

    setUploading(true)
    setError(null)

    try {
      // Read file as text (for demo - in production would send XML)
      const xmlContent = await file.text()

      const res = await fetch('/api/sta/submit', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          cadoc_code: cadoc,
          data_base: `${period}-01`,
          xml: xmlContent,
          cnpj: 'demo',
        }),
      })

      if (!res.ok) {
        const data = await res.json().catch(() => ({ error: 'Upload failed' }))
        throw new Error(data.error || 'Upload failed')
      }

      setSuccess(true)
      setTimeout(() => {
        onSuccess()
        onClose()
        setSuccess(false)
        setCadoc('')
        setPeriod('')
        setFile(null)
      }, 1500)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao fazer upload')
    } finally {
      setUploading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />
      
      {/* Modal */}
      <div className="relative bg-surface-raised rounded-xl shadow-2xl w-full max-w-md mx-4 border border-border">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-border">
          <div className="flex items-center gap-3">
            <div className="size-10 rounded-lg bg-accent-600 flex items-center justify-center">
              <Upload className="size-5 text-white" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-ink">Novo Envio STA</h2>
              <p className="text-xs text-ink-muted">Envio de documento regulatório</p>
            </div>
          </div>
          <button 
            onClick={onClose}
            className="size-8 rounded-lg hover:bg-surface-sunken flex items-center justify-center text-ink-muted hover:text-ink transition-colors"
          >
            <X className="size-4" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {/* CADOC Select */}
          <div className="space-y-2">
            <label className="text-xs font-medium text-ink-muted uppercase tracking-wider">
              Tipo de Documento (CADOC)
            </label>
            <select
              value={cadoc}
              onChange={(e) => setCadoc(e.target.value)}
              className={cn(
                'w-full h-10 px-3 rounded-lg border border-border bg-surface',
                'text-sm text-ink',
                'focus:outline-none focus:ring-2 focus:ring-accent-500 focus:border-accent-500',
                'transition-colors'
              )}
            >
              <option value="">Selecione o CADOC...</option>
              {cadocs.map((c) => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          </div>

          {/* Period */}
          <div className="space-y-2">
            <label className="text-xs font-medium text-ink-muted uppercase tracking-wider">
              Mês de Referência
            </label>
            <input
              type="month"
              value={period}
              onChange={(e) => setPeriod(e.target.value)}
              className={cn(
                'w-full h-10 px-3 rounded-lg border border-border bg-surface',
                'text-sm text-ink',
                'focus:outline-none focus:ring-2 focus:ring-accent-500 focus:border-accent-500',
                'transition-colors'
              )}
            />
          </div>

          {/* File Upload */}
          <div className="space-y-2">
            <label className="text-xs font-medium text-ink-muted uppercase tracking-wider">
              Arquivo XML
            </label>
            <div
              className={cn(
                'border-2 border-dashed rounded-lg p-6 text-center',
                'hover:border-accent-400 hover:bg-accent-50/50 dark:hover:bg-accent-950/20',
                'transition-colors cursor-pointer',
                file ? 'border-accent-400 bg-accent-50/30 dark:bg-accent-950/20' : 'border-border'
              )}
              onClick={() => document.getElementById('file-input')?.click()}
            >
              <input
                id="file-input"
                type="file"
                accept=".xml"
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0]
                  if (f) setFile(f)
                }}
              />
              {file ? (
                <div className="flex items-center justify-center gap-2 text-accent-600 dark:text-accent-400">
                  <FileText className="size-5" />
                  <span className="text-sm font-medium">{file.name}</span>
                </div>
              ) : (
                <div className="text-ink-muted">
                  <Upload className="size-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">Clique ou arraste o arquivo XML aqui</p>
                  <p className="text-xs mt-1">Máximo 10MB</p>
                </div>
              )}
            </div>
          </div>

          {/* Error */}
          {error && (
            <div className="flex items-start gap-2 p-3 rounded-lg bg-critical-50 dark:bg-critical-950 border border-critical-200 dark:border-critical-800 text-critical-700 dark:text-critical-300 text-sm">
              <AlertCircle className="size-4 mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Success */}
          {success && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-success-50 dark:bg-success-950 border border-success-200 dark:border-success-800 text-success-700 dark:text-success-300 text-sm">
              <CheckCircle2 className="size-4 shrink-0" />
              <span>Envio realizado com sucesso!</span>
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-3 pt-2">
            <Button
              type="button"
              variant="ghost"
              onClick={onClose}
              className="flex-1"
            >
              Cancelar
            </Button>
            <Button
              type="submit"
              variant="primary"
              loading={uploading}
              disabled={!cadoc || !period || !file || uploading || success}
              className="flex-1"
            >
              {uploading ? 'Enviando...' : 'Enviar'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}
