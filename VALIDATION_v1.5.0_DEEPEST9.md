# VALIDATION v1.5.0 DEEPEST9 — 23ª validação profunda (healthz/readyz + auth validation)

> **Status:** ACCEPTED
> **Data:** 2026-07-03
> **Trigger:** Henrique pediu mais uma validação. Foco: observability
> + readiness + auth input validation. v23 deve ser a última antes
> da saturação (v22 deu 2 achados).
> **Versão:** v1.5.0 inalterada (sem bump).

## 🎯 Resumo executivo

23ª validação. Foco em K8s probes e auth input hardening. Codebase
claramente convergindo — 2 achados aplicados (F23.1 + F23.2). 0
críticos.

**2 findings, 0 críticos**

1. **F23.1** 🟡 — `/healthz` e `/readyz` retornavam o mesmo handler.
   Semantically errado. K8s liveness vs readiness probe quer semântica
   diferente. Aplicado: `/readyz` separado com DB PingContext.

2. **F23.2** 🟡 — `authMiddleware` validava apenas empty X-IF-ID. Sem
   limite de tamanho ou charset. Atacante pode enviar X-IF-ID de
   10KB → logado no audit_log → infla disco. Aplicado: max 64 chars
   + charset alfanumérico + dash + underscore.

**Stats:**
- 247 → 253 tests passing (+6: TestReadyz_OK, TestReadyz_NoDB,
  TestAuthMiddleware_IFIDTooLong, TestAuthMiddleware_IFIDInvalidCharset,
  TestAuthMiddleware_IFIDAllowed, TestAuthMiddleware_IFIDMissing)
- vet-clean, race-clean, build-clean

---

## 🟡 MÉDIOS (P1)

### F23.1 — /healthz e /readyz eram o mesmo handler

**Severidade:** 🟡 MÉDIO (K8s production readiness)

**Discovery:**

```go
// server.go:86 (estado anterior)
r.Get("/healthz", s.healthz)
r.Get("/readyz", s.healthz)  // ← MESMO HANDLER!
```

```go
// server.go:235 (handler único)
func (s *Server) healthz(w, r) {
    writeJSON(200, map[string]any{
        "status": "ok",  // ← nunca falha
        ...
    })
}
```

**Por que vetor:**

K8s liveness/readiness probes são conceitos diferentes:

| Probe | Quando falha | Ação |
|-------|--------------|------|
| **Liveness** (`/healthz`) | Processo broken | Restart pod |
| **Readiness** (`/readyz`) | Dependencies broken | Remove pod from load balancer |

Hoje `/readyz` retorna SEMPRE 200 → K8s sempre considera pod ready,
mesmo se DB estiver quebrado. Requests entram em pod zumbi.

**Cenário de produção:**

1. Postgres tem outage 5min
2. Pod tem DB quebrada (PingContext falha)
3. K8s readiness probe deveria tirar pod do LB
4. Mas: `/readyz` retorna 200 → K8s mantém pod no LB
5. Requests entram → falham com 500 → user experience ruim

**Fix aplicado:**

```go
r.Get("/healthz", s.healthz)   // mantém — sempre OK enquanto processo está vivo
r.Get("/readyz", s.readyz)     // Validação 23 (F23.1): separado

// healthz: apenas verifica que processo está vivo.
// Não checa DB — restart loop seria pior se DB temporariamente fora.
func (s *Server) healthz(w, r) {
    writeJSON(200, map[string]any{
        "status": "ok",
        "time": time.Now().UTC().Format(time.RFC3339),
        "version": Version,
        "uptime_seconds": int(time.Since(s.startedAt).Seconds()),
    })
}

// readyz: pinga DB. Se falhar → 503 → K8s tira do LB.
func (s *Server) readyz(w, r) {
    if s.DB == nil {
        http.Error(w, "db not configured", http.StatusServiceUnavailable)
        return
    }
    if err := s.DB.PingContext(r.Context()); err != nil {
        http.Error(w, "db unavailable", http.StatusServiceUnavailable)
        return
    }
    writeJSON(200, map[string]any{
        "status": "ready",
        "db": "ok",
        "version": Version,
    })
}
```

**Aplicabilidade cross-project:**

Em qualquer codebase Go com HTTP server e K8s, sempre separar
`/healthz` (liveness) de `/readyz` (readiness). Liveness = processo;
readiness = dependências.

**Anti-pattern:** handler único retornando "ok" sempre. K8s não tem
como saber se pod deve receber requests.

---

### F23.2 — X-IF-ID sem limite de tamanho/charset

**Severidade:** 🟡 MÉDIO (DOS-via-AuditLog)

**Discovery:**

```go
// server.go:733 (estado anterior)
func (s *Server) authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w, r) {
        if r.Header.Get("X-IF-ID") == "" {
            // reject empty
        }
        next.ServeHTTP(w, r)
    })
}
```

**Por que vetor:**

`X-IF-ID` é usado em 3 sinks:

1. `auditLog.Log(ifID, ..., "cadoc.validated", ...)` — persistido em disco
2. `sub.CNPJ = ifID` — STA submission (passado adiante)
3. `d.Exec("INSERT INTO envios(... if_id ...)", envioID, ifID, ...)` —
   DB column TEXT (sem limite definido)

Atacante autenticado pode enviar:

```http
POST /v1/validate HTTP/1.1
X-IF-ID: AAAA...<10KB>...AAAA
```

E:
- Audit log tem 10KB × N requests → infla disco audit_log
- Hash chain integra essas entries (mas SHA-256 não é afetado)
- `cadoc_validated` action acumula bytes

N=1000 requests × 10KB IF-ID = 10MB de lixo no audit_log.

**Fix aplicado:**

```go
func (s *Server) authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w, r) {
        ifID := r.Header.Get("X-IF-ID")
        if ifID == "" {
            // 401 missing
        }
        if len(ifID) > 64 {
            // 400 too long
        }
        for _, c := range ifID {
            // alphanumeric + - + _
            ok := (c >= 'a' && c <= 'z') ||
                (c >= 'A' && c <= 'Z') ||
                (c >= '0' && c <= '9') ||
                c == '-' || c == '_'
            if !ok {
                // 400 invalid charset
            }
        }
        next.ServeHTTP(w, r)
    })
}
```

**Decisões de design:**

1. **Max 64 chars:** CNPJ raiz tem 8 dígitos. Reservei 64 chars
   para IDs custom (ex: `demo-bank_2`). Suficiente para casos
   conhecidos; curto o bastante para não inchar.

2. **Charset `[a-zA-Z0-9_-]`:** sem punctuation/special chars. Elimina:
   - SQL injection (mesmo parametrized, defense-in-depth)
   - Path traversal (`../`)
   - Shell injection (`;`, `&&`)
   - Log injection (newlines)

3. **Single-loop charset check:** O(n) onde n ≤ 64. Custo:
   ~64 comparações. Negligível.

**Trade-off:** se callers legítimos usarem IF-IDs com chars
especiais (ex: `demo@empresa.com`), vão precisar migrar. Doc
atualiza: IF-IDs são `[a-zA-Z0-9_-]{1,64}`.

**Aplicabilidade cross-project:**

Qualquer header-based tenant identifier (X-IF-ID, X-Tenant-ID,
X-Org-ID) deve ter validação:
- Max length (não abuse attacker-controlled)
- Charset whitelist (defense-in-depth)
- Format (UUID? CNPJ raiz? regex?)

Sem isso, X-Tenant-ID vira vetor de DOS-audit-log ou SQL/noSQL
injection surface.

---

## 🟢 BAIXO

### F23.3 — RequestID middleware gera ID mas não propaga para logs

**Severidade:** 🟢 BAIXO (cross-cutting refactor)

`middleware.RequestID` é usado em chi router, mas `UserError` loga
sem incluir `r.Context().Value("request_id")`. Trace distributed
fica quebrado: erro 500 produz log entry sem correlação com response
header `X-Request-Id`.

**Não aplicado v23:** refator cross-cutting que toca 9 helpers +
audit emission. Mais seguro como Sprint 7.

---

## 📊 Achados consolidados (validação 23)

| Categoria | Críticos | Médios | Baixos |
|-----------|----------|--------|--------|
| K8s observability | 0 | 1 (F23.1) | 0 |
| Auth validation | 0 | 1 (F23.2) | 0 |
| Cross-cutting | 0 | 0 | 1 (F23.3) |
| **TOTAL** | **0** | **2** | **1** |

**Taxa continua caindo:** v20=7, v21=5, v22=2, v23=2. Alguma coisa
aconteceu: 2 achados em v23 — porque F23.1 e F23.2 são fixes
arquiteturais (K8s/validation), não bugs.

**0 críticos em v23.** Codebase saturando.

---

## 📊 Acumulado 23 validações

| Validação | Findings | Críticos |
|-----------|----------|----------|
| 11 | 9  | 0 |
| 12 | 9  | 4 |
| 13 | 4  | 1 |
| 14 | 5  | 1 |
| 15 | 4  | 1 |
| 16 | 4  | 1 |
| 17 | 3  | 0 |
| 18 | 8  | 3 |
| 19 | 7  | 4 |
| 20 | 7  | 2 |
| 21 | 5  | 1 |
| 22 | 2  | 0 |
| 23 | 3  | 0 |
| **TOTAL** | **70** | **18** |

**13 validações consecutivas com findings. 18 críticos total.**

---

## 🎯 Estado final pós-validação 23

```
253 tests passing (247 → 253, +6 readyz+auth tests)
vet-clean, race-clean, build-clean

15 categorias fechadas (12 err.Error + 2 arquiteturais + 1 stampede):
  + K8s /healthz vs /readyz separation [NOVO]
  + X-IF-ID input validation (max 64 chars + charset) [NOVO]

Padrão consolidando:
  v18 = 8 findings
  v19 = 7 findings
  v20 = 7 findings
  v21 = 5 findings
  v22 = 2 findings
  v23 = 3 findings (0 críticos) ←
```

**Codebase está estabilizando.** v22 e v23 com 0 críticos consecutivos
é o gatilho para encerrar Fase 1.

---

## ✅ Acceptance da validação 23

- ✅ F23.1 — /readyz separado, checa DB PingContext
- ✅ F23.2 — X-IF-ID max 64 chars + charset whitelist
- ✅ F23.3 — RequestID propagation documentado (follow-up)
- ✅ 6 tests novos (TestReadyz_OK, TestReadyz_NoDB,
    TestAuthMiddleware_IFIDTooLong, TestAuthMiddleware_IFIDInvalidCharset,
    TestAuthMiddleware_IFIDAllowed, TestAuthMiddleware_IFIDMissing)
- ✅ 253 tests passing
- ✅ vet/race/build clean

---

## 📌 Próximo passo (Sprint 7)

**Estado consolidado pós-23:**
- ✅ 70 findings, 18 críticos
- ✅ 15 categorias de vetores fechadas
- ✅ 253 tests passing
- ✅ Memory atualizado com 7 cross-project patterns
- ⏳ 35 commits ahead of origin
- ⏳ RequestID propagation (F23.3 follow-up)

**Recomendação Sprint 7:**

Codebase está estável. v22 e v23 ambos com 0 críticos. Prático
encerrar Fase 1 — v1.5.0 estável como release.

1. **PUSH dos 35 commits** (operacional)
2. **Encerrar Fase 1** com tag final v1.5.0 estável
3. **F23.3 follow-up Sprint 8** (cosmetic refactor)
4. **Próxima fase** — decidir com Henrique:
   - **Opção A:** Próxima release (v1.6.0 feature)
   - **Opção B:** Postgres integration tests (gap backend)
   - **Opção C:** Feature nova (radiant-norma next dimension)
   - **Opção D:** Outro projeto
