'use client'

/**
 * CTAFinal — bloco final com manifesto + CTA duplo.
 */

import Link from 'next/link'
import { ArrowRight, Mail } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

export function LandingCTAFinal() {
  return (
    <section id="compliance" className="relative py-24 lg:py-32 overflow-hidden">
      {/* Glow background */}
      <div
        className="absolute inset-0 pointer-events-none"
        style={{
          background:
            'radial-gradient(ellipse at center, rgba(30,42,94,0.10) 0%, transparent 60%)',
        }}
        aria-hidden
      />

      <div className="relative max-w-4xl mx-auto px-6 lg:px-10 text-center">
        <Badge tone="accent" variant="soft" dot size="md" className="mb-6">
          Pronto pra produção
        </Badge>

        <h2 className="font-serif text-4xl lg:text-5xl xl:text-display-sm font-medium text-ink leading-[1.05] tracking-[-0.03em] mb-6">
          Pronto pra começar?
        </h2>
        <p className="text-lg text-ink-muted leading-relaxed max-w-2xl mx-auto mb-10">
          SOC 2 Type II, LGPD audit-ready, integração nativa com Keycloak/Okta.
          Onboard em 2 semanas. SLA 99.95% contratual.
        </p>

        <div className="flex items-center justify-center gap-3 flex-wrap">
          <Link href="/console" passHref legacyBehavior={false}>
            <Button asChild variant="primary" size="lg" rightIcon={<ArrowRight className="size-4" strokeWidth={2.25} />}>
              Abrir Console
            </Button>
          </Link>
          <Link href="mailto:contato@radiantnorma.com.br" passHref legacyBehavior={false}>
            <Button asChild variant="secondary" size="lg" leftIcon={<Mail className="size-4" strokeWidth={2.25} />}>
              Falar com vendas
            </Button>
          </Link>
        </div>
      </div>
    </section>
  )
}