/**
 * Features — grid de features do produto.
 *
 * Layout editorial: eyebrow → 10 cards em grid 5×2. Cada card com glyph,
 * título serif, descrição muted, e (opcional) micro-visual.
 */

import {
  Radar,
  BookCheck,
  ShieldCheck,
  Sparkles,
  Send,
  History,
  Wand2,
  Database,
  Webhook,
  Store,
  CheckCircle2,
} from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { SectionHeader } from '@/components/ui/section-header'
import { Divider } from '@/components/ui/divider'
import { cn } from '@/lib/utils'

interface Feature {
  icon: React.ComponentType<{ className?: string; strokeWidth?: number | string }>
  title: string
  description: string
  badge?: string
  visual?: React.ReactNode
}

const FEATURES: Feature[] = [
  {
    icon: Wand2,
    title: 'Motor de Geração',
    description:
      '10 generators de CADOC (3040 pronto, 9 planejados). Recebe CanonicalDocument IF-agnóstico, gera XML validado pelo schema registry.',
    badge: 'v3.36',
    visual: <GeneratorVisual />,
  },
  {
    icon: Database,
    title: 'Ingest multi-fonte',
    description:
      '5 conectores (Manual, API, File, DB, MCP) produzem CanonicalDocument. Wizard guiado para configurar cada fonte.',
    badge: '5 adapters',
    visual: <IngestVisual />,
  },
  {
    icon: Sparkles,
    title: 'Wizard guiado',
    description:
      'Wizard multi-etapas: escolher CADOC → mapear campos → pré-validar → gerar. Onboarding 0 → primeiro envio em 15 min.',
    badge: 'Novo',
    visual: <WizardVisual />,
  },
  {
    icon: CheckCircle2,
    title: 'Validação L1 → L4',
    description:
      'XSD (L1) → semântico (L2) → cross-doc (L3) → histórico (L4). 1.099 regras. L4 diff mostra mudanças vs envio anterior.',
    badge: '1.099 regras',
    visual: <ValidationVisual />,
  },
  {
    icon: Sparkles,
    title: 'AI Insights',
    description:
      'LLM interpreta audit_log em linguagem natural. Recomendações heurísticas determinísticas. Streaming SSE. Opt-in por tenant.',
    badge: 'IA',
    visual: <InsightsVisual />,
  },
  {
    icon: Radar,
    title: 'Radar Regulatório',
    description:
      'Varredura automática a cada 6h em URLs oficiais do BACEN. Detecta mudanças em leiautes, instruções e prazos.',
    badge: 'Live',
    visual: <RadarVisual />,
  },
  {
    icon: BookCheck,
    title: 'Catálogo de regras',
    description:
      'Regras tipadas: estruturais, formato, campos, semânticas. Cobertura 76% no 3040, 100% no 3050. Toggle individual por IF.',
    visual: <RulesVisual />,
  },
  {
    icon: Send,
    title: 'Envios STA',
    description:
      'Pipeline STA-h/STA-ws nativo. Retry exponencial, DLQ, chunked upload (range API). Protocol STA rastreado.',
    visual: <EnvioVisual />,
  },
  {
    icon: Webhook,
    title: 'Webhooks outbound',
    description:
      'Registry de callbacks HTTP disparados em eventos. HMAC-SHA256 assinado. Delivery log + retry. Conecta a sistemas da IF sem polling.',
    badge: 'HMAC',
    visual: <WebhookVisual />,
  },
  {
    icon: Store,
    title: 'Marketplace de regras',
    description:
      'IFs publicam regras customizadas; outras IFs instalam. Rating por estrela. Versionamento. Catálogo de casos especiais do mercado.',
    badge: 'Comunidade',
    visual: <MarketplaceVisual />,
  },
  {
    icon: History,
    title: 'Auditoria LGPD / SOC 2',
    description:
      'Audit log tamper-evident com SHA-256 hash chain. Verificável por SELECT. Pilot3 ESG-first via IN BCB 694/2025.',
    badge: 'Verifiable',
    visual: <HistoryVisual />,
  },
  {
    icon: Send,
    title: 'Envios STA',
    description:
      'Pipeline completo de envio, validação e confirmação. Deduplicação automática, retry inteligente e audit log imutável por envio.',
    visual: <EnvioVisual />,
  },
  {
    icon: Sparkles,
    title: 'Insights operacionais',
    description:
      'Anomalia, tendência e recomendação geradas por heurística própria. Top regras falhando, mapa de calor CADOC × dia, sugestões acionáveis.',
    badge: 'IA',
    visual: <InsightsVisual />,
  },
  {
    icon: ShieldCheck,
    title: 'Compliance LGPD / SOC 2',
    description:
      'Audit log tamper-evident com SHA-256 hash chain. Logs não contêm dados pessoais sensíveis. Retention configurável, padrão 5 anos.',
    visual: <ComplianceVisual />,
  },
  {
    icon: History,
    title: 'Auditoria completa',
    description:
      'Linha do tempo de toda mutação na plataforma. Filtros por ação, IF, período. Exportação CSV/JSON pra integrações externas.',
    visual: <HistoryVisual />,
  },
]

export function LandingFeatures() {
  return (
    <section id="features" className="relative py-24 lg:py-32">
      <div className="max-w-7xl mx-auto px-6 lg:px-10">
        <SectionHeader
          eyebrow="Capacidades"
          title="Tudo o que sua IF precisa pra operar CADOC sem dor"
          description="Uma plataforma desenhada por quem entende o ciclo regulatório brasileiro — não um wrapper de ferramenta genérica."
          align="between"
        />

        <div className="mt-14 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
          {FEATURES.map((feature, i) => {
            const Icon = feature.icon
            return (
              <Card
                key={feature.title}
                padding="none"
                className="group hover:border-border-strong hover:shadow-md transition-all duration-240 ease-out-expo hover:-translate-y-px animate-fade-up"
                style={{ animationDelay: `${i * 60}ms` }}
              >
                <div className="p-6">
                  <div className="flex items-start justify-between mb-5">
                    <div className="size-11 rounded-lg bg-gradient-to-br from-accent-50 to-magenta-500/10 dark:from-accent-950/40 dark:to-magenta-500/10 ring-1 ring-inset ring-accent-200/60 dark:ring-accent-800/40 flex items-center justify-center text-accent-600 dark:text-accent-300 group-hover:scale-105 transition-transform">
                      <Icon className="size-5" strokeWidth={1.75} />
                    </div>
                    {feature.badge && (
                      <Badge tone="accent" variant="soft" dot size="sm">
                        {feature.badge}
                      </Badge>
                    )}
                  </div>

                  <h3 className="font-serif text-lg font-medium text-ink tracking-tight mb-2">
                    {feature.title}
                  </h3>
                  <p className="text-sm text-ink-muted leading-relaxed">
                    {feature.description}
                  </p>
                </div>

                {feature.visual && (
                  <div className="border-t border-border-subtle bg-surface-sunken/40 px-6 py-4">
                    {feature.visual}
                  </div>
                )}
              </Card>
            )
          })}
        </div>
      </div>
    </section>
  )
}

/* ──────────── Micro-visuals (estilo terminal/dashboard) ──────────── */

function WizardVisual() {
  return (
    <div className="flex items-center gap-1.5">
      {['1', '2', '3', '4'].map((n, i) => (
        <div
          key={n}
          className="flex items-center gap-1.5"
        >
          <div
            className={cn(
              'size-7 rounded-md flex items-center justify-center text-2xs font-mono font-medium',
              i < 2
                ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/40 dark:text-accent-300 ring-1 ring-inset ring-accent-200/60 dark:ring-accent-800/40'
                : 'bg-surface-sunken text-ink-subtle ring-1 ring-inset ring-border-subtle',
            )}
          >
            {n}
          </div>
          {i < 3 && (
            <div className="w-3 h-px bg-border" />
          )}
        </div>
      ))}
    </div>
  )
}

function ValidationVisual() {
  return (
    <div className="space-y-1.5">
      {['L1 XSD', 'L2 Semântico', 'L3 Cross-doc', 'L4 Histórico'].map((layer, i) => (
        <div key={layer} className="flex items-center gap-2 text-2xs">
          <span
            className={cn(
              'size-3.5 rounded-sm flex items-center justify-center text-2xs font-mono',
              i < 2
                ? 'bg-accent-500 text-white'
                : 'bg-surface-sunken text-ink-muted ring-1 ring-inset ring-border',
            )}
          >
            {i + 1}
          </span>
          <span className="text-ink-muted">{layer}</span>
        </div>
      ))}
    </div>
  )
}

function WebhookVisual() {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2 text-2xs">
        <div className="size-1.5 rounded-full bg-success-500 animate-pulse-soft" />
        <span className="text-ink-muted">POST /webhook</span>
        <span className="text-2xs font-mono text-ink-subtle ml-auto">200</span>
      </div>
      <div className="flex items-center gap-2 text-2xs">
        <div className="size-1.5 rounded-full bg-success-500 animate-pulse-soft" />
        <span className="text-ink-muted">envio.approved</span>
        <span className="text-2xs font-mono text-ink-subtle ml-auto">HMAC ✓</span>
      </div>
      <div className="flex items-center gap-2 text-2xs">
        <div className="size-1.5 rounded-full bg-warning-500 animate-pulse-soft" />
        <span className="text-ink-muted">retry...</span>
        <span className="text-2xs font-mono text-ink-subtle ml-auto">3×</span>
      </div>
    </div>
  )
}

function MarketplaceVisual() {
  return (
    <div className="space-y-1.5">
      {['F23 (3050)', 'S05 (3040)', 'C04 (3040)'].map((rule, i) => (
        <div key={rule} className="flex items-center gap-2 text-2xs">
          <span className="font-mono text-accent-600 dark:text-accent-400 shrink-0">
            {rule}
          </span>
          <div className="flex gap-0.5 ml-auto">
            {[1, 2, 3, 4, 5].map((s) => (
              <span
                key={s}
                className={cn(
                  'size-1.5 rounded-full',
                  s <= 4 - i ? 'bg-warning-500' : 'bg-ink-subtle/30',
                )}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function GeneratorVisual() {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-1.5">
        <div className="flex items-center gap-1 px-1.5 py-0.5 rounded text-2xs font-mono bg-accent-50 text-accent-700 dark:bg-accent-950/40 dark:text-accent-300">
          <span className="size-1 rounded-full bg-accent-500" />
          3040
        </div>
        <div className="flex-1 h-px bg-gradient-to-r from-accent-300 to-magenta-300 dark:from-accent-700 dark:to-magenta-500" />
        <Sparkles className="size-2.5 text-magenta-500" />
        <div className="flex-1 h-px bg-gradient-to-r from-magenta-300 to-accent-300 dark:from-magenta-500 dark:to-accent-700" />
        <div className="flex items-center gap-1 px-1.5 py-0.5 rounded text-2xs font-mono bg-magenta-500/10 text-magenta-600 dark:text-magenta-300">
          <span className="size-1 rounded-full bg-magenta-500" />
          XML
        </div>
      </div>
      <div className="text-2xs font-mono text-ink-subtle mt-1.5">
        canonical → generate → validate (275 regras)
      </div>
    </div>
  )
}

function IngestVisual() {
  return (
    <div className="grid grid-cols-5 gap-1.5">
      {[
        { label: 'Manual', active: true },
        { label: 'API', active: true },
        { label: 'File', active: true },
        { label: 'DB', active: false },
        { label: 'MCP', active: false },
      ].map((s) => (
        <div
          key={s.label}
          className={cn(
            'rounded text-2xs font-mono text-center py-1 ring-1 ring-inset',
            s.active
              ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/40 dark:text-accent-300 ring-accent-200/60 dark:ring-accent-800/40'
              : 'bg-surface-sunken/40 text-ink-subtle ring-border-subtle',
          )}
        >
          {s.label}
        </div>
      ))}
    </div>
  )
}

function RadarVisual() {
  return (
    <div className="space-y-2.5">
      {[
        { severity: 'critical', label: 'Nova versão layout 3040', time: 'há 12 min' },
        { severity: 'warning', label: 'Instrução normativa atualizada', time: 'há 2h' },
        { severity: 'info', label: 'CMN 4.966 — sem mudanças', time: 'há 6h' },
      ].map((row, i) => (
        <div key={i} className="flex items-center gap-2.5 text-xs">
          <span className={cn(
            'size-1.5 rounded-full shrink-0',
            row.severity === 'critical' ? 'bg-critical-500' :
            row.severity === 'warning' ? 'bg-warning-500' : 'bg-info-500'
          )} />
          <span className="flex-1 truncate text-ink-muted">{row.label}</span>
          <span className="text-2xs font-mono text-ink-subtle shrink-0">{row.time}</span>
        </div>
      ))}
    </div>
  )
}

function RulesVisual() {
  return (
    <div className="grid grid-cols-7 gap-1">
      {Array.from({ length: 21 }, (_, i) => {
        const intensity = Math.sin(i * 0.7) * 0.5 + 0.5
        const op = 0.1 + intensity * 0.6
        return (
          <div
            key={i}
            className="h-6 rounded-sm bg-accent-500"
            style={{ opacity: op }}
          />
        )
      })}
    </div>
  )
}

function EnvioVisual() {
  return (
    <div className="flex items-center gap-2 text-xs">
      <div className="flex-1 flex items-center gap-1.5 px-2 py-1.5 rounded bg-surface-raised border border-border">
        <Send className="size-3 text-info-600" strokeWidth={2.25} />
        <span className="text-2xs font-mono text-ink-muted">3040</span>
        <span className="ml-auto text-2xs font-mono text-ink-subtle">2,4s</span>
      </div>
      <div className="text-ink-subtle">→</div>
      <div className="flex-1 flex items-center gap-1.5 px-2 py-1.5 rounded bg-success-50 border border-success-200/60">
        <ShieldCheck className="size-3 text-success-600" strokeWidth={2.25} />
        <span className="text-2xs font-medium text-success-700">aprovado</span>
      </div>
    </div>
  )
}

function InsightsVisual() {
  return (
    <div className="flex items-end gap-1 h-12">
      {[0.4, 0.55, 0.45, 0.7, 0.6, 0.85, 0.75, 0.95].map((h, i) => (
        <div
          key={i}
          className="flex-1 rounded-sm bg-gradient-to-t from-accent-500 to-magenta-500"
          style={{ height: `${h * 100}%` }}
        />
      ))}
    </div>
  )
}

function ComplianceVisual() {
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 flex items-center gap-2">
        {[1, 2, 3, 4, 5].map((n) => (
          <div
            key={n}
            className="size-5 rounded-full bg-success-50 ring-1 ring-inset ring-success-200/60 flex items-center justify-center"
          >
            <span className="text-2xs font-mono font-medium text-success-700">{n}</span>
          </div>
        ))}
        <div className="ml-2 text-2xs font-mono text-ink-muted">SHA-256 chain ✓</div>
      </div>
    </div>
  )
}

function HistoryVisual() {
  return (
    <div className="relative pl-4">
      <div className="absolute left-1 top-2 bottom-2 w-px bg-gradient-to-b from-border via-border to-transparent" aria-hidden />
      {[
        { label: 'Envio aprovado', time: 'há 5 min' },
        { label: 'Radar atualizado', time: 'há 12 min' },
        { label: 'Login IF', time: 'há 1h' },
      ].map((row, i) => (
        <div key={i} className="relative flex items-center gap-2 py-0.5 text-2xs">
          <span className="absolute -left-3 size-1.5 rounded-full bg-accent-500 ring-2 ring-surface" />
          <span className="flex-1 text-ink-muted truncate">{row.label}</span>
          <span className="font-mono text-ink-subtle shrink-0">{row.time}</span>
        </div>
      ))}
    </div>
  )
}