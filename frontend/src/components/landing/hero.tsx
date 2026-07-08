'use client'

/**
 * Hero — landing section principal.
 *
 * Editorial premium: wordmark serif + manifesto em Fraunces italic +
 * 2 CTAs (primário gradient + ghost). Visual rhythm: whitespace generoso,
 * linha tracking negativo, hierarchy clara.
 */

import Link from 'next/link'
import {
  ArrowRight,
  Sparkles,
  ShieldCheck,
  Lock,
  Activity,
  Radar,
  BookCheck,
  Send,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Kbd } from '@/components/ui/kbd'
import { cn } from '@/lib/utils'

export function LandingHero() {
  return (
    <section className="relative overflow-hidden">
      {/* Background: paper warm + grain + radial glows */}
      <div className="absolute inset-0 pattern-grid opacity-50" aria-hidden />
      <div
        className="absolute -top-40 left-1/2 -translate-x-1/2 w-[1100px] h-[700px] rounded-full blur-3xl pointer-events-none"
        style={{
          background:
            'radial-gradient(circle, rgba(124,58,237,0.18) 0%, rgba(217,70,239,0.08) 40%, transparent 70%)',
        }}
        aria-hidden
      />
      <div
        className="absolute top-1/3 -right-40 w-[600px] h-[600px] rounded-full blur-3xl pointer-events-none animate-gradient-pan"
        style={{
          background:
            'radial-gradient(circle, rgba(217,70,239,0.12) 0%, transparent 70%)',
        }}
        aria-hidden
      />

      {/* Top nav */}
      <nav className="relative z-10 max-w-7xl mx-auto px-6 lg:px-10 pt-8 flex items-center justify-between">
        <Link href="/" className="flex items-center gap-3 group">
          <div
            className="size-9 rounded-md bg-gradient-to-br from-accent-600 to-magenta-500 flex items-center justify-center text-white font-serif text-base font-medium shadow-glow-accent-sm group-hover:scale-105 transition-transform"
            aria-hidden
          >
            R
          </div>
          <div className="flex flex-col leading-none">
            <span className="font-serif text-base font-medium text-ink tracking-tight">
              Radiant Norma
            </span>
            <span className="text-2xs uppercase tracking-[0.18em] text-ink-subtle font-mono mt-0.5">
              Console Regulatório
            </span>
          </div>
        </Link>

        <div className="flex items-center gap-2">
          <a href="#features" className="hidden sm:block text-sm text-ink-muted hover:text-ink transition-colors px-3 py-2">
            Features
          </a>
          <a href="#how-it-works" className="hidden sm:block text-sm text-ink-muted hover:text-ink transition-colors px-3 py-2">
            Como funciona
          </a>
          <a href="#compliance" className="hidden md:block text-sm text-ink-muted hover:text-ink transition-colors px-3 py-2">
            Compliance
          </a>
          <span className="hidden md:block h-5 w-px bg-border mx-1" />
          <Link href="/login" passHref legacyBehavior={false}>
            <Button asChild variant="ghost" size="sm">
              Entrar
            </Button>
          </Link>
          <Link href="/console" passHref legacyBehavior={false}>
            <Button asChild variant="primary" size="sm" rightIcon={<ArrowRight className="size-3.5" strokeWidth={2.25} />}>
              Abrir Console
            </Button>
          </Link>
        </div>
      </nav>

      {/* Hero copy */}
      <div className="relative z-10 max-w-5xl mx-auto px-6 lg:px-10 pt-24 pb-20 lg:pt-36 lg:pb-32 text-center">
        <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-surface-raised border border-border shadow-xs mb-8">
          <span className="relative flex size-1.5">
            <span className="absolute inset-0 rounded-full bg-success-500 animate-ping opacity-60" />
            <span className="relative size-1.5 rounded-full bg-success-500" />
          </span>
          <span className="text-2xs uppercase tracking-[0.18em] font-mono text-ink-muted">
            Em conformidade · BACEN / CMN 4.966 / IFRS 9
          </span>
        </div>

        <h1 className="font-serif text-5xl sm:text-6xl lg:text-7xl xl:text-display-md font-medium text-ink leading-[0.96] tracking-[-0.035em] mb-8">
          Validação CADOC que{' '}
          <span className="text-gradient-accent italic">pensa junto</span>
          {' '}com você.
        </h1>

        <p className="text-lg lg:text-xl text-ink-muted leading-relaxed max-w-2xl mx-auto mb-10">
          Radar regulatório, catálogo de regras tipadas e auditoria LGPD
          em uma plataforma desenhada para o ciclo regulatório brasileiro.
          {' '}<span className="text-ink">60 regras 3040 monitoradas em tempo real.</span>
        </p>

        <div className="flex items-center justify-center gap-3 flex-wrap mb-8">
          <Link href="/console" passHref legacyBehavior={false}>
            <Button asChild variant="primary" size="lg" rightIcon={<ArrowRight className="size-4" strokeWidth={2.25} />}>
              Abrir Console
            </Button>
          </Link>
          <Link href="/#features" passHref legacyBehavior={false}>
            <Button asChild variant="secondary" size="lg">
              Ver features
            </Button>
          </Link>
        </div>

        <div className="flex items-center justify-center gap-4 text-2xs text-ink-subtle font-mono uppercase tracking-[0.14em]">
          <span className="flex items-center gap-1.5">
            <ShieldCheck className="size-3" strokeWidth={2.25} />
            SOC 2
          </span>
          <span className="size-1 rounded-full bg-border" />
          <span className="flex items-center gap-1.5">
            <Lock className="size-3" strokeWidth={2.25} />
            LGPD
          </span>
          <span className="size-1 rounded-full bg-border" />
          <span className="flex items-center gap-1.5">
            <Sparkles className="size-3" strokeWidth={2.25} />
            BACEN Ready
          </span>
        </div>
      </div>

      {/* Product preview mockup */}
      <div className="relative z-10 max-w-6xl mx-auto px-6 lg:px-10 pb-24">
        <ProductPreviewMockup />
      </div>
    </section>
  )
}

/* ──────────── Product preview mockup ──────────── */

function ProductPreviewMockup() {
  return (
    <div className="relative">
      {/* Glow atrás do card */}
      <div
        className="absolute -inset-x-12 -inset-y-8 rounded-3xl blur-2xl opacity-50"
        style={{
          background:
            'radial-gradient(ellipse, rgba(124,58,237,0.25) 0%, rgba(217,70,239,0.10) 50%, transparent 80%)',
        }}
        aria-hidden
      />

      {/* Browser chrome */}
      <div className="relative rounded-2xl border border-border bg-surface-raised shadow-2xl overflow-hidden">
        <div className="flex items-center gap-2 px-4 h-10 bg-surface-sunken border-b border-border-subtle">
          <div className="flex gap-1.5">
            <span className="size-2.5 rounded-full bg-critical-500/60" />
            <span className="size-2.5 rounded-full bg-warning-500/60" />
            <span className="size-2.5 rounded-full bg-success-500/60" />
          </div>
          <div className="flex-1 mx-6">
            <div className="max-w-md mx-auto h-6 rounded-md bg-surface-raised border border-border-subtle flex items-center px-3 gap-2">
              <Lock className="size-3 text-ink-subtle" />
              <span className="text-2xs font-mono text-ink-subtle">
                console.radiantnorma.com.br
              </span>
            </div>
          </div>
        </div>

        {/* App preview */}
        <div className="grid grid-cols-12 min-h-[420px]">
          {/* Mini sidebar */}
          <aside className="col-span-2 border-r border-border-subtle p-3 space-y-3 bg-surface-raised">
            <div className="flex items-center gap-2 mb-4">
              <div className="size-7 rounded bg-gradient-to-br from-accent-600 to-magenta-500 flex items-center justify-center text-white font-serif text-xs">
                R
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-2xs font-medium text-ink truncate">Radiant Norma</div>
                <div className="text-2xs text-ink-subtle font-mono uppercase tracking-wider truncate">Console</div>
              </div>
            </div>
            {[
              { label: 'Dashboard', active: true },
              { label: 'Envios', active: false },
              { label: 'Radar', active: false, dot: true },
              { label: 'Regras', active: false },
              { label: 'Insights', active: false },
              { label: 'Auditoria', active: false },
            ].map((item) => (
              <div
                key={item.label}
                className={cn(
                  'flex items-center gap-2 px-2 h-7 rounded text-2xs font-medium',
                  item.active
                    ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/50 dark:text-accent-300'
                    : 'text-ink-muted',
                )}
              >
                <div className={cn('size-1 rounded-full', item.active ? 'bg-accent-500' : 'bg-ink-subtle/40')} />
                <span>{item.label}</span>
                {item.dot && (
                  <span className="ml-auto size-1 rounded-full bg-success-500 animate-pulse-soft" />
                )}
              </div>
            ))}
          </aside>

          {/* Mini main */}
          <main className="col-span-10 p-6 space-y-4 bg-surface">
            {/* Header */}
            <div className="flex items-baseline justify-between">
              <div>
                <div className="text-2xs uppercase tracking-[0.18em] font-mono text-ink-subtle mb-1">
                  Status operacional
                </div>
                <div className="text-base font-serif font-medium text-ink">
                  2 alertas críticos exigem ação imediata
                </div>
              </div>
              <div className="px-2 py-0.5 rounded-full bg-critical-50 text-critical-700 text-2xs font-medium uppercase tracking-wide ring-1 ring-inset ring-critical-200/60 flex items-center gap-1">
                <span className="size-1 rounded-full bg-critical-500" />
                ação imediata
              </div>
            </div>

            {/* KPIs */}
            <div className="grid grid-cols-4 gap-3">
              <MiniStat label="Envios (30d)" value="1.247" tone="accent" delta="+12,3%" />
              <MiniStat label="Alertas ativos" value="12" tone="critical" />
              <MiniStat label="Aprovação" value="98,4%" tone="success" delta="+0,4pp" />
              <MiniStat label="CADOCs" value="8" tone="neutral" />
            </div>

            {/* Alert preview */}
            <div className="rounded-lg border border-border bg-surface-raised p-3 relative overflow-hidden">
              <div className="absolute left-0 top-0 bottom-0 w-[3px] bg-critical-500" aria-hidden />
              <div className="flex items-start gap-3 pl-2">
                <div className="size-7 rounded-md bg-critical-50 text-critical-600 flex items-center justify-center shrink-0">
                  <Radar className="size-3.5" strokeWidth={2.25} />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-1.5 mb-1">
                    <span className="px-1.5 py-0.5 rounded-full bg-critical-50 text-critical-700 text-2xs font-medium uppercase ring-1 ring-inset ring-critical-200/60">
                      Crítico
                    </span>
                    <span className="text-2xs font-mono text-ink-muted">CADOC 3040</span>
                    <span className="text-2xs text-ink-subtle ml-auto font-mono">há 12 min</span>
                  </div>
                  <div className="text-xs font-serif font-medium text-ink leading-snug">
                    Nova versão do layout BACEN publicada — reanálise automática em curso
                  </div>
                </div>
              </div>
            </div>
          </main>
        </div>
      </div>

      {/* Floating badges em volta do mockup */}
      <div className="hidden lg:block absolute -left-8 top-1/3 animate-fade-up" style={{ animationDelay: '200ms' }}>
        <div className="px-3 py-2 rounded-lg bg-surface-raised border border-border shadow-md flex items-center gap-2">
          <Activity className="size-3.5 text-success-600" strokeWidth={2.25} />
          <span className="text-xs font-medium text-ink">Audit chain OK</span>
        </div>
      </div>
      <div className="hidden lg:block absolute -right-6 top-2/3 animate-fade-up" style={{ animationDelay: '400ms' }}>
        <div className="px-3 py-2 rounded-lg bg-surface-raised border border-border shadow-md flex items-center gap-2">
          <Sparkles className="size-3.5 text-accent-600" strokeWidth={2.25} />
          <span className="text-xs font-medium text-ink">SHA-256 hash chain</span>
        </div>
      </div>
    </div>
  )
}

function MiniStat({
  label,
  value,
  tone,
  delta,
}: {
  label: string
  value: string
  tone: 'neutral' | 'accent' | 'success' | 'warning' | 'critical'
  delta?: string
}) {
  const toneClass: Record<typeof tone, string> = {
    neutral: 'text-ink',
    accent: 'text-gradient-accent',
    success: 'text-success-700 dark:text-success-300',
    warning: 'text-warning-700 dark:text-warning-300',
    critical: 'text-critical-700 dark:text-critical-300',
  }
  return (
    <div className="rounded-lg border border-border bg-surface-raised p-3">
      <div className="text-2xs uppercase tracking-[0.14em] font-mono font-medium text-ink-subtle mb-2">
        {label}
      </div>
      <div className="flex items-baseline gap-2">
        <span className={cn('text-xl font-serif font-medium tracking-tight', toneClass[tone])}>
          {value}
        </span>
        {delta && (
          <span className="text-2xs font-mono text-success-600">↑ {delta}</span>
        )}
      </div>
    </div>
  )
}