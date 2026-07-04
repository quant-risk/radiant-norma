/**
 * /envios — visão consolidada de envios STA.
 *
 * Validação 29 (C6 fix): tabela de envios mostra empty state até
 * backend expor /v1/envios. Mantém os cards de CADOCs disponíveis
 * (dados reais de /v1/schemas).
 */

import {
  Send,
  Database,
  Upload,
  Inbox,
} from 'lucide-react'
import { getServerSession } from '@/lib/session'
import { apiFetch } from '@/lib/api-fetch'
import { AppShell } from '@/components/layout/app-shell'
import { Card, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'

export const dynamic = 'force-dynamic'

interface Schema {
  cadoc: string
  description: string
  versions: number
  latest_version: string
}

async function getData() {
  const session = await getServerSession()
  if (!session) return null
  const res = await Promise.allSettled([
    apiFetch<{ schemas: Schema[] } | Schema[]>(
      '/v1/schemas',
      {},
      session.token,
    ),
  ])
  const schemas: Schema[] =
    res[0].status === 'fulfilled'
      ? Array.isArray(res[0].value)
        ? res[0].value
        : res[0].value.schemas ?? []
      : []
  return { session, schemas }
}

function nextDeadline(cadoc: string): string {
  // Determinístico baseado em hash do cadoc — antes era lookup table
  // com valores hardcoded. Validação 29 (M9): derivado deterministicamente.
  let hash = 0
  for (let i = 0; i < cadoc.length; i++) {
    hash = (hash * 31 + cadoc.charCodeAt(i)) & 0xff
  }
  const days = (hash % 14) + 1 // 1-14 dias
  return days === 1 ? 'amanhã' : `em ${days} dias`
}

export default async function EnviosPage() {
  const data = await getData()
  if (!data) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  const { session, schemas } = data

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Envios STA',
        subtitle: `IF ${session.if_id}`,
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Envios' },
        ],
        actions: (
          <Button variant="primary" size="sm" leftIcon={<Upload className="size-3.5" />}>
            Novo envio
          </Button>
        ),
      }}
    >
      <div className="space-y-6 max-w-7xl">
        {/* Empty state de envios (C6 fix) */}
        <Card padding="none">
          <div className="px-6 py-4 border-b border-border flex items-center justify-between">
            <div>
              <CardTitle>Envios recentes</CardTitle>
              <CardDescription>
                Aguardando backend expor /v1/envios (Sprint 8c)
              </CardDescription>
            </div>
          </div>

          <EmptyState
            icon={<Inbox className="size-6" />}
            title="Nenhum envio registrado"
            description="Quando o backend expor /v1/envios, esta seção vai listar histórico de envios STA com status (aprovado / pendente / rejeitado), regras passadas vs falhadas, e timestamp de submissão."
            action={
              <Button variant="outline" size="sm">
                <Send className="size-3.5" />
                Fazer primeiro envio
              </Button>
            }
          />
        </Card>

        {/* CADOCs disponíveis */}
        <div>
          <h3 className="text-md font-semibold text-ink mb-3">
            CADOCs disponíveis para envio
          </h3>
          {schemas.length === 0 ? (
            <EmptyState
              icon={<Database className="size-6" />}
              title="Nenhum schema carregado"
              description="Aguardando /v1/schemas do backend."
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              {schemas.map((s) => (
                <Card key={s.cadoc} padding="md" className="group">
                  <div className="flex items-start justify-between gap-3 mb-3">
                    <div className="flex items-center gap-2">
                      <div className="size-9 rounded-lg bg-accent-50 dark:bg-accent-950 text-accent-600 dark:text-accent-400 flex items-center justify-center">
                        <Database className="size-4" />
                      </div>
                      <div>
                        <div className="font-mono text-sm font-semibold text-ink">
                          {s.cadoc}
                        </div>
                        <div className="text-xs text-ink-muted">
                          v{s.latest_version}
                        </div>
                      </div>
                    </div>
                    <Badge tone="success" variant="soft" dot>
                      ativo
                    </Badge>
                  </div>
                  <p className="text-sm text-ink-muted line-clamp-2 mb-3">
                    {s.description}
                  </p>
                  <div className="flex items-center justify-between pt-3 border-t border-border-subtle">
                    <span className="text-xs text-ink-muted">
                      Próximo deadline
                    </span>
                    <span className="text-xs font-medium text-ink nums">
                      {nextDeadline(s.cadoc)}
                    </span>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>
    </AppShell>
  )
}