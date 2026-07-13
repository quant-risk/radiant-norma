'use client'

/**
 * WizardClient — client island que gerencia o estado do wizard.
 * 6 passos: upload → mapear → preview → validar → gerar → enviar STA
 *
 * Estado persiste na URL (?step=upload&cadoc=3040) — refresh não perde estado.
 * Recebe cadoc via ?cadoc=3040 do link em /console/generate.
 */
import { useState, useCallback, useEffect } from 'react'
import { useSearchParams, useRouter, usePathname } from 'next/navigation'
import {
  Upload, Wand2, CheckCircle2, AlertCircle,
  ChevronRight, ChevronLeft, Download, RefreshCw,
  FileSpreadsheet, Table2, Code2, Send, ArrowRight,
  Map as MapIcon,
} from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { SectionHeader } from '@/components/ui/section-header'
import { cn } from '@/lib/utils'

// ─── Canonical field descriptors for mapping UI ──────────────────────────────────

const CANONICAL_FIELDS = [
  { value: '', label: '— não mapear —' },
  { value: 'id', label: 'ID da operação' },
  { value: 'modalidade', label: 'Modalidade (código COSIF)' },
  { value: 'tipo_pessoa', label: 'Tipo pessoa (PF/PJ)' },
  { value: 'tipo_operacao', label: 'Tipo operação (C/CFL/D)' },
  { value: 'numero_contrato', label: 'Número contrato' },
  { value: 'valor', label: 'Valor principal' },
  { value: 'encargos', label: 'Encargos totais' },
  { value: 'iof', label: 'IOF' },
  { value: 'valor_atualizado', label: 'Valor atualizado' },
  { value: 'taxa_juros', label: 'Taxa de juros' },
  { value: 'taxa_spread', label: 'Taxa spread' },
  { value: 'percentual_indexador', label: 'Percentual indexador' },
  { value: 'percentual_provisao', label: 'Percentual provisão' },
  { value: 'indexador', label: 'Indexador (CDI/IPCA/PRE)' },
  { value: 'faixa_vencimento', label: 'Faixa vencimento (V0-V5)' },
  { value: 'nivel_risco', label: 'Nível risco (AA/B/C…)' },
  { value: 'classificacao_if', label: 'Classificação IF' },
  { value: 'uf', label: 'UF' },
  { value: 'pais', label: 'País' },
  { value: 'data_vencimento', label: 'Data vencimento' },
  { value: 'data_constituicao', label: 'Data constituição' },
] as const

// ─── Types ────────────────────────────────────────────────────────────────────

type WizardStep = 'upload' | 'map' | 'preview' | 'validate' | 'generate' | 'sta'

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
  { id: 'map', label: 'Mapear', icon: MapIcon },
  { id: 'preview', label: 'Preview', icon: Table2 },
  { id: 'validate', label: 'Validar', icon: CheckCircle2 },
  { id: 'generate', label: 'Gerar XML', icon: Code2 },
  { id: 'sta', label: 'Enviar STA', icon: Send },
]

// ─── Storage key for mapping profiles ────────────────────────────────────────

const MAPPING_STORAGE_KEY = 'radiant:mapping_profiles'

type ColumnMapping = Record<string, string> // column → canonical field

function loadMappingProfile(cadoc: string): ColumnMapping | null {
  try {
    const raw = localStorage.getItem(MAPPING_STORAGE_KEY)
    if (!raw) return null
    const profiles: Record<string, ColumnMapping> = JSON.parse(raw)
    return profiles[cadoc] ?? null
  } catch {
    return null
  }
}

function saveMappingProfile(cadoc: string, mapping: ColumnMapping) {
  try {
    const raw = localStorage.getItem(MAPPING_STORAGE_KEY)
    const profiles: Record<string, ColumnMapping> = raw ? JSON.parse(raw) : {}
    profiles[cadoc] = mapping
    localStorage.setItem(MAPPING_STORAGE_KEY, JSON.stringify(profiles))
  } catch {
    // ignore
  }
}

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

interface UploadStepProps {
  defaultCadoc?: string
  onNext: (doc: ParsedDoc, file: File) => void
}

function UploadStep({ defaultCadoc, onNext }: UploadStepProps) {
  const [cadoc, setCadoc] = useState(defaultCadoc ?? '3040')
  const [period, setPeriod] = useState('2026-07')
  const [file, setFile] = useState<File | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isDragOver, setIsDragOver] = useState(false)

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragOver(false)
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
      onNext(data.document as ParsedDoc, file)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <SectionHeader
        eyebrow="Passo 1 de 6"
        title="Upload do arquivo"
        description="Selecione o CADOC, mês de referência e faça upload da planilha."
      />
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
            'border-2 border-dashed rounded-xl p-10 text-center cursor-pointer transition-all',
            isDragOver
              ? 'border-accent-500 bg-accent-50/40 dark:bg-accent-950/30 scale-[1.01]'
              : 'border-border hover:border-accent-400 hover:bg-accent-50/20 dark:hover:bg-accent-950/15',
            file ? 'border-accent-400 bg-accent-50/10' : '',
          )}
          onDragEnter={() => setIsDragOver(true)}
          onDragLeave={() => setIsDragOver(false)}
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
              <p className="text-xs mt-1">CSV ou XLSX · Máximo 50MB</p>
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
        Parsear e continuar
      </Button>
    </div>
  )
}

// ─── Step 2: Map ──────────────────────────────────────────────────────────────

interface MapStepProps {
  file: File
  cadoc: string
  onNext: (mapping: ColumnMapping) => void
  onBack: () => void
}

function MapStep({ file, cadoc, onNext, onBack }: MapStepProps) {
  const [columns, setColumns] = useState<string[]>([])
  const [mapping, setMapping] = useState<ColumnMapping>({})
  const [loading, setLoading] = useState(true)

  // Load saved mapping or auto-detect from file
  useEffect(() => {
    const saved = loadMappingProfile(cadoc)
    if (saved) {
      setMapping(saved)
    }
    // Auto-detect columns by parsing first line only
    const reader = new FileReader()
    reader.onload = (e) => {
      const text = e.target?.result as string
      const firstLine = text.split('\n')[0]
      const cols = firstLine.split(',').map((c) => c.trim().replace(/^"|"$/g, ''))
      setColumns(cols)
      // Auto-map columns that match canonical field names
      const auto: ColumnMapping = {}
      for (const col of cols) {
        const normalized = col.toLowerCase().replace(/[_\s-]/g, '')
        for (const field of CANONICAL_FIELDS) {
          if (field.value && normalized === field.value.replace(/_/g, '')) {
            auto[col] = field.value
            break
          }
        }
      }
      if (Object.keys(auto).length > 0) {
        setMapping(auto)
      }
      setLoading(false)
    }
    reader.readAsText(file.slice(0, 4096))
  }, [file, cadoc])

  const handleFieldChange = (col: string, field: string) => {
    setMapping((prev) => {
      const next = { ...prev }
      if (field) {
        next[col] = field
      } else {
        delete next[col]
      }
      return next
    })
  }

  const handleSaveAndContinue = () => {
    saveMappingProfile(cadoc, mapping)
    onNext(mapping)
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12">
        <RefreshCw className="size-6 animate-spin text-ink-muted" />
      </div>
    )
  }

  const mappedCount = Object.keys(mapping).length

  return (
    <div className="space-y-6 max-w-4xl">
      <SectionHeader
        eyebrow="Passo 2 de 6"
        title="Mapear colunas"
        description={`${columns.length} coluna(s) detectadas. Mapeie cada coluna do seu arquivo para o campo canônico correspondente. Perfil salvo automaticamente para ${cadoc}.`}
        actions={
          <Badge tone="accent" variant="soft" size="sm">
            {mappedCount}/{columns.length} mapeadas
          </Badge>
        }
      />
      <Card padding="none" className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-sunken">
                <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted">Coluna do arquivo</th>
                <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted">Campo canônico</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border-subtle">
              {columns.map((col) => (
                <tr key={col} className="hover:bg-surface-sunken/30">
                  <td className="px-4 py-2.5 font-mono text-xs text-ink bg-surface-raised">{col}</td>
                  <td className="px-4 py-2.5">
                    <select
                      value={mapping[col] ?? ''}
                      onChange={(e) => handleFieldChange(col, e.target.value)}
                      className="w-full h-8 px-2 rounded border border-border bg-surface text-xs text-ink focus:outline-none focus:ring-1 focus:ring-accent-500"
                    >
                      {CANONICAL_FIELDS.map((f) => (
                        <option key={f.value} value={f.value}>
                          {f.label}
                        </option>
                      ))}
                    </select>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
      <div className="flex gap-3">
        <Button variant="ghost" size="md" onClick={onBack} leftIcon={<ChevronLeft className="size-4" />}>
          Voltar
        </Button>
        <Button
          variant="primary"
          size="md"
          onClick={handleSaveAndContinue}
          rightIcon={<ChevronRight className="size-4" />}
        >
          Salvar perfil e continuar
        </Button>
      </div>
    </div>
  )
}

// ─── Step 3: Preview ──────────────────────────────────────────────────────────

interface PreviewStepProps {
  doc: ParsedDoc
  onBack: () => void
  onNext: () => void
}

function PreviewStep({ doc, onBack, onNext }: PreviewStepProps) {
  return (
    <div className="space-y-6">
      <SectionHeader
        eyebrow="Passo 3 de 6"
        title="Revisar documento"
        description={`${doc.operacoes.length} operação(ões) pronta(s) para geração. Valide os dados antes de prosseguir.`}
      />
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
          <div className="overflow-x-auto max-h-80">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-sunken">
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted sticky top-0 bg-surface-sunken">ID</th>
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted sticky top-0 bg-surface-sunken">Modalidade</th>
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted sticky top-0 bg-surface-sunken">Tipo</th>
                  <th className="text-right px-4 py-2.5 text-xs font-medium text-ink-muted sticky top-0 bg-surface-sunken">Valor Principal</th>
                  <th className="text-right px-4 py-2.5 text-xs font-medium text-ink-muted sticky top-0 bg-surface-sunken">Taxa Juros</th>
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted sticky top-0 bg-surface-sunken">Indexador</th>
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-ink-muted sticky top-0 bg-surface-sunken">UF</th>
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
          Validar
        </Button>
      </div>
    </div>
  )
}

// ─── Step 4: Validate ─────────────────────────────────────────────────────────

interface ValidationResult {
  passed: boolean
  errors: Array<{ code: string; severity: string; message: string }>
  warnings: Array<{ code: string; severity: string; message: string }>
  rules_run: string[]
}

interface ValidateStepProps {
  doc: ParsedDoc
  cadoc: string
  onBack: () => void
  onNext: (xml: string, sha256: string) => void
}

function ValidateStep({ doc, cadoc, onBack, onNext }: ValidateStepProps) {
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<ValidationResult | null>(null)
  const [error, setError] = useState<string | null>(null)

  const runValidation = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      // First generate the XML
      const genRes = await fetch('/api/generate/xml', {
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
      const genData = await genRes.json()
      if (!genRes.ok) throw new Error(genData.message || genData.error || 'Erro ao gerar')
      const xml = atob(genData.generated?.xml || '')
      const sha256 = genData.generated?.sha256 || ''

      // Then validate
      const valRes = await fetch('/v1/validate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-IF-ID': doc.if_id || 'demo',
        },
        body: JSON.stringify({
          cadoc_code: cadoc,
          data_base: doc.extra?.data_base ?? new Date().toISOString().slice(0, 7),
          xml,
        }),
      })
      const valData = await valRes.json()
      setResult({
        passed: valData.passed ?? valData.errors?.length === 0,
        errors: valData.errors ?? [],
        warnings: valData.warnings ?? [],
        rules_run: valData.rules_run ?? [],
      })
      // Store XML for next step
      ;(window as any).__wizardXml = xml
      ;(window as any).__wizardSha256 = sha256
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setLoading(false)
    }
  }, [doc, cadoc])

  const handleContinue = () => {
    const xml = (window as any).__wizardXml
    const sha256 = (window as any).__wizardSha256
    if (xml) onNext(xml, sha256)
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <SectionHeader
        eyebrow="Passo 4 de 6"
        title="Validação L1-L4"
        description="Validação semântica, cross-documento e histórica antes da geração final."
      />
      {!result && !loading && !error && (
        <div className="rounded-xl border border-dashed border-border p-10 text-center">
          <CheckCircle2 className="size-10 mx-auto mb-3 text-accent-600 opacity-60" />
          <h3 className="text-base font-serif font-medium text-ink mb-2">Pronto para validar</h3>
          <p className="text-sm text-ink-muted mb-5">Serão executadas até 1.099 regras de validação.</p>
          <Button variant="primary" size="lg" onClick={runValidation} leftIcon={<CheckCircle2 className="size-4" />}>
            Iniciar validação
          </Button>
        </div>
      )}
      {loading && (
        <div className="flex flex-col items-center gap-3 p-10">
          <RefreshCw className="size-6 animate-spin text-accent-600" />
          <p className="text-sm text-ink-muted">Validando…</p>
        </div>
      )}
      {error && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-critical-50 dark:bg-critical-950 border border-critical-200 dark:border-critical-800 text-critical-700 dark:text-critical-300 text-sm">
          <AlertCircle className="size-4 mt-0.5 shrink-0" />
          <span>{error}</span>
        </div>
      )}
      {result && (
        <div className="space-y-4">
          {result.passed ? (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-success-50 dark:bg-success-950 border border-success-200 dark:border-success-800 text-success-700 dark:text-success-300 text-sm">
              <CheckCircle2 className="size-4 shrink-0" />
              <span>Validação aprovada — {result.rules_run.length} regra(s) executada(s)</span>
            </div>
          ) : (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-critical-50 dark:bg-critical-950 border border-critical-200 dark:border-critical-800 text-critical-700 dark:text-critical-300 text-sm">
              <AlertCircle className="size-4 shrink-0" />
              <span>{result.errors.length} erro(s) encontrado(s)</span>
            </div>
          )}
          {result.errors.length > 0 && (
            <Card padding="none" className="overflow-hidden">
              <div className="px-4 py-2.5 bg-critical-50 dark:bg-critical-950 border-b border-critical-200 dark:border-critical-800">
                <span className="text-xs font-medium text-critical-700 dark:text-critical-300 uppercase tracking-wider">
                  Erros ({result.errors.length})
                </span>
              </div>
              <div className="divide-y divide-border-subtle">
                {result.errors.map((e, i) => (
                  <div key={i} className="px-4 py-2.5">
                    <div className="flex items-center gap-2 mb-1">
                      <Badge tone="critical" variant="soft" size="sm">{e.code}</Badge>
                      <Badge tone="critical" variant="outline" size="sm">{e.severity}</Badge>
                    </div>
                    <p className="text-xs text-ink-muted">{e.message}</p>
                  </div>
                ))}
              </div>
            </Card>
          )}
          {result.warnings.length > 0 && (
            <Card padding="none" className="overflow-hidden">
              <div className="px-4 py-2.5 bg-warning-50 dark:bg-warning-950 border-b border-warning-200 dark:border-warning-800">
                <span className="text-xs font-medium text-warning-700 dark:text-warning-300 uppercase tracking-wider">
                  Avisos ({result.warnings.length})
                </span>
              </div>
              <div className="divide-y divide-border-subtle">
                {result.warnings.slice(0, 10).map((w, i) => (
                  <div key={i} className="px-4 py-2.5">
                    <div className="flex items-center gap-2 mb-1">
                      <Badge tone="warning" variant="soft" size="sm">{w.code}</Badge>
                    </div>
                    <p className="text-xs text-ink-muted">{w.message}</p>
                  </div>
                ))}
                {result.warnings.length > 10 && (
                  <div className="px-4 py-2.5 text-xs text-ink-muted text-center">
                    +{result.warnings.length - 10} outro(s) aviso(s)
                  </div>
                )}
              </div>
            </Card>
          )}
          <div className="flex gap-3">
            <Button variant="ghost" size="md" onClick={onBack} leftIcon={<ChevronLeft className="size-4" />}>
              Voltar
            </Button>
            <Button
              variant="primary"
              size="md"
              onClick={handleContinue}
              rightIcon={<ArrowRight className="size-4" />}
              disabled={!result.passed}
            >
              {result.passed ? 'Continuar para geração' : 'Verificar erros'}
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

// ─── Step 5: Generate ─────────────────────────────────────────────────────────

interface GenerateStepProps {
  doc: ParsedDoc
  cadoc: string
  onBack: () => void
  onNext: (xml: string, sha256: string) => void
}

function GenerateStep({ doc, cadoc, onBack, onNext }: GenerateStepProps) {
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
      const xml = atob(data.generated?.xml || '')
      const sha256 = data.generated?.sha256 || ''
      setResult({ xml, sha256, cadoc_code: cadoc, status: 'ok', message: data.message })
      ;(window as any).__wizardXml = xml
      ;(window as any).__wizardSha256 = sha256
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
    <div className="space-y-4 max-w-3xl">
      <SectionHeader
        eyebrow="Passo 5 de 6"
        title="Geração do XML"
        description="XML gerado a partir do CanonicalDocument. Pronto para download ou envio direto ao BACEN."
      />
      {!result ? (
        <div className="space-y-4 max-w-lg">
          <div className="rounded-xl border border-dashed border-border p-8 text-center">
            <Wand2 className="size-10 mx-auto mb-3 text-accent-600 opacity-60" />
            <h3 className="text-base font-serif font-medium text-ink mb-2">Pronto para gerar</h3>
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
              <pre className="bg-[#0d1117] text-green-400 text-xs p-4 overflow-x-auto max-h-64 leading-relaxed">
                <code>{result.xml.slice(0, 3000)}{result.xml.length > 3000 && '\n…'}</code>
              </pre>
            </Card>
          </div>
          <div className="flex gap-3">
            <Button variant="secondary" size="md" onClick={handleDownload} leftIcon={<Download className="size-4" />}>
              Baixar XML
            </Button>
            <Button
              variant="primary"
              size="md"
              onClick={() => onNext(result.xml, result.sha256)}
              rightIcon={<ChevronRight className="size-4" />}
            >
              Enviar ao BACEN
            </Button>
          </div>
        </div>
      )}
      <Button variant="ghost" size="md" onClick={onBack} leftIcon={<ChevronLeft className="size-4" />}>
        Voltar
      </Button>
    </div>
  )
}

// ─── Step 6: STA ──────────────────────────────────────────────────────────────

interface STAStepProps {
  xml: string
  sha256: string
  cadoc: string
  doc: ParsedDoc
  onBack: () => void
}

function STAStep({ xml, sha256, cadoc, doc, onBack }: STAStepProps) {
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<{ protocolo?: string; accepted?: boolean; message?: string } | null>(null)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/sta/submit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          cadoc_code: cadoc,
          data_base: doc.extra?.data_base ?? new Date().toISOString().slice(0, 7),
          xml: btoa(xml),
          cnpj: doc.header.cnpj,
          validate_first: true,
        }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.message || data.error || 'Erro ao submeter')
      setResult(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <SectionHeader
        eyebrow="Passo 6 de 6"
        title="Envio ao BACEN"
        description="Submeta o XML gerado diretamente ao sistema STA do BACEN."
      />
      {!result ? (
        <div className="space-y-4">
          <Card padding="md" className="space-y-3">
            <div className="flex items-center justify-between text-sm">
              <span className="text-ink-muted">Documento</span>
              <span className="font-mono text-ink">{cadoc}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-ink-muted">Hash</span>
              <span className="font-mono text-ink text-xs">{sha256.slice(0, 24)}…</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-ink-muted">Tamanho</span>
              <span className="font-mono text-ink">{(xml.length / 1024).toFixed(1)} KB</span>
            </div>
          </Card>
          <Button
            variant="primary"
            size="lg"
            onClick={handleSubmit}
            loading={loading}
            leftIcon={<Send className="size-4" />}
          >
            Submeter ao BACEN
          </Button>
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
            <span>{result.message || 'Submetido com sucesso'}</span>
          </div>
          {result.protocolo && (
            <Card padding="md">
              <div className="text-xs text-ink-muted mb-1">Protocolo STA</div>
              <div className="font-mono text-2xl font-medium text-ink">{result.protocolo}</div>
            </Card>
          )}
        </div>
      )}
      <Button variant="ghost" size="md" onClick={onBack} leftIcon={<ChevronLeft className="size-4" />}>
        Voltar
      </Button>
    </div>
  )
}

// ─── Wizard Client Component ───────────────────────────────────────────────────

export function WizardClient() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const pathname = usePathname()

  const defaultCadoc = searchParams.get('cadoc') ?? undefined

  const [step, setStep] = useState<WizardStep>('upload')
  const [parsedDoc, setParsedDoc] = useState<ParsedDoc | null>(null)
  const [uploadedFile, setUploadedFile] = useState<File | null>(null)
  const [columnMapping, setColumnMapping] = useState<ColumnMapping>({})
  const [generatedXml, setGeneratedXml] = useState<{ xml: string; sha256: string } | null>(null)

  // Sync step to URL
  const pushStep = useCallback(
    (s: WizardStep) => {
      const params = new URLSearchParams(searchParams.toString())
      params.set('step', s)
      router.replace(`${pathname}?${params.toString()}`, { scroll: false })
    },
    [router, pathname, searchParams],
  )

  // Restore step from URL on mount
  useEffect(() => {
    const urlStep = searchParams.get('step') as WizardStep | null
    if (urlStep && STEPS.find((s) => s.id === urlStep)) {
      setStep(urlStep)
    }
  }, [searchParams])

  const handleStepChange = (next: WizardStep) => {
    setStep(next)
    pushStep(next)
  }

  const handleUploadNext = (doc: ParsedDoc, file: File) => {
    setParsedDoc(doc)
    setUploadedFile(file)
    handleStepChange('map')
  }

  const handleMapNext = (mapping: ColumnMapping) => {
    setColumnMapping(mapping)
    handleStepChange('preview')
  }

  const handlePreviewNext = () => {
    handleStepChange('validate')
  }

  const handleValidateNext = (xml: string, sha256: string) => {
    setGeneratedXml({ xml, sha256 })
    handleStepChange('generate')
  }

  const handleGenerateNext = (xml: string, sha256: string) => {
    setGeneratedXml({ xml, sha256 })
    handleStepChange('sta')
  }

  return (
    <div className="space-y-8 max-w-5xl">
      <StepIndicator current={step} steps={STEPS} />
      <div className="mt-6">
        {step === 'upload' && (
          <UploadStep defaultCadoc={defaultCadoc} onNext={handleUploadNext} />
        )}
        {step === 'map' && uploadedFile && (
          <MapStep
            file={uploadedFile}
            cadoc={parsedDoc?.cadoc_code ?? '3040'}
            onNext={handleMapNext}
            onBack={() => handleStepChange('upload')}
          />
        )}
        {step === 'preview' && parsedDoc && (
          <PreviewStep
            doc={parsedDoc}
            onBack={() => handleStepChange('map')}
            onNext={handlePreviewNext}
          />
        )}
        {step === 'validate' && parsedDoc && (
          <ValidateStep
            doc={parsedDoc}
            cadoc={parsedDoc.cadoc_code}
            onBack={() => handleStepChange('preview')}
            onNext={handleValidateNext}
          />
        )}
        {step === 'generate' && parsedDoc && generatedXml && (
          <GenerateStep
            doc={parsedDoc}
            cadoc={parsedDoc.cadoc_code}
            onBack={() => handleStepChange('validate')}
            onNext={handleGenerateNext}
          />
        )}
        {step === 'sta' && parsedDoc && generatedXml && (
          <STAStep
            xml={generatedXml.xml}
            sha256={generatedXml.sha256}
            cadoc={parsedDoc.cadoc_code}
            doc={parsedDoc}
            onBack={() => handleStepChange('generate')}
          />
        )}
      </div>
    </div>
  )
}
