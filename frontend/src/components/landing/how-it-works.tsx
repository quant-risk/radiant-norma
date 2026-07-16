/**
 * HowItWorks — pipeline editorial 7 etapas.
 *
 * Ingest → Canonical → Wizard → Generator → Validação → AI/Insights → STA + Webhooks.
 */

import { Database, FileText, Wand2, CheckCircle2, Send, Sparkles, Webhook } from 'lucide-react'
import { SectionHeader } from '@/components/ui/section-header'

const STEPS = [
  {
    n: '01',
    icon: Database,
    title: 'Conecte uma fonte',
    desc:
      '5 conectores: Manual (formulário), API (REST/webhook), File (PDF/XLSX/DOCX), DB (Postgres/Oracle/MySQL), MCP (LLM tools).',
    detail: [
      'Manual funcional',
      'API + File em stub',
      'DB + MCP planejados',
    ],
  },
  {
    n: '02',
    icon: FileText,
    title: 'Produza o Canonical',
    desc:
      'Cada conector normaliza dados brutos no CanonicalDocument — JSON tipado, IF-agnóstico. LLM nunca escreve XML direto.',
    detail: [
      'Schema-first',
      'IF-agnostic',
      'Versionado por CADOC',
    ],
  },
  {
    n: '03',
    icon: Wand2,
    title: 'Wizard guiado',
    desc:
      'Wizard multi-etapas para config guiada (escolher CADOC → mapear campos → validar antes de gerar). Reduz 0 → primeiro envio.',
    detail: [
      'Onboarding 15 min',
      'Pré-validação',
      'Templates salvos',
    ],
  },
  {
    n: '04',
    icon: Send,
    title: 'Gere o CADOC',
    desc:
      'Cada CADOC (3040, 3050, 4111…) tem seu próprio generator que consome o Canonical e emite XML validado. 3040 pronto, 9 planejados.',
    detail: [
      '10 generators',
      '1 pronto (3040)',
      'Batch generation',
    ],
  },
  {
    n: '05',
    icon: CheckCircle2,
    title: 'Valide L1 → L4',
    desc:
      'XSD (L1) → semântico (L2) → cross-doc (L3) → histórico (L4). 1.099 regras, 76% coverage no 3040. Explica cada falha campo-a-campo.',
    detail: [
      '4 camadas',
      'L4 diff histórico',
      'Explainability',
    ],
  },
  {
    n: '06',
    icon: Sparkles,
    title: 'AI Insights',
    desc:
      'LLM interpreta audit_log e responde em linguagem natural. Cross-doc L3 + heurística determinística. Opt-in por tenant.',
    detail: [
      'Chat por audit',
      'Recomendações',
      'Opt-in por IF',
    ],
  },
  {
    n: '07',
    icon: Webhook,
    title: 'STA + Webhooks',
    desc:
      'Push nativo pro BACEN via STA-h/STA-ws. Webhooks outbound HMAC-SHA256 para sistemas da IF. Retry exponencial + DLQ.',
    detail: [
      'STA-h + STA-ws',
      'Webhooks',
      'DLQ + retry',
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
          title="Pipeline 7 etapas. 0 mágica."
          description="Determinístico e auditável. Você sempre sabe o que está rodando, contra qual schema, e por quê."
        />

        <div className="mt-14 grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-4 relative">
          <div
            className="hidden lg:block absolute top-10 left-[7%] right-[7%] h-px bg-gradient-to-r from-border via-accent-300 to-border"
            aria-hidden
          />

          {STEPS.map((step, i) => {
            const Icon = step.icon
            return (
              <div
                key={step.n}
                className="relative animate-fade-up"
                style={{ animationDelay: `${i * 80}ms` }}
              >
                <div className="flex flex-col gap-3 mb-5">
                  <div className="size-10 rounded-lg bg-surface-raised border border-border shadow-xs flex items-center justify-center relative z-10">
                    <Icon
                      className="size-4 text-accent-600 dark:text-accent-400"
                      strokeWidth={1.75}
                    />
                  </div>
                  <span className="font-serif text-2xl font-medium text-ink-subtle tracking-tight leading-none">
                    {step.n}
                  </span>
                </div>

                <h3 className="font-serif text-base font-medium text-ink tracking-tight mb-2">
                  {step.title}
                </h3>
                <p className="text-xs text-ink-muted leading-relaxed mb-3">
                  {step.desc}
                </p>

                <ul className="space-y-1.5">
                  {step.detail.map((item) => (
                    <li
                      key={item}
                      className="flex items-start gap-1.5 text-2xs text-ink-muted"
                    >
                      <span className="size-1 rounded-full bg-accent-500 shrink-0 mt-1.5" />
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