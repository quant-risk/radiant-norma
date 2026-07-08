'use client'

/**
 * HowItWorks — 3 passos numerados com visual editorial.
 */

import { Database, Cpu, Send } from 'lucide-react'
import { SectionHeader } from '@/components/ui/section-header'

const STEPS = [
  {
    n: '01',
    icon: Database,
    title: 'Conecte seus CADOCs',
    description:
      'Importe o catálogo do BACEN em 1 click. O Radiant Norma detecta automaticamente as versões ativas e configura o pipeline de validação.',
    detail: ['60+ schemas versionados', 'Auto-sync via BACEN', 'Cobertura por CADOC'],
  },
  {
    n: '02',
    icon: Cpu,
    title: 'Valide antes de enviar',
    description:
      'Cada envio passa por 60 regras tipadas antes de tocar no BACEN. Erros bloqueiam, alertas avisam, infos documentam — você decide o limiar.',
    detail: ['Catálogo B/F/C/S', 'Toggle individual por regra', 'Exemplos inline'],
  },
  {
    n: '03',
    icon: Send,
    title: 'Envie e monitore',
    description:
      'Confirmação automática, audit log imutável e dashboard em tempo real. Se algo mudar no BACEN, o Radar te avisa antes do próximo envio.',
    detail: ['Audit chain SHA-256', 'Radar regulatório', 'SSE em tempo real'],
  },
]

export function LandingHowItWorks() {
  return (
    <section id="how-it-works" className="relative py-24 lg:py-32 bg-surface-sunken/30">
      <div className="absolute inset-0 pattern-grid opacity-30" aria-hidden />
      <div className="relative max-w-7xl mx-auto px-6 lg:px-10">
        <SectionHeader
          eyebrow="Como funciona"
          title="Três passos. Zero mágica."
          description="Pipeline determinístico e auditável — você sempre sabe o que está rodando, contra qual versão, e por quê."
        />

        <div className="mt-14 grid grid-cols-1 md:grid-cols-3 gap-6 relative">
          {/* Linha conectora (desktop) */}
          <div className="hidden md:block absolute top-12 left-[16.6%] right-[16.6%] h-px bg-gradient-to-r from-border via-accent-300 to-border" aria-hidden />

          {STEPS.map((step, i) => {
            const Icon = step.icon
            return (
              <div
                key={step.n}
                className="relative animate-fade-up"
                style={{ animationDelay: `${i * 100}ms` }}
              >
                {/* Número editorial */}
                <div className="flex items-center gap-4 mb-6">
                  <div className="size-12 rounded-lg bg-surface-raised border border-border shadow-xs flex items-center justify-center relative z-10">
                    <Icon className="size-5 text-accent-600 dark:text-accent-400" strokeWidth={1.75} />
                  </div>
                  <span className="font-serif text-3xl font-medium text-ink-subtle tracking-tight">
                    {step.n}
                  </span>
                </div>

                <h3 className="font-serif text-xl font-medium text-ink tracking-tight mb-3">
                  {step.title}
                </h3>
                <p className="text-sm text-ink-muted leading-relaxed mb-5">
                  {step.description}
                </p>

                <ul className="space-y-2">
                  {step.detail.map((item) => (
                    <li key={item} className="flex items-center gap-2 text-xs text-ink-muted">
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