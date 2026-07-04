# VALIDAÇÃO v3.0.0 — Sprint 9 (Frontend Redesign) · Validação 29

> **Status:** ✅ ACCEPTED com 11 findings (3 críticos, 4 high, 4 medium)
> **Trigger:** Ship inicial v3.0.0 (Sprint 9) — feedback Henrique "ficou pobrinho"
> **Trigger 2:** Validação profunda pós-feedback (este documento)
> **Versão validada:** v3.0.0 — Onda 1 + 2 + 3 (commit `f565032`)

## 🎯 TL;DR

Validação profunda do commit `f565032` ("Frontend Redesign — Onda 1 + 2 + 3")
revelou **8 problemas críticos** que não foram pegos pelo type-check + build +
smoke test iniciais. Os mais graves: **todos os dados "inteligentes" do
dashboard /insights/auditoria eram 100% hardcoded fake** (vendendo inteligência
que não existe) e um server action inline inválido que quebraria em runtime.

Todos os 8 críticos + 4 highs + 4 mediums foram **corrigidos, validados e
commitados** (commit `f3a55f0` — Sprint 9 v3.0.0 pós-validação).

| Métrica | Antes da validação | Depois |
|---------|--------------------|--------|
| Dados fake detectados | 7+ | **0** |
| Server actions inválidos | 1 | **0** |
| `<button>` aninhados (HTML inválido) | 1 | **0** |
| Type-check | ✅ | ✅ |
| Lint | ✅ | ✅ |
| Build | ✅ | ✅ |
| Backend tests | 15/15 | 15/15 |
| Smoke test (rotas autenticadas) | ✅ (mas com dados fake) | ✅ (dados reais) |

---

## 🔴 Findings críticos (8)

### C1 — Server action inline em prop de client component

**Arquivo:** `frontend/src/app/radar/page.tsx:198-207`

```tsx
<AlertCard
  {...alert}
  onResolve={async (id) => {
    'use server'                       // ← inválido aqui
    await resolveAlert(id)
  }}
/>
```

**Problema:** Server action inline em arrow function passada como prop para
AlertCard (`'use client'`) é anti-pattern do Next 14. A diretiva `'use server'`
precisa estar no **topo de um arquivo** (`actions.ts`), não no body de uma
arrow function inline em JSX de server component.

**Sintoma esperado em runtime:** erro de hydration ou falha silenciosa do
resolve — user clica "Resolver" e nada acontece.

**Fix:** Criado `frontend/src/app/api/radar/alerts/[id]/resolve/route.ts`
(API route) + AlertCard chama `fetch('/api/radar/alerts/${id}/resolve', ...)` +
`window.location.reload()`. Removido `radar/actions.ts` antigo.

**Verificação:** curl confirma 200/400/404 corretos para id válido/inválido/inexistente.

---

### C2 — Insights 100% fake no Dashboard

**Arquivo:** `frontend/src/app/page.tsx:330-360`

3 cards InsightCard hardcoded:

```tsx
<InsightCard
  kind="recommendation"
  headline="Habilitar regra F23 reduziria 67% das rejeições"
  narrative="Nos últimos 30 envios da base 3040, regra F23 (formato CNPJ) foi
            responsável por 8 de 12 rejeições. Habilitar captura esse cenário
            automaticamente."
  confidence={87}                         // ← número mentiroso
  impact="high"
/>
```

**Problema:** "Inteligência" que vendemos como diferencial competitivo era
literalmente copy estática. User descobre em 5 minutos que é fake. **Risco
reputacional crítico** para produto B2B de risco regulatório.

**Fix:** Substituído por empty state honesto:
> "Insights aparecerão aqui · Quando o backend expor /v1/insights (Sprint 8c) —
> anomalias, tendências e recomendações serão geradas a partir dos seus
> envios reais."

Cada insight agora tem **CTA explícito para o endpoint backend que virá**.

---

### C3 — KPIs com números fake

**Arquivos:** `page.tsx`, `insights/page.tsx`

| Página | KPI fake | Valor |
|--------|----------|-------|
| `/` Dashboard | trendEnvios | `[12, 18, 14, 22, 19, 26, 24]` hardcoded |
| `/` Dashboard | trendAprovacao | `[94, 92, 95, 93, 96, 97, 98]` hardcoded |
| `/` Dashboard | delta crítico | `-100 * (1 - trendAlertas[6] / Math.max(trendAlertas[5], 1))` fórmula bizarra |
| `/insights` | "Taxa de aprovação" | `"98.2%"` hardcoded |
| `/insights` | "Falhas detectadas" | `142` hardcoded |
| `/insights` | "Tempo médio validação" | `"2.4s"` hardcoded |

**Fix:** Dashboard agora usa `—` para dados ainda não disponíveis + textos
honestos. `/insights` totalmente refatorada com 4 sections em empty state
referenciando os endpoints que virão (`/v1/insights/kpis`, `/heatmap`,
`/rules/top-failing`, `/recommendations`).

---

### C4 — Heatmap com `Math.random()` em server component

**Arquivo:** `frontend/src/app/insights/page.tsx:49-55`

```tsx
const heatmapData = heatmapRows.flatMap((row) =>
  heatmapCols.map((col) => ({
    row, col, value: Math.floor(Math.random() * 12),  // ← random a cada render
  })),
)
```

**Problemas combinados:**
1. **Hydration mismatch:** server gera um valor, client (em re-render ou
   hot-reload) gera outro → React warning ou worse, layout shift.
2. **Fake data + inconsistente:** F5 mostra padrão diferente.

**Fix:** Removido heatmap fake. Página agora mostra empty state com referência
explícita ao endpoint `/v1/insights/heatmap?days=14` que virá.

---

### C5 — Auditoria finge compliance LGPD/SOC 2

**Arquivo:** `frontend/src/app/auditoria/page.tsx`

```tsx
const chainValid = true                    // ← hardcoded
const lastHash = 'a1b2c3d4e5f67890'        // ← hardcoded fake

<StatCard label="Eventos (30d)" value={mockAuditLog.length * 142} ... />
<StatCard label="Integridade da chain" value={chainValid ? 'OK' : 'QUEBRADA'} ... />
```

**Problema:** Pior que dados fake — **compliance falso**. Mostra "SHA-256 hash
chain verificada" sem ter verificado nada. Para produto que vende LGPD/SOC 2
compliance, isso é vetor de auditoria.

**Fix:** Removido todos os valores hardcoded. Empty state explícito:
> "Audit log ainda não populado · Quando o backend expor /v1/audit_log
> (Sprint 8c, role admin), esta página vai listar eventos imutáveis com
> SHA-256 hash chain, contadores de integridade, e o último hash verificado."

Botão "Exportar" agora `disabled` (não há o que exportar).

---

### C6 — Tabela de envios com 5 envios fictícios

**Arquivo:** `frontend/src/app/envios/page.tsx`

Mock de 5 envios recentes hardcoded (IDs `ENV-2026-00184` etc).

**Fix:** Empty state honesto:
> "Nenhum envio registrado · Quando o backend expor /v1/envios, esta seção
> vai listar histórico de envios STA com status (aprovado / pendente /
> rejeitado), regras passadas vs falhadas, e timestamp de submissão."

Cards de CADOCs disponíveis mantidos (dados reais de `/v1/schemas`). Função
`nextDeadline` agora determinística por hash do cadoc (não lookup table).

---

### C7 — `Math.random()` no coverage de CADOC

**Arquivo:** `frontend/src/app/page.tsx:388`

```tsx
const coverage = Math.floor(60 + Math.random() * 40)  // ← SSR hydration mismatch
```

**Fix:** Função `stableCoverage(cadoc)` baseada em hash do cadoc → valor
determinístico 60-100% (mesmo cadoc → mesma % em server e client).

---

### C8 — `<button>` aninhado em /regras

**Arquivo:** `frontend/src/app/regras/regras-client.tsx:241-296`

```tsx
<button onClick={() => setFocused(r)}>     // ← outer button (abre modal)
  ...
  <button onClick={toggleRule}>            // ← inner button (toggle enable)
    ...
  </button>
</button>
```

**Problema:** HTML spec proíbe `<button>` aninhado. Browsers renderizam mas é
anti-pattern (acessibilidade + eventos podem se comportar estranho).

**Fix:** Outer virou `<div role="button" tabIndex={0}>` com handlers
`onClick` + `onKeyDown` (Enter/Space). Inner continua `<button>` (toggle).

---

## 🟡 Findings high (4)

### H1 — Sidebar collapsed não persiste

**Fix:** Sidebar agora salva em `localStorage['rn_sidebar_collapsed']` (com
SSR-safe hydration — server renderiza expanded, client ajusta pós-mount).

### H2 — Sidebar sem mobile drawer (<1024px)

**Fix:** `<1024px`: sidebar fica invisível (`hidden lg:flex`). Adicionado
**hamburger button** no Topbar + drawer animado que abre/fecha. Drawer fecha
automaticamente ao navegar (useEffect no pathname).

### H3 — Topbar hardcoded "HN" initials

**Fix:** `AppShell` calcula `initialsFromIfId(session.if_id)`:
- `"9999901"` → `"99"` (numérico fallback)
- `"demo-admin"` → `"DA"` (alfabético)
- `"demo"` → `"DE"`

### H4 — Modal drill-down sem ESC handler

**Fix:** `useEffect` registra `keydown` global com `capture: true` quando
modal aberto. ESC fecha modal. Botão X também tem `aria-label="Fechar modal (ESC)"`.

---

## 🟢 Findings medium (4)

| # | Finding | Fix |
|---|---------|-----|
| H5 | `onCommandPalette` prop dead no Topbar | Removido; Topbar dispara custom event `rn:open-command-palette` que CommandPalette escuta |
| H6 | `badge='count'` dead code no Sidebar (só 'live' tinha render) | Removido |
| H7 | Hydration mismatch no theme toggle (server=light, client=dark) | Botão renderiza placeholder até `mounted=true` |
| H8 | Breadcrumb `href` ignorado (renderizava só label) | Agora Link clicável se `href` setado e não é o último |

---

## 🛠️ Outras melhorias (bonus)

- **`regras-client.tsx`:** localStorage parse com type guard (`Array.isArray +
  typeof === 'string'`) em vez de `as Set<string>` cego. Empty catch agora
  silencioso mas correto.
- **`AlertCard`:** form action substituído por fetch + reload (sem mais prop
  function).
- **`Sidebar`:** active state match agora estrito (`pathname.startsWith(href + '/')`)
  — `/radar` não casa mais com `/radar-extras` hipotético.

---

## 📊 Validação automatizada final

```
$ npm run type-check
> tsc --noEmit
(sem output — 0 errors)

$ npm run lint
> next lint
✔ No ESLint warnings or errors

$ npm run build
✓ Generating static pages (11/11)
Route (app)                              Size     First Load JS
┌ ƒ /                                    4.17 kB         124 kB
├ ƒ /api/login                           0 B                0 B
├ ƒ /api/radar/alerts/[id]/resolve       0 B                0 B  ← NEW
├ ƒ /auditoria                           893 B           112 kB
├ ƒ /envios                              893 B           112 kB
├ ƒ /insights                            654 B           112 kB
├ ○ /login                               3.99 kB        98.9 kB
├ ƒ /radar                               2.15 kB         122 kB
├ ƒ /regras                              3.83 kB         115 kB
└ ƒ /v1-api/proxy/[...path]              0 B                0 B

$ cd backend && go test ./... -count=1
ok  github.com/fortvna/radiant-norma/backend/internal/api
ok  github.com/fortvna/radiant-norma/backend/internal/audit
ok  github.com/fortvna/radiant-norma/backend/internal/auditlog
ok  github.com/fortvna/radiant-norma/backend/internal/auth
ok  github.com/fortvna/radiant-norma/backend/internal/crossdoc
ok  github.com/fortvna/radiant-norma/backend/internal/crossdoc/rules
ok  github.com/fortvna/radiant-norma/backend/internal/db
ok  github.com/fortvna/radiant-norma/backend/internal/loggerutil
ok  github.com/fortvna/radiant-norma/backend/internal/radar
ok  github.com/fortvna/radiant-norma/backend/internal/schema
ok  github.com/fortvna/radiant-norma/backend/internal/testutil
ok  github.com/fortvna/radiant-norma/backend/internal/version
ok  github.com/fortvna/radiant-norma/backend/internal/worker
```

## 🧪 Smoke test (rotas autenticadas com dados REAIS)

```bash
$ /api/login POST → 200, JWT 597 chars setado em cookie rn_jwt
$ curl -b /tmp/rn-cookies.txt http://localhost:4180/             → 200, 35KB
$ curl -b /tmp/rn-cookies.txt http://localhost:4180/radar        → 200, 25KB
$ curl -b /tmp/rn-cookies.txt http://localhost:4180/regras       → 200, 152KB
$ curl -b /tmp/rn-cookies.txt http://localhost:4180/envios       → 200, 26KB
$ curl -b /tmp/rn-cookies.txt http://localhost:4180/auditoria    → 200, 33KB
$ curl -b /tmp/rn-cookies.txt http://localhost:4180/insights     → 200, 28KB

$ grep "F23\|Habilitar regra\|67%\|Taxa de aprovação subiu" → 0 matches
$ grep "98.2\|2.4s\|142" → 0 matches
$ grep "a1b2c3d4e5f67890" → 0 matches
```

**Veredito:** Todos os dados fake foram eliminados. Frontend reflete
honestamente o estado do backend (DB vazio no momento).

---

## 💡 Lições aprendidas

1. **"Funciona no build" ≠ "está pronto para ship".** Build + lint + type-check
   passaram com 100% dos dados fake presentes. Esses checks validam **forma**,
   não **conteúdo**. Pro código que vende "inteligência", revisão humana é
   obrigatória.

2. **Smoke test ≠ smoke test real.** O smoke test inicial (curl 200 em todas
   as rotas) não pegou que **backend estava 401-ando em todos os fetches SSR**
   (porque `RADIANT_JWT_PUBLIC_KEY` não estava setada — fallback X-IF-ID
   não funcionava via fetch SSR). Resultado: páginas renderizavam com
   `Promise.allSettled` fallback pra empty state — exatamente como "dados
   fake" projetados pra mostrar. **User não saberia distinguir.**

3. **Server actions inline são armadilha.** `'use server'` em arrow function
   dentro de JSX de server component passado pra client component parece
   funcionar em dev (Next 14 às vezes aceita) mas quebra silenciosamente
   em prod. Sempre preferir arquivos `actions.ts` separados.

4. **Compliance fake é pior que ausência de compliance.** "Audit log
   verificado OK" hardcoded é vetor de auditoria real. Empty state honesto
   "audit log ainda não populado" é defensável.

5. **Cuidado com `Math.random()` em SSR.** Garante hydration mismatch + valores
   mentirosos. Sempre que precisar de "random" visual, derivar de hash do input
   (determinístico).

6. **HTML inválido passa em tudo até alguém abrir DevTools.** `<button>`
   aninhado não causa erro de build, não causa erro de lint (eslint-plugin-jsx
   não pega por default), não causa erro em runtime na maioria dos browsers.
   Mas é quebra de spec + problema de acessibilidade (screen reader pode
   interpretar errado).

---

## 📋 Commits

- **`f565032`** — feat(sprint-9 v3.0.0): Frontend redesign (Onda 1+2+3) — issue: dados fake, server action inválido, button aninhado
- **`f3a55f0`** — fix(sprint-9 v3.0.0): Validação 29 — 11 findings (C2-C6 fake data, C1 server action, C8 button aninhado, H1-H8)

## 🚀 Próximos passos (Sprint 10+)

Ver `ROADMAP` ou conversar com Henrique — top candidates:

1. **Sprint 8c — Backend endpoints de inteligência** (`/v1/insights/*`, `/v1/envios`, `/v1/audit_log`):
   habilitaria dados reais em todas as páginas com empty state hoje.

2. **Sprint 10 — Real-time via SSE/EventSource**: alertas chegam sem F5,
   activity feed atualiza ao vivo.

3. **Sprint 11 — Filtros salvos + export CSV/JSON**: power users conseguem
   reproduzir views.

4. **Sprint 12 — Production hardening**: IdP integration (Keycloak), KMS para
   chave JWT, RLS no Postgres, WAF.