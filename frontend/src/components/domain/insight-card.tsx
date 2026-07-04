'use client'

/**
 * InsightCard — cartão de inteligência / recomendação.
 *
 * Padrão: 1 ícone (estado), 1 headline (curta), 1 narrativa (o que tá
 * acontecendo), 1 CTA (ação). Sem exceções. Sem insights sem ação.
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
    icon: React.ComponentType<{ className?: string }>
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
  confidence?: number // 0-100
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
      'bg-critical-50 dark:bg-critical-950 text-critical-600 dark:text-critical-400',
    warning:
      'bg-warning-50 dark:bg-warning-950 text-warning-600 dark:text-warning-400',
    accent: 'bg-accent-50 dark:bg-accent-950 text-accent-600 dark:text-accent-400',
    success:
      'bg-success-50 dark:bg-success-950 text-success-600 dark:text-success-400',
    info: 'bg-info-50 dark:bg-info-950 text-info-600 dark:text-info-400',
  }

  return (
    <Card padding="md" className="group">
      <div className="flex items-start gap-4">
        <div
          className={cn(
            'shrink-0 size-9 rounded-lg flex items-center justify-center',
            toneClasses[meta.tone],
            '[&_svg]:size-4',
          )}
        >
          <Icon />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1.5 flex-wrap">
            <Badge tone={meta.tone} variant="soft">
              {meta.label}
            </Badge>
            {impact && (
              <Badge tone={impactMeta[impact].tone} variant="outline">
                Impacto {impactMeta[impact].label}
              </Badge>
            )}
            {confidence !== undefined && (
              <span className="text-2xs text-ink-subtle ml-auto">
                {confidence}% confiança
              </span>
            )}
          </div>

          <h3 className="text-sm font-semibold text-ink mb-1 leading-snug">
            {headline}
          </h3>
          <p className="text-sm text-ink-muted leading-relaxed mb-3">
            {narrative}
          </p>

          {cta && (
            <Button
              size="sm"
              variant="outline"
              rightIcon={<ArrowRight className="size-3.5" />}
              onClick={cta.onClick}
            >
              {cta.label}
            </Button>
          )}
        </div>
      </div>
    </Card>
  )
}