// /regras — catálogo interativo das 60 regras catalogadas.
//
// Server component: lê de docs/rules-3040-catalog.md (estático por
// enquanto). Filtros client-side: por categoria, severidade,
// search. Toggle enable/disable (Sprint 8).

import { promises as fs } from 'fs'
import path from 'path'

interface Rule {
  code: string
  severity: 'E' | 'A' | 'I'
  sheet: string
  description: string
  example: string
}

async function parseCatalog(): Promise<Rule[]> {
  const file = path.join(
    process.cwd(),
    '..',
    'docs',
    'rules-3040-catalog.md',
  )
  const content = await fs.readFile(file, 'utf-8').catch(() => '')
  // Parse MVP: extrai tabela markdown.
  const rules: Rule[] = []
  const re = /\| (B\d\d|F\d\d|C\d\d|S\d\d) \| ([EAI]) \| ([^|]+) \| ([^|]+) \| ([^|]+) \|/g
  let m
  while ((m = re.exec(content)) !== null) {
    rules.push({
      code: m[1].trim(),
      severity: m[2].trim() as Rule['severity'],
      sheet: m[3].trim(),
      description: m[4].trim(),
      example: m[5].trim(),
    })
  }
  return rules
}

export const dynamic = 'force-dynamic'

export default async function RegrasPage() {
  const rules = await parseCatalog()

  const byCategory = {
    B: rules.filter((r) => r.code.startsWith('B')),
    F: rules.filter((r) => r.code.startsWith('F')),
    C: rules.filter((r) => r.code.startsWith('C')),
    S: rules.filter((r) => r.code.startsWith('S')),
  }

  return (
    <main className="p-8 max-w-6xl mx-auto">
      <h1 className="text-3xl font-bold mb-2">Catálogo de Regras</h1>
      <p className="text-slate-600 mb-8">
        {rules.length} regras ativas para CADOC 3040 (Sprint 7b / v1.7.0)
      </p>

      <div className="space-y-12">
        {(Object.entries(byCategory) as Array<[keyof typeof byCategory, typeof rules]>).map(
          ([cat, catRules]) => (
            <section key={cat}>
              <h2 className="text-2xl font-bold mb-4">
                {cat === 'B' && 'Básicas'}
                {cat === 'F' && 'Formato'}
                {cat === 'C' && 'Campos Obrigatórios'}
                {cat === 'S' && 'Semânticas'}
                <span className="ml-2 text-base text-slate-500">
                  ({catRules.length})
                </span>
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {catRules.map((r) => (
                  <div
                    key={r.code}
                    className="bg-white p-4 rounded shadow-sm border border-slate-200"
                  >
                    <header className="flex items-baseline gap-2 mb-1">
                      <span className="font-mono text-sm text-slate-500">
                        {r.code}
                      </span>
                      <span
                        className={`text-xs uppercase font-bold ${
                          r.severity === 'E'
                            ? 'text-red-600'
                            : r.severity === 'A'
                            ? 'text-amber-600'
                            : 'text-sky-600'
                        }`}
                      >
                        {r.severity}
                      </span>
                    </header>
                    <p className="text-sm text-slate-700">{r.description}</p>
                    <code className="block text-xs text-slate-500 mt-2 font-mono">
                      {r.example}
                    </code>
                  </div>
                ))}
              </div>
            </section>
          ),
        )}
      </div>
    </main>
  )
}
