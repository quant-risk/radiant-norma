'use client'

/**
 * LandingFooter — footer institucional.
 */

import Link from 'next/link'
import type { Route } from 'next'
import { Lock, ShieldCheck } from 'lucide-react'

interface FooterLink {
  label: string
  href: string
  external?: boolean
}

const COLUMNS: Array<{ label: string; links: FooterLink[] }> = [
  {
    label: 'Produto',
    links: [
      { label: 'Features', href: '#features' },
      { label: 'Como funciona', href: '#how-it-works' },
      { label: 'Compliance', href: '#compliance' },
      { label: 'Console', href: '/console' },
    ],
  },
  {
    label: 'Recursos',
    links: [
      { label: 'Regras 3040', href: '/regras' },
      { label: 'API Reference', href: 'mailto:api@radiantnorma.com.br', external: true },
      { label: 'Status', href: 'mailto:status@radiantnorma.com.br', external: true },
      { label: 'Changelog', href: '/CHANGELOG.md' },
    ],
  },
  {
    label: 'Empresa',
    links: [
      { label: 'Sobre', href: 'mailto:contato@radiantnorma.com.br', external: true },
      { label: 'Vendas', href: 'mailto:contato@radiantnorma.com.br', external: true },
      { label: 'Privacidade', href: 'mailto:privacidade@radiantnorma.com.br', external: true },
      { label: 'Termos', href: 'mailto:contato@radiantnorma.com.br', external: true },
    ],
  },
]

export function LandingFooter() {
  return (
    <footer className="border-t border-border bg-surface">
      <div className="max-w-7xl mx-auto px-6 lg:px-10 py-16">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-8 mb-12">
          {/* Brand */}
          <div className="col-span-2">
            <Link href="/" className="flex items-center gap-3 mb-4">
              <div
                className="size-9 rounded-md bg-gradient-to-br from-accent-600 to-magenta-500 flex items-center justify-center text-white font-serif text-base font-medium shadow-glow-accent-sm"
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
            <p className="text-sm text-ink-muted leading-relaxed max-w-xs mb-6">
              Plataforma de validação CADOC e monitoramento regulatório
              para Instituições Financeiras brasileiras.
            </p>
            <div className="flex items-center gap-3">
              <span className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-surface-sunken border border-border text-2xs font-medium text-ink-muted">
                <ShieldCheck className="size-3 text-success-600" strokeWidth={2.25} />
                SOC 2 Type II
              </span>
              <span className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-surface-sunken border border-border text-2xs font-medium text-ink-muted">
                <Lock className="size-3 text-info-600" strokeWidth={2.25} />
                LGPD
              </span>
            </div>
          </div>

          {/* Link columns */}
          {COLUMNS.map((col) => (
            <div key={col.label}>
              <h4 className="text-2xs uppercase tracking-[0.18em] font-mono font-medium text-ink-subtle mb-4">
                {col.label}
              </h4>
              <ul className="space-y-2.5">
                {col.links.map((link) => (
                  <li key={link.label}>
                    {link.external ? (
                      <a
                        href={link.href}
                        className="text-sm text-ink-muted hover:text-ink transition-colors"
                      >
                        {link.label}
                      </a>
                    ) : (
                      <Link
                        href={link.href as Route}
                        className="text-sm text-ink-muted hover:text-ink transition-colors"
                      >
                        {link.label}
                      </Link>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom bar */}
        <div className="pt-8 border-t border-border-subtle flex flex-col sm:flex-row items-center justify-between gap-3">
          <p className="text-2xs text-ink-subtle font-mono uppercase tracking-wider">
            © 2026 Radiant Norma · Todos os direitos reservados
          </p>
          <p className="text-2xs text-ink-subtle font-mono uppercase tracking-wider">
            Feito em São Paulo · v3.36.0
          </p>
        </div>
      </div>
    </footer>
  )
}