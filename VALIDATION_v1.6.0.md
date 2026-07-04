# VALIDATION v1.6.0 — Sprint 7a (Auth JWT real)

> **Status:** ACCEPTED
> **Data:** 2026-07-03
> **Trigger:** Henrique pediu substituição do placeholder X-IF-ID por
> autenticação JWT real (Sprint 7a).
> **Versão:** v1.6.0 (minor — auth infra)

## 🎯 Resumo executivo

Substitui `X-IF-ID` placeholder (string trust) por **JWT bearer
RS256 com claims role/if_id/iss**. Migration preservada via
`RADIANT_DEV_AUTH=1` flag para clients legados durante janela de
adaptação.

**5 findings, 1 crítico (vetor original era CRÍTICO)**

1. **F24.1** 🔴 CRÍTICO — F23.2 (X-IF-ID max 64 chars) era **PLUG**:
   vetor real não era 10KB injection — era auth bypass total.
   Atacante tinha acesso a TODOS os endpoints multi-tenant sem
   saber nenhum secret. **Apply F24.1 = JWT real.**

2. **F24.2** 🟡 — Dev flag `RADIANT_DEV_AUTH=1` para migration
   (X-IF-ID fallback). Documentado em env var.

3. **F24.3** 🟡 — Key rotation grace policy implementado
   (Keyring.Rotate). Tokens antigos verify até expiry.

4. **F24.4** 🟢 — `cmd/jwt-mint` dev tool — gera tokens para test/demo.

5. **F24.5** 🟢 — Issuer pinning (iss claim obrigatório) previne
   tokens cross-tenant.

**Stats:**
- 253 → 270 tests passing (+17 sprint 7a)
- vet-clean, race-clean, build-clean

---

## 🔴 CRÍTICO (P0) — F24.1

### F24.1 — X-IF-ID era placeholder + F23.2 não era auth

**Severidade:** 🔴 CRÍTICO — auth bypass total

**Discovery via re-read:**

F23.2 (validação 23) fechou vetor de DOS-via-X-IF-ID-10KB
mitigando compression attack no audit_log. Mas a verdadeira
questão era: **o sistema NÃO tinha autenticação real**.

```go
// server.go antigo (F23 era assim):
func authMiddleware(next) http.Handler {
    return http.HandlerFunc(func(w, r) {
        if r.Header.Get("X-IF-ID") == "" {
            // 401 missing
        }
        // ⚠️ Apenas checa se ESTÁ VAZIO. Não valida formato/sign/issuer.
        next.ServeHTTP(w, r)
    })
}
```

Consequência:
- Atacante escolhe qualquer string como X-IF-ID.
- No audit log: `if_id="attacker-chosen-string"`.
- Envios STA: `INSERT INTO envios(if_id, ...)` com valor controlado.
- Cross-tenant: atacante cria IFs fantasma e usa seus IDs.

Vetor real confirmado. Em produção, esse sistema era PoC-only.

**Fix aplicado — JWT bearer RS256:**

```go
type Claims struct {
    Sub  string  `validate:"required"`
    IFID string  `validate:"required,max=64"`
    Role Role    `validate:"required,oneof=if admin readonly"`
    Iss  string  `validate:"required"`     // issuer pinning
    Exp  time.Time `validate:"required"`   // expiry (goland-jwt)
    Iat  time.Time
    Jti  string  `validate:"omitempty"`
}

type Verifier struct {
    config  Config
    keyring *Keyring
    parser  *jwt.Parser  // golang-jwt/v5 RS256 only
}

func (v *Verifier) Verify(tokenString string) (*Claims, error) {
    // 1. Parse — ValidateMethod(RS256), WithIssuer, WithExpirationRequired
    // 2. keyFunc lookup com kid — ausente = reject
    // 3. MapClaims → Claims struct
    // 4. claims.Validate() — checks runtime invariants
}
```

**Plus:**

- **Constant-time compare** built-in (jwt.Equal strings).
- **Issuer pinning** — regeita tokens com iss ≠ "radiant-norma".
- **Audience reserve** (config — pronta para Sprint 8).
- **Kid obrigatório** — kid vazio = reject (defense rotation).
- **Sign method whitelist** (`jwt.WithValidMethods(["RS256"])`) —
  HS256 attack rejected (Token Compromise).

**Vetores cobertos pelos tests:**
1. Token válido (happy path)
2. Expired → 401 (exp rejection)
3. Wrong issuer → 401 (iss pinning)
4. Wrong algorithm (HS256 attack) → 401
5. Unknown kid → 401 (rotation safety)
6. Wrong signature (key compromise) → 401
7. Malformed JWT → 401
8. Missing sub/iss/role → Claims.Validate failure
9. IFID > 64 chars → Claims.Validate failure
10. Key rotation grace (old kid still verifies durante grace period)

**Migration strategy:**

Default **JWT obrigatório**. Para migration helper:

```go
// dev: RADIANT_DEV_AUTH=1 aceita X-IF-ID como fallback.
// prod: set RADIANT_JWT_PUBLIC_KEY (PEM) + RADIANT_JWT_ISSUER.
```

Tests existentes (legacy X-IF-ID) passam via `RADIANT_DEV_AUTH=1`
setado em `newTestServer`. Em produção real, JWT obrigatório.

---

## 🟡 MÉDIOS (P1)

### F24.2 — Dev flag migration

Documentado em tests + env vars.

### F24.3 — Key rotation grace

`Keyring.Rotate` substitui active, antiga fica retired.
Grace period = max(token TTL) × 2 (recomendação).

```go
func (k *Keyring) Rotate(newActive *Key) error {
    // 1. Marca active atual como retired
    // 2. Adiciona newActive com kid novo
    // 3. Lock até completion
    //
    // Tokens emitidos antes da rotação (com old kid) AINDA verificam.
}
```

Test coverage: `TestKeyring_Rotate_GraceForOldToken`.

---

## 🟢 BAIXO

### F24.4 — cmd/jwt-mint dev tool

```bash
go run ./cmd/jwt-mint \
  --private-key=dev-private.pem \
  --kid=k1 \
  --issuer=radiant-norma \
  --if=demo \
  --role=if \
  --ttl=24h
```

Genera JWT imprimido em stdout. Read token via env file for security
(não aceita flag para private key — só file path).

### F24.5 — Issuer pinning

`Iss claim obrigatório` + `jwt.WithIssuer(config.Issuer)` rejeita
tokens cross-issuer. Vetor cross-tenant fechado.

---

## 📊 Achados consolidados (validação 24)

| Categoria | Críticos | Médios | Baixos |
|-----------|----------|--------|--------|
| Auth bypass | 1 (F24.1) | 0 | 0 |
| Dev mode | 0 | 1 (F24.2) | 0 |
| Key rotation | 0 | 1 (F24.3) | 0 |
| Dev tools | 0 | 0 | 1 (F24.4) |
| Issuer pinning | 0 | 0 | 1 (F24.5) |
| **TOTAL** | **1** | **2** | **2** |

---

## 📊 Acumulado 24 validações

| Val | Findings | Críticos |
|-----|----------|----------|
| 11-23 | 73 | 18 |
| **24** | 5 | 1 |
| **TOTAL** | **78** | **19** |

---

## 🎯 Estado final pós-validação 24

```
270 tests passing (253 → 270, +17 sprint 7a)
vet-clean, race-clean, build-clean

16 categorias vetores fechados (15 v15-v23 + 1 sprint 7a):
  + Auth real JWT RS256 [NOVO]

Pacote internal/auth: 4 arquivos (jwt, claims, keyring, middleware)
+ cmd/jwt-mint (dev tool)
+ cmd/api/main.go setup com env var
v1.6.0 release preparado.
```

---

## ✅ Acceptance da validação 24

- ✅ F24.1 — JWT RS256 substitui X-IF-ID placeholder
- ✅ F24.2 — Dev mode migration (RADIANT_DEV_AUTH=1)
- ✅ F24.3 — Key rotation grace (Keyring.Rotate)
- ✅ F24.4 — cmd/jwt-mint (dev tool)
- ✅ F24.5 — Issuer pinning
- ✅ 270 tests passing
- ✅ vet/race/build clean

---

## 📌 Próximo passo (Sprint 7b — Regras 3040)

Após release v1.6.0, sprint 7b começa com:
- Validação 25 (revisão dos tests de regressão JWT)
- Implementação de 75+ regras 3040 adicionais (alvo: 25 → 100)
- Fuzz testing em `iterXMLElements`
- Documentation de regras (cada uma com código/descrição/exemplo)

Continua-se sem pausa, conforme instruído por Henrique.
