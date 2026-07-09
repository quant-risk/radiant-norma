'use client'

/**
 * WizardClient — client island que gerencia o estado do wizard.
 * Recebe session data como props (serializáveis) do server component.
 */
import { useState } from 'react'
import {
  Upload, Wand2, CheckCircle2, AlertCircle,
  ChevronRight, ChevronLeft, Download, RefreshCw,
  FileSpreadsheet, Table2, Code2,
} from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

// ─── Types ────────────────────────────────────────────────────────────────────

type WizardStep = 'upload' | 'preview' | 'generate'

interface ParsedDoc {
  if_id: string
  cadoc_code: string
  header: {
    cnpj: string
    nome_if: string
  }
  operacoes: Array<{
    id: string
    modalidade: string
    tipo_pessoa: string
    valor_principal: { valor: number; moeda: string }
    taxa_juros?: number
    indexador?: string
    uf?: string
  }>
  extra: Record<string, unknown>
}

interface GeneratedResult {
  xml: string
  sha256: string
  cadoc_code: string
  status: string
  message: string
}

const STEPS: { id: WizardStep; label: string; icon: React.ElementType }[] = [
  { id: 'upload', label: 'Upload', icon: Upload },
  { id: 'preview', label: 'Preview', icon: Table2 },
  { id: 'generate', label: 'Gerar XML', icon: Code2 },
]

// ─── Step Indicator ───────────────────────────────────────────────────────────

function StepIndicator({ current, steps }: { current: WizardStep; steps: typeof STEPS }) {
  const currentIdx = steps.findIndex((s) => s.id === current)
  return (
    <div className="flex items-center gap-0">
      {steps.map((step, i) => {
        const Icon = step.icon
        const done = i < currentIdx
        const active = i === currentIdx
        return (
          <div key={step.id} className="flex items-center">
            <div
              className={cn(
                'flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium transition-all',
                done && 'bg-accent-100 dark:bg-accent-900 text-accent-700 dark:text-accent-300',
                active && 'bg-accent-600 text-white shadow-sm',
                !done && !active && 'text-ink-muted bg-surface-sunken',
              )}
            >
              {done ? <CheckCircle2 className="size-3.5" /> : <Icon className="size-3.5" />}
              <span>{step.label}</span>
            </div>
            {i < steps.length - 1 && (
              <ChevronRight className="size-3.5 text-ink-subtle mx-1" />
            )}
          </div>
        )
      })}
    </div>
  )
}

// ─── Step 1: Upload ───────────────────────────────────────────────────────────

function UploadStep({ onParsed }: { onParsed: (doc: ParsedDoc) => void }) {
  const [cadoc, setCadoc] = useState('3040')
  const [period, setPeriod] = useState('2026-07')
  const [file, setFile] = useState<File | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    const f = e.dataTransfer.files[0]
    if (f) setFile(f)
  }

  const handleSubmit = async () => {
    if (!file || !cadoc) return
    setLoading(true)
    setError(null)
    const fd = new FormData()
    fd.append('cadoc', cadoc)
    fd.append('data_base', `${period}-01`)
    fd.append('file', file)
    fd.append('has_header', 'true')
    try {
      const res = await fetch('/api/generate/wizard', { method: 'POST', body: fd })
      const data = await res.json()
      if (!res.ok) throw new Error(data.message || data.error || 'Erro ao parsear')
      onParsed(data.document as ParsedDoc)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div className="space-y-2">
        <label className="text-xs font-medium text-ink-muted uppercase tracking-wider">Tipo de Documento</label>
        <select
          value={cadoc}
          onChange={(e) => setCadoc(e.target.value)}
          className="w-full h-10 px-3 rounded-lg border border-border bg-surface text-sm text-ink focus:outline-none focus:ring-2 focus:ring-accent-500"
        >
          <option value="3040">3040 — SCR (Risco de Crédito)</option>
          <option value="3050">3050 — TXB (Taxas Bancárias)</option>
        </select>
      </div>

      <div className="space-y-2">
        <label className="text-xs font-medium text-ink-muted uppercase tracking-wider">Mês de Referência</label>
        <input
          type="month"
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
          className="w-full h-10 px-3 rounded-lg border border-border bg-surface text-sm text-ink focus:outline-none focus:ring-2 focus:ring-accent-500"
        />
      </div>

      <div className="space-y-2">
        <label className="text-xs font-medium text-ink-muted uppercase tracking-wider">Arquivo (CSV ou XLSX)</label>
        <div
          className={cn(
            'border-2 border-dashed rounded-xl p-10 text-center cursor-pointer transition-colors',
            'hover:border-accent-400 hover:bg-accent-50/30 dark:hover:bg-accent-950/20',
            file ? 'border-accent-400 bg-accent-50/20' : 'border-border',
          )}
          onDragOver={(e) => e.preventDefault()}
          onDrop={handleDrop}
          onClick={() => {
            const input = document.createElement('input')
            input.type = 'file'
            input.accept = '.csv,.xlsx'
            input.onchange = (e) => {
              const f = (e.target as HTMLInputElement).files?.[0]
              if (f) setFile(f)
            }
            input.click()
          }}
        >
          {file ? (
            <div className="flex flex-col items-center gap-2">
              <FileSpreadsheet className="size-8 text-accent-600" />
              <span className="text-sm font-medium text-ink">{file.name}</span>
              <span className="text-xs text-ink-muted">{(file.size / 1024).toFixed(1)} KB</span>
            </div>
          ) : (
            <div className="text-ink-muted">
              <Upload className="size-8 mx-auto mb-2 opacity-50" />
              <p className="text-sm">Arraste ou clique para selecionar</p>
              <p className="text-xs mt-1">CSV ou XLSX · Máximo 10MB</p>
            </div>
          )}
        </div>
      </div>

      {error && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-critical-50 dark:bg-critical-950 border border-critical-200 dark:border-critical-800 text-critical-700 dark:text-critical-300 text-sm">
          <AlertCircle className="size-4 mt-0.5 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      <Button
        variant="primary"
        size="md"
        onClick={handleSubmit}
        loading={loading}
        disabled={!file || !cadoc}
        rightIcon={<ChevronRight className="size-4" />}
      >
        Parsear arquivo
      </Button>
    </div>
  )
}

// ─── Step 2: Preview ──────────────────────────────────────────────────────────

function PreviewStep({ doc, onBack, onNext }: { doc: ParsedDoc; onBack: () => void; onNext: () => void }) {
  return (
    <div className="space-y-6">
      <Card padding="md" className="max-w-2xl">
        <div className="flex items-center gap-3 mb-3">
          <CheckCircle2 className="size-5 text-success-600" />
          <span className="text-sm font-medium text-ink">Documento parseado</span>
        </div>
        <div className="grid grid-cols-3 gap-4 text-sm">
          <div>
            <div className="text-xs text-ink-muted mb-1">CNPJ</div>
            <div className="font-mono text-ink">{doc.header.cnpj || '—'}</div>
          </div>
          <div>
            <div className="text-xs text-ink-muted mb-1">Instituição</div>
            <div className="text-ink truncate">{doc.header.nome_if || '—'}</div>
          </div>
          <div>
            <div className="text-xs text-ink-muted mb-1">Operações</div>
            <div className="font-mono text-ink">{doc.operacoes.length}</div>
          </div>
        </div>
      </Card>

      <div className="space-y-2">
        <h3 className="text-xs font-medium text-ink-muted uppercase tracking-wider">
          Operações ({doc.operacoes.length})
        </h3>
        <Card padding="none" className="overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-sunken">
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted">ID</th>
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted">Modalidade</th>
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted">Tipo</th>
                  <th className="text-right px-4 py-2.5 text-xs font-medium text-ink-muted">Valor Principal</th>
                  <th className="text-right px-4 py-2.5 text-xs font-medium text-ink-muted">Taxa Juros</th>
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted">Indexador</th>
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted">UF</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border-subtle">
                {doc.operacoes.map((op, i) => (
                  <tr key={op.id || i} className="hover:bg-surface-sunken/30 transition-colors">
                    <td className="px-4 py-2.5 font-mono text-xs text-ink-muted">{op.id || `op-${i + 1}`}</td>
                    <td className="px-4 py-2.5 font-mono text-xs text-ink">{op.modalidade || '—'}</td>
                    <td className="px-4 py-2.5 text-xs text-ink">{op.tipo_pessoa || '—'}</td>
                    <td className="px-4 py-2.5 text-right font-mono text-xs text-ink">
                      {op.valor_principal
                        ? op.valor_principal.valor.toLocaleString('pt-BR', {
                            style: 'currency',
                            currency: op.valor_principal.moeda || 'BRL',
                          })
                        : '—'}
                    </td>
                    <td className="px-4 py-2.5 text-right font-mono text-xs text-ink">
                      {op.taxa_juros != null
                        ? `${(op.taxa_juros * 100).toFixed(4)}%`
                        : '—'}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-ink-muted">{op.indexador || '—'}</td>
                    <td className="px-4 py-2.5 text-xs text-ink">{op.uf || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      </div>

      <div className="flex gap-3">
        <Button variant="ghost" size="md" onClick={onBack} leftIcon={<ChevronLeft className="size-4" />}>
          Voltar
        </Button>
        <Button variant="primary" size="md" onClick={onNext} rightIcon={<ChevronRight className="size-4" />}>
          Gerar XML
        </Button>
      </div>
    </div>
  )
}

// ─── Step 3: Generate ─────────────────────────────────────────────────────────

function GenerateStep({ doc, cadoc, onBack }: { doc: ParsedDoc; cadoc: string; onBack: () => void }) {
  const [result, setResult] = useState<GeneratedResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleGenerate = async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/generate/xml', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          cadoc_code: cadoc,
          if_id: doc.if_id,
          cnpj: doc.header.cnpj,
          nome_if: doc.header.nome_if,
          extra: doc.extra,
          operacoes: doc.operacoes,
        }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.message || data.error || 'Erro ao gerar')
      setResult({
        xml: atob(data.generated?.xml || ''),
        sha256: data.generated?.sha256 || '',
        cadoc_code: data.cadoc_code,
        status: data.status,
        message: data.message,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = () => {
    if (!result) return
    const blob = new Blob([result.xml], { type: 'application/xml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${cadoc}_${new Date().toISOString().slice(0, 7)}.xml`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="space-y-4">
      {!result ? (
        <div className="space-y-4 max-w-lg">
          <div className="rounded-xl border border-dashed border-border p-8 text-center">
            <Wand2 className="size-10 mx-auto mb-3 text-accent-600 opacity-60" />
            <h3 className="text-base font-serif font-medium text-ink mb-2">
              Pronto para gerar o XML
            </h3>
            <p className="text-sm text-ink-muted mb-5">
              {doc.operacoes.length} operação(ões) · CADOC {cadoc}
            </p>
            <Button
              variant="primary"
              size="lg"
              onClick={handleGenerate}
              loading={loading}
              leftIcon={<Wand2 className="size-4" />}
            >
              Gerar XML
            </Button>
          </div>
          {error && (
            <div className="flex items-start gap-2 p-3 rounded-lg bg-critical-50 dark:bg-critical-950 border border-critical-200 dark:border-critical-800 text-critical-700 dark:text-critical-300 text-sm">
              <AlertCircle className="size-4 mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}
        </div>
      ) : (
        <div className="space-y-4">
          <div className="flex items-center gap-2 p-3 rounded-lg bg-success-50 dark:bg-success-950 border border-success-200 dark:border-success-800 text-success-700 dark:text-success-300 text-sm">
            <CheckCircle2 className="size-4 shrink-0" />
            <span>{result.message}</span>
            <span className="ml-auto font-mono text-xs opacity-70">{result.sha256.slice(0, 12)}…</span>
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-medium text-ink-muted uppercase tracking-wider">XML Gerado</h3>
              <Badge tone="success" variant="soft" size="sm">
                {(result.xml.length / 1024).toFixed(1)} KB
              </Badge>
            </div>
            <Card padding="none" className="overflow-hidden">
              <pre className="bg-[#0d1117] text-green-400 text-xs p-4 overflow-x-auto max-h-80 leading-relaxed">
                <code>{result.xml.slice(0, 4000)}{result.xml.length > 4000 && '\n… (truncated preview)'}</code>
              </pre>
            </Card>
          </div>
          <div className="flex gap-3">
            <Button variant="primary" size="md" onClick={handleDownload} leftIcon={<Download className="size-4" />}>
              Baixar XML
            </Button>
            <Button variant="ghost" size="md" onClick={() => setResult(null)} leftIcon={<RefreshCw className="size-4" />}>
              Regenerar
            </Button>
          </div>
        </div>
      )}
      <Button variant="ghost" size="md" onClick={onBack} leftIcon={<ChevronLeft className="size-4" />}>
        Voltar ao preview
      </Button>
    </div>
  )
}

// ─── Wizard Client Component ───────────────────────────────────────────────────

export function WizardClient() {
  const [step, setStep] = useState<WizardStep>('upload')
  const [parsedDoc, setParsedDoc] = useState<ParsedDoc | null>(null)

  return (
    <div className="space-y-8 max-w-4xl">
      <StepIndicator current={step} steps={STEPS} />
      <div className="mt-6">
        {step === 'upload' && (
          <UploadStep onParsed={(doc) => { setParsedDoc(doc); setStep('preview') }} />
        )}
        {step === 'preview' && parsedDoc && (
          <PreviewStep
            doc={parsedDoc}
            onBack={() => setStep('upload')}
            onNext={() => setStep('generate')}
          />
        )}
        {step === 'generate' && parsedDoc && (
          <GenerateStep
            doc={parsedDoc}
            cadoc={parsedDoc.cadoc_code}
            onBack={() => setStep('preview')}
          />
        )}
      </div>
    </div>
  )
}
