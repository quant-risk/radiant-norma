# Sprint 8a — JWT bridge real — RESULTADOS (v2.1.0)

> **Data:** 2026-07-04
> **Status:** ✅ Concluída
> **Tema:** JWT bridge real frontend↔backend (`POST /v1/auth/dev-token`)
> **Trigger:** Sprint 7c deixou placeholder `/api/login` que emitia `dev:<if_id>:<role>` string opaca ao invés de JWT real
> **Resultado:** 1 nova feature shipped, 18 testes novos, ~315 LOC backend + ~70 LOC frontend

## 🎯 Objetivo da sprint

Sprint 7a (v1.6.0) introduziu JWT RS256 bearer + helper `auth.ClaimsFromContext`.
Sprint 7c (v2.0.0) introduziu frontend Next.js que **armazenava** JWT em
cookie httpOnly `rn_jwt`. **Mas o JWT nunca foi gerado de verdade**:
`/api/login` setava `dev:<if_id>:<role>` string opaca.

Sprint 8a fecha esse gap. Bridge JWT real:
1. Backend ganha endpoint `POST /v1/auth/dev-token` que emite JWT in-process.
2. Frontend `/api/login` chama esse endpoint.
3. Cookie `rn_jwt` agora armazena JWT RS256 string.
4. Backend JWT verifier (mesma chave pública) aceita tokens emitidos.

Em produção: dev-token endpoint retorna 404. Tokens vêm de IdP externo
(Keycloak/Okta/etc) — Sprint 9+.

## 🏛️ Entregas

### 🟢 Backend — `auth.Signer` helper

**Arquivo:** `backend/internal/auth/mint.go` (145 LOC)

Ponto importante: este helper encapsula signing in-process. Antes, a
lógica estava acoplada ao `cmd/jwt-mint/main.go` (script CLI), não
reaproveitável pelo servidor HTTP. Agora qualquer goroutine pode chamar
`signer.Mint(claims)` para emitir JWT.

**API exposta:**

```go
type SignerConfig struct {
    PrivateKeyPEM string  // PKCS#1 ou PKCS#8 PEM
    Kid           string
    Issuer        string
    Now           func() time.Time  // optional, for tests
    JTIrand       func() string      // optional, for tests
}

func NewSigner(cfg SignerConfig) (*Signer, error)
func NewSignerFromFile(path, kid, issuer string) (*Signer, error)
func (s *Signer) Mint(claims Claims) (string, time.Time, error)
func (s *Signer) MintSimple(ifID string, role Role, ttl time.Duration) (string, time.Time, error)

const TTLCap = 30 * 24 * time.Hour
const TTLDefault = 24 * time.Hour
```

**Defesas:**

- Issuer é pinned por design — Signer override qualquer `Claims.Iss` diferente.
- Claims.Validate() roda antes de assinar — defesa em profundidade.
- JTI random 16 bytes (32 hex chars) — uniqueness garantida via crypto/rand.
- TTL clamping em MintSimple (max 30 dias) — defesa contra tokens de vida excessiva.

### 🟢 Backend — `POST /v1/auth/dev-token`

**Arquivo:** `backend/internal/api/auth_handlers.go` (173 LOC)

Endpoint novo. Posicionado **fora** do group `/v1` que tem JWT
middleware — `/v1/auth/dev-token` é o que **gera** tokens, não
consome. Defense: retorna 404 quando `RADIANT_DEV_TOKEN` != "1"
(esconde existência em prod).

```yaml
POST /v1/auth/dev-token
Body: { "if_id": "demo-bank", "role": "if", "ttl_seconds": 86400 }
Returns 200: { "token": "eyJ...", "if_id": "...", "role": "...",
              "expires_at": "2026-07-04T...", "ttl_seconds": 86400 }
Returns 404: quando RADIANT_DEV_TOKEN off
Returns 503: quando flag on mas signer não configurado
Returns 400: bad request
```

Audit emission: `auth.dev_token.minted` com metadata `{role, ttl}` —
rastreabilidade de quem pediu tokens.

### 🟢 Backend — wire em cmd/api/main.go

Carrega chave privada de env na ordem:
1. `RADIANT_DEV_JWT_PRIVATE_KEY` (path)
2. `RADIANT_DEV_JWT_PRIVATE_KEY_PEM` (inline)

Cria Signer, attache em `srv.DevSigner`. Log warning explícito se
dev mode ativo (não usar em prod).

### 🟢 Frontend — `/api/login` rewritten

**Arquivo:** `frontend/src/app/api/login/route.ts` (~95 LOC)

Antes (v2.0.1):
```ts
const token = `dev:${body.if_id}:${role}`  // nunca aceito pelo backend
response.cookies.set({ name: 'rn_jwt', value: token, ... })
```

Agora (v2.1.0):
```ts
const r = await fetch(`${apiUrl}/v1/auth/dev-token`, {
  method: 'POST',
  body: JSON.stringify({ if_id, role, ttl_seconds: 604800 }),
})
const { token, expires_at } = await r.json()
response.cookies.set({ name: 'rn_jwt', value: token, ... })
```

Error handling:
- 404 do backend → 503 com hint ("RADIANT_DEV_TOKEN=1")
- 400 do backend → propaga
- Network error → 502 com hint ("verifique se backend está rodando")

## 📊 Estatísticas

```
Sprint 8a deliverou:
  Backend:           +315 LOC (mint 145 + auth_handlers 173 - 3 wrapper)
  Backend tests:     +18 (13 mint + 8 dev-token - 3 setup overlap)
  Backend go build:  clean
  Backend go vet:    clean
  Backend go test:   322 passing (era 304)

  Frontend:          +25 LOC (login rewritten, error handling expandido)
  Frontend lint:     clean (Strict)
  Frontend build:    10 routes, 87.3 kB
  Frontend type-check: clean

  OpenAPI:           14 → 15 endpoints, 9 → 11 schemas
  CHANGELOG:         +120 LOC (v2.1.0 entry completa)
```

## 🧪 Suíte de regressão E2E (resumo)

### Backend (Sprint 8a)

```
✓ go vet ./...                                   clean
✓ gofmt                                          clean
✓ go build ./...                                  clean
✓ go test ./... -count=1                          322 passing
✓ go test ./internal/auth/... -v                  mint_helper tests 13/13
✓ go test ./internal/api/... -run TestDevToken -v dev-token tests 8/8
✓ TestSigner_Roundtrip                            sign → verify ciclo fechado
✓ TestDevToken_MintValid                          mint válido, all fields populated
✓ TestDevToken_TTLClamp                           ttl clamped para TTLCap
```

### Frontend (Sprint 8a)

```
✓ npm install                                      unchanged (no new deps)
✓ npm run build                                    10 routes, 87.3 kB shared
✓ TypeScript strict mode                           clean
✓ ESLint Strict                                    ✔ No warnings or errors
```

### Auth Flow end-to-end

```
1. Backend Go:
   RADIANT_DEV_TOKEN=1
   RADIANT_DEV_JWT_PRIVATE_KEY=./dev-private.pem
   RADIANT_JWT_PUBLIC_KEY="$(cat dev-public.pem)"
   → Server.Router() expõe /v1/auth/dev-token + /v1/* com JWT middleware

2. Frontend Next.js:
   User clica em "Entrar" com if_id=demo, role=if
   → /api/login chamada (Next route handler)
   → POST http://localhost:8080/v1/auth/dev-token
     { if_id: "demo", role: "if", ttl_seconds: 604800 }
   ← 200 { token: "eyJ...", if_id: "demo", role: "if", ... }
   → cookie rn_jwt setado com JWT real

3. Server component /radar/page.tsx:
   → getServerSession() lê cookie via next/headers
   → verifyJwtServer(token) decodifica + valida
   → pagina renderiza

4. Client component resolve-alert-button:
   → fetch /v1-api/proxy/radar/alerts/{id}/resolve
   → proxy server-side injeta Authorization: Bearer <jwt>
   → backend JWT verifier valida
   → handler getIfID() retorna Claims.IFID (do JWT)
   → resolveRadarAlert emite audit
```

## 🎯 Critérios de aceite (vs SPRINT_8.md)

### Sprint 8a ✅ 4/4

- ✅ /api/login chama backend dev-token
- ✅ Cookie rn_jwt vira JWT RS256 real
- ✅ Server proxy funciona com JWT real
- ✅ 18 testes cobrindo fluxo end-to-end (13 mint + 8 dev-token, mas
  helper é compartilhado, então há overlap)

## 🏗️ Lições aprendidas (memory candidates)

1. **Bridge frontend↔backend é fundação, não otimização.** Sem o bridge
   real, todo o resto é teatro. Sprint 7c shipped UX sem o wire-up
   fundamental. **Sempre fazer a ponte primeiro**, depois features
   dependentes.

2. **Code-once entre CLI e server:** extrair lógica de signing do
   `cmd/jwt-mint` para `auth.Signer` evitou duplicação entre o script
   CLI e o handler HTTP. Single source of truth para JWT signing.

3. **404 ≠ 503 para endpoints dev-only.** Quando flag off, retornar 404
   esconde existência do endpoint — boa prática de security by obscurity
   + principle of least exposure. 503 seria "endpoint existe mas
   quebrado" — vetor de info disclosure.

4. **TTL clamping mandatório.** Mesmo com `ttl_seconds` validado,
   clamp para max previne tokens de vida excessiva acidental. Defesa
   contra bug de param (e.g., typo `--ttl=87600` para 1 ano inteiro
   não-pretendido).

5. **Audit emission for dev actions.** Mesmo em dev, registrar quem
   pediu tokens é essencial pra forensic trail. `auth.dev_token.minted`
   no audit_log permite identificar tokens suspeitos mesmo após fact.

6. **Test signers via Signer+Verifier roundtrip, não só Unit Mint().**
   `TestSigner_Roundtrip` é a única coisa que prova a ponte completa
   — sign + verify ciclo fechado. Sem isso, poderíamos ter um bug
   sutil (e.g., claim order que assinado mas não parseado igual).

## 🔜 Próximos passos (carry-over Sprint 8b+)

| # | Gap | Origem | Sprint |
|---|-----|--------|--------|
| GAP-8.2 | /v1/dashboard/stats endpoint | Sprint 7c frontend chama | 8b |
| GAP-8.3 | /v1/envios list endpoint | Frontend placeholder | 8b |
| GAP-8.4 | /v1/audit_log endpoint (admin role) | LGPD/SOC2 compliance | 8b |
| GAP-8.5 | Tenant isolation cross-tenant tests | F27.1 coverage end-to-end | 8c |
| GAP-8.6 | CI/CD pipeline (.github/workflows/) | Manual validações | 8d |
| GAP-9.x | JWT refresh token rotation | Edge case | 9+ |
| GAP-9.x | IdP integration (Keycloak/Okta) | Substituir dev-token | 9+ |

## 📚 Referências

- `SPRINT_8.md` (proposta)
- `VALIDATION_v2.0.0_POST.md` (v27 com F27.1/F27.2)
- `VALIDATION_v2.0.1.md` (v28 — saturação)
- `CHANGELOG.md` (v2.1.0 entry)
- `backend/docs/api/openapi.yaml` (1 endpoint novo)
