'use client'

/**
 * HowItWorks — pipeline editorial 5 etapas.
 *
 * Ingest → Canonical → Generator → Validação → STA.
 * Cada etapa com ícone, título, descrição e bullet points.
 */

import { Database, FileText, Wand2, CheckCircle2, Send } from 'lucide-react'
import { SectionHeader } from '@/components/ui/section-header'

const STEPS = [
  {
    n: '01',
    icon: Database,
    title: 'Conecte uma fonte',
    desc:
      '5 conectores disponíveis: Manual (formulário), API (REST/webhook), File (PDF/XLSX/DOCX), DB (Postgres/Oracle/MySQL), MCP (LLM tools).',
    detail: [
      'Manual funcional',
      'API + File beta',
      'DB + MCP planejados',
    ],
  },
  {
    n: '02',
    icon: FileText,
    title: 'Produza o Canonical',
    desc:
      'Cada conector normaliza dados brutos no CanonicalDocument — JSON tipado, IF-agnóstico. LLM nunca escreve XML direto, sempre passa pelo schema.',
    detail: [
      'Schema-first',
      'IF-agnostic',
      'Versionado por CADOC',
    ],
  },
  {
    n: '03',
    icon: Wand2,
    title: 'Gere o CADOC',
    desc:
      'Cada CADOC (3040, 3050, 4111…) tem seu próprio generator que consome o Canonical e emite XML validado. 3040 pronto, 9 planejados.',
    detail: [
      '10 generators planejados',
      '1 pronto (3040)',
      'Adapter pattern',
    ],
  },
  {
    n: '04',
    icon: CheckCircle2,
    title: 'Valide L1 → L4',
    desc:
      'XSD (L1) → semântico (L2) → cross-doc (L3) → histórico (L4). 1.099 regras, 90% coverage. Explica cada falha campo-a-campo.',
    detail: [
      '4 camadas',
      '1.099 regras',
      'Explainability',
    ],
  },
  {
    n: '05',
    icon: Send,
    title: 'Envie pro STA',
    desc:
      'Push nativo pro BACEN via STA-h/STA-ws. Retry exponencial, DLQ em caso de falha, audit chain imutável por envio.',
    detail: [
      'STA-h + STA-ws',
      'Retry exponencial',
      'Audit chain SHA-256',
    ],
  },
]

export function LandingHowItWorks() {
  return (
    <section
      id="how-it-works"
      className="relative py-24 lg:py-32 bg-surface-sunken/30"
    >
      <div className="absolute inset-0 pattern-grid opacity-30" aria-hidden />
      <div className="relative max-w-7xl mx-auto px-6 lg:px-10">
        <SectionHeader
          eyebrow="Como funciona"
          title="Pipeline 5 etapas. 0 mágica."
          description="Determinístico e auditável. Você sempre sabe o que está rodando, contra qual schema, e por quê."
        />

        <div className="mt-14 grid grid-cols-1 md:grid-cols-5 gap-6 relative">
          {/* Linha conectora (desktop) */}
          <div
            className="hidden md:block absolute top-12 left-[10%] right-[10%] h-px bg-gradient-to-r from-border via-accent-300 to-border"
            aria-hidden
          />

          {STEPS.map((step, i) => {
            const Icon = step.icon
            return (
              <div
                key={step.n}
                className="relative animate-fade-up"
                style={{ animationDelay: `${i * 100}ms` }}
              >
                <div className="flex items-center gap-4 mb-6">
                  <div className="size-12 rounded-lg bg-surface-raised border border-border shadow-xs flex items-center justify-center relative z-10">
                    <Icon
                      className="size-5 text-accent-600 dark:text-accent-400"
                      strokeWidth={1.75}
                    />
                  </div>
                  <span className="font-serif text-3xl font-medium text-ink-subtle tracking-tight">
                    {step.n}
                  </span>
                </div>

                <h3 className="font-serif text-xl font-medium text-ink tracking-tight mb-3">
                  {step.title}
                </h3>
                <p className="text-sm text-ink-muted leading-relaxed mb-5">
                  {step.desc}
                </p>

                <ul className="space-y-2">
                  {step.detail.map((item) => (
                    <li
                      key={item}
                      className="flex items-center gap-2 text-xs text-ink-muted"
                    >
                      <span className="size-1 rounded-full bg-accent-500 shrink-0" />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}