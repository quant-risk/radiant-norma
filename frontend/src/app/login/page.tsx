'use client'

/**
 * /login — escolha de IF para dev (Sprint 7c v2.0.0 MVP).
 *
 * Visual: card central sobre fundo com pattern grid sutil + gradient
 * accent. Apresenta 3 demo IFs como cards selecionáveis (não select
 * nativo), cada um com role badge. Empty space generoso (whitespace
 * pesado tipo Linear).
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
  Sparkles,
  AlertCircle,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

interface DemoIF {
  id: string
  role: 'if' | 'admin' | 'auditor'
  label: string
  description: string
  icon: React.ComponentType<{ className?: string }>
}

const DEMO_IFS: DemoIF[] = [
  {
    id: 'demo',
    role: 'if',
    label: 'Demo IF',
    description: 'Sociedade de Crédito Direto (SCD)',
    icon: Building2,
  },
  {
    id: 'demo-banco',
    role: 'if',
    label: 'Demo Banco',
    description: 'Instituição bancária multi-propósito',
    icon: Landmark,
  },
  {
    id: 'demo-admin',
    role: 'admin',
    label: 'Demo Admin',
    description: 'Acesso administrativo (regulador interno)',
    icon: ShieldCheck,
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
      {/* Left: brand panel */}
      <aside className="relative lg:w-1/2 xl:w-[55%] bg-slate-950 text-white overflow-hidden flex flex-col">
        {/* Subtle grid + glow */}
        <div className="absolute inset-0 pattern-grid opacity-30" aria-hidden />
        <div
          className="absolute -top-32 -left-32 size-96 rounded-full bg-accent-600/20 blur-3xl"
          aria-hidden
        />
        <div
          className="absolute -bottom-32 -right-32 size-96 rounded-full bg-accent-400/10 blur-3xl"
          aria-hidden
        />

        <div className="relative flex flex-col h-full p-10 lg:p-16">
          {/* Logo */}
          <div className="flex items-center gap-3 mb-16">
            <div
              className="size-10 rounded-xl bg-gradient-to-br from-accent-500 to-accent-700 flex items-center justify-center shadow-lg"
              aria-hidden
            >
              <svg
                viewBox="0 0 24 24"
                className="size-6 text-white"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M12 2L2 7l10 5 10-5-10-5z" />
                <path d="M2 17l10 5 10-5" />
                <path d="M2 12l10 5 10-5" />
              </svg>
            </div>
            <div>
              <div className="text-base font-semibold">Radiant Norma</div>
              <div className="text-2xs text-slate-400 uppercase tracking-wider">
                Console regulatório
              </div>
            </div>
          </div>

          {/* Hero copy */}
          <div className="flex-1 flex flex-col justify-center max-w-md">
            <Badge tone="accent" variant="soft" className="self-start mb-6">
              <Sparkles className="size-3" />
              v2.1 · Sprint 8
            </Badge>
            <h1 className="text-4xl lg:text-5xl font-semibold leading-[1.1] tracking-tight mb-6">
              Validação CADOC que <span className="text-gradient-accent">pensa junto</span> com você.
            </h1>
            <p className="text-base text-slate-300 leading-relaxed mb-10">
              Radar regulatório, catálogo de regras tipadas e auditoria LGPD
              em uma plataforma desenhada para o ciclo regulatório brasileiro:
              CMN 4.966, IFRS 9, Basileia.
            </p>

            <ul className="space-y-3 text-sm text-slate-300">
              {[
                '60 regras tipadas para CADOC 3040',
                'Detecção automática de mudanças BACEN',
                'Insights de risco baseados nos seus envios',
              ].map((item, i) => (
                <li key={i} className="flex items-center gap-2.5">
                  <span
                    className="size-1.5 rounded-full bg-accent-400"
                    aria-hidden
                  />
                  {item}
                </li>
              ))}
            </ul>
          </div>

          <footer className="text-2xs text-slate-500 mt-10">
            Dev mode · RADIANT_DEV_TOKEN=1 ativo
          </footer>
        </div>
      </aside>

      {/* Right: form */}
      <main className="flex-1 flex items-center justify-center p-6 lg:p-12">
        <div className="w-full max-w-md animate-fade-in">
          <div className="mb-8">
            <h2 className="text-2xl font-semibold text-ink mb-2 tracking-tight">
              Entrar no console
            </h2>
            <p className="text-sm text-ink-muted">
              Selecione a IF para entrar no ambiente de demonstração.
            </p>
          </div>

          <form onSubmit={handleLogin} className="space-y-6">
            <div className="space-y-2">
              <label className="text-xs font-medium text-ink-muted uppercase tracking-wider">
                Instituição
              </label>
              <div className="space-y-2">
                {DEMO_IFS.map((demo) => {
                  const Icon = demo.icon
                  const isSelected = selected.id === demo.id
                  return (
                    <button
                      key={demo.id}
                      type="button"
                      onClick={() => setSelected(demo)}
                      className={cn(
                        'w-full flex items-center gap-3 p-3 rounded-lg border text-left',
                        'transition-all duration-150',
                        isSelected
                          ? 'border-accent-400 bg-accent-50 dark:bg-accent-950 ring-2 ring-accent-400/30'
                          : 'border-border bg-surface-raised hover:border-border-strong hover:bg-surface-sunken',
                      )}
                    >
                      <div
                        className={cn(
                          'size-10 rounded-lg flex items-center justify-center shrink-0',
                          isSelected
                            ? 'bg-accent-600 text-white'
                            : 'bg-surface-sunken text-ink-muted',
                        )}
                      >
                        <Icon className="size-5" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-semibold text-ink">
                            {demo.label}
                          </span>
                          <Badge
                            tone={
                              demo.role === 'admin' ? 'accent' : 'neutral'
                            }
                            variant="soft"
                          >
                            {demo.role}
                          </Badge>
                        </div>
                        <p className="text-xs text-ink-muted truncate">
                          {demo.description}
                        </p>
                      </div>
                      <div
                        className={cn(
                          'size-4 rounded-full border-2 transition-all shrink-0',
                          isSelected
                            ? 'border-accent-600 bg-accent-600'
                            : 'border-border-strong',
                        )}
                        aria-hidden
                      >
                        {isSelected && (
                          <svg
                            viewBox="0 0 12 12"
                            className="size-full text-white"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="2"
                          >
                            <path d="M3 6l2 2 4-4" strokeLinecap="round" strokeLinejoin="round" />
                          </svg>
                        )}
                      </div>
                    </button>
                  )
                })}
              </div>
            </div>

            {error && (
              <div className="flex items-start gap-2 p-3 rounded-md bg-critical-50 dark:bg-critical-950 border border-critical-200 dark:border-critical-800 text-critical-700 dark:text-critical-300 text-xs">
                <AlertCircle className="size-3.5 mt-0.5 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <Button
              type="submit"
              loading={loading}
              fullWidth
              size="lg"
              rightIcon={<ArrowRight className="size-4" />}
            >
              Entrar como {selected.label}
            </Button>

            <div className="text-center pt-2">
              <p className="text-2xs text-ink-subtle">
                Em produção, autenticação via Keycloak / Okta
                <br />
                com MFA e rotação automática de chaves
              </p>
            </div>
          </form>
        </div>
      </main>
    </div>
  )
}