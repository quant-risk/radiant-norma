# Sprint 8 — Auth wire-up real + Endpoints complementares + Test isolation

> **Status:** 📋 Proposta (consolidar para início após validação 27 fechar)
> **Data proposta:** 2026-07-04 (em sequência a Sprint 7c + validação 27)
> **Tema:** JWT bridge real, endpoints complementares (envios/audit/dashboard), tenant isolation
> **Trigger:** Gaps remanescentes de Sprint 7a/b/c + findings críticos F27.1/F27.2 fechados em validação 27

## 🎯 Objetivo

Sprint 7 entregou backend (JWT + 60 regras) e frontend Next.js, mas deixou
gaps funcionais que impedem produção real:

1. **JWT bridge real frontend↔backend**: hoje `rn_jwt` é cookie `dev:<if_id>:<role>`
   string. Backend rejeita em prod. Precisa gerar JWT RS256 real no /api/login
   com chave privada do backend.
2. **Endpoints complementares**: frontend chama `/v1/dashboard/envios`,
   `/v1/envios`, `/v1/audit_log` que não existem no backend. Sprint 8 adiciona.
3. **Tenant isolation tests**: validar F27.1 com cenários cross-tenant reais
   (atualmente só testamos helper isolado).
4. **CI/CD**: pipeline .github/workflows para rodar `go test`, `npm run lint`,
   `npm run build` em PRs.

## 🏛️ Entregas propostas (4 frentes em sequência)

### 🔴 Frente 1 — JWT bridge real (Sprint 8a → v2.1.0)

**Problema atual:** `/api/login` retorna `dev:<if_id>:<role>` string opaca.
Backend JWT verifier não consegue validar (string não é JWT real).

**Fix:**

1. `/api/login` (Next route) chama backend `POST /v1/auth/dev-token` (NOVO).
   - Backend recebe if_id + role.
   - Backend gera JWT RS256 usando private key local (em dev `cmd/jwt-mint`
     já faz; reutilizar).
   - Backend retorna JWT string + if_id + role + expires_at.
2. Frontend armazena JWT real em cookie `rn_jwt` httpOnly.
3. Server proxy `/v1-api/proxy/[...path]` repassa JWT no Authorization header.
4. Backend JWT verifier aceita. Tenant isolation OK.

**Novos endpoints:**
- `POST /v1/auth/dev-token` — dev only (RADIANT_DEV_AUTH=1 necessário).
  Recebe `{if_id, role, ttl}`. Retorna `{token, expires_at}`. Só `admin role`
  consegue gerar (defesa em profundidade).
- `POST /v1/auth/refresh` — refresh token rotation (GAP-7.12 backlog).
  Recebe token válido perto de expiry. Retorna novo.

**Tests:** 6 testes novos — dev-token geração, role-based access, refresh
flow, expiry check.

### 🟡 Frente 2 — Endpoints complementares (Sprint 8b → v2.1.0)

**Problema atual:** Frontend chama 3 endpoints que não existem:
- `GET /v1/dashboard/stats` (page.tsx:25-26)
- `GET /v1/envios` (placeholder, mas UI esperando)
- `GET /v1/audit_log` (placeholder, mas UI esperando)

**Fix — 3 endpoints:**

#### `GET /v1/dashboard/stats` (admin ou self-only)
```json
{
  "envios_total": 123,
  "envios_aprovados_24h": 12,
  "envios_rejeitados_24h": 3,
  "alertas_ativas": 7,
  "regras_ativas": 60,
  "ultimo_envio_at": "2026-07-04T..."
}
```
Backend lê tabela `envios` + `audit_log` + cache de cadocs.

#### `GET /v1/envios?if_id=X&status=Y&cadoc=Z&limit=50&offset=0`
```json
{
  "envios": [
    {"id": "...", "if_id": "demo", "cadoc_code": "3040",
     "data_base": "2026-06", "status": "approved",
     "sent_at": "...", "protocol_sta": "STA-..."}
  ],
  "total": 245,
  "limit": 50,
  "offset": 0
}
```
Backend lê tabela `envios` com multi-tenant filter.

#### `GET /v1/audit_log?if_id=X&action=Y&from=Z&to=W&admin=true`
```json
{
  "entries": [
    {"id": 1234, "actor": "...", "action": "cadoc.validated",
     "target": "3040", "if_id": "demo", "metadata": {...},
     "prev_hash": "...", "entry_hash": "...",
     "created_at": "..."}
  ],
  "total": 5680
}
```
Backend lê `audit_log` com paginação. **Apenas role=admin.**

**Tests:** ~15 testes — happy paths, filtros, paginação, role check.

### 🟢 Frente 3 — Tenant isolation (Sprint 8c → v2.1.0)

**Problema atual:** F27.1 fechou o vetor (helper getIfID prioriza Claims),
mas só temos 4 testes unitários. Falta teste end-to-end que prova:
- Cliente A (JWT tenant A) NÃO consegue ver dados de tenant B.
- Audit log emitido por tenant A grava como "A", nunca como "B" mesmo
  com header X-IF-ID malicioso.

**Fix:**

1. Setup de test que sobe 2 JWT tokens (tenant A, tenant B).
2. Suite de testes cross-tenant:
   - `TestCrossTenant_DashboardStats`: A vê só envios de A.
   - `TestCrossTenant_EnviosList`: A vê só envios de A.
   - `TestCrossTenant_AuditLog`: A não acessa audit log (precisa admin),
     admin vê entries de TODOS os tenants.
   - `TestCrossTenant_XIFIDHeaderInjettion`: A envia JWT válido + header
     X-IF-ID=B. Verifica que audit_log grava if_id=A (do JWT).

**~6 testes novos.** Complementa F27.1 com cobertura end-to-end.

### 🔵 Frente 4 — CI/CD pipeline (Sprint 8d → v2.2.0)

**Problema atual:** Sem pipeline. Cada validação é manual.

**Fix:** `.github/workflows/ci.yml` com jobs:
- `backend-test`: go build + go test -race + go vet + gofmt
- `frontend-test`: npm ci + npm run lint + npm run type-check + npm run build
- `lint`: golangci-lint + eslint strict
- `docker-build`: docker build do backend (smoke)
- Trigger: PR open, push to main

**Adicionar:** `Makefile` target `make ci` que orquestra tudo.

## 📊 Estimativa de tamanho

```
Sprint 8a (JWT bridge real):  ~200 LOC backend + 30 LOC frontend + 6 tests
Sprint 8b (3 endpoints):      ~300 LOC backend + ~100 LOC frontend
                                 + 15 tests
Sprint 8c (Tenant isolation): ~150 LOC tests (config helper + fixtures)
Sprint 8d (CI/CD):            ~80 LOC YAML + ~30 LOC Makefile
─────────────────────────────────────────────────────────────
Total:                         ~860 LOC + ~21 tests novos
```

## 🎯 Critérios de aceite (propostos)

### Sprint 8a ✅ 4/4
- /api/login chama backend dev-token
- Cookie rn_jwt vira JWT RS256 real
- Server proxy funciona com JWT real
- 6 testes cobrindo fluxo end-to-end

### Sprint 8b ✅ 4/4
- /v1/dashboard/stats retorna shape documentado
- /v1/envios com filtros + paginação
- /v1/audit_log com role check (admin only)
- 15 testes cobrindo cada combinação

### Sprint 8c ✅ 3/3
- TestCrossTenant_* suite (6 testes)
- Helper de test setup com 2 JWT tokens
- Regressão F27.1 coberta end-to-end

### Sprint 8d ✅ 3/3
- GitHub Actions workflow
- Makefile `make ci` local
- docker-compose para test isolation

## 🚦 Decisões pendentes

1. **JWT bridge**: queremos chamada síncrona ao backend `dev-token`
   ou geração local de JWT no frontend usando chave pública? Sync é
   mais simples mas adiciona latência. Local é mais rápido mas cria
   superfície de ataque.
2. **Audit log pagination**: default 50 ou cursor-based? Cursor mais
   resiliente em datasets grandes.
3. **CI runner**: ubuntu-latest ou macos-latest (já que desenvolvemos
   em mac)? Ubuntu é mais barato.

## 📚 Carry-over (para Sprint 9+)

- GAP-7.3: Postgres integration tests (testcontainers)
- GAP-7.6: Cross-doc engine goroutine pool limit
- GAP-7.10: RequestID propagation logs
- GAP-7.15: Frontend E2E tests Playwright
- GAP-7.16: Frontend Dockerfile multi-stage
- Frontend typedRoutes migration para Next 15
- IdP real (Keycloak/Okta) integração
