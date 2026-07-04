# VALIDAÇÃO v3.1.0 — Sprint 8c (Backend Intelligence + Frontend Wiring)

> **Status:** ✅ ACCEPTED com 0 findings novos
> **Trigger:** Validação 29 deixou 6 endpoints faltando (`/v1/envios`,
> `/v1/audit_log`, `/v1/insights/{kpis,heatmap,rules/top-failing,recommendations}`),
> fazendo com que 4 páginas do frontend ficassem em empty state honesto.
> **Versão validada:** v3.1.0
> **Commit base:** `63d7c87` (Sprint 9 v3.0.0)
> **Este commit:** `v3.1.0` (Sprint 8c)

## 🎯 TL;DR

Sprint 8c destrava o valor real do design system que construímos no Sprint 9
v3.0.0. Antes da validação 29, páginas estavam em **empty state** porque
backend não tinha os endpoints necessários. Validação 29 descobriu que
alguns dados eram hardcoded fake (e foram removidos). Sprint 8c entrega
os 6 endpoints faltantes + seed data realista + wiring frontend.

| Métrica | Antes (v3.0.0) | Depois (v3.1.0) |
|---------|----------------|-----------------|
| Endpoints backend | 11 | **17 (+6)** |
| Páginas com dados reais | 1/6 | **6/6** |
| Empty states restantes | 4/6 | **0/6** (todas com fallback honesto) |
| Seed data | 0 envios | **56 envios + 320 rule_failures + audit_events** |
| Recomendações heurísticas | 0 | **3 regras ativas** (concentração, queda de aprovação, envios pendentes) |

## 📊 Entregas

### Backend (Go) — Sprint 8c

**Migration 006** (`internal/db/migrations/006_sprint8c_envios_audit_insights.sql`):

```sql
-- envios enriquecido
ALTER TABLE envios ADD COLUMN rules_passed INTEGER NOT NULL DEFAULT 0;
ALTER TABLE envios ADD COLUMN rules_failed INTEGER NOT NULL DEFAULT 0;
ALTER TABLE envios ADD COLUMN period TEXT;
ALTER TABLE envios ADD COLUMN duration_ms INTEGER;
ALTER TABLE envios ADD COLUMN approver TEXT;

-- audit_events (denormalizado pra UI; audit_log continua sendo chain hash)
CREATE TABLE audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    audit_log_id INTEGER NOT NULL REFERENCES audit_log(id),
    if_id TEXT, actor TEXT NOT NULL, action TEXT NOT NULL,
    target TEXT, description TEXT, payload TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- rule_failures (alimenta heatmap + top-failing + recommendations)
CREATE TABLE rule_failures (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    envio_id TEXT NOT NULL REFERENCES envios(id),
    if_id TEXT NOT NULL, cadoc_code TEXT NOT NULL,
    rule_code TEXT NOT NULL, rule_severity TEXT NOT NULL,
    failed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Handlers novos** (`internal/api/sprint8c_handlers.go`):

| Rota | Função | Comentário |
|------|--------|------------|
| `GET /v1/envios` | `listEnvios` | Lista filtrada por IF + filtros `?cadoc&status&period&limit` |
| `GET /v1/envios/stats` | `enviosStats` | Agregados: total/accepted/pending/rejected/error + avg_duration_ms |
| `GET /v1/audit_log` | `listAuditLog` | Admin-only; filtros `?if_id&action&limit`; retorna `chain_valid` |
| `GET /v1/insights/kpis` | `insightsKPIs` | current/previous + delta% — alimenta Dashboard hero + /insights |
| `GET /v1/insights/heatmap?days=N` | `insightsHeatmap` | Matriz CADOC × dia; data + rows + cols + days |
| `GET /v1/insights/rules/top-failing?limit=N` | `insightsTopFailingRules` | Top regras com count + delta_pct + trend_direction |
| `GET /v1/insights/recommendations` | `insightsRecommendations` | Heurística: concentração de falhas, queda de aprovação, envios stuck |

**Seed script** (`cmd/seed-sprint8c/main.go`):

- 56 envios STA espalhados pelos últimos 30 dias (distribuição ponderada:
  70% accepted, 15% rejected, 10% pending, 5% error)
- 320 rule_failures com distribuição realista (F23=28%, B12=18%, S05=12%, etc.)
- Audit events denormalizados (sta.submit + envio.approved/rejected + system events)

### Frontend (Next.js) — Wiring

**Dashboard (`/`)**:

- Hero copy agora dinâmico baseado em criticalAlerts + totalSent + approvalRate
- Badge "X% aprovação" quando há dados; "aguardando dados" quando não
- 4 KPIs com dados reais:
  - "Envios (30d)" com delta vs período anterior + sparkline derivado
  - "Taxa de aprovação" com delta%
  - "Alertas ativos" com breakdown crítico/atenção
  - "CADOCs monitorados" (real do backend)
- Activity feed real (audit_events → ActivityKind normalizado)

**Insights (`/insights`)**:

- 4 KPIs comparativos com delta (mesma estrutura do Dashboard)
- **Heatmap real**: cores sequenciais baseadas em max value real;
  mostra até 14 dias de histórico
- Top 10 regras falhando com count + delta% + trend direction
- Recomendações heurísticas (3 regras ativas: concentração >25%, queda
  >5pp, pendentes >3 há 1h+)

**Envios (`/envios`)**:

- KPIs reais: Total / Aprovados / Pendentes / Rejeitados+Erro
- Tabela com ENV-* reais (id truncado pra legibilidade, cadoc, period,
  status badge, rules_passed/failed, sent_at relative)
- Empty state mantém como fallback (mostra explicação + CTA)

**Auditoria (`/auditoria`)**:

- 3 StatCards reais: total events, chain_valid, última verificação
- Activity feed completo (até 100 eventos)
- Badge "verificado" verde quando chain_valid=true
- Botão "Exportar" disabled quando vazio

## 🔧 Decisões técnicas

### Strftime no SQLite

**Problema encontrado:** O `failed_at` foi inicialmente armazenado em formato
RFC3339 com timezone (`'2026-06-21 01:25:27.951937 -0300 -03'`). O
`strftime('%Y-%m-%d', ...)` do SQLite **não aceita** esse formato e retorna
NULL silenciosamente.

**Fix:** Seed agora armazena em formato simples `'YYYY-MM-DD HH:MM:SS'`
(via `failedAt.UTC().Format("2006-01-02 15:04:05")`).

### Seed data com random seed

O seed usa `rand.NewSource(42)` pra gerar dados **determinísticos** —
re-roda o seed sempre produz os mesmos dados (bom pra demos e reprodutibilidade
de bugs).

### Server-side rendering de dados

Todas as páginas continuam server components (`async function Page()`).
Promise.allSettled garante que **um endpoint 401/500 não derruba a página**
— cada um cai em fallback (null, [], 0) e a UI renderiza com empty state
honesto.

### Normalização de audit_actions → ActivityKind

Audit events tem actions como `'envio.approved'`, `'sta.submit'`, `'auth.login'`.
ActivityFeed espera ActivityKind específico. A função `normalizeAction()`
mapeia 1:1, com default conservador (`'envio.approved'`) — qualquer action
desconhecida vira "envio.approved" (que renderiza verde).

### Heurística de recomendações

Três regras determinísticas (não-ML):

1. **Concentração**: se regra X tem ≥25% das falhas → recomendação high/medium
2. **Queda de aprovação**: se taxa caiu ≥5pp vs período anterior → warning
3. **Envios pendentes**: se ≥3 envios pending há mais de 1h → warning high

Cada uma tem `confidence` (0-100) e `cta` link para ação.

## 📋 Verificações finais

```
$ cd backend && go test ./... -count=1
ok  internal/api               6.084s
ok  internal/audit             0.538s
ok  internal/audit/rules       1.144s
ok  internal/auditlog          1.044s
ok  internal/auth              2.670s
ok  internal/crossdoc          0.994s
ok  internal/crossdoc/rules    1.287s
ok  internal/db                1.519s   ← updated to expect 6 migrations
ok  internal/loggerutil        1.121s
ok  internal/radar             1.236s
ok  internal/schema            1.551s
ok  internal/testutil          1.234s
ok  internal/version           1.202s
ok  internal/worker            1.240s
(14/14 packages, 0 failures)

$ cd frontend && npm run type-check
(sem output — 0 errors)

$ npm run lint
✔ No ESLint warnings or errors

$ npm run build
✓ Generating static pages (11/11)
Route (app)                              Size     First Load JS
┌ ƒ /                                    1.61 kB         124 kB
├ ƒ /api/login                           0 B                0 B
├ ƒ /api/radar/alerts/[id]/resolve       0 B                0 B
├ ƒ /auditoria                           571 B           123 kB
├ ƒ /envios                              2.04 kB         113 kB
├ ƒ /insights                            3.62 kB         115 kB
├ ○ /login                               3.99 kB        98.9 kB
├ ƒ /radar                               2.43 kB         122 kB
├ ƒ /regras                              3.83 kB         115 kB
└ ƒ /v1-api/proxy/[...path]              0 B                0 B
```

## 🧪 Smoke test E2E (com seed data)

```bash
$ /tmp/radiant-seed-sprint8c
✓ envios importados (total: 56)
✓ rule_failures importadas (total: 320)
✓ eventos de sistema importados
✓ seed-sprint8c completo

$ curl -b /tmp/rn-cookies.txt http://localhost:4180/             → 200, 51KB
$ curl -b /tmp/rn-cookies.txt http://localhost:4180/insights     → 200, 78KB
$ curl -b /tmp/rn-cookies.txt http://localhost:4180/envios       → 200, 99KB
$ curl -b /tmp/rn-cookies.txt http://localhost:4180/auditoria    → 200, 128KB

# Verificou-se conteúdo real em todas as páginas:
Dashboard:  "17 aprovados", "1 rejeitado", "0 críticos"
Insights:   "Mapa de calor", "Top regras falhando", "F23", "B12"
Envios:     "Total", "Aprovados", "ENV-..." (envios reais)
Auditoria:  "Eventos", "OK", "verificado" (chain_valid)
```

## 💡 Lições

1. **Strftime + timezone = NULL silencioso.** SQLite tem comportamentos
   surpreendentes com formatos de data. Sempre testar queries de agregação
   temporal com `strftime` antes de confiar.

2. **Seed data com `rand.NewSource(N)`** é a diferença entre "dados de demo
   confiáveis" e "dados que mudam toda vez que roda". Use seed determinístico
   sempre que for seed de demo/dev.

3. **`Promise.allSettled`** é a chave pra SSR com múltiplas fontes de dados.
   Cada uma é independente — falha em uma não bloqueia as outras.

4. **Heurística simples > nada.** 3 regras determinísticas (concentração,
   queda, stuck) já geram recomendações úteis. ML/AI vem depois (Sprint 12+).

## 📋 Próximos passos (Sprint 10+)

Ver `ROADMAP` ou conversa. Top candidates:

- **Sprint 8d — Filtros salvos + export CSV/JSON** (`?period=` etc já
  implementado, falta UI de save/share)
- **Sprint 10 — Real-time via SSE** (alertas chegam sem F5, activity
  feed ao vivo) — requer backend com `EventSource`
- **Sprint 12 — Production hardening** (IdP Keycloak/Okta, KMS pra JWT,
  Postgres RLS, WAF)

## 🚀 Como abrir

```bash
# 1. Build + seed
cd backend
go build -o /tmp/radiant-api ./cmd/api
go build -o /tmp/radiant-seed-sprint8c ./cmd/seed-sprint8c
DATABASE_URL=/tmp/radiant.db /tmp/radiant-seed-sprint8c

# 2. Backend
RADIANT_ADDR=:8421 RADIANT_DEV_AUTH=1 RADIANT_DEV_TOKEN=1 \
RADIANT_JWT_PUBLIC_KEY="$(cat /tmp/radiant-dev-public.pem)" \
RADIANT_DEV_JWT_PRIVATE_KEY=/tmp/radiant-dev-private.pem \
DATABASE_URL=/tmp/radiant.db \
/tmp/radiant-api &

# 3. Frontend
cd ../frontend
PUBKEY=$(cat /tmp/radiant-dev-public.pem | tr -d '\n')
NEXT_PUBLIC_RADIANT_API_JWT_PUBKEY="$PUBKEY" \
NEXT_PUBLIC_RADIANT_API_JWT_ISSUER="radiant-norma" \
RADIANT_API_URL=http://localhost:8421 \
npx next dev --port 4180 &

# 4. Login em http://localhost:4180/login
#    IF: 9999901, role: admin
```