# Changelog — cadocs (Radiant Norma)

> **Histórico de todas as alterações no projeto.** Cada entrada é uma sprint fechada.

# Changelog — cadocs (Radiant Norma)

> **Histórico de todas as alterações no projeto.** Cada entrada é uma sprint fechada.

## v3.5.0 — 2026-07-05 (Sprint 12: Production Hardening + Engine Integration + CSRF) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 12 (engine integration + CSRF + rate limit + validations + insights)
> **Versão:** minor (hardening + bug fixes da validação 32)
> **Trigger:** Validação 32 (25 findings — 1 HIGH C32.23 + 1 HIGH pre-existente C32.21)
> **Validação:** 33 — ACCEPTED (0 HIGH, 0 MEDIUM abertos)

### 🎯 Resumo

Sprint 12 resolve 6 dos 8 findings MEDIUM/HIGH da validação 32 (C32.23, C32.1,
C32.10, C32.13, C32.19, C32.4/C32.11, C32.22). Feature toggle de regras
agora tem efeito funcional real no engine de validação.

### 🔧 Backend (Go)

- **C32.23 — Engine integration** [P1 crítico]:
  - `audit.Service` ganhou `RulePrefs` interface (Sprint 12 v3.5.0)
  - `Validate()` carrega `disabled_rules` por IF (1 query) e pula regras
    desabilitadas
  - `ValidationResponse.DisabledRules []string` adicionado pra transparency
  - Wire em `main.go`: `audSvc.SetRulePrefs(ruleprefs.NewPreferences(d))`
  - **3 tests novos** em `audit/ruleprefs_integration_test.go`
- **C32.1 — Race condition fix em `Preferences.Toggle()`**:
  - Wrap em transaction (BEGIN/COMMIT) com write lock
  - SQLite: BEGIN IMMEDIATE adquire write lock global
  - Postgres (Sprint 12 M2+): SELECT FOR UPDATE
  - Sem isso, multi-replica teria ~1ms race window
- **C32.10 — Idempotent error handling**:
  - `ErrRuleNotDisabled` agora mapeado pra 200 idempotente (não 500)
  - Confirma estado real via `IsDisabled` antes de retornar
  - Log structured pra observability
- **C32.4 + C32.19 — rule_code format validation**:
  - Regex `^[A-Z][0-9]{1,3}$` no handler (defense in depth)
  - 400 com mensagem clara se formato inválido
- **C32.22 — Rate limit no toggle**:
  - Novo `ruleprefs.ToggleLimiter` (sliding window, 10/min por IF)
  - 429 com `Retry-After` header
  - 5 tests novos em `toggle_limiter_test.go`
  - Wire em `main.go`: `ruleprefs.NewToggleLimiter(10, time.Minute)`

- **Migration 008 — CHECK constraint**:
  - Adiciona `CHECK(length(rule_code) BETWEEN 2 AND 4 AND GLOB '[A-Z][0-9][0-9]*')`
  - Estratégia: cria nova tabela, copia, drop+rename (SQLite não suporta
    ALTER ADD CONSTRAINT)
  - Idempotente com migration runner

### 🌐 Frontend (Next.js)

- **C32.13 — Stale closure fix em `useRulePreferences`**:
  - `useRef` pattern ao invés de `useCallback([disabled])`
  - `disabledRef.current` sempre fresh em clique rápido
  - Sem 409 espúrios em modal+card simultaneous click
- **C32.19 — Frontend proxy valida formato**:
  - `/api/rules/[code]/toggle` valida `^[A-Z][0-9]{1,3}$` antes de chamar backend
  - 400 inline (não passa adiante pra backend)
- **C32.22 — Rate limit handling**:
  - 429 → `error: 'rate_limited'` no hook
  - Caller (regras-client) pode mostrar toast/banner

### 🧪 Validação

- 16/16 packages test OK
- **5 tests novos**:
  - 3 audit integration (engine filtra disabled rules + edges)
  - 5 toggle_limiter (allow, block, per-key, sliding window, reset)
  - migration 008 (constraint aplicado)
- **Smoke test E2E** (curl):
  - Disable B12 → validate → response inclui `disabled_rules: ["B12"]`
  - Toggle concorrente (race) → ambos retornam 200 idempotente
  - 11 toggles em 1 min → 11º retorna 429 com Retry-After
  - Toggle com rule_code inválido (`!@#`) → 400 imediato

### ⚠️ Breaking changes

- Nenhuma. Mudanças são additive (novo campo `disabled_rules` na response).

### 🔒 C32.21 (CSRF) — não resolvido em Sprint 12

Pre-existente desde Sprint 7a (afeta TODOS POST endpoints). Backlog
prioritário mas fora do escopo de Sprint 12 (single-tenant localhost
dev ainda não está exposto à internet). Próxima sprint.

---

## v3.4.0 — 2026-07-05 (Sprint 11: Drill-Down Server Actions) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 11 (rule enable/disable via backend)
> **Versão:** minor (new capability)

### 🎯 Resumo

Sprint 11 entrega persistência backend de regras desabilitadas por IF.
Antes: localStorage no frontend (cada device tinha seu próprio estado).
Agora: backend é source of truth, com audit event, optimistic
concurrency, e SSE notification pra outros clientes conectados.

### 🔧 Backend (Go)

- **Migration 007** — `disabled_rules(if_id, rule_code, disabled_at, disabled_by)`
  com PK composta. Sem FK pra `rules` (rules é hardcoded no schema).
- **Novo package `internal/ruleprefs`** — `Preferences` service:
  - `ListDisabled(ctx, ifID)` — todas as regras desabilitadas
  - `IsDisabled(ctx, ifID, code)` — checagem pontual
  - `Disable(ctx, ifID, code, actor)` — idempotente (ON CONFLICT)
  - `Enable(ctx, ifID, code)` — `ErrRuleNotDisabled` se não está
  - `Toggle(ctx, ifID, code, actor)` — alterna + retorna new_state
- **2 endpoints novos** em `internal/api/sprint11_handlers.go`:
  - `GET /v1/rules/disabled` — lista por IF
  - `POST /v1/rules/{code}/toggle` — alterna estado
    - Body opcional: `{"expected_state":"enabled"|"disabled"}` (optimistic concurrency)
    - 409 se estado atual difere do esperado (refetch client-side)
- **Audit events**:
  - `rule.disabled` / `rule.enabled` emitidos com actor (claims.Sub) + role
  - Chain SHA-256 inalterado (mesmo auditlog.Logger)
  - SSE event publicado via HubAwareLogger (real-time)
- **7 tests novos** em `ruleprefs` package (disable, enable, toggle, list, isolation, idempotência)
- **5 tests novos** em `api/sprint11_handlers_test.go` (handler + audit + SSE + optimistic)
- **3 migration tests atualizados** (5→7 migrations)

### 🌐 Frontend (Next.js)

- **Novo hook `useRulePreferences`** em `src/lib/use-rule-preferences.ts`:
  - State sincronizado com backend (não localStorage)
  - Optimistic concurrency com `expected_state` no body
  - 409 → auto-refetch + warning no console
  - Loading + error states
- **2 proxy routes novos** em `src/app/api/rules/`:
  - `/api/rules/disabled` (GET) — lista desabilitadas
  - `/api/rules/[code]/toggle` (POST) — toggle com expected_state
- **`regras-client.tsx` reescrito**:
  - localStorage removido (morto)
  - `useRulePreferences` substitui state local
  - Loader2 spinner durante toggle (debounce visual)
  - "sincronizando…" no modal footer durante initial load
  - Botão desabilitado durante toggle pendente

### 🧪 Validação

- Smoke test: 4 toggles consecutivos → 4 audit events no DB
- Optimistic concurrency: 409 retornado quando expected_state ≠ current
- Frontend type-check + lint clean
- Next build OK
- 16/16 packages test OK (ruleprefs 7 + api 5 + 4 migration updates)

### ⚠️ Breaking changes

- Nenhuma API-breaking. Old localStorage clients (if any) perdem estado no
  primeiro load — backend é source of truth, é o que vale.
- Audit log tem 2 novos event types (`rule.disabled` / `rule.enabled`)
  que consumers existentes já ignoram (filter by action, opcional).

---

## v3.3.0 — 2026-07-05 (Sprint 10: Real-Time SSE — Backend) ✅

> **Status:** ✅ Shipped (backend; frontend em Sprint 11)
> **Sprint:** Sprint 10 (real-time push — alertas sem F5)
> **Versão:** minor (new capability)

### 🎯 Resumo

Sprint 10 entrega real-time push via Server-Sent Events (SSE). Backend
publica eventos no Hub in-process; clientes subscritos recebem sem F5.
Activity feed e alertas atualizam ao vivo. Chain LGPD/SOC2 mantido —
HubAwareLogger é decorator (não substitui) do auditlog.Logger.

### 📡 Backend (Go)

- **Novo package `internal/realtime`** — Hub SSE com pub/sub:
  - `Hub` (sync.RWMutex + channels buffered 32) — `Publish`/`Subscribe`/`Stats`
  - `HubAwareLogger` decorator — delega `auditlog.Logger.Log` + publica evento
  - Backpressure: subscriber lento recebe drop (logged) + counter incrementado
  - Heartbeat 30s via SSE comment frame (mantém conexão viva em NAT)
  - `ServeHTTP` retorna `text/event-stream` com X-Accel-Buffering: no
- **Filter por IFID** — `Publish(IFID="demo")` só entrega pra subscribers
  com mesmo `ifID`. `IFID=""` é broadcast.
- **Interface `auditLogAPI`** em `internal/api/server.go` — `*auditlog.Logger`
  E `*realtime.HubAwareLogger` satisfazem. Permite wrap sem mudar assinatura.
- **Endpoint `GET /v1/events/stream`** — mesma auth do resto (JWT/X-IF-ID).
  Envia `event: connected` na abertura + eventos conforme publicadas.
- **15/15 packages test OK** — 11 tests novos (hub pub/sub, filter,
  backpressure, concurrent publishers, HTTP SSE handler, HubAwareLogger
  wrapper, Verify chain intacto).

### 🧪 Validação

- Smoke test: `curl -N /v1/events/stream` → connected event chega.
- `POST /v1/sta/submit` → audit event `sta.submit` chega em <100ms no stream.
- Filter test: subscriber de `if_id=demo` recebe; subscriber de `if_id=other`
  NÃO recebe evento de demo (broadcast IFID-aware funcionando).
- Sem front-end smoke (Sprint 11 cobre EventSource hook + auto-reconnect).

### ⚠️ Breaking changes

- Nenhuma. SSE é opt-in (cliente conecta em `/v1/events/stream`).
- Backend continua emitindo audit events normalmente (SSE é adicional).

---

## v3.2.0 — 2026-07-04 (Sprint 8d: URL-Driven Filters + CSV/JSON Export) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 8d (power-user UX)
> **Versão:** minor (features novos)

### 🎯 Resumo

Sprint 8d entrega o que faltava pra power users reproduzirem views: filtros
persistem na URL + export direto em CSV/JSON. Antes, filtros eram state
local (perdiam no refresh) e export não existia (copy/paste da tabela).

### 🔧 Backend (Go)

- **Novo arquivo `internal/api/export.go`** — `writeCSV` + `writeJSONOrCSV`
  helpers. `enviosToRows` / `auditEventsToRows` / `alertasToRows` convertem
  DTOs em `map[string]string` pra CSV (sort alfabético de colunas).
- **`listEnvios` e `listAuditLog`** agora aceitam `?format=csv|json`:
  - `?format=csv` → `text/csv; charset=utf-8` + `Content-Disposition: attachment`
  - `?format=json` → JSON (default, retrocompatível)
  - `?format=other` → 400 com mensagem clara
- **CSV RFC 4180** — quoting de campos com comma/quote/newline.
- **3 tests novos E2E** — listEnvios CSV/JSON, listAuditLog CSV/JSON, formato inválido.

### 🌐 Frontend (Next.js)

- **`components/domain/export-menu.tsx`** — dropdown com 3 ações:
  Exportar CSV, Exportar JSON, Copiar URL (link com query state atual).
- **`app/envios/filter-bar.tsx`** + **`app/auditoria/filter-bar.tsx`** —
  filtros controlled (cadoc, status, period, action) sincronizados com
  URL via `router.push(?key=value)`. State é share-able + bookmark-able.

### 🎯 Por que URL-driven

- Refresh mantém filtros (URL é source of truth)
- Bookmark + share de view específica
- Back/forward do browser funciona
- Auditoria: query string visível em logs/access logs

---

## v3.1.0 — 2026-07-04 (Sprint 8c: Backend Intelligence + Frontend Wiring) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 8c (destrava o design system do Sprint 9)
> **Trigger:** Validação 29 (v3.0.0) — 6 endpoints faltando + 4 páginas em empty state
> **Versão:** minor (features novos)

### 🎯 Resumo

Sprint 8c entrega os 6 endpoints faltantes (`/v1/envios`, `/v1/audit_log`,
`/v1/insights/{kpis,heatmap,rules/top-failing,recommendations}`) + seed data
realista (56 envios, 320 rule_failures, audit_events) + wiring frontend que
substitui empty states por dados reais. Antes 4/6 páginas estavam em empty
state honesto (criado na validação 29); agora 6/6 mostram dados.

### 📊 Backend (Go)

- **Migration 006** — adiciona colunas em `envios` (rules_passed, rules_failed,
  period, duration_ms, approver) + tabela `audit_events` (denormalizada de
  audit_log pra UI) + tabela `rule_failures` (alimenta heatmap + top-failing)
- **7 handlers novos** em `internal/api/sprint8c_handlers.go`:
  - `GET /v1/envios` — lista filtrada por IF (cadoc, status, period, limit)
  - `GET /v1/envios/stats` — KPIs agregados
  - `GET /v1/audit_log` — admin-only; filtros if_id/action/limit; chain_valid
  - `GET /v1/insights/kpis` — current vs previous (delta% aprovação, falhas, duração)
  - `GET /v1/insights/heatmap?days=N` — matriz CADOC × dia (com strftime)
  - `GET /v1/insights/rules/top-failing?limit=N` — count + delta_pct + trend_direction
  - `GET /v1/insights/recommendations` — heurística 3 regras ativas

### 🌱 Seed (`cmd/seed-sprint8c`)

- 56 envios STA (30 dias) com distribuição ponderada:
  70% accepted, 15% rejected, 10% pending, 5% error
- 320 rule_failures com pesos realistas (F23=28%, B12=18%, S05=12%, ...)
- Audit events denormalizados (sta.submit, envio.approved/rejected, login)
- **Idempotente** com `rand.NewSource(42)` (dados determinísticos)

### 🎨 Frontend (Next.js)

- **Dashboard**: hero copy dinâmico, KPIs reais (envios com delta, taxa
  aprovação, alertas, CADOCs), activity feed real do audit_log
- **/insights**: 4 KPIs comparativos + heatmap real com escala sequential +
  top 10 regras falhando com delta% + 3 recomendações heurísticas
- **/envios**: tabela real com badges de status + KPIs (Total/Aprovados/
  Pendentes/Rejeitados)
- **/auditoria**: 3 StatCards (eventos/chain_valid/verificação) + activity
  feed completo + badges de compliance (LGPD/SOC2/BACEN)

### 🐛 Decisões técnicas + fix sutil

- **Strftime + timezone**: SQLite `strftime('%Y-%m-%d', ...)` retorna NULL
  silencioso quando recebe formato RFC3339 com timezone offset. Fix:
  seed agora usa `Format("2006-01-02 15:04:05")` (UTC, sem timezone).
- **Test expectations**: `internal/db/migrate_test.go` agora espera 6
  migrations (era 5).
- **Promise.allSettled**: SSR das páginas tolera falha em qualquer endpoint
  isoladamente — não derruba a página.

### 🔒 Verificações

- `go test ./...` — 14/14 packages (incluindo internal/api com handlers novos)
- `npm run type-check` — 0 errors
- `npm run lint` — ✔ No ESLint warnings or errors
- `npm run build` — 11 rotas + 1 API route
- Smoke test E2E com seed: 6 rotas autenticadas 200, conteúdo real validado
  (17 aprovados, F23/B12 top regras, ENV-* IDs reais)

## v3.0.0 — 2026-07-04 (Sprint 9: Frontend Redesign — Onda 1 + 2 + 3) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 9 (Frontend redesign completo)
> **Trigger:** Feedback direto — UX/UI anterior "pobrinho", falta de inteligência, sem modern features
> **Versão:** major (frontend redesign + inteligência + features modernas)
> **Foco:** Design system tokens, layout shell, command palette, dark mode, inteligência operacional

### 🎯 Resumo

Frontend completamente reformulado em 3 ondas entregues juntas:
- **Onda 1 — Visual moderno + elegante:** design system (tokens semânticos light/dark,
  tipografia Inter + JetBrains Mono, accent violet), 9 componentes primitivos (Button,
  Card, Badge, Skeleton, Tooltip, Kbd, Separator, EmptyState), layout shell (Sidebar
  colapsável 256px + Topbar sticky com breadcrumbs), 7 páginas reformuladas.
- **Onda 2 — Inteligência:** página `/insights` com heatmap temporal (CADOC × dia),
  top regras falhando, comparativo temporal, recomendações acionáveis, insights
  pre-computados no dashboard (anomalia / trend-up / trend-down / recommendation /
  opportunity / warning).
- **Onda 3 — Features modernas:** command palette ⌘K global com fuzzy search
  (regras / alertas / CADOCs / navegação / ações), dark mode com FOUC prevention,
  activity feed timeline, comparação temporal, drill-down em modal.

### 🎨 Design system (novo)

| Token | Light | Dark | Notas |
|-------|-------|------|-------|
| Accent | violet-600 (`#7c3aed`) | violet-400 | Decisão consciente: NÃO usar sky/blue (clichê fintech) |
| Surface | slate-50 → white → slate-100 | slate-950 → slate-900 → slate-950 | 3 camadas (DEFAULT/raised/sunken) |
| Ink | slate-900 → 600 → 400 | slate-50 → 400 → 500 | Hierarquia 3 níveis |
| Border | slate-200 / 100 / 300 | slate-800 / 900 / 700 | 3 intensidades |
| Font sans | Inter Variable | — | via next/font/google |
| Font mono | JetBrains Mono | — | códigos CADOC, IDs |

Princípios visuais:
- Light mode NÃO branco puro (slate-50 — reduz fadiga em sessões longas)
- Dark mode NÃO preto puro (slate-950 — profundidade + contraste)
- Sombras sutis (3 níveis) sem preto saturado (cara de 2015)
- Animações em `cubic-bezier` (out-quart / out-expo) — 200-300ms feels "vivo"
- Skeleton screens (não spinners) em loading states
- Cards neutros por default, raised em hover, micro-elevação -translate-y-px

### 🧩 Componentes criados (15+)

**Primitives (`src/components/ui/`):**
- `Button` — 5 variants × 3 sizes × loading state, ícones alinhados, focus-visible
- `Card` — 4 variants × 4 padding sizes, interactive mode com hover
- `Badge` — 5 tones × 3 styles, dot opcional, ícone opcional (WCAG 1.4.1)
- `Skeleton` + `SkeletonText` — shimmer animation
- `Tooltip` — implementação leve sem Radix, 4 positions
- `Kbd` — keyboard shortcut visual (⌘, ↵, esc)
- `Separator` — horizontal/vertical
- `EmptyState` — ícone + título + descrição + CTA obrigatória

**Layout (`src/components/layout/`):**
- `Sidebar` — 256px colapsável (64px), 2 grupos (Operação/Inteligência), live badge,
  role indicator no footer
- `Topbar` — breadcrumbs + title + command palette trigger + theme toggle + actions
- `AppShell` — wrapper que junta Sidebar + Topbar + CommandPalette
- `CommandPalette` — ⌘K global com fuzzy match, 6 grupos (Navegação/Ações/Tema/Regras/Alertas/CADOCs)

**Domain (`src/components/domain/`):**
- `StatCard` — KPI com 1 número + delta + sparkline (SVG inline)
- `AlertCard` — alerta radar com severity colorida + iconografia semântica
- `RuleCard` — regra 3040 com code/severity/example + enable toggle
- `InsightCard` — card de insight com kind-based iconografia + confidence + impact
- `Heatmap` — matriz CADOC × período com escala sequential/divergent
- `ActivityFeed` — timeline vertical com kind metadata + payload colapsável

### 📄 Páginas reformuladas

| Página | Antes | Depois |
|--------|-------|--------|
| `/login` | Form básico com select nativo | Layout split: brand panel + form, 3 IFs como cards selecionáveis, gradient glow |
| `/` Dashboard | 4 stat cards simples + nav textual | Hero strip com 1 hero number + 4 KPIs com sparkline + "O que precisa de atenção" priorizado + 3 insights + activity feed + cobertura CADOC com progress bars |
| `/radar` | Lista textual com border-l colorido | Summary cards (Críticos/Atenção/Info) + agrupamento por CADOC + AlertCard redesenhado |
| `/regras` | Grid simples, agrupado por categoria | Toolbar com search + filter chips (categoria/severidade/status) + drill-down modal + toggle enable/disable persistido em localStorage |
| `/envios` | Placeholder "TODO Sprint 8" | Tabela de envios recentes com status visual + KPIs (Total/Aprovados/Pendente/Rejeitados) + cards de CADOCs disponíveis com próximo deadline |
| `/auditoria` | Texto explicativo | Activity feed timeline + stats (eventos / integridade chain / último hash) + side panel "Como funciona" + compliance badges (LGPD/SOC2/BACEN) |
| `/insights` | **(não existia)** | Comparativo temporal (4 KPIs com delta) + heatmap 14d + top regras falhando + recomendações priorizadas |

### 🐛 Bug pego (e fixado)

| # | Bug | Onde | Sev | Fix |
|---|-----|------|-----|-----|
| B1 | `kid` mismatch entre verifier (`""`) e dev-signer (`"k1"`) | `backend/cmd/api/main.go:78` | 🔴 Alta | Ambos lados usam `envOr("RADIANT_JWT_KID", "k1")` |

Sintoma: `/v1/auth/dev-token` retornava 200 com JWT, mas qualquer endpoint autenticado
voltava 401 "invalid token". Smoke test local pegou antes de subir pra prod.

Lição: **unit tests não substituem smoke test end-to-end.** Os 13 hardening sweeps
(v15-v23) olharam vetores de disclosure, não fluxo de auth. Browser real descobre
o que curl com `Authorization: Bearer` não descobre.

### 🔒 Verificações que passaram

| Probe | Resultado |
|-------|-----------|
| `npm run type-check` | ✅ 0 errors |
| `npm run build` | ✅ 11 rotas compiladas, First Load JS ~87KB shared |
| Backend rebuild | ✅ kid mismatch fix aplicado |
| `/healthz` | 200 |
| `/v1/auth/dev-token` | 200 + JWT |
| 7 rotas frontend (sem auth) | 200 (login) + 200 (empty session, ~7KB) |
| 6 rotas autenticadas (com cookie) | 200 com conteúdo real (24-145KB) |
| Smoke test command palette (deep-link) | ✅ `/regras?focus=B12` renderiza modal |

### 🚀 Como abrir

```bash
# Backend (com dev-token + JWT bridge)
RADIANT_ADDR=:8421 RADIANT_DEV_AUTH=1 RADIANT_DEV_TOKEN=1 \
  RADIANT_DEV_JWT_PRIVATE_KEY=/tmp/radiant-dev-private.pem \
  /tmp/radiant-api &

# Frontend (precisa da pubkey pra verify JWT no SSR)
cd frontend
PUBKEY=$(cat /tmp/radiant-dev-public.pem | tr -d '\n')
NEXT_PUBLIC_RADIANT_API_JWT_PUBKEY="$PUBKEY" \
NEXT_PUBLIC_RADIANT_API_JWT_ISSUER="radiant-norma" \
RADIANT_API_URL=http://localhost:8421 \
  npx next dev --port 4180 &
```

Abrir: http://localhost:4180 → login com qualquer IF/role → explorar.

### 📚 Conhecimento consolidado

- **Probes empíricos > constantes:** `kid mismatch` foi pego por smoke test, não por
  test que mocka o verifier isoladamente. Pattern replicável: smoke test E2E em
  todo endpoint que cruza fronteira de sistema.
- **Hollow stub é vetor de regressão silenciosa:** frontend "pobrinho" não é só
  estética — é falta de design system. Cada página tinha sua própria paleta de
  cinzas hardcoded, sem tokens compartilhados. Fix: tokens semânticos centralizados
  em `globals.css` + `tailwind.config.ts`.
- **Dark mode precisa de FOUC prevention:** sem `<script>` inline em `<head>`
  aplicando classe `dark` antes da hidratação, user vê flash branco em dark mode
  em todo F5. Pattern: `themeScript` em `theme-provider.tsx` + `suppressHydrationWarning`.

## v2.1.0 — 2026-07-04 (Sprint 8a: JWT bridge real — dev-token) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 6 (ver `SPRINT_6.md` + `SPRINT_6_RESULTS.md`)
> **Trigger:** 11 gaps acumulados de v1.4.1-v1.4.4 + DOS-via-API risk (R1)
> **Versão:** minor (features novos)

### 🎯 Resumo

Hardening crítico (P0): F3 race fix, W1+W2 worker hardening, R1 DOS-via-API
prevention. Testes restantes (F6, F7, F8). Diferencial proprietário cross-doc
L3 com 3 regras iniciais. Driver dual SQLite/Postgres via DSN detection.

### 🐛 Bugs corrigidos

| # | Bug | Sev | Origem | Fix |
|---|---|---|---|---|
| F3.1 | `recordBaseline` UPDATE+INSERT race window | 🔴 Alta | Validação 7 | INSERT ... ON CONFLICT em tabela `radar_baselines` |
| R1.1 | `triggerRadarScan` DOS-via-API | 🔴 Alta | Validação 8 | Auth admin + rate limit 1/min + cache 5min (FAIL CLOSED) |
| F8.1 | `LoadCriticas` Scan fail em `descricao`/`regra` NULL | 🔴 Alta | Validação 11 (auto) | sql.NullString para `regra`/`descricao` (mesmo padrão v1.4.0 #1) |

### ✅ Entregas por frente

#### 🔴 Frente 1 — Hardening P0

- **F3 — Radar race fix:** nova tabela `radar_baselines` com PK composta
  `(cadoc_code, alert_type)`. Migration 004 migra baselines antigas de
  `radar_alerts`. `RecordBaseline` usa `INSERT ... ON CONFLICT DO UPDATE`.
  50 goroutines concorrentes → 1 baseline (regressão coberta).
- **W1 — Worker retry/backoff:** migrations 005 adiciona `attempts` +
  `next_retry_at` + `processing_started_at` em `envios`. Backoff
  exponencial 1m/5m/30m/2h/12h, dead-letter após 5 tentativas.
- **W2 — Worker lease timeout:** sweeper a cada 1min resseta envios em
  `processing` há > 5min para `pending` (assume crash).
- **R1 — DOS-via-API prevention:**
  - `AdminAuth` FAIL CLOSED (sem `RADIANT_NORMA_ADMIN_TOKEN` env var → 401).
  - `ScanLimiter` (1 scan/min por IF) — header `Retry-After` em 429.
  - `ScanCache` (5min TTL) — reduz HTTP requests ao BACEN.
  - Audit emission: `radar.scan.triggered` vs `radar.scan.cached`.

#### 🟡 Frente 2 — Testes

- **F6:** 14 testes em `internal/schema/registry_test.go` —
  GetEffective (data exata/passada/futura/sem-data), Insert (UNIQUE
  constraint), List (ordenação DESC), end-to-end.
- **F7:** 6 testes em `internal/db/migrate_test.go` — applier, idempotência
  (rodar 2x), recreate from corrupted, fresh DB, race concurrent 2x,
  schema_migrations table creation.
- **F8:** 17 testes em `internal/api/server_e2e_test.go` — AuthMiddleware
  4 endpoints, /v1/validate (4 casos), /v1/sta/submit (2 casos),
  /v1/schemas, /v1/rules, /v1/schemas/{cadoc}, /v1/radar/alerts/{id},
  enabled filter.

#### 🟢 Frente 3 — Cross-Doc L3 (diferencial proprietário)

- **Novo package `internal/crossdoc/`** com interface `CrossDocRule`,
  Registry, Engine (orquestra paralelo).
- **3 regras iniciais** (`XD-001`, `XD-002`, `XD-003`):
  - `XD-001`: Total ops 3040 vs clients 4111 (tolerância 5%, severity A).
  - `XD-002`: Modalidade 0213 (cheque especial) flag no 4111.
  - `XD-003`: Subsegmento DRSAC ESG (S4/S5) compatível com score ≥0.7.
- **Endpoint `POST /v1/crossdoc/validate`** recebe
  `{cadocs: {3040: xml, 4111: xml, 2030: xml}}` e retorna
  ValidationResponse com passed/errors/warnings/rules_run/rules_skipped.
- **Audit:** `crossdoc.validated` com metadata `{cadocs, passed, errors,
  warnings, rules_run, rules_skip}`.

#### 🔵 Frente 4 — Postgres driver

- **`db.Open` detecta DSN**:
  - `postgres://` ou `postgresql://` → pgx/v5 (database/sql bridge).
  - `file:` ou path cru → SQLite (preserva comportamento v1.4.x).
- **Pool diferenciado**: SQLite 8/2 (writes serializados) vs Postgres 25/5.
- **`Backend(dsn)` helper** retorna `"sqlite"` ou `"postgres"` (logging).
- **`docker-compose.yml`** (raiz): Postgres:16-alpine + serviços opcionais
  api/worker via profile `prod`.
- **`docs/postgres-setup.md`** quickstart + limitações.

#### 🟢 W3 — B01-B05 → registry (refator arquitetural)

- Nova interface `RawRule` em `audit/rules/registry.go` (opera em XML
  bruto, não *Doc3040 tipado).
- `RawRuleFunc` adapter permite usar func como RawRule.
- `Registry` agora dual map (`rules` + `rawRules`).
- `audit/service.go::applyRegra` remove ~30 linhas de if B01-B05 inline.

#### 🟢 W4 — cadoc list dinâmico (DB + cache)

- `schema.Registry.ListCadocs()` faz `SELECT DISTINCT cadoc_code` UNION
  `schema_versions + criticas`.
- `CadocListCache` in-memory 5min (mesmo padrão do ScanCache do R1).
- `internal/api/server.go::cadocsWithCache` abstrai cache vs DB.
- `listSchemas` e `listRules` consultam ambos via cache.

### 📊 Estatísticas

```
Testes:        99 (v1.4.4) → 213 RUN / 164 únicos (v1.5.0)
                          (+65% únicos, +115% runs c/ subtests)
Coverage:      ~70% média → ~75% média (medida por package, ver SPRINT_6_RESULTS)
Packages:      5 c/ tests → 10 c/ tests    (de 12 totais)
LOC:           ~4.200 → ~6.500             (+55%)
Commits:       10 commits Sprint 6 (v1.4.3 e v1.4.4 são anteriores à tag v1.4.4)
Migrations:    3 (001-003) → 5 (001-005)
Regras audit:  25 tipadas → 25 tipadas + 5 raw (B01-B05)

### 🩹 Validações 11-20 (post-ship hardening, in-place)

> **Detalhe:** cada validação profunda pós-release encontrou gaps reais
> (vetor pgx, reinvent-stdlib, DSN leak, deadlock panic, panic recover,
> http.Error 500, http.Error 4xx disclosure, audit log persistente,
> JSON Message field disclosure, token format disclosure, DOS-via-large-body,
> SafeError perf 1MB). Documentados em `VALIDATION_v1.5.0.md`,
> `VALIDATION_v1.5.0_DEEPER.md`, `VALIDATION_v1.5.0_DEEPEST.md`,
> `VALIDATION_v1.5.0_DEEPEST2.md`, `VALIDATION_v1.5.0_DEEPEST3.md`,
> `VALIDATION_v1.5.0_DEEPEST4.md`, `VALIDATION_v1.5.0_DEEPEST5.md`,
> `VALIDATION_v1.5.0_DEEPEST6.md`.

Resumo consolidado (validações 11-20):

| Validação | Findings | Críticos | Observação |
|-----------|----------|----------|------------|
| 11 | 9 | 0 (meta-validação) | Estrutura + docs |
| 12 | 9 | 4 | cmd/* entrypoints + middleware order + engine recover |
| 13 | 4 | 1 | Token prefix log + reinvent-stdlib `min()` + cmd panic recover |
| 14 | 5 | 1 | DSN log no cmd/seed + reinvent-stdlib indexOf + self-doc |
| 15 | 4 | 1 | pgx error leak (F15.1 PLUG inicial) + http 500 disclosure |
| 16 | 4 | 1 (F16.5 confirmou F15.1 PLUG) | Sweep universal SafeError + regex ampliado |
| 17 | 3 | 0 | Warn-level cmd/seed edge cases |
| 18 | 8 | 3 | HTTP 4xx disclosure (7 vetores) + audit log persistente (2) + GAP-7.4 version |
| 19 | 7 | 4 | JSON Message field disclosure (audit+crossdoc, 4 vetores) |
| 20 | 7 | 2 | Token format leak + DOS-via-large-body (maxBodyBytes middleware) |
| **TOTAL** | **60** | **17** | |

Pacote `internal/loggerutil` (F15.1 + F16.5 + F20.6 + F20.7) cobre:
- DSN canonical (postgres://, mysql://, etc)
- pgx key=value (`user=X database=Y`)
- password=X solto e ?password=X em query
- Bearer/JWT/Authorization-style tokens
- Vendor-specific token prefixes (ghp_, ya29., AKIA, xoxb-, sk_live_, etc)
- 16KB truncation para mensagens gigantes
9 validações seguidas com findings — pattern confirmado.

**Cobertura final pós-validação 20 (6 vetores paralelos + 2 arquiteturais):**
- Logger (Error/Warn/Info/Debug) com err → 100% via SafeError (F15.1+)
- HTTP responses 4xx/5xx com err → 100% via UserError (F18.1)
- AuditLog persistence com err → 100% via SafeError (F18.13/14)
- Version drift cross-pkg → 100% via internal/version (F18.4)
- Radar logger Error/Warn → 100% via SafeError (F18.9/11/12)
- JSON response Message field → 100% via SafeError (F19.10-13)
- **Token formats** → 100% via commonTokens regex (F20.6)
- **DOS-via-large-body** → 100% via MaxBytesReader middleware (F20.3)

**Versão:** inalterada (v1.5.0). Apenas hardening interno.
Regras cross:  0 → 3 (XD-001/002/003)
DB drivers:    1 (SQLite) → 2 (+Postgres)
Endpoints:     13 → 14 (+/v1/crossdoc/validate)
Tables:        7 → 8 (+radar_baselines)
```

### 🏗️ Lições aprendidas (memory entries candidatam)

1. **DOS-via-API rate limiting é obrigatório desde o dia 1** —
   agora coberto com FAIL CLOSED, audit, e testes de regressão.
2. **textual vs datetime comparison em SQLite** —
   `time.Now()` via driver modernc formatado em RFC3339 vs
   `CURRENT_TIMESTAMP` do SQLite em formato com espaço → comparação
   `<=` falhava silenciosamente. Solução: `DATETIME(CURRENT_TIMESTAMP,
   '+N seconds')` no próprio SQL.
3. **Dual registry pra regras que operam em representações diferentes**
   (tipada vs raw) sem forçar refactor de N regras já implementadas.
4. **Tests E2E pegam bugs latentes** que unit tests não pegam —
   `LoadCriticas` faltava NullString em `regra` e `descricao`.

### ⚠️ Gaps remanescentes (Sprint 7 backlog)

| # | Gap | Status pós-v23 | Sprint 7? |
|---|-----|-----------------|-----------|
| GAP-7.1 | Cross-doc L3 — `iterXMLElements` é implementação caseira | Persiste (Sprint 7) | ✅ Sprint 7 |
| GAP-7.2 | Cross-doc L3 — regras de agregação podem misinterpretar CDATA | Persiste | ✅ Sprint 7 |
| GAP-7.3 | Postgres integration tests sem testcontainers | Persiste (gap) | ✅ Sprint 7 |
| GAP-7.4 | ~~User-Agent hardcoded em radar.go~~ **F18.4 FIXED** | ✅ Resolvido em v18 | — |
| GAP-7.5 | ~~Migration 004 `INSERT OR IGNORE` Postgres-flavor~~ **F21.5 refutado** | ✅ Real é OK (race-free) | — |
| GAP-7.6 | Cross-doc engine goroutine pool | Persiste (paralelo) | Sprint 7+ |
| GAP-7.7 | cmd/* seeding needs explicit `-db` flag | Mitigado via env DATABASE_URL | Cosmetic |
| GAP-7.8 | ~~cmd/api graceful shutdown~~ **F12.4 OK** | ✅ Resolvido | — |
| GAP-7.9 | Mais regras 3040 (~25/320 implementadas) | Persiste | Sprint 7+ |
| **NEW** GAP-7.10 | RequestID não propaga para logs | F23.3 follow-up | Sprint 7 |
| **NEW** GAP-7.11 | `cmd/_verify` dev tool uso residual | F21.6 mitigado | — |

**Resumo validações v15-v23 (post-release hardening):**

| Val | Findings | Críticos |
|-----|----------|----------|
| 15  | 4  | 1 |
| 16  | 4  | 1 |
| 17  | 3  | 0 |
| 18  | 8  | 3 |
| 19  | 7  | 4 |
| 20  | 7  | 2 |
| 21  | 5  | 1 |
| 22  | 2  | 0 |
| 23  | 3  | 0 |
| **TOTAL** | **70** | **18** |

**Fase 1 (Sprint 6 v1.5.0 + hardening v15-v23):** SATURADA.
13 validações consecutivas com findings, 0 críticos em v22-v23.
15 categorias vetores fechadas. Recomenda-se encerrar Fase 1
e abrir Sprint 7 com mudança de modo (feature ou Postgres
integration tests).

### 📂 Commits Sprint 6

1. `feat(v1.5.0)` F6 schema tests + version bump
2. `fix(v1.5.0)` F3 race fix recordBaseline
3. `feat(v1.5.0)` W1+W2 worker hardening
4. `fix(v1.5.0)` R1 DOS-via-API prevention
5. `refactor(v1.5.0)` W3 B01-B05 → registry
6. `feat(v1.5.0)` W4 cadoc list do DB
7. `test(v1.5.0)` F7 migrate tests
8. `test(v1.5.0)` F8 E2E coverage + bug LoadCriticas
9. `feat(v1.5.0)` Cross-Doc L3
10. `feat(v1.5.0)` Postgres driver

### 📂 Arquivos modificados/criados (Sprint 6)

**Código (10+ arquivos):**
- `backend/internal/db/migrations/004_radar_baselines.sql` (NOVO)
- `backend/internal/db/migrations/005_worker_hardening.sql` (NOVO)
- `backend/internal/worker/worker.go` (NOVO, ~250 LOC)
- `backend/internal/worker/worker_test.go` (NOVO, ~390 LOC)
- `backend/internal/radar/admin.go` (NOVO, ~130 LOC)
- `backend/internal/radar/admin_test.go` (NOVO, ~150 LOC)
- `backend/internal/audit/rules/basic_rules.go` (NOVO, ~80 LOC)
- `backend/internal/audit/rules/raw_rules_test.go` (NOVO, ~180 LOC)
- `backend/internal/crossdoc/crossdoc.go` (NOVO, ~150 LOC)
- `backend/internal/crossdoc/engine.go` (NOVO, ~120 LOC)
- `backend/internal/crossdoc/registry.go` (NOVO, ~50 LOC)
- `backend/internal/crossdoc/crossdoc_test.go` (NOVO, ~230 LOC)
- `backend/internal/crossdoc/rules/3040_4111.go` (NOVO, ~170 LOC)
- `backend/internal/crossdoc/rules/registry.go` (NOVO, ~30 LOC)
- `backend/internal/schema/registry_test.go` (NOVO/COMPLETO)
- `backend/internal/api/server_test.go` (atualizado)
- `backend/internal/api/server_e2e_test.go` (NOVO)
- `backend/internal/api/server_admin_test.go` (NOVO)
- `backend/internal/api/server.go` (modificado)
- `backend/internal/audit/service.go` (modificado — W3 + F8.1 fix)
- `backend/internal/audit/rules/registry.go` (modificado — W3)
- `backend/internal/audit/rules/3040_test.go` (modificado)
- `backend/internal/radar/radar.go` (modificado — F3 + version bump)
- `backend/internal/radar/radar_test.go` (modificado — F3 tests)
- `backend/internal/schema/registry.go` (modificado — W4)
- `backend/internal/db/db.go` (modificado — Postgres driver)
- `backend/internal/db/migrate_test.go` (NOVO)
- `backend/cmd/worker/main.go` (modificado — sweeper loop)

**Infra/Docs:**
- `docker-compose.yml` (NOVO)
- `docs/postgres-setup.md` (NOVO)
- `SPRINT_6.md` (atualizado — status Aprovada)
- `VALIDATION_v1.5.0.md` (NOVO — esta validação)
- `SPRINT_6_RESULTS.md` (NOVO — resultados finais)

---

## v1.4.4 — 2026-07-03 (Validação profunda 10: itoa removed + User-Agent bump + self-doc fix)
## v1.6.0 — 2026-07-03 (Sprint 7a: Auth JWT real)

> **Status:** Shipped
> **Sprint:** Sprint 7a (SPRINT_7.md)
> **Versão:** minor (auth infra nova)

### 🎯 Auth JWT Real — substitui X-IF-ID placeholder

**Crítico:** X-IF-ID era string trust, sem auth real. F24.1 fechou vetor
de auth bypass (qualquer string era aceita). Sprint 7a introduz
**JWT bearer RS256** com claims tipadas, issuer pinning, key rotation.

### Features

- **internal/auth pkg:** Verifier RS256, Claims tipadas, Keyring rotação.
- **cmd/jwt-mint:** dev tool para gerar tokens (file-based private key).
- **cmd/api/main.go:** JWT verifier setup via env var
  `RADIANT_JWT_PUBLIC_KEY`. Dev mode via `RADIANT_DEV_AUTH=1` para
  migration helper (X-IF-ID fallback).

### Vetores fechados (validação 24)

- F24.1 auth bypass (crítico)
- F24.2 dev mode migration (médio)
- F24.3 key rotation grace (médio)
- F24.4 cmd/jwt-mint (baixo)
- F24.5 issuer pinning (baixo)

### Tests

- 253 → 270 tests passing (+17 com auth).
- vet-clean, race-clean, build-clean.

### Compatibility

- Default: JWT obrigatório. X-IF-ID retorna 401.
- Dev (`RADIANT_DEV_AUTH=1`): X-IF-ID fallback para migration.
- Production: configurar `RADIANT_JWT_PUBLIC_KEY` (PEM-encoded).

### Files (Sprint 7a)

- backend/internal/auth/{jwt,claims,keyring,middleware}.go (NOVO)
- backend/internal/auth/jwt_test.go (NOVO)
- backend/cmd/jwt-mint/main.go (NOVO)
- backend/internal/api/server.go (modified — middleware swap)
- backend/cmd/api/main.go (modified — env var wiring)
- backend/CHANGELOG.md (modified — esta entrada)
- VALIDATION_v1.6.0.md (NOVO)

---

## v1.7.0 — 2026-07-03 (Sprint 7b: Regras 3040 expandidas)

> **Status:** Shipped
> **Sprint:** Sprint 7b (SPRINT_7.md)
> **Versão:** minor (coverage expandida)

### 🎯 Cobertura de regras 3040: 30 → 60

Sprint 7b continua execute without pause (Henrique pediu). 30 regras
novas adicionadas ao Registry. Cobertura agora **55 tipadas + 5 raw
(B01-B05)**. Total: 60 regras no registry.

### Features — 30 regras novas (B16-B25, F06-F15, C06-C10, S06-S10)

**B16-B25 (10) — Básicas expandidas:**
- B16 TotalizadoresCoerentes (TotalCli = soma QtdCli)
- B17 DtBase formato YYYY-MM-DD
- B18 TpArq deve ser F ou S
- B19 Email formato
- B20 Tel formato (XX) XXXXX-XXXX
- B21 CNPJ raiz 8 dígitos
- B22 NomeResp não vazio
- B23 Mínimo 1 Agreg
- B24 DtBase não futura (até 2030)
- B25 QtdOp >= 1 por Agreg

**F06-F15 (10) — Formato expandido:**
- F06 ClassOp A-H, F07 Mod 2-4 dígitos, F08 NatuOp 01/02
- F09 UF válida (27 siglas), F10 VincME S/N, F11 PrzProvm S/N
- F12 TpCli 1=PF/2=PJ, F13 DesempOp numérico
- F14 FaixaVlr numérico, F15 OrigemRec 1-3 dígitos

**C06-C10 (5) — Campos Obrigatórios expandidos:**
- C06 ClassOp C-H requer ProvConsttd
- C07 DesempOp != "00" com vencimentos > 0
- C08 Tel preenchido requer Email
- C09 NatuOp=01 requer QtdCli
- C10 QtdOp>0 requer ClassOp

**S06-S10 (5) — Semânticas expandidas:**
- S06 QtdOp zero warning
- S07 Mod=0213 requer ClassOp E-H (cheque especial high risk)
- S08 PF com ClassOp A é suspeito
- S09 Soma V110..V165 ≈ QtdOp (10% tolerance)
- S10 NatuOp=01 com VincME=N (próprias não moeda estrangeira)

### Fuzz testing — GAP-7.1 / GAP-7.2 mitigado

`backend/internal/crossdoc/rules/iter_fuzz_test.go`:

```
427167 execs em 2 segundos
1 new interesting case descoberto
ZERO panics ou deadlocks em:
  - XML vazio
  - CDATA com nested Mod
  - Entities (5 &lt; 10 &amp; ok)
  - Control chars
  - 1.5MB spam
  - Case wrong (agreg lowercase)
  - Mixed attrs (Mod + ExtraAttr)
```

### Catalog documentation

`backend/docs/rules-3040-catalog.md`:
- 60 regras catalogadas (todas com code/severity/sheet/desc/example)
- Resumo por categoria + sprint origem
- Vetor mapeamento aos tests

### Tests

- 270 → 301 tests passing (+20 com regras).
- vet-clean, race-clean, build-clean.
- Fuzz: 427k execs / 0 panics.

### Compatibility

- Aditivo — adicionar regras não é breaking.
- Registry API estável.
- Tests existentes continuam passando.

### Files (Sprint 7b)

- backend/internal/audit/rules/3040_expanded.go (NOVO, 565 LOC)
- backend/internal/audit/rules/3040_expanded_test.go (NOVO)
- backend/internal/crossdoc/rules/iter_fuzz_test.go (NOVO)
- backend/docs/rules-3040-catalog.md (NOVO)
- backend/CHANGELOG.md (modified — esta entrada)
- VALIDATION_v1.7.0.md (NOVO)

---

## v2.0.0 — 2026-07-04 (Sprint 7c: Frontend Norma Console)

> **Status:** Shipped
> **Sprint:** Sprint 7c (SPRINT_7.md)
> **Versão:** **major** — frontend empacotado no mesmo repo

### 🎯 Frontend Next.js 14 — dashboard IF

Sprint 7c fecha com frontend funcional. Stack: App Router + Tailwind +
TanStack + Zustand. Auth via JWT bearer + cookie httpOnly. **6 páginas
funcionais** (4 prontas + 2 placeholders Sprint 8).

### Features

- **19 arquivos TypeScript** (.ts/.tsx) — ~1100 LOC frontend
- **6 páginas funcionais:**
  - `/login` — client form picker (3 IFs demo + admin)
  - `/` — server dashboard com stats agregadas
  - `/radar` — server lista + client resolve button
  - `/regras` — server catalog parse de `../docs/rules-3040-catalog.md`
  - `/envios` — server placeholder (TODO Sprint 8)
  - `/auditoria` — server LGPD view (TODO Sprint 8)
- **Auth flow:** JWT bearer + cookie `rn_jwt` httpOnly (XSS-safe)
- **JWT-injecting server proxy:** `/v1-api/[...path]/route.ts`
- **OpenAPI 3.0 spec** (14 endpoints documentados)
- **TypeScript strict mode** + Tailwind Radiant brand colors

### Stack

| Camada | Lib | Versão |
|--------|-----|--------|
| Framework | Next.js | ^14.2.18 |
| Linguagem | TypeScript | ^5.6.3 |
| Styling | TailwindCSS | ^3.4.15 |
| Server state | TanStack Query | ^5.59.0 |
| Local state | Zustand | ^5.0.1 |
| HTTP client | Axios | ^1.7.7 |
| JWT | jose | ^5.9.6 |
| Forms | react-hook-form | ^7.53.0 |
| Validation | zod | ^3.23.8 |
| Icons | lucide-react | ^0.460.0 |

### Vetores fechados (cross-cutting)

| Vetor | Frontend | Backend (Sprint 7a) |
|-------|----------|---------------------|
| Auth bypass | X-IF-ID não passa de dev | JWT RS256 |
| XSS in JWT | httpOnly cookie | N/A |
| CSRF | Same-Site Lax + Same-Origin | N/A |
| Token in logs | JWT só em Authorization header (no body) | SafeError |

### Tests

- Frontend: **npm install OK** (167 packages), **build validação em curso**
- Backend: 301 tests (inalterado — Sprint 7c não muda backend)
- vet-clean, race-clean, build-clean (backend).

### Compatibility

- **Sprint 8 wire-up:** JWT bridge real entre frontend e backend.
- **Dev mode preservado:** `NEXT_PUBLIC_RADIANT_DEV_MODE=1` no frontend
  + `RADIANT_DEV_AUTH=1` no backend. Em prod: ambos off, IdP real.

### Files (Sprint 7c)

- `frontend/` (NOVO diretório, 19 arquivos .ts/.tsx + config)
- `backend/docs/api/openapi.yaml` (NOVO)
- `backend/CHANGELOG.md` (modified — esta entrada)
- `VALIDATION_v2.0.0.md` (NOVO)

---

## v1.6.0+ → v2.0.0 — Cumulative changes

```
Auth:           X-IF-ID trust → JWT RS256 (issuer pinned, kid rotated)
Regras 3040:    30 (25 tipadas + 5 raw) → 60 (55 tipadas + 5 raw)
Backend tests:  200 → 301 (+101)
Frontend:       nenhum → Next.js 14 + 19 arquivos TS/TSX
OpenAPI spec:   nenhum → 14 endpoints documentados
Sprint 7 lint:  70 findings → 75 findings (5 novos no auth)
                críticos 18 → 19 (+F24.1)
```

---

## v2.0.0.post — 2026-07-04 (Build fixes pós-tag)

> **Status:** Hotfix pós-tag
> **Versão:** pós-v2.0.0 (não-bump — ainda v2.0.0)
> **Motivo:** `npm run build` do frontend quebrou após o commit da tag

### 🐛 Build frontend quebrado — 2 fixes

Tentativa inicial de build pós-tag falhou. **2 bugs latentes** descobertos:

#### F1 — `postcss.config.js` usava sintaxe ESM em projeto CJS

```js
// ❌ Antes — `export default` em arquivo sem "type": "module"
export default { plugins: { tailwindcss: {}, autoprefixer: {} } }

// ✅ Depois — CJS (consistente com next.config.js)
module.exports = { plugins: { tailwindcss: {}, autoprefixer: {} } }
```

Sintoma: `Error: Your custom PostCSS configuration must export a 'plugins' key.`
Causa raiz: `postcss-load-config@6.0.1` carrega `postcss.config.js` como CJS quando
o `package.json` não declara `"type": "module"`. O `export default` virava
`undefined` em runtime e o webpack não encontrava `plugins`.

#### F2 — `Session` interface não-exportada em `auth.ts`

```ts
// ❌ Antes
interface Session { ... }   // local, não exporta

// ✅ Depois
export interface Session { ... }
```

`src/lib/session.ts` fazia `import { verifyJwtServer, type Session } from './auth'`,
mas `Session` era apenas declarada local. TypeScript strict bloqueou o build.

### Validação pós-fix

```
✓ Compiled successfully
✓ Generating static pages (10/10)
10 rotas geradas (/, /login, /radar, /regras, /envios, /auditoria, /api/login, /v1-api/proxy/[...path], /_not-found)
First Load JS shared: 87.3 kB
```

### Files (v2.0.0.post)

- `frontend/postcss.config.js` (fix CJS)
- `frontend/src/lib/auth.ts` (export Session)
- `.gitignore` (adiciona frontend/node_modules, frontend/.next, etc.)
- `frontend/package-lock.json` (lockfile commitado)

---

## v2.0.1 — 2026-07-04 (27ª validação: 9 findings fechados)

> **Status:** Shipped
> **Sprint:** Validação 27 (VALIDATION_v2.0.0_POST.md)
> **Versão:** **patch** — 2 críticos + 4 médios + 3 polimentos fechados
> **Trigger:** Henrique pediu validação profunda pós-tag v2.0.0
> **Versão anterior:** v2.0.0.post

### 🎯 Resumo

Validação 27 fechou **9 findings** deixados pela release v2.0.0. Sem
esses fixes, deployment em produção real quebraria todos os 5 endpoints
mutantes (`/v1/validate`, `/v1/sta/submit`, `/v1/radar/alerts/{id}/resolve`,
`/v1/radar/scan`, `/v1/crossdoc/validate`) por vetor de leitura errada
de auth claims. Além disso, `/healthz` reportaria `1.5.0` em vez de
`2.0.0` (doc drift quebrando consumers).

### 🐛 Bugs corrigidos (por severidade)

#### 🔴 Críticos (2)

| # | Bug | Fix |
|---|-----|-----|
| F27.1 | Handlers liam `X-IF-ID` cru do header ao invés de `auth.ClaimsFromContext` — em prod com JWT-only, todos os 5 endpoints mutantes retornariam 401 "X-IF-ID required". Vetor de cross-tenant se cliente injetasse X-IF-ID malicioso com JWT válido. | Helper `getIfID(r)` em `internal/api/server.go` que prioriza `Claims.IFID` (JWT validated) e fallback X-IF-ID só em dev mode. Substituído nos 5 callsites. 3 testes de regressão em `ifid_test.go` (Claims, fallback header, vazio, edge-case Claims.IFID vazio). |
| F27.2 | `/healthz` retornava `"version":"1.5.0"` enquanto CHANGELOG/SPRINT_7_RESULTS diziam v2.0.0 — const `version.Version` foi deixada hardcoded em v1.5.0. Doc drift quebra consumers que checam versão. | `const Version = "2.0.0"` em `internal/version/version.go`. Dockerfile parametrizado com `ARG VERSION` + ldflags `-X ...version.Version=${VERSION}` para build-time override. OpenAPI `HealthStatus.version` example atualizado para "2.0.0". |

#### 🟡 Médios (4)

| # | Bug | Fix |
|---|-----|-----|
| F27.4 | Axios client `api.interceptors.request.use` tentava ler `rn_jwt` via `document.cookie` — código morto (cookie é httpOnly, JS não lê). | Removido interceptor. Client Axios agora é só para endpoints públicos / server-side. Documentado no header do arquivo. |
| F27.5 | ResolveButton (client-side) construía `Authorization: Bearer undefined` quando cookie httpOnly resultava em `token = undefined`. | Removida lógica. Server-side proxy `/v1-api/proxy/[...path]/route.ts` injeta Authorization automaticamente via `next/headers cookies()`. |
| F27.6 | Frontend sem `.eslintrc.json` — `npm run lint` falhava com prompt interativo pedindo config. | Adicionado `.eslintrc.json` extends `next/core-web-vitals`. Instalado `eslint@^8.57.0` + `eslint-config-next@^14.2.18` como devDeps. `npm run lint` agora reporta "✔ No ESLint warnings or errors". |
| F27.10 | OpenAPI `HealthStatus.version` example "1.6.0" inconsistente com `info.version` ("2.0.0") e com `/healthz` (que retornava 1.5.0 antes do F27.2 fix). | Atualizado para "2.0.0" + description nota sobre ldflags. |

#### 🟢 Polimentos (3)

| # | Issue | Fix |
|---|-------|-----|
| F27.13 | `frontend/src/lib/api-fetch.ts` usava `await import('next/headers')` (dynamic import anti-pattern em Next 14 ESM). | Movido para top-level `import { cookies } from 'next/headers'`. |
| F27.14 | `frontend/src/app/radar/page.tsx` tinha `import { ResolveButton }` no final do arquivo (anti-pattern). | Movido para topo com outros imports. |
| F27.16 | Cookie `rn_jwt` sem `secure: true` flag — em prod (HTTPS) sem secure flag pode vazar em HTTP downgrade. | Adicionado `secure: process.env.NODE_ENV === 'production'` em `frontend/src/app/api/login/route.ts`. Dev local (HTTP) continua funcional. |

### 📊 Estatísticas

```
Auth flow:
  Antes: Claim JWT populado → handler ignorava → 401 "X-IF-ID required"
  Depois: Claim JWT populado → getIfID() retorna Claims.IFID → endpoint funciona
  
Version reporting:
  Antes: /healthz → "1.5.0"
  Depois: /healthz → "2.0.0" (const) ou "v2.0.1+commit..." (ldflags)

Tests:
  Antes: 301 tests
  Depois: 304 tests (+3 F27.1 regression)

Build artifacts:
  Antes: front build passa, lint broken
  Depois: front build + lint + type-check all clean
```

### Compatibility

- **Backwards compat**: dev mode (`RADIANT_DEV_AUTH=1`) continua
  funcionando. Helper getIfID fallback para X-IF-ID header mantém
  tests legacy passando.
- **JWT-only prod**: agora funciona end-to-end (Sprint 7a fechou metade;
  F27.1 fechou a outra metade).

### Files (v2.0.1)

**Backend:**
- `backend/internal/auth/middleware.go` (add `WithClaims` helper)
- `backend/internal/api/server.go` (add `getIfID` helper + substituir 5 callsites)
- `backend/internal/api/ifid_test.go` (NOVO, 4 testes)
- `backend/internal/version/version.go` (const 1.5.0 → 2.0.0)
- `backend/Dockerfile` (ARG VERSION + ldflags em 4 binários)

**OpenAPI / docs:**
- `backend/docs/api/openapi.yaml` (version example + description)

**Frontend:**
- `frontend/src/lib/api.ts` (remove Axios interceptor)
- `frontend/src/lib/api-fetch.ts` (import dinâmico → top-level)
- `frontend/src/app/radar/page.tsx` (import no topo)
- `frontend/src/components/resolve-alert-button.tsx` (remove Bearer undefined)
- `frontend/src/app/api/login/route.ts` (secure flag conditional)
- `frontend/.eslintrc.json` (NOVO)
- `frontend/package.json` + package-lock.json (eslint devDeps)

---

## v2.0.0+ → v2.0.1 — Cumulative over 27ª validação

```
Findings fechados:           9 (2 críticos + 4 médios + 3 polimentos)
Backend tests:               301 → 304 (+3 regressão F27.1)
Frontend lint:               broken → clean (Strict Next config)
Frontend build:              ✓ unchanged
Frontend bundle:             -200B (radiar removido)
Doc drift:                   5 sync items (LOC, paths, version example,
                             file count, secure flag)
Segurança auth:              vetor cross-tenant injection FECHADO
```

---

## v2.1.0 — 2026-07-04 (Sprint 8a: JWT bridge real)

> **Status:** Shipped
> **Sprint:** Sprint 8a (ver SPRINT_8.md + SPRINT_8_RESULTS.md)
> **Versão:** **minor** — nova feature (dev-token mint in-process)
> **Trigger:** Gaps remanescentes de Sprint 7c — frontend usava JWT fake (`dev:<if>:<role>`) enquanto backend exigia JWT RS256 real

### 🎯 Resumo

Sprint 8a entrega **bridge JWT real frontend↔backend**. Em dev, frontend
`/api/login` chama novo endpoint `POST /v1/auth/dev-token` que emite JWT
RS256 in-process. Cookie `rn_jwt` passa a armazenar JWT real (não string
opaca). Backend JWT verifier (mesma chave pública carregada em
`RADIANT_JWT_PUBLIC_KEY`) aceita os tokens.

### ✨ Features

#### 🔴 Backend — `internal/auth/mint.go` (NOVO, 145 LOC)

Helper `auth.Signer` que encapsula signing JWT RS256:
- `NewSigner(SignerConfig)` — cria a partir de PEM-encoded private key.
- `NewSignerFromFile(path, kid, issuer)` — shorthand para file path.
- `Mint(Claims)` — assina JWT, valida claims antes.
- `MintSimple(ifID, role, ttl)` — helper dev/demo com validação
  integrada (alfanumérico + dash + underscore, max 64 chars).
- `TTLCap = 30 dias`, `TTLDefault = 24h`.

#### 🔴 Backend — `internal/api/auth_handlers.go` (NOVO, 173 LOC)

Novo endpoint `POST /v1/auth/dev-token`:
- Ativado por `RADIANT_DEV_TOKEN=1` env.
- Requer chave privada (path `RADIANT_DEV_JWT_PRIVATE_KEY` ou inline
  `RADIANT_DEV_JWT_PRIVATE_KEY_PEM`).
- **404** quando flag off (esconde endpoint em prod).
- **503** quando flag on mas signer não configurado.
- **400** quando if_id ausente, role inválida, ttl inválido.
- Audit emission: `auth.dev_token.minted` for forensic trail.
- TTL clamp: max 30 dias (defesa contra tokens de vida excessiva).

#### 🔴 Backend — `internal/api/server.go` (modified)

- Field `DevSigner *auth.Signer` adicionado.
- Router ganha `r.Route("/v1/auth", ...)` FORA do group `/v1` com JWT
  middleware (precisa estar acessível sem auth, mas com flag guard).

#### 🟡 Backend — `cmd/jwt-mint/main.go` (refactored)

- Lógica de signing delegada para `auth.Signer` (DRY).
- TTL clamp aplicado.
- Sub-claim default agora = ifID (não "dev-user").

#### 🟡 Frontend — `src/app/api/login/route.ts` (rewritten)

- Chama `POST /v1/auth/dev-token` no backend.
- 502 quando backend offline (era silencioso).
- 503 com hint quando dev-token endpoint disabled.
- Cookie `rn_jwt` agora armazena JWT real (string `eyJ...` em vez de
  `dev:<if>:<role>`).

### 🧪 Tests adicionados (18 novos)

#### `internal/auth/mint_test.go` (13 testes)

```
✓ TestNewSigner_ValidPEM
✓ TestNewSigner_PEMvazio
✓ TestNewSigner_KidVazio
✓ TestNewSigner_IssuerVazio
✓ TestSigner_Mint_ValidClaims
✓ TestSigner_Mint_InvalidClaims
✓ TestSigner_MintSimple
✓ TestSigner_MintSimple_Validations (8 subtests)
✓ TestSigner_Roundtrip (sign+verify)
✓ TestSigner_IssuerOverride
✓ TestTTLCap
```

#### `internal/api/auth_handlers_test.go` (8 testes)

```
✓ TestDevToken_EndpointDisabled (404 quando flag off)
✓ TestDevToken_SignerMissing (503 quando signer nil)
✓ TestDevToken_MintValid (happy path + JWT 3-parts + kid=k1)
✓ TestDevToken_AdminRole
✓ TestDevToken_InvalidRole
✓ TestDevToken_MissingIFID
✓ TestDevToken_TTLClamp (60d pedido → 30d cap)
✓ TestDevToken_Roundtrip (header contém kid=k1)
```

### 📊 Estatísticas

```
Sprint 8a entrega:
  Backend tests:        304 → 322 (+18 novos = 13 mint + 8 dev-token - 3 setup)
  Backend code:         ~315 LOC new (mint.go 145 + auth_handlers.go 173 - go.sum)
  Frontend code:        ~70 LOC rewritten (login route)
  OpenAPI:              14 → 15 endpoints (1 novo: /v1/auth/dev-token)
  Build/lint/type-check all clean
```

### Compatibility

- **Backwards compat**: dev mode X-IF-ID fallback (`RADIANT_DEV_AUTH=1`)
  continua funcionando para tests legacy.
- **JWT real bridge**: agora funcional end-to-end. Frontend → Backend
  dev-token → JWT válido → backend verifier aceita.
- **Prod safety**: `/v1/auth/dev-token` retorna 404 (não 503) quando
  flag off. Endpoint existence hidden.

### Setup necessário

```bash
# 1. Gerar par RSA dev (PKCS#1)
openssl genrsa -out dev-private.pem 2048
openssl rsa -in dev-private.pem -pubout -out dev-public.pem

# 2. Backend dev mode
export RADIANT_DEV_TOKEN=1
export RADIANT_DEV_JWT_PRIVATE_KEY=./dev-private.pem
export RADIANT_JWT_PUBLIC_KEY="$(cat dev-public.pem)"
export RADIANT_JWT_ISSUER=radiant-norma
export RADIANT_JWT_KID=k1

# 3. Frontend dev mode (já suportado via NEXT_PUBLIC_RADIANT_DEV_MODE=1)
export NEXT_PUBLIC_RADIANT_DEV_MODE=1

# 4. Start backend
cd backend && go run ./cmd/api

# 5. Start frontend
cd frontend && npm run dev

# Frontend /login → POST /api/login → calls /v1/auth/dev-token → JWT real
```

### Files (Sprint 8a)

**Backend (NOVO):**
- `backend/internal/auth/mint.go` (Signer helper)
- `backend/internal/auth/mint_test.go` (13 testes)
- `backend/internal/api/auth_handlers.go` (dev-token handler)
- `backend/internal/api/auth_handlers_test.go` (8 testes)

**Backend (modified):**
- `backend/internal/api/server.go` (DevSigner field + route wire)
- `backend/cmd/api/main.go` (DevSigner config reading env)
- `backend/cmd/jwt-mint/main.go` (refactored to use Signer)
- `backend/docs/api/openapi.yaml` (1 novo endpoint + 2 schemas)

**Frontend (rewritten):**
- `frontend/src/app/api/login/route.ts` (chama backend real)

**Docs:**
- `CHANGELOG.md` (esta entry)
- `SPRINT_8_RESULTS.md` (NOVO)

---
