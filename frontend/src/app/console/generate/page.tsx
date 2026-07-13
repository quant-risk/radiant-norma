/**
 * /console/generate — Motor de Geração de CADOCs.
 *
 * Visualiza o motor (Gen3040 MVP, outros 9 planejados) +
 * status de cada generator por CADOC + acesso ao wizard de geração.
 *
 * Sprint 57 (v3.36.0): motor 3040 + canonical + adapters + API REST
 * já implementado no backend (`internal/generator/gen3040/`).
 * Esta página é a fachada UI do motor.
 *
 * v3.36.2: removed silent fetch to /v1/generate/stats (endpoint
 * inexistente). 3040 ainda é MVP — validação XSD é feita em camada
 * separada (L1), não pelo generator.
 */

import Link from 'next/link'
import {
  Wand2,
  Database,
  Send,
  CheckCircle2,
  Clock,
  ArrowUpRight,
  Sparkles,
  FileText,
  Cog,
} from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { StatCard } from '@/components/domain/stat-card'
import { Card, CardTitle, CardEyebrow, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SectionHeader } from '@/components/ui/section-header'
import { Divider } from '@/components/ui/divider'

export const dynamic = 'force-dynamic'

interface CadocStatus {
  cadoc: string
  name: string
  generator: 'ready' | 'beta' | 'planned'
  rules: number
  description: string
}

const CADOCS: CadocStatus[] = [
  {
    cadoc: '3040',
    name: 'SCR — Risco de Crédito',
    generator: 'ready',
    rules: 275,
    description:
      'MVP funcional. Gera XML 3040 v3.4 a partir de CanonicalDocument. Validação XSD é feita em camada separada (L1).',
  },
  {
    cadoc: '3050',
    name: 'Estatísticas Agregadas',
    generator: 'planned',
    rules: 170,
    description: 'Regras validadas (Sprint 33). Generator em backlog Sprint 58.',
  },
  {
    cadoc: '3044',
    name: 'Eventos de Crédito (JSON)',
    generator: 'planned',
    rules: 17,
    description: 'Engine JSON implementado. Generator em backlog.',
  },
  {
    cadoc: '2061',
    name: 'DLO — Limites Operacionais',
    generator: 'planned',
    rules: 24,
    description: 'Regras validadas (Sprint 38). Generator planejado.',
  },
  {
    cadoc: '2070',
    name: 'DDR — Requerimento Capital',
    generator: 'planned',
    rules: 11,
    description: 'Cross-doc implementado. Generator planejado.',
  },
  {
    cadoc: '2160',
    name: 'DRL — Liquidez (LCR)',
    generator: 'planned',
    rules: 11,
    description: 'Regras validadas (Sprint 40). Generator planejado.',
  },
  {
    cadoc: '2170',
    name: 'DLP — Liquidez LP (NSFR)',
    generator: 'planned',
    rules: 10,
    description: 'Regras validadas (Sprint 41). Generator planejado.',
  },
  {
    cadoc: '2062',
    name: 'DLI — Limites Individuais',
    generator: 'planned',
    rules: 18,
    description: 'Parser implementado (Sprint 58). Generator planejado.',
  },
  {
    cadoc: '2060',
    name: 'DRM — Risco de Mercado',
    generator: 'planned',
    rules: 22,
    description: 'Regras validadas. Generator em planejamento.',
  },
  {
    cadoc: '4111',
    name: 'COSIF — Plano Contábil',
    generator: 'planned',
    rules: 30,
    description: 'Parser genérico (Sprint 51). Generator planejado.',
  },
]

const STATUS_TONE: Record<CadocStatus['generator'], 'success' | 'info' | 'neutral'> = {
  ready: 'success',
  beta: 'info',
  planned: 'neutral',
}

const STATUS_LABEL: Record<CadocStatus['generator'], string> = {
  ready: 'pronto',
  beta: 'beta',
  planned: 'planejado',
}

export default async function GeneratePage() {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-12 text-center">
        <p className="text-ink-muted">Sessão expirada.</p>
      </div>
    )
  }

  // TODO(v3.36.2): GET /v1/generate/stats endpoint — por ora 0.
  // Backend não tem este endpoint ainda. Removida a call que retornava
  // 404 silencioso a cada render. Adicionar quando /v1/generate/stats
  // existir no backend.
  const totalGenerated = 0

  const readyCount = CADOCS.filter((c) => c.generator === 'ready').length
  const plannedCount = CADOCS.filter((c) => c.generator === 'planned').length

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Generator',
        subtitle: 'Motor de geração de CADOCs a partir de dados brutos',
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Console' },
          { label: 'Generator' },
        ],
      }}
    >
      <div className="space-y-10 max-w-7xl">
        {/* Hero strip */}
        <section>
          <SectionHeader
            eyebrow="Motor de geração"
            title="De dados brutos ao CADOC pronto pra STA"
            description="Recebe dados de qualquer fonte (planilha, API, banco, MCP) e produz o documento XML pronto pra envio. Validação L1-L4 automática. Explainability campo a campo."
            actions={
              <Link href="/console/ingest" passHref legacyBehavior={false}>
                <Button asChild variant="secondary" size="md" rightIcon={<ArrowUpRight className="size-3.5" strokeWidth={2.25} />}>
                  Conectar fonte de dados
                </Button>
              </Link>
            }
          />

          <div className="mt-8 grid grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              label="Generators"
              value={`${readyCount}/10`}
              tone="accent"
              icon={<Wand2 className="size-4" strokeWidth={2.25} />}
              helpText="CADOCs com motor pronto"
            />
            <StatCard
              label="Documentos gerados"
              value={totalGenerated}
              tone="neutral"
              icon={<FileText className="size-4" strokeWidth={2.25} />}
              helpText="Total no ciclo atual"
            />
            <StatCard
              label="Conectores"
              value="5"
              tone="neutral"
              icon={<Database className="size-4" strokeWidth={2.25} />}
              helpText="Manual · API · File · DB · MCP"
            />
            <StatCard
              label="Pipeline"
              value="0 → 15 min"
              tone="success"
              icon={<Clock className="size-4" strokeWidth={2.25} />}
              helpText="Onboarding self-service"
            />
          </div>
        </section>

        <Divider />

        {/* Como funciona */}
        <section className="space-y-5">
          <SectionHeader
            eyebrow="Pipeline"
            title="Como funciona"
            description="5 etapas, 0 mágica. Conector → Canonical → Generator → Validação → STA."
          />

          <div className="grid grid-cols-1 md:grid-cols-5 gap-3">
            {[
              {
                n: '1',
                title: 'Ingest',
                desc: '5 conectores (Manual, API, File, DB, MCP) produzem CanonicalDocument IF-agnóstico',
                icon: Database,
              },
              {
                n: '2',
                title: 'Canonical',
                desc: 'JSON tipado, sem XML. LLM nunca escreve XML direto — sempre passa pelo schema',
                icon: FileText,
              },
              {
                n: '3',
                title: 'Generator',
                desc: 'Cada CADOC (3040, 3050, 4111) tem seu generator que consome schema + emite XML',
                icon: Wand2,
              },
              {
                n: '4',
                title: 'Validação L1-L4',
                desc: 'XSD → semântico → cross-doc → histórico. 1.099 regras, 90% coverage',
                icon: CheckCircle2,
              },
              {
                n: '5',
                title: 'STA Submit',
                desc: 'Push nativo pro BACEN via STA-h/STA-ws. Retry exponencial + DLQ + audit chain',
                icon: Send,
              },
            ].map((step, i) => {
              const Icon = step.icon
              return (
                <div
                  key={step.n}
                  className="relative rounded-lg border border-border bg-surface-raised p-5 animate-fade-up"
                  style={{ animationDelay: `${i * 60}ms` }}
                >
                  <div className="flex items-baseline gap-3 mb-3">
                    <span className="font-serif text-3xl font-medium text-accent-600 dark:text-accent-400 tracking-tight">
                      {step.n}
                    </span>
                    <Icon className="size-4 text-ink-muted ml-auto" strokeWidth={2.25} />
                  </div>
                  <h3 className="font-serif text-base font-medium text-ink mb-1.5 tracking-tight">
                    {step.title}
                  </h3>
                  <p className="text-xs text-ink-muted leading-relaxed">
                    {step.desc}
                  </p>
                </div>
              )
            })}
          </div>
        </section>

        <Divider />

        {/* CADOCs table */}
        <section className="space-y-5">
          <SectionHeader
            eyebrow="Catálogo"
            title="Generators por CADOC"
            description="Status de cada generator. Pronto (production-ready), Planejado (em backlog Sprint 58+)."
          />

          <Card padding="none">
            <div className="divide-y divide-border-subtle">
              {CADOCS.map((c) => (
                <div
                  key={c.cadoc}
                  className="px-6 py-5 flex items-center gap-6 hover:bg-surface-sunken/30 transition-colors"
                >
                  <div className="flex items-center gap-3 shrink-0">
                    <span className="font-mono text-sm font-medium text-accent-600 dark:text-accent-400 w-12">
                      {c.cadoc}
                    </span>
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-serif text-sm font-medium text-ink tracking-tight">
                        {c.name}
                      </span>
                    </div>
                    <p className="text-xs text-ink-muted leading-relaxed">
                      {c.description}
                    </p>
                  </div>

                  <div className="flex items-center gap-6 shrink-0">
                    <div className="text-right">
                      <div className="text-2xs uppercase tracking-wider font-mono text-ink-subtle">
                        regras
                      </div>
                      <div className="text-sm font-mono font-medium text-ink nums">
                        {c.rules}
                      </div>
                    </div>

                    <Badge
                      tone={STATUS_TONE[c.generator]}
                      variant="soft"
                      dot
                      size="sm"
                    >
                      {STATUS_LABEL[c.generator]}
                    </Badge>

                    {c.generator === 'ready' ? (
                      <Link href={`/console/generate/wizard?cadoc=${c.cadoc}`} passHref legacyBehavior={false}>
                        <Button variant="primary" size="sm">
                          Gerar CADOC
                        </Button>
                      </Link>
                    ) : (
                      <Button variant="ghost" size="sm" disabled>
                        Em breve
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </section>

        <Divider />

        {/* CTA */}
        <section className="rounded-2xl border border-border bg-gradient-to-br from-accent-50/40 to-magenta-500/5 dark:from-accent-950/20 dark:to-magenta-500/5 p-10 md:p-14 text-center">
          <Sparkles className="size-6 mx-auto mb-4 text-accent-600" strokeWidth={1.5} />
          <h3 className="font-serif text-2xl md:text-3xl font-medium text-ink tracking-tight mb-3">
            Primeira geração em 15 minutos
          </h3>
          <p className="text-sm text-ink-muted max-w-xl mx-auto leading-relaxed mb-7">
            Conecte uma fonte de dados, escolha o CADOC, gere o documento
            validado. Wizard guiado, sem código. 3040 está pronto —
            os outros 9 vêm em sequência.
          </p>
          <div className="flex items-center justify-center gap-3 flex-wrap">
            <Link href="/console/ingest" passHref legacyBehavior={false}>
              <Button asChild variant="primary" size="md" rightIcon={<ArrowUpRight className="size-3.5" strokeWidth={2.25} />}>
                Conectar fonte de dados
              </Button>
            </Link>
            <Link href="mailto:contato@radiantnorma.com.br" passHref legacyBehavior={false}>
              <Button asChild variant="secondary" size="md" leftIcon={<Cog className="size-4" strokeWidth={2.25} />}>
                Falar com engenharia
              </Button>
            </Link>
          </div>
        </section>
      </div>
    </AppShell>
  )
}