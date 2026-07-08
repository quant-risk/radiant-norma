'use client'

/**
 * /login — capa editorial split-panel.
 *
 * Inspiração: capa de revista de negócios + terminal institucional.
 * Lado esquerdo: hero editorial com wordmark Fraunces + headline + manifesto.
 * Lado direito: form minimalista com cards selecionáveis (sem select nativo).
 *
 * Em produção: integrado com IdP (Keycloak/Okta).
 */

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import {
  Building2,
  Landmark,
  ShieldCheck,
  ArrowRight,
  ArrowUpRight,
  Shield,
  Lock,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

interface DemoIF {
  id: string
  role: 'if' | 'admin' | 'auditor'
  label: string
  description: string
  icon: React.ComponentType<{ className?: string; strokeWidth?: number | string }>
  cif: string
}

const DEMO_IFS: DemoIF[] = [
  {
    id: 'demo',
    role: 'if',
    label: 'Demo IF',
    description: 'Sociedade de Crédito Direto (SCD)',
    icon: Building2,
    cif: 'SCD-001',
  },
  {
    id: 'demo-banco',
    role: 'if',
    label: 'Demo Banco',
    description: 'Instituição bancária multi-propósito',
    icon: Landmark,
    cif: 'BCO-014',
  },
  {
    id: 'demo-admin',
    role: 'admin',
    label: 'Demo Admin',
    description: 'Acesso administrativo (regulador interno)',
    icon: ShieldCheck,
    cif: 'REG-099',
  },
]

export default function LoginPage() {
  const [selected, setSelected] = useState<DemoIF>(DEMO_IFS[0])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const router = useRouter()

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)
    try {
      const r = await fetch('/api/login', {
        method: 'POST',
        body: JSON.stringify({ if_id: selected.id, role: selected.role }),
        headers: { 'Content-Type': 'application/json' },
      })
      if (!r.ok) {
        const err = await r.json().catch(() => ({ error: 'login failed' }))
        setError(err.error || 'Falha no login')
        return
      }
      router.push('/')
      router.refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Erro de rede')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex flex-col lg:flex-row bg-surface">
      {/* Left: brand panel — editorial hero */}
      <aside className="relative lg:w-[58%] xl:w-[60%] bg-ink text-ink-inverse overflow-hidden flex flex-col">
        {/* Pattern grid sutil + glows de accent */}
        <div className="absolute inset-0 pattern-grid opacity-[0.06]" aria-hidden />
        <div
          className="absolute -top-40 -left-40 size-[28rem] rounded-full bg-accent-600/30 blur-3xl animate-gradient-pan"
          style={{ backgroundImage: 'radial-gradient(circle, rgba(124,58,237,0.4) 0%, rgba(217,70,239,0.1) 50%, transparent 70%)' }}
          aria-hidden
        />
        <div
          className="absolute -bottom-40 -right-40 size-[32rem] rounded-full blur-3xl"
          style={{ backgroundImage: 'radial-gradient(circle, rgba(217,70,239,0.18) 0%, transparent 70%)' }}
          aria-hidden
        />

        {/* Hairline accent diagonal */}
        <div
          className="absolute top-0 right-0 w-px h-full bg-gradient-to-b from-transparent via-accent-500/40 to-transparent"
          aria-hidden
        />

        <div className="relative flex flex-col h-full p-10 lg:p-14 xl:p-20">
          {/* Wordmark */}
          <div className="flex items-center gap-3">
            <div
              className="size-11 rounded-md bg-gradient-to-br from-accent-500 to-magenta-500 flex items-center justify-center text-white font-serif text-lg font-medium shadow-glow-accent"
              aria-hidden
            >
              R
            </div>
            <div className="flex flex-col leading-none">
              <span className="font-serif text-lg font-medium tracking-tight">
                Radiant Norma
              </span>
              <span className="text-2xs uppercase tracking-[0.22em] text-ink-subtle font-mono mt-1">
                Console Regulatório
              </span>
            </div>
          </div>

          {/* Eyebrow */}
          <div className="mt-auto pt-20">
            <div className="inline-flex items-center gap-2 mb-7">
              <span className="size-1.5 rounded-full bg-accent-400 animate-pulse-soft" />
              <span className="text-2xs uppercase tracking-[0.22em] font-mono text-accent-300">
                v2.1 · Em conformidade com BACEN / CMN 4.966
              </span>
            </div>

            {/* Hero copy */}
            <h1 className="font-serif text-[3rem] lg:text-[3.75rem] xl:text-[4.5rem] leading-[0.98] tracking-[-0.035em] font-medium max-w-2xl mb-8">
              Validação CADOC que{' '}
              <span className="text-gradient-accent italic">pensa junto</span>
              {' '}com você.
            </h1>
            <p className="text-base lg:text-lg text-ink-muted leading-relaxed max-w-xl mb-12">
              Radar regulatório, catálogo de regras tipadas e auditoria LGPD
              em uma plataforma desenhada para o ciclo regulatório brasileiro:
              CMN 4.966, IFRS 9, Basileia.
            </p>

            {/* Manifest — features */}
            <ul className="space-y-3.5 text-sm text-ink-inverse">
              {[
                '60 regras tipadas para CADOC 3040',
                'Detecção automática de mudanças BACEN',
                'Insights de risco baseados nos seus envios',
              ].map((item, i) => (
                <li key={i} className="flex items-center gap-3">
                  <span className="size-5 rounded-md bg-gradient-to-br from-accent-500 to-magenta-500 flex items-center justify-center shrink-0">
                    <ArrowRight className="size-3 text-white" strokeWidth={2.5} />
                  </span>
                  <span className="tracking-tight">{item}</span>
                </li>
              ))}
            </ul>

            {/* Trust badges */}
            <div className="mt-12 flex items-center gap-3 flex-wrap">
              <Badge tone="neutral" variant="outline" size="sm" className="bg-transparent border-border-strong text-ink-muted">
                <Lock className="size-3" />
                SOC 2
              </Badge>
              <Badge tone="neutral" variant="outline" size="sm" className="bg-transparent border-border-strong text-ink-muted">
                <Shield className="size-3" />
                LGPD
              </Badge>
              <Badge tone="neutral" variant="outline" size="sm" className="bg-transparent border-border-strong text-ink-muted">
                BACEN Ready
              </Badge>
            </div>
          </div>

          <footer className="mt-16 flex items-center justify-between text-2xs text-ink-subtle font-mono">
            <span>© 2026 Radiant Norma</span>
            <span className="uppercase tracking-[0.18em]">
              Dev mode · RADIANT_DEV_TOKEN=1
            </span>
          </footer>
        </div>
      </aside>

      {/* Right: form panel */}
      <main className="flex-1 flex items-center justify-center p-8 lg:p-12 xl:p-16 bg-surface">
        <div className="w-full max-w-md animate-fade-up">
          <div className="mb-10">
            <p className="eyebrow mb-3">Acesso seguro</p>
            <h2 className="font-serif text-3xl lg:text-4xl font-medium text-ink mb-2 tracking-tight">
              Entrar no console
            </h2>
            <p className="text-sm text-ink-muted leading-relaxed">
              Selecione a IF para entrar no ambiente de demonstração.
              Em produção, autenticação via Keycloak / Okta com MFA.
            </p>
          </div>

          <form onSubmit={handleLogin} className="space-y-6">
            <div>
              <div className="flex items-baseline justify-between mb-3">
                <label className="text-xs font-medium text-ink-muted uppercase tracking-[0.14em] font-mono">
                  Instituição
                </label>
                <span className="text-2xs text-ink-subtle font-mono">
                  {DEMO_IFS.length} disponíveis
                </span>
              </div>
              <div className="space-y-2.5">
                {DEMO_IFS.map((demo) => {
                  const Icon = demo.icon
                  const isSelected = selected.id === demo.id
                  return (
                    <button
                      key={demo.id}
                      type="button"
                      onClick={() => setSelected(demo)}
                      className={cn(
                        'group relative w-full flex items-center gap-4 p-4 rounded-xl text-left',
                        'border bg-surface-raised',
                        'transition-all duration-240 ease-out-expo',
                        isSelected
                          ? 'border-accent-500 bg-accent-50/50 dark:bg-accent-950/20 ring-1 ring-accent-500/30 shadow-sm'
                          : 'border-border hover:border-border-strong hover:bg-surface-sunken/40',
                      )}
                    >
                      {/* Selected indicator rail */}
                      {isSelected && (
                        <span
                          className="absolute left-0 top-3 bottom-3 w-[3px] rounded-r-full bg-gradient-to-b from-accent-500 to-magenta-500"
                          aria-hidden
                        />
                      )}

                      <div
                        className={cn(
                          'size-11 rounded-lg flex items-center justify-center shrink-0 transition-all duration-240',
                          isSelected
                            ? 'bg-gradient-to-br from-accent-600 to-magenta-500 text-white shadow-glow-accent-sm'
                            : 'bg-surface-sunken text-ink-muted ring-1 ring-inset ring-border',
                        )}
                      >
                        <Icon className="size-5" strokeWidth={isSelected ? 2.25 : 1.75} />
                      </div>

                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-0.5">
                          <span className="text-sm font-medium text-ink tracking-tight">
                            {demo.label}
                          </span>
                          <span className="text-2xs font-mono text-ink-subtle uppercase tracking-wider">
                            {demo.cif}
                          </span>
                        </div>
                        <p className="text-xs text-ink-muted truncate">
                          {demo.description}
                        </p>
                      </div>

                      <div
                        className={cn(
                          'size-5 rounded-full border-2 flex items-center justify-center shrink-0 transition-all duration-240',
                          isSelected
                            ? 'border-accent-600 bg-accent-600 scale-100'
                            : 'border-border-strong scale-90 group-hover:scale-100',
                        )}
                        aria-hidden
                      >
                        {isSelected && (
                          <svg
                            viewBox="0 0 12 12"
                            className="size-full text-white"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="2.5"
                          >
                            <path
                              d="M3 6l2 2 4-4"
                              strokeLinecap="round"
                              strokeLinejoin="round"
                            />
                          </svg>
                        )}
                      </div>
                    </button>
                  )
                })}
              </div>
            </div>

            {error && (
              <div className="flex items-start gap-2.5 p-3.5 rounded-lg bg-critical-50 dark:bg-critical-950/30 border border-critical-200/60 dark:border-critical-800/40 text-critical-700 dark:text-critical-300 text-xs animate-fade-in">
                <svg
                  className="size-3.5 mt-0.5 shrink-0"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.25"
                >
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="8" x2="12" y2="12" />
                  <line x1="12" y1="16" x2="12.01" y2="16" />
                </svg>
                <span>{error}</span>
              </div>
            )}

            <Button
              type="submit"
              loading={loading}
              fullWidth
              size="lg"
              rightIcon={<ArrowRight className="size-4" strokeWidth={2.25} />}
            >
              Entrar como {selected.label}
            </Button>

            <div className="flex items-center gap-3 pt-2">
              <span className="flex-1 h-px bg-border" />
              <span className="text-2xs uppercase tracking-[0.18em] text-ink-subtle font-mono">
                Em produção
              </span>
              <span className="flex-1 h-px bg-border" />
            </div>

            <p className="text-center text-xs text-ink-subtle leading-relaxed">
              Autenticação federada via Keycloak / Okta
              <br />
              com MFA e rotação automática de chaves.
              <a
                href="#"
                className="inline-flex items-center gap-1 mt-2 text-accent-600 dark:text-accent-400 hover:underline font-medium"
              >
                Documentação
                <ArrowUpRight className="size-3" />
              </a>
            </p>
          </form>
        </div>
      </main>
    </div>
  )
}