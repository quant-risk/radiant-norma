'use client'

/**
 * InsightCard — cartão de inteligência / recomendação, editorial.
 *
 * Visual: glyph com ring colorido · eyebrow (kind + impact) ·
 * headline serif · narrative · CTA.
 */
import {
  TrendingUp,
  TrendingDown,
  AlertTriangle,
  Sparkles,
  Lightbulb,
  ShieldAlert,
  ArrowRight,
} from 'lucide-react'
import * as React from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

type InsightKind =
  | 'anomaly'
  | 'trend-up'
  | 'trend-down'
  | 'recommendation'
  | 'opportunity'
  | 'warning'

const kindMeta: Record<
  InsightKind,
  {
    icon: React.ComponentType<{ className?: string; strokeWidth?: number | string }>
    tone: 'critical' | 'warning' | 'accent' | 'success' | 'info'
    label: string
  }
> = {
  anomaly: { icon: AlertTriangle, tone: 'critical', label: 'Anomalia' },
  'trend-up': { icon: TrendingUp, tone: 'info', label: 'Tendência' },
  'trend-down': { icon: TrendingDown, tone: 'warning', label: 'Atenção' },
  recommendation: {
    icon: Lightbulb,
    tone: 'accent',
    label: 'Recomendação',
  },
  opportunity: { icon: Sparkles, tone: 'success', label: 'Oportunidade' },
  warning: { icon: ShieldAlert, tone: 'warning', label: 'Alerta' },
}

export interface InsightCardProps {
  id: string
  kind: InsightKind
  headline: string
  narrative: string
  confidence?: number
  impact?: 'low' | 'medium' | 'high'
  cta?: {
    label: string
    onClick?: () => void
    href?: string
  }
  onDismiss?: () => void
}

const impactMeta = {
  low: { label: 'Baixo', tone: 'neutral' as const },
  medium: { label: 'Médio', tone: 'info' as const },
  high: { label: 'Alto', tone: 'warning' as const },
}

export function InsightCard({
  kind,
  headline,
  narrative,
  confidence,
  impact,
  cta,
}: InsightCardProps) {
  const meta = kindMeta[kind]
  const Icon = meta.icon

  const toneClasses: Record<typeof meta.tone, string> = {
    critical:
      'bg-critical-50 text-critical-600 dark:bg-critical-950/40 dark:text-critical-300 ring-critical-200/60 dark:ring-critical-800/40',
    warning:
      'bg-warning-50 text-warning-600 dark:bg-warning-950/40 dark:text-warning-300 ring-warning-200/60 dark:ring-warning-800/40',
    accent:
      'bg-accent-50 text-accent-600 dark:bg-accent-950/40 dark:text-accent-300 ring-accent-200/60 dark:ring-accent-800/40',
    success:
      'bg-success-50 text-success-600 dark:bg-success-950/40 dark:text-success-300 ring-success-200/60 dark:ring-success-800/40',
    info: 'bg-info-50 text-info-600 dark:bg-info-950/40 dark:text-info-300 ring-info-200/60 dark:ring-info-800/40',
  }

  return (
    <Card padding="md" className="group hover:shadow-md transition-all duration-240 ease-out-expo">
      <div className="flex items-start gap-4">
        <div
          className={cn(
            'shrink-0 size-10 rounded-lg flex items-center justify-center ring-1 ring-inset',
            toneClasses[meta.tone],
            '[&_svg]:size-4',
          )}
        >
          <Icon strokeWidth={2.25} />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2 flex-wrap">
            <Badge tone={meta.tone} variant="soft" size="sm">
              {meta.label}
            </Badge>
            {impact && (
              <Badge tone={impactMeta[impact].tone} variant="outline" size="sm">
                Impacto {impactMeta[impact].label}
              </Badge>
            )}
            {confidence !== undefined && (
              <span className="text-2xs text-ink-subtle ml-auto font-mono uppercase tracking-wider">
                {confidence}% confiança
              </span>
            )}
          </div>

          <h3 className="font-serif text-base font-medium text-ink mb-1.5 leading-snug tracking-tight">
            {headline}
          </h3>
          <p className="text-sm text-ink-muted leading-relaxed mb-4">
            {narrative}
          </p>

          {cta && (
            <Button
              size="sm"
              variant="ghost"
              rightIcon={<ArrowRight className="size-3.5" strokeWidth={2.25} />}
              onClick={cta.onClick}
              className="-ml-3"
            >
              {cta.label}
            </Button>
          )}
        </div>
      </div>
    </Card>
  )
}