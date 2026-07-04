# Sprint 7 — Auth JWT + Regras 3040 + Frontend Norma Console

> **Data:** 2026-07-04 (iniciada após PUSH Sprint 6 Fase 1 + tag v1.5.0)
> **Status:** 📋 Planejamento — aguardando decisão de escopo
> **Tema:** Substituir auth X-IF-ID por JWT real + escalar regras 3040 +
> bootstrap do Frontend Norma Console (Next.js).
> **Versão alvo:** v1.6.0 (JWT), v1.7.0 (Regras), v2.0.0 (Frontend)
> **Trigger:** Sprint 6 v1.5.0 estável + 13 validações pós-release +
> saturação confirmada (compound-validation saturation pattern).
> **Decisão:** Escopo pragmático — começa com v1.6.0 (JWT) e queue
> subsequente. Cada sprint mantém invariants:
>       - testes passing (race-clean, vet-clean)
>       - CHANGELOG atualizado
>       - migration path backward-compatible
>       - pelo menos 1 validação profunda no fecho.

## 🎯 Por que Sprint 7 AGORA

**Sprint 6 saturado:** v15-v23 fechou 70 findings, 18 críticos.
0 críticos em v22-v23. Codebase convergindo. Mover para trabalho
que entrega valor ao usuário, não mais hardening defensivo.

**3 features pedidas por Henrique (comprimento-de-tendão):**

| Feature | Escopo | Release |
|---------|--------|---------|
| **GAP-7.8 Frontend Norma Console (Next.js)** | Frontend dashboard para IFs verem validações, alertas radar, stats. Projeto separado. | **v2.0.0** |
| **GAP-7.9 Mais regras 3040 (~320 total, ~25 hoje)** | Backend: 25 → 100 regras tipadas + raw. Tests fuzzer para XML pathology. | **v1.7.0** |
| **Auth JWT real (substitui X-IF-ID)** | Backend: JWT middleware (`Authorization: Bearer <jwt>`) + claims (sub, if_id, role). X-IF-ID deprecated em dev. | **v1.6.0** |

Por que essa ordem:
1. **JWT primeiro** — afeta API surface, todas as features futuras
   constroem em cima.
2. **Regras depois** — JWT + tenant context estabiliza antes de
   escalar regra que precisa de tenant differentiation.
3. **Frontend por último** — depende de API estável. v1.6.0 e
   v1.7.0 entregam backend completo antes do frontend consumir.

---

## 🔷 Sprint 7a (v1.6.0) — Auth JWT Real

**Duration estimate:** 1-2 semanas (com 1 validação profunda)

### Mudanças

1. **Novo pacote `internal/auth`**:
   - `JWTVerifier` com validação via chave pública (RS256/Ed25519)
   - `Claims { sub, if_id, role, exp, iat, iss }`
   - Key rotation policy (rotate a cada 90d, ver cadence)
   - Issuer pinning (rejeita tokens não-our-issuer)

2. **Middleware `jwtMiddleware`** (substitui `authMiddleware`):
   - Header: `Authorization: Bearer <token>`
   - Claim `if_id` substitui `X-IF-ID` no audit log + DB columns
   - Reject role mismatch (e.g. admin para endpoint non-admin)
   - Constant-time compare para HMAC/HMAC-related

3. **Dev convenience**:
   - `X-IF-ID` ainda aceito em modo dev (env var `RADIANT_DEV_AUTH=1`)
   - CLI `cmd/jwt-mint` para gerar tokens de dev (sem secret prod)
   - Migration helper: rodar `cmd/migrate-jwt` se DB tem X-IF-ID sem
     mapping para user

4. **Tests**:
   - Verify token expired → 401
   - Verify token signed by wrong issuer → 401
   - Verify role mismatch → 403
   - Verify constant-time compare (timing attack)
   - Verify concurrent (token rotation sem race)

### Compatibilidade / Migration path

- **Default:** JWT obrigatório (X-IF-ID → 401)
- **Dev override:** `RADIANT_DEV_AUTH=1` aceita X-IF-ID
- **Production:** quebra clientes que ainda usam X-IF-ID.
  **Sprint 7a emite WARN nos logs** se detectar requests com
  X-IF-ID por 30 dias antes de deprecar completamente.

### Validation approach

- `VALIDATION_v1.6.0.md` documentando features
- 1 validação profunda obrigatória no fecho (validate_jwt_*,
  test middleware integration, test timing-attack resistance,
  test rotation policy)

---

## 🔷 Sprint 7b (v1.7.0) — Regras 3040 (25 → 100)

**Duration estimate:** 2-3 semanas

### Mudanças

1. **Regras B06-B30** (raw registry, +9 rules com testes):
   - B06-B15: regras estruturais simples (5 regras)
   - B16-B25: regras de agregação (10 regras)
   - B26-B30: regras cross-cadoc (§+10)

2. **Regras tipadas F-rules** (25 → 60):
   - F-rules em estrutura hierárquica: F01 (Mês inválido),
     F02 (campo obrigatório), F03 (formato), etc.
   - **Target:** 60 regras tipadas no total (incluindo os 25
     atuais).

3. **Cruzando o limiar 100**: target = 100 regras total no Registry.
   Architecture dual-registry (tipada + raw) suporta ambos.

4. **Fuzz testing** (resolvendo GAP-7.2 parcial):
   - go test -fuzz sobre `iterXMLElements`
   - XML pathology: CDATA, entities, malformed UTF-8
   - Regressão coverage XML real BACEN

5. **Documentation**:
   - `docs/rules-3040-catalog.md` — listagem completa de regras
   - Cada regra tem: código, descrição, severidade, exemplos

### Compatibilidade

- Não breaking — adicionar regras é aditivo
- Tests existentes devem continuar passando
- Registry API estável

### Validation approach

- Mutation test do Registry (registry tests não cobrem todas
  as regras — cobrir via codegen template? ou hand-write?)
- 1 validação profunda no fecho (mutation tests + fuzz coverage)

---

## 🔷 Sprint 7c (v2.0.0) — Frontend Norma Console

**Duration estimate:** 3-4 semanas
**Repo:** Projeto separado `radiant-norma-console/` (Next.js 14)

### Stack decisions

- **Framework:** Next.js 14 (App Router)
- **Styling:** TailwindCSS 3 + shadcn/ui (componentes copy-paste)
- **Auth:** NextAuth.js with JWT bridge to radiant-norma backend
- **State:** TanStack Query (server state) + Zustand (local)
- **Charts:** Recharts (já usado em BCA dashboards)
- **Forms:** react-hook-form + zod

### Páginas MVP

- `/` Dashboard: métricas agregadas (envios/dia, taxa
  passing, alertas radar ativas)
- `/envios` Lista de envios STA com filtros (status, IF, CADOC,
  período), batch retry, audit log inline
- `/radar/alertas` Lista de alertas regulatórios com severity
  filter e "snooze for 7 days" (similar a GitHub)
- `/regras` Catálogo de regras 3040 com toggle enable/disable
  (multi-tenant)
- `/auditoria` View-only audit log (LGPD report)
- `/login` Login com X-IF-ID ou JWT dev

### Backend integration

- Reusa `internal/api` API (sem duplicação)
- NextAuth.js provider custom que valida JWT contra
  `internal/auth` (backend-issued)
- Server components para data fetching direto do backend
  (sem intermediário frontend-only)

### Security considerations

- CSRF token (Next.js default)
- CSP header (Google Fonts CDN whitelist)
- Rate limit (`next-rate-limit` ou middleware custom)
- No client-side encryption de secrets (zero trust)

### Repo setup (estimado)

```
radiant-norma-console/         # nova repo
├── app/                      # Next.js App Router
├── components/              # shadcn/ui + custom
├── lib/                     # API client (OpenAPI gen)
├── server/                  # NextAuth.js + JWT bridge
├── public/
├── tailwind.config.ts
├── package.json
├── tsconfig.json
└── README.md
```

### Validation approach

Frontend tem validation orthogonal ao backend. Vou aplicar
princípios cross-project memory: validar DEPOIS de MVP funcional,
não theoretical.

---

## 🔷 Backlog consolidado (Sprint 7 final)

Após Sprint 7a/b/c, GAPs remanescentes:

| # | Gap | Status |
|---|-----|--------|
| GAP-7.1 | Cross-doc L3 — iterXMLElements caseira | Mitigado v7b (fuzz test) |
| GAP-7.2 | Cross-doc L3 — CDATA/entity | Mitigado v7b (fuzz test) |
| GAP-7.3 | Postgres integration tests | Follow-up (Sprint 8) |
| GAP-7.4 | ~~User-Agent hardcoded~~ | ✅ v18 |
| GAP-7.5 | ~~INSERT OR IGNORE~~ | ✅ v21 refutado |
| GAP-7.6 | Cross-doc engine goroutine pool limit | Follow-up Sprint 8 |
| GAP-7.10 | RequestID propagation logs | Follow-up Sprint 8 |
| GAP-7.11 | cmd/_verify dev tool | ✅ v21 |
| NEW GAP-7.12 | JWT refresh token rotation | Sprint 7a edge case |
| NEW GAP-7.13 | Tenant context isolation tests | Sprint 7b feature |

---

## 📂 Arquivos Sprint 7a (v1.6.0)

### Criar (novos)

- `backend/internal/auth/jwt.go` — JWT verification
- `backend/internal/auth/claims.go` — Claims struct + validation
- `backend/internal/auth/middleware.go` — chi middleware
- `backend/internal/auth/keyring.go` — key rotation
- `backend/cmd/jwt-mint/main.go` — dev tool
- `backend/internal/auth/jwt_test.go` — tests

### Modificar

- `backend/internal/api/server.go` — middleware order change
- `backend/cmd/api/main.go` — JWT setup + dev flag
- `backend/CHANGELOG.md` — v1.6.0 entry
- `backend/SPRINT_7_RESULTS.md` — synthesis

### Migration path

```sql
-- 001 NÃO PENDENTE: tenant_id agora vem do JWT claim, não X-IF-ID
-- Nenhuma migration DB necessária.
-- IF-IDs existentes no DB continuam compatíveis.
```

---

## 🪜 Como começar Sprint 7a AGORA

1. **Criar `internal/auth/jwt.go`** com `Claims` struct + `Verify(token)`
2. **Test first:** escrever `jwt_test.go` com vetores:
   - valid token, expired, malformed, wrong issuer, wrong sig
3. **Middleware chi** (`middleware.go`):
   - constant-time compare
   - claims extraction
   - role validation
4. **Integration test:** server.go end-to-end com JWT real
5. **Dev tool** (`cmd/jwt-mint/main.go`):
   - Hardcoded dev key
   - Uso: `go run ./cmd/jwt-mint --if=demo --role=admin`

### Decisões em aberto

- **Algoritmo:** RS256 (assimetrico) vs Ed25519 vs HMAC?
  - RS256: standard, broad tooling, PKI infra requise
  - Ed25519: moderno, rápido, mas tooling-limited
  - HMAC: simples, mas 1 key serves both sign + verify
  - **Recomendação:** RS256 (broad compat, AWS Cognito/GCP
    compat)
- **Key storage:** Vault, AWS KMS, ou env var?
  - Sprint 7a: env var (start small)
  - Sprint 8: AWS KMS integration
- **Issuance:** backend-issued only ou aceitar 3rd-party?
  - Backend-issued only no Sprint 7a
  - 3rd-party OIDC em Sprint 7a+1b (forward)

---

## ✅ Resumo executivo

**Status:** 📋 Planejamento completo. Pronto pra Sprint 7a (JWT)
imediatamente após Henrique confirmar.

**3 features ambiciosas** com escopo realista:

- Sprint 7a (v1.6.0) — JWT real, 1-2 semanas
- Sprint 7b (v1.7.0) — Regras 3040 (25 → 100), 2-3 semanas
- Sprint 7c (v2.0.0) — Frontend Next.js, 3-4 semanas

**Timing total:** ~6-9 semanas se sequencial.

**Confirmação:** vou começar Sprint 7a agora?

Recomendo começar pelo **Sprint 7a (v1.6.0 — JWT real)** porque:
1. É a fundação para as demais (tenant context auth)
2. É a feature mais isolada (1-2 semanas vs 2-3 vs 3-4)
3. Tem 1 validação profunda natural no fecho

Concordas? Ou quer outro caminho?
