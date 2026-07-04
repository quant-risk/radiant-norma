/**
 * /regras — server entrypoint. Carrega regras (do backend ou fallback
 * estático do catálogo markdown), depois delega ao client component
 * que faz filtros e drill-down.
 */
import { promises as fs } from 'fs'
import path from 'path'
import { getServerSession } from '@/lib/session'
import { AppShell } from '@/components/layout/app-shell'
import { RegrasClient } from './regras-client'

export const dynamic = 'force-dynamic'

interface Rule {
  code: string
  severity: 'E' | 'A' | 'I'
  sheet?: string
  description: string
  example?: string
  category: 'B' | 'F' | 'C' | 'S'
}

async function loadRulesFromCatalog(): Promise<Rule[]> {
  const file = path.join(
    process.cwd(),
    '..',
    'docs',
    'rules-3040-catalog.md',
  )
  const content = await fs.readFile(file, 'utf-8').catch(() => '')
  const rules: Rule[] = []
  const re = /\| (B\d\d|F\d\d|C\d\d|S\d\d) \| ([EAI]) \| ([^|]+) \| ([^|]+) \| ([^|]+) \|/g
  let m
  while ((m = re.exec(content)) !== null) {
    const code = m[1].trim()
    rules.push({
      code,
      severity: m[2].trim() as Rule['severity'],
      sheet: m[3].trim(),
      description: m[4].trim(),
      example: m[5].trim(),
      category: code[0] as Rule['category'],
    })
  }
  return rules
}

export default async function RegrasPage() {
  const session = await getServerSession()
  if (!session) {
    return (
      <div className="p-12 text-center">
        <p>Sessão expirada.</p>
      </div>
    )
  }

  const rules = await loadRulesFromCatalog()

  return (
    <AppShell
      session={session}
      topbar={{
        title: 'Catálogo de Regras',
        subtitle: `${rules.length} regras ativas · CADOC 3040`,
        breadcrumbs: [
          { label: 'Radiant Norma', href: '/' },
          { label: 'Regras' },
        ],
      }}
      commandData={{
        rules: rules.map((r) => ({
          code: r.code,
          description: r.description,
          severity: r.severity,
        })),
      }}
    >
      <div className="max-w-7xl">
        <RegrasClient rules={rules} />
      </div>
    </AppShell>
  )
}