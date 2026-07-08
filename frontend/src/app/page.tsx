/**
 * / — Landing page pública.
 *
 * Server component: sempre renderiza a landing institucional, mesmo
 * para usuários autenticados (UX padrão SaaS premium: visitante vê
 * marketing, logado acessa console via /console).
 *
 * Console autenticado: /console
 */

import { LandingHero } from '@/components/landing/hero'
import { LandingFeatures } from '@/components/landing/features'
import { LandingHowItWorks } from '@/components/landing/how-it-works'
import { LandingCTAFinal } from '@/components/landing/cta-final'
import { LandingFooter } from '@/components/landing/footer'

export const dynamic = 'force-dynamic'

export default function HomePage() {
  return (
    <>
      <LandingHero />
      <LandingFeatures />
      <LandingHowItWorks />
      <LandingCTAFinal />
      <LandingFooter />
    </>
  )
}