# VALIDATION v1.5.0 DEEPEST3 — 17ª validação profunda (post-v16 sweep)

> **Status:** ACCEPTED
> **Data:** 2026-07-03
> **Trigger:** Henrique pediu mais uma validação. Foco: cross-check
> transversal do que v16 entregou, auditar edge cases ainda não cobertos.
> **Versão:** v1.5.0 inalterada (sem bump — apenas hardening).

## 🎯 Resumo executivo

Sweep TOTAL pós-commit da validação 16 (F16.1 + F16.5). Procura
consistente por vetores residuais.

**3 gaps encontrados:**

1. **F17.1** 🟡 MÉDIO — 3 ocorrências Warn-level em `cmd/seed/main.go`
   (insert críticas, decode leiaute, insert schema) usando `err`
   cru sem `SafeError`. Aplicado F17.1.

2. **F17.2** 🟢 BAIXO — 3 tests adicionados ao `safe_test.go`
   cobrindo edge cases que as validações anteriores não tinham:
   - `TestSafeError_DriverConstraintError` — INSERT errors
   - `TestSafeError_JsonDecodeError` — json.Unmarshal errors
   - `TestSafeError_FmtErrorfChain` — `fmt.Errorf("%w", ...)` chains

3. **F17.3** 🟢 BAIXO — Memory pattern cross-project atualizado para
   iteração 3: "logger.Warn também precisa SafeError".

**Total hoje:**
- 3 substituições aplicadas (logger.Warn → loggerutil.SafeError)
- 3 tests novos em safe_test.go (15 total, vs 12 da v16)
- CHANGELOG atualizado com tabela consolidada validações 11-17

---

## 🟡 MÉDIOS (P1)

### F17.1 — 3 logger.Warn sem SafeError em cmd/seed (escape do F16.1)

**Severidade:** 🟡 MÉDIO (consistência; vetor real pequeno)

**Discovery via sweep ampliado:**

```bash
grep -rnE 'logger\.(Error|Warn|Info|Debug).*"err"' backend/cmd/ backend/internal/ \
  | grep -v _test.go | grep -v SafeError
```

**Output (3 matches):**

```
backend/cmd/seed/main.go:192: logger.Warn("insert falhou", "codigo", c.Codigo, "err", err)
backend/cmd/seed/main.go:262: logger.Warn("decode leiaute falhou", "cadoc", cadoc, "err", err)
backend/cmd/seed/main.go:287: logger.Warn("insert schema falhou", "cadoc", cadoc, "err", err)
```

**Por que escapou de F16.1:**

F16.1 cobriu **Error-level** apenas. Warn passou despercebido.
A regex do sweep era:

```bash
grep -rnE 'logger\.Error.*"err",\s*err[,)]'
```

Não pegava `logger.Warn`.

**Avaliação do vetor real:**

- `stmt.Exec` (INSERT críticos, INSERT schemas) — erros são constraint
  violations tipo `UNIQUE constraint failed: table.col` ou
  `pq: duplicate key`. **NÃO vazam DSN** por design.
- `json.Unmarshal(rawLei, &lei)` — erro é tipo `invalid character 'x'
  looking for beginning of value`. **NÃO vaza DSN**.

Argumentavelmente o vetor REAL é pequeno, mas:
1. **Consistência** — todo log de erro deve passar por SafeError
2. **Defense-in-depth** — se algum driver futuro passar a incluir DSN
   em erros de constraint, já estamos cobertos
3. **Memory pattern** — "algum nível de logger é diferente" é armadilha
   (já caiu com F13.8 que era Info-level)

**Fix aplicado:**

```go
// cmd/seed/main.go:192
logger.Warn("insert falhou", "codigo", c.Codigo,
    "err", loggerutil.SafeError(err))

// cmd/seed/main.go:262
logger.Warn("decode leiaute falhou", "cadoc", cadoc,
    "err", loggerutil.SafeError(err))

// cmd/seed/main.go:287
logger.Warn("insert schema falhou", "cadoc", cadoc,
    "err", loggerutil.SafeError(err))
```

`loggerutil` já estava importado em `cmd/seed/main.go`.

**Final sweep pós-F17.1 confirma:**

```bash
$ grep -rnE 'logger\.(Error|Warn|Info|Debug).*"err"' backend/cmd/ backend/internal/ \
    --include="*.go" | grep -v _test.go | grep -v SafeError
0 matches
```

**Aplicabilidade:** Universal. Cobertura agora é `Error|Warn|Info|Debug`
— qualquer logger call com `"err"` field passa por SafeError.

---

## 🟢 BAIXO

### F17.2 — 3 tests em safe_test.go (iteração 3)

**Severidade:** 🟢 (test coverage)

Adicionados 3 tests focados nos vetores expandidos pós-F16.5:

```go
// TestSafeError_DriverConstraintError
// INSERT errors NÃO vazam DSN, mas test garante que mudanças no
// regex não corrompam mensagens legítimas tipo "UNIQUE constraint..."
err := errors.New("UNIQUE constraint failed: schema_versions.cadoc_code")
got := loggerutil.SafeError(err)
// expect: idêntico, sem modificação

// TestSafeError_JsonDecodeError
// Erros de json.Unmarshal também devem passar limpos.
err := errors.New("invalid character 'x' looking for beginning of value")
got := loggerutil.SafeError(err)
// expect: idêntico, sem modificação

// TestSafeError_FmtErrorfChain
// fmt.Errorf("%w", original) preserva mensagem do inner.
// Em Go 1.13+, fmt.Errorf com %w permite Unwrap(). Mas SafeError
// recebe error.Error() (concatena tudo).
inner := errors.New("connect: postgres://u:pa55@primary:5432/db")
outer := fmt.Errorf("first attempt: %w; retrying", inner)
got := loggerutil.SafeError(outer)
// expect: DSN encadeada sanitizada
```

**Total tests loggerutil:** 12 → **15**.

**Por que esses vetores:** `cmd/seed` é o único pkg que faz logging
diferenciado. v17 confirmou que seed não vaza (insert + decode mantêm-se
limpos). Mas o teste serve como baseline e proteção contra mudanças
acidentais no regex.

---

### F17.3 — Memory pattern iteração 3

Padrão de sanitization iterado pela 3ª vez:

**Iteração 1 (F15.1):** "Crie SafeError helper"

**Iteração 2 (F16.1 + F16.5):** "Aplique em TODOS os logger.Error"

**Iteração 3 (F17.1):** "Aplique em TODOS os logger.*: Error + Warn +
Info + Debug — exceto quando `err` é info, aí use Info-only fallback
sem err field."

Pattern consolidado: **NENHUMA chamada logger com `"err"` field pode
passar error cru**. Reconhecidos falsos positivos:

1. `logger.Info("✓ seed completo")` — sem err, não passa pelo SafeError.
2. `logger.Debug(...)` — mesmo critério.
3. `logger.Info("api listening", "addr", addr)` — sem err, info disclosure
   não é problema aqui (addr:port não é credencial).

**Aplicabilidade (universal):**

- Todo codebase que faz `logger.Error/Warn/Info/Debug` com field "err"
  → se err.Error() pode conter secret, use wrapper de sanitization.

- Não confie em "esse nível específico não vaza" — **uniform coverage**.

- **Caveat:** F13.8 tinha Info-level leak (token prefix). F13.8 passa pelo
  padrão "se err.Error() toca credenciais, sanitize". F16.1 cobriu
  Error+runtime. F17.1 fecha Warn também.

---

## 📊 Findings consolidados (validação 17)

| Categoria | Críticos | Médios | Baixos |
|-----------|----------|--------|--------|
| Logger.X sem SafeError | 0 | 1 (F17.1) | 0 |
| Test coverage | 0 | 0 | 1 (F17.2) |
| Memory pattern | 0 | 0 | 1 (F17.3) |
| **TOTAL** | **0** | **1** | **2** |

**Crítico:** 0 — codebase continua estável após v16.
**Médio:** 1 — pequena gap de consistência, fix aplicado.
**Baixo:** 2 — test coverage + memory.

---

## 🎯 Validação: balanço das 17 validações

**Estado pós-validação 17:**

```
228 tests passing (15 loggerutil / 213 outros)
vet-clean, build-clean, race-clean
6 packages com health checks ativos
20 logger calls com "err" → TODOS via SafeError
0 logger.Error/Warn/Info/Debug com err cru
0 fmt.Println/Printf/Print diretos em pkg críticos
0 DSN residual em fmt.Errorf
```

**Pattern cross-project de err sanitization estabilizou:**

1. **Helper (F15.1):** `loggerutil.SafeError(err) string`
2. **Cobertura (F16.1):** TODOS os logger.Error
3. **Robustez regex (F16.5):** dsnCanonical + pgxKeyValue + passwordKV
   + passwordInQuery (ordem importa)
4. **Universal (F17.1):** TODOS os logger.* níveis
5. **Regression tests (F17.2):** vetor REAL + edge cases
6. **Memory (F17.3):** iteração 3 documentada

---

## 📌 Próximo passo (Sprint 7)

Sprint 7 backlog consolidado de validações 11-17 (já existia —
atualizado com F17.1/F17.3):

1. **F16.11** — Sanitize regex patterns adicionais (Sprint 7)
2. **F12.10** — ScanLimiter LRU
3. **F12.14/F13.15** — Cross-doc XML decoder robusto
4. **F12.21** — singleflight em CadocListCache
5. **F12.22** — crossdoc/rules tests
6. **F12.6 follow-up** — Postgres integration tests
7. **GAP-7.4** — User-Agent hardcoded (refactor internal/version/)
8. **F14.4** — Server struct Logger field
9. **F14.8** — HTTP 4xx err.Error() review
10. **F15.5** — Test de regressão cmd/* log NÃO contém secrets (in-process)

Prioridade: F16.11 ou GAP-7.4 (refactor internal/version).

---

## ✅ Acceptance da validação 17

- ✅ F17.1 — 3 logger.Warn cmd/seed substituídos via SafeError
- ✅ F17.2 — 3 tests novos safe_test.go (15 total)
- ✅ F17.3 — Memory pattern iteração 3
- ✅ 228 tests passing, race-clean, vet-clean, build-clean
- ✅ 0 vetor residual em logger.* com "err"
- ✅ 0 fmt.Errorf com DSN residual
- ✅ CHANGELOG atualizado com tabela validações 11-17

---

## 🔄 Comando para validar (replayável)

```bash
cd backend/
go test ./... -count=1 -race                                  # 228 passing
go vet ./... && go build ./... && echo "clean"
grep -rnE 'logger\.(Error|Warn|Info|Debug).*"err"' \
    cmd/ internal/ --include="*.go" | grep -v _test.go \
    | grep -v SafeError                                       # 0 matches
```
