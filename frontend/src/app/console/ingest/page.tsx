/**
 * /console/ingest — Conectores de dados.
 *
 * Lista os 5 adapters (Manual, API, File, DB, MCP) e permite configurar
 * cada fonte. Cada conector produz CanonicalDocument que alimenta o
 * Generator.
 *
 * Sprint 57 (v3.36.0): adapter Manual funcional. Os outros 4 (File, API,
 * DB, MCP) estão com Fetch ainda em stub (return ErrNotImplemented) —
 * configuração ValidateConfig parcialmente funcional, geração real virá
 * na Sprint 57 fase 2. Use POST /v1/generate/{cadoc} direto como
 * work-around para adapters stub.
 * 4 stubs). Esta página é a fachada UI.
 */

import Link from 'next/link'
import {
  Database,
  Globe,
  FileSpreadsheet,
  HardDrive,
  Cpu,
  ArrowUpRight,
  Sparkles,
  Plus,
  CheckCircle2,
} from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { Card, CardEyebrow } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SectionHeader } from '@/components/ui/section-header'
import { Divider } from '@/components/ui/divider'

export const dynamic = 'force-dynamic'

interface Adapter {
  id: string
  name: string
  description: string
  icon: React.ComponentType<{ className?: string; strokeWidth?: number | string }>
  status: 'ready' | 'beta' | 'stub'
  format: string
  examples: string[]
}

const ADAPTERS: Adapter[] = [
  {
    id: 'manual',
    name: 'Manual',
    description:
      'Formulário web guiado. Compliance officer preenche campos, generator consome CanonicalDocument.',
    icon: Plus,
    status: 'ready',
    format: 'Form UI → JSON',
    examples: ['Form wizard por CADOC', 'Templates salvos', 'Bulk paste (CSV)'],
  },
  {
    id: 'file',
    name: 'File (XLSX / CSV / JSON)',
    description:
      'Upload de planilhas. Adapter com ValidateConfig implementado; Fetch ainda em stub (Sprint 57 fase 2).',
    icon: FileSpreadsheet,
    status: 'stub',
    format: 'XLSX / CSV / JSON → JSON',
    examples: ['XLSX com colunas mapeadas', 'CSV com header row', 'JSON com schema pré-definido'],
  },
  {
    id: 'api',
    name: 'API REST',
    description:
      'Webhook + POST endpoint. Adapter registrado mas Fetch ainda retorna ErrNotImplemented. Use /v1/generate/{cadoc} direto enquanto isso.',
    icon: Globe,
    status: 'stub',
    format: 'JSON via HTTPS',
    examples: ['POST /v1/generate/3040 (direto)', 'Webhook planejado', 'Polling endpoint configurável'],
  },
  {
    id: 'db',
    name: 'Database (PostgreSQL / Oracle / MySQL)',
    description:
      'Conexão read-only com banco da IF. SQL queries mapeiam colunas → campos canônicos.',
    icon: HardDrive,
    status: 'stub',
    format: 'SQL SELECT → JSON',
    examples: ['Connection string', 'Schema mapping UI', 'CDC opcional'],
  },
  {
    id: 'mcp',
    name: 'MCP (Model Context Protocol)',
    description:
      'Agentes IA da IF chamam tools MCP. LLM recebe schema canônico + docs BACEN como contexto.',
    icon: Cpu,
    status: 'stub',
    format: 'MCP tool call',
    examples: ['Claude / GPT integra via MCP', 'Tools: generate, validate, submit', 'Contexto: docs BACEN'],
  },
]

const STATUS_TONE: Record<Adapter['status'], 'success' | 'info' | 'neutral'> = {
  ready: 'success',
  beta: 'info',
  stub: 'neutral',
}

const STATUS_LABEL: Record<Adapter['status'], string> = {
  ready: 'funcional',
  beta: 'beta',
  stub: 'planejado',
}

export default async function IngestPage() {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-12 text-center">
        <p className="text-ink-muted">Sessão expirada.</p>
      </div>
    )
  }

  const readyCount = ADAPTERS.filter((a) => a.status === 'ready').length
  const plannedCount = ADAPTERS.filter((a) => a.status !== 'ready').length

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Ingest',
        subtitle: 'Conectores de dados que alimentam o Generator',
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Console' },
          { label: 'Ingest' },
        ],
      }}
    >
      <div className="space-y-10 max-w-7xl">
        {/* Hero strip */}
        <section>
          <SectionHeader
            eyebrow="Conectores de dados"
            title="5 fontes de dados que produzem CanonicalDocument"
            description="Cada adapter normaliza dados brutos (planilha, API, banco, MCP) no modelo canônico IF-agnóstico. O Generator consome esse modelo — sem tocar no motor quando você troca de fonte."
            actions={
              <Link href="/console/generate" passHref legacyBehavior={false}>
                <Button asChild variant="secondary" size="md" rightIcon={<ArrowUpRight className="size-3.5" strokeWidth={2.25} />}>
                  Ver generators
                </Button>
              </Link>
            }
          />

          <div className="mt-8 grid grid-cols-3 gap-4">
            <div className="rounded-lg border border-border bg-surface-raised p-5">
              <div className="flex items-center gap-2.5 mb-2">
                <CheckCircle2 className="size-4 text-success-600" strokeWidth={2.25} />
                <span className="text-2xs uppercase tracking-wider font-mono text-ink-subtle">
                  Funcionais
                </span>
              </div>
              <div className="text-3xl font-serif font-medium text-ink tracking-tight nums">
                {readyCount}
              </div>
              <div className="text-xs text-ink-muted mt-1">pronto pra produção</div>
            </div>

            <div className="rounded-lg border border-border bg-surface-raised p-5">
              <div className="flex items-center gap-2.5 mb-2">
                <Sparkles className="size-4 text-ink-muted" strokeWidth={2.25} />
                <span className="text-2xs uppercase tracking-wider font-mono text-ink-subtle">
                  Planejados
                </span>
              </div>
              <div className="text-3xl font-serif font-medium text-ink tracking-tight nums">
                {plannedCount}
              </div>
              <div className="text-xs text-ink-muted mt-1">em roadmap</div>
            </div>

            <div className="rounded-lg border border-border bg-surface-raised p-5">
              <div className="flex items-center gap-2.5 mb-2">
                <Database className="size-4 text-ink-muted" strokeWidth={2.25} />
                <span className="text-2xs uppercase tracking-wider font-mono text-ink-subtle">
                  Total
                </span>
              </div>
              <div className="text-3xl font-serif font-medium text-ink tracking-tight nums">
                {ADAPTERS.length}
              </div>
              <div className="text-xs text-ink-muted mt-1">conectores disponíveis</div>
            </div>
          </div>
        </section>

        <Divider />

        {/* Adapters grid */}
        <section className="space-y-5">
          <SectionHeader
            eyebrow="Catálogo"
            title="5 conectores"
            description="Cada adapter é um plugin independente — adicione um novo sem tocar no motor."
          />

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {ADAPTERS.map((a, i) => {
              const Icon = a.icon
              return (
                <Card
                  key={a.id}
                  padding="none"
                  className="group hover:border-border-strong hover:shadow-md transition-all duration-240 ease-out-expo hover:-translate-y-px animate-fade-up"
                  style={{ animationDelay: `${i * 60}ms` }}
                >
                  <div className="p-6">
                    <div className="flex items-start justify-between gap-3 mb-4">
                      <div className="size-11 rounded-lg bg-gradient-to-br from-accent-50 to-magenta-500/10 dark:from-accent-950/40 dark:to-magenta-500/10 ring-1 ring-inset ring-accent-200/60 dark:ring-accent-800/40 flex items-center justify-center text-accent-600 dark:text-accent-300 group-hover:scale-105 transition-transform">
                        <Icon className="size-5" strokeWidth={1.75} />
                      </div>
                      <Badge
                        tone={STATUS_TONE[a.status]}
                        variant="soft"
                        dot
                        size="sm"
                      >
                        {STATUS_LABEL[a.status]}
                      </Badge>
                    </div>

                    <h3 className="font-serif text-base font-medium text-ink tracking-tight mb-2">
                      {a.name}
                    </h3>
                    <p className="text-xs text-ink-muted leading-relaxed mb-4">
                      {a.description}
                    </p>

                    <div className="space-y-2 pt-3 border-t border-border-subtle">
                      <div className="flex items-center gap-2 text-2xs font-mono">
                        <span className="uppercase tracking-wider text-ink-subtle">
                          formato
                        </span>
                        <span className="text-ink-muted">{a.format}</span>
                      </div>
                      <ul className="space-y-1">
                        {a.examples.map((ex) => (
                          <li
                            key={ex}
                            className="text-2xs text-ink-muted flex items-center gap-1.5"
                          >
                            <span className="size-0.5 rounded-full bg-ink-subtle" />
                            {ex}
                          </li>
                        ))}
                      </ul>
                    </div>
                  </div>

                  <div className="border-t border-border-subtle px-6 py-3 bg-surface-sunken/30 flex items-center justify-between">
                    {a.status === 'ready' ? (
                      <Button variant="primary" size="sm" fullWidth>
                        Configurar
                      </Button>
                    ) : a.status === 'beta' ? (
                      <Button variant="secondary" size="sm" fullWidth>
                        Solicitar acesso beta
                      </Button>
                    ) : (
                      <Button variant="ghost" size="sm" fullWidth disabled>
                        Em breve
                      </Button>
                    )}
                  </div>
                </Card>
              )
            })}
          </div>
        </section>
      </div>
    </AppShell>
  )
}