# VALIDATION v2.0.0_POST — 27ª validação profunda (Sprint 7c post-tag)

> **Status:** 🟡 FIXES EM ANDAMENTO
> **Data:** 2026-07-04
> **Trigger:** Henrique pediu validada profunda — "passa tudo, código, docs, estrutura"
> **Versão-alvo:** v2.0.0.post (cumulativa sobre v2.0.0)
> **Marco:** primeira validação pós-Sprint 7 (frontend + auth + 60 regras)

## 🎯 Resumo executivo

Revisão completa após release v2.0.0 cobriu **código, arquitetura, docs,
build, auth flow, contratos OpenAPI ↔ frontend ↔ backend**. **9 findings
encontrados, 2 CRÍTICOS, 4 MÉDIOS, 3 DOC DRIFT.** Critical findings
introduziriam **quebra funcional em produção** se não corrigidos antes
de deploy real (não-dev-mode).

**Stats validados:**

```
Backend:
  ✓ go build ./...              clean
  ✓ go vet ./...                clean
  ✓ go test ./... -count=1      301 tests passing
  ✓ go test ./... -race         301 tests passing (race detector)
  ✗ /healthz retorna "1.5.0"    deveria ser 2.0.0  → F27.2

Frontend:
  ✓ npm run build               OK (10 rotas, 87.3 kB First Load JS)
  ✓ tsc --noEmit                clean (TypeScript strict)
  ✗ npm run lint                NÃO CONFIGURADO     → F27.6
  ✗ LOC: 953 (não 1100)         doc drift            → F27.10
  ✗ Arquivos: 17 .ts/.tsx (não 19) doc drift          → F27.9

Contratos:
  ✓ OpenAPI 3.0 spec            14 endpoints
  ✗ Frontend chama /v1/dashboard/envios (não existe) → F27.3
  ✗ Frontend chama /v1/envios, /v1/audit_log (não existem) → F27.8
```

## 🔴 F27.1 — Vetor crítico: handlers leem X-IF-ID cru ao invés de Claims

**Severidade:** 🔴 ALTA — quebra funcional em produção

**Localização:** `backend/internal/api/server.go:380, 433, 591, 617, 708`

**Contexto:**

Sprint 7a (v1.6.0) introduziu JWT bearer RS256 com middleware
`auth.Middleware(s.Auth)` que popula `context.Claims` após JWT válido.
Helper `auth.ClaimsFromContext(ctx)` foi criado (middleware.go:103).

**Vetor:**

Handers ativos (`validate`, `staSubmit`, `resolveRadarAlert`,
`triggerRadarScan`, `crossdocValidate`) chamam `r.Header.Get("X-IF-ID")`
diretamente, IGNORANDO o context.Claims populado pelo middleware.

Em produção (`RADIANT_JWT_PUBLIC_KEY` setado, `RADIANT_DEV_AUTH` off):

1. Cliente envia `Authorization: Bearer <jwt válido, sub=demo>`
2. Middleware valida JWT, popula Claims.IFID="demo"
3. Handler chama `r.Header.Get("X-IF-ID")` → retorna ""
4. Handler retorna 401 "X-IF-ID required"

**Resultado:** sistema JWT-only não funciona — todos os 5 endpoints
quebram em produção. Tests passam porque `RADIANT_DEV_AUTH=1` está setado
em todos os tests (testutil pattern legacy de X-IF-ID).

**Mais grave (cross-tenant se auth falhar):** Se cliente injeta
X-IF-ID customizado em conjunto com JWT válido para outro tenant:
- JWT Claims.IFID="attacker"
- Header X-IF-ID="victim_if_id"
- Handler usa `X-IF-ID` direto → audit log registra como "victim_if_id"
- Defense-in-depth quebrada: o vetor existe se app sub-usar o header.

**Fix:** Helper `getIfID(r *http.Request) string` em server.go que
prioriza Claims (JWT) e fallback para X-IF-ID header (dev mode legado
em tests).

```go
func getIfID(r *http.Request) string {
    if claims, err := auth.ClaimsFromContext(r.Context()); err == nil && claims != nil {
        return claims.IFID
    }
    return r.Header.Get("X-IF-ID")
}
```

Substituir 5 callsites. Tests passam com dev mode porque Claims fallback.

## 🔴 F27.2 — `/healthz` retorna `"version":"1.5.0"` em vez de `2.0.0`

**Severidade:** 🔴 ALTA — doc drift visível em produção

**Localização:** `backend/internal/version/version.go:28`

```go
const Version = "1.5.0"
```

**Constatação:**

- Código diz 1.5.0 (linha 28 do package version)
- CHANGELOG diz v2.0.0 + v2.0.0.post
- SPRINT_7_RESULTS.md diz v2.0.0
- OpenAPI `info.version` está em "2.0.0"
- Dockerfile NÃO usa ldflags para override
- Tag git está em v2.0.0 (`9ce7880`)

Resultado: `/healthz` (e `/readyz`) reportam `1.5.0` —
handshake version negotiation falha em consumers que checam.

**Fix:**

```go
const Version = "2.0.0"
```

E aplicar ldflags no Dockerfile:
```dockerfile
ARG VERSION=2.0.0
RUN ... -ldflags "-s -w -X 'github.com/fortvna/radiant-norma/backend/internal/version.Version=${VERSION}'"
```

**Atualizar também:** exemplo `version: 1.6.0` na OpenAPI schema
`HealthStatus.version` (linha 330) para "2.0.0".

## 🟡 F27.3 — Frontend chama `/v1/dashboard/envios` que não existe

**Severidade:** 🟡 MÉDIA — quebra silenciosa em runtime

**Localização:** `frontend/src/app/page.tsx:25`

```ts
apiFetch<{ total: number }>('/v1/dashboard/envios', {}, session.token)
```

**Contexto:**

OpenAPI documenta 14 endpoints. Nenhum é `/v1/dashboard/envios`.
Backend atualmente expõe stats apenas via `/v1/radar/alerts` (mas não
`/v1/radar/summary`).

**Impacto:** ao logar no frontend, dashboard quebra em runtime com
erro 404 silencioso. `apiFetch` joga `throw new Error(...)` que não
é capturado pelo `Promise.all` — unhandled promise rejection.

**Fix:** Implementar `GET /v1/dashboard/stats` no backend OU degradar
graciosamente no frontend (`stats = null` em caso de erro).

**Decisão:** Sprint 8 implementa endpoint real. Sprint 7+post adota
fallback gracioso para não bloquear demo.

## 🟡 F27.4 — Axios client interceptor busca cookie httpOnly (no-op)

**Severidade:** 🟡 MÉDIA — código morto

**Localização:** `frontend/src/lib/api.ts:23-29`

```ts
api.interceptors.request.use((config) => {
  const token = getCookie('rn_jwt')  // ← httpOnly: false no doc.cookie
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
})
```

**Vetor:**

`rn_jwt` é `httpOnly: true` (login/route.ts:62-66) → JavaScript browser
NÃO consegue ler `document.cookie`. Interceptor é silenciosamente no-op
em produção quando o cookie é necessário para authorization.

Em dev (`httpOnly` ainda é true), o cookie httpOnly fica disponível
apenas para o proxy server-side `/v1-api/proxy/[...path]/route.ts` que
usa `next/headers cookies()`.

**Fix:** Remover interceptor de Authorization (Axios client não consegue
ler de qualquer jeito). Server-side proxy é a única ponte de auth.

## 🟡 F27.5 — ResolveButton client-side envia `Bearer undefined`

**Severidade:** 🟡 MÉDIA — header inválido

**Localização:** `frontend/src/components/resolve-alert-button.tsx:13-19`

```ts
const token = document.cookie
  .split('; ')
  .find((c) => c.startsWith('rn_jwt='))
  ?.split('=')[1]
const r = await fetch(`/v1-api/proxy/radar/alerts/${id}/resolve`, {
  method: 'POST',
  headers: token ? { Authorization: `Bearer ${token}` } : {},
})
```

**Vetor:**

Mesmo problema F27.4: cookie httpOnly → `document.cookie` retorna
undefined → `token` é undefined → header NÃO enviado.

Mas se um dia mudarmos `httpOnly: false` (não recomendável), o header
vai como `Bearer undefined` (string literal) — backend vai parsear e
rejeitar por formato inválido.

**Fix:** Remover lógica de Authorization do client. Server-side proxy
injeta automaticamente.

## 🟡 F27.6 — Frontend sem ESLint configurado

**Severidade:** 🟡 MÉDIA — `npm run lint` falha

**Localização:** `frontend/`

**Constatação:**

SPRINT_7_RESULTS.md diz:
> ✓ TypeScript strict mode → clean

Mas `npm run lint` foi rodado interativamente pedindo setup. Não há
`.eslintrc.json` nem `eslint.config.js`. Sem lint configurado, regressões
de qualidade passam silenciosas.

**Fix:** Adicionar `.eslintrc.json` com Next.js Strict config:

```json
{
  "extends": ["next/core-web-vitals"]
}
```

## 🟢 F27.7 — Doc drift: contagem de arquivos frontend

**Severidade:** 🟢 BAIXA — cosmético

**Constatação:**

- Real: 17 arquivos .ts/.tsx em `frontend/src/`
- CHANGELOG v2.0.0 diz "19 arquivos TypeScript"
- SPRINT_7_RESULTS diz "19 arquivos .ts/.tsx novos"
- VALIDATION_v2.0.0.md diz "19 arquivos .ts/.tsx"

Off-by-2 (provavelmente esqueceram de contar `tailwind.config.ts` e
`next-env.d.ts` no original, ou contar coisas que foram integradas).

**Fix:** Atualizar docs para 17.

## 🟢 F27.8 — Doc drift: LOC frontend

**Severidade:** 🟢 BAIXA — cosmético

**Constatação:**

- Real: `find frontend/src -name "*.ts*" | xargs wc -l` = 953 LOC
  (incluindo configs e CSS)
  - Só .ts/.tsx: 892 LOC
- CHANGELOG: "~1100 LOC"
- SPRINT_7_RESULTS: "~1100 LOC"

**Fix:** Atualizar docs para ~950 LOC (~900 .ts/.tsx + ~50 configs).

## 🟢 F27.9 — Doc drift: postgres-setup.md path

**Severidade:** 🟢 BAIXA — link quebrado

**Localização:** `docs/postgres-setup.md` (raiz)

**Contexto:**

CHANGELOG v1.5.0 diz `backend/docs/postgres-setup.md`. Mas arquivo
real está em `docs/postgres-setup.md` (raiz do repo).

**Fix:** Corrigir path no CHANGELOG.

## 🟢 F27.10 — OpenAPI HealthStatus.version exemplo diverge

**Severidade:** 🟢 BAIXA — example inconsistente

**Localização:** `backend/docs/api/openapi.yaml:330`

```yaml
HealthStatus:
  properties:
    version: {type: string, example: "1.6.0"}
```

Mas o código retorna `"1.5.0"` (até fix do F27.2). E o OpenAPI `info.version`
diz "2.0.0".

**Fix:** Atualizar example para "2.0.0" junto com F27.2.

## 🟢 F27.11 — Cookies sem flag `secure` em prod

**Severidade:** 🟢 BAIXA — config de deployment

**Localização:** `frontend/src/app/api/login/route.ts:60-67`

```ts
response.cookies.set({
  name: 'rn_jwt',
  value: token,
  httpOnly: true,
  sameSite: 'lax',
  path: '/',
  maxAge: 7 * 24 * 60 * 60,
})
```

**Vetor:**

Sem `secure: true`, cookie pode vazar em HTTP (não HTTPS). Em dev local
(http) é OK. Em prod deve ser `secure: true` se servido em HTTPS.

**Fix:** Conditional baseada em NODE_ENV:

```ts
secure: process.env.NODE_ENV === 'production',
```

## 🟢 F27.12 — `next.config.js` typedRoutes em experimental

**Severidade:** 🟢 BAIXA — vai quebrar upgrade Next 15

**Localização:** `frontend/next.config.js:4-6`

```js
experimental: {
  typedRoutes: true,
},
```

TypedRoutes em experimental foi marcado estável no Next 14.2.x mas está
deprecated em Next 15 (move para top-level). Não quebra agora, mas
próximo upgrade precisa migrar.

**Fix:** Manter como está (Next 14 atual). Documentar para Sprint X.

## 🟢 F27.13 — Login dev expõe role admin sem gate

**Severidade:** 🟢 BAIXA — dev-only, mas vetor se flag vaza

**Localização:** `frontend/src/app/login/page.tsx:11-15`

```ts
const DEMO_IFS = [
  { id: 'demo', role: 'if', label: 'Demo IF (SCD)' },
  { id: 'demo-banco', role: 'if', label: 'Demo Banco (BC)' },
  { id: 'demo-admin', role: 'admin', label: 'Demo Admin' },  // ← público
]
```

**Vetor:**

Em dev, `NEXT_PUBLIC_RADIANT_DEV_MODE=1` → qualquer um pode logar como
admin. Se flag vaza para prod (via `NEXT_PUBLIC_*`), qualquer visitante
vira admin. Mitigação: server-side `/api/login` deve recusar role=admin
em prod.

**Fix:** Backend `/api/login` (se reescrito) deve validar se role=admin
é permitido em prod. Hoje o frontend decide sozinho.

## 🟢 F27.14 — Anti-patterns menores

**Severidade:** 🟢 BAIXA — cosmetic

| Local | Anti-pattern | Fix |
|-------|--------------|-----|
| `frontend/src/lib/api-fetch.ts:39-41` | `const { cookies } = await import('next/headers')` (import dinâmico) | Mover import para topo do arquivo |
| `frontend/src/app/radar/page.tsx:90` | `import { ResolveButton } from '@/components/...'` no final do arquivo | Mover para topo (junto com outros imports) |
| `frontend/src/app/login/page.tsx:75` | `<p>Sprint 7c (v2.0.0)</p>` no login | Atualizar para "Sprint 7 completo" |

## 📊 Stats consolidadas da validada

```
Sprint 7 (a/b/c) + v2.0.0.post — entregou:
  ✓ JWT RS256 auth (Sprint 7a) — funciona em dev mode (RADIANT_DEV_AUTH=1)
  ✗ JWT-only em prod — handlers não leem context.Claims (F27.1) [CRÍTICO]
  ✓ 60 regras 3040 tipadas + 5 raw
  ✓ Frontend Next.js 14 dashboard — 6 páginas, 4 funcionais
  ⚠️ 2 páginas (envios, auditoria) — placeholders reais (F27.8 Sprint 8)
  ✓ OpenAPI 3.0 spec — 14 endpoints
  ✗ Backend version const hardcoded 1.5.0 (F27.2) [CRÍTICO]
  ✓ 301 backend tests passing
  ✓ -race clean

Findings count by sprint:
  Sprint 6 (v1.5.0 v15-v23): 70 findings (18 críticos) [histórico]
  Sprint 7 (v24-v26):        5 findings (1 crítico F24.1)
  Sprint 7c post (v27):      12 findings (2 críticos F27.1/F27.2)

Doc drift tracks:
  - LOC count (Sprint 7c): 1100 → 953 real
  - File count (Sprint 7c): 19 → 17 real
  - Path postgres-setup: backend/docs/ → docs/
  - Version const: 1.5.0 (não bumpou)
```

## 🎯 Decisões aplicadas nesta validação

1. **F27.1 corrigido** antes de declarar Sprint 7 officially closed em
   produção. Sem isso, deployment real quebra todos os 5 endpoints.
2. **F27.2 corrigido** antes do CHANGELOG notar a v2.0.0.post.
3. **F27.4 / F27.5 corrigidos** removendo código morto (Axios interceptor
   + Bearer undefined injection).
4. **F27.6 adicionado ESLint Strict** para previnir regressões futuras.
5. **F27.7-F27.10 docs sincronizados** com real-state.

## 🚀 Carry-over para Sprint 8

| # | Gap | Origem | Sprint 8? |
|---|-----|--------|-----------|
| GAP-7.3 | Postgres integration tests | Sprint 7 | ✅ Sprint 8 |
| GAP-7.6 | Cross-doc engine goroutine pool limit | Sprint 6 | ✅ Sprint 8 |
| GAP-7.10 | RequestID propagation logs | Sprint 7 | ✅ Sprint 8 |
| GAP-7.13 | Tenant context isolation tests | Sprint 7b | ✅ Sprint 8 |
| GAP-7.15 | Frontend tests E2E (Playwright) | Sprint 7c | ✅ Sprint 8+ |
| GAP-7.16 | CI/CD npm install + frontend docker | Sprint 7c | ✅ Sprint 8 |
| GAP-7.17 | Backend endpoints envios/audit/dashboard | Sprint 7c | ✅ Sprint 8 |
| GAP-7.18 | Tenant isolation tests cross-tenant | v27 | ✅ Sprint 8 |
| GAP-7.19 | Frontend E2E real auth (JWT bridge) | Sprint 7c | ✅ Sprint 8 |

## 📚 Referências

- 27ª validação profunda — esta sessão
- 26ª validação (frontend) — `VALIDATION_v2.0.0.md`
- 25ª validação (regras) — `VALIDATION_v1.7.0.md`
- 24ª validação (auth) — `VALIDATION_v1.6.0.md`
- v1.4.x validações 7-23 — `VALIDATION_v1.5.0_DEEPEST*.md`
