# VALIDATION v1.5.0 DEEPER — 14ª validação profunda (auditoria de follow-up)

> **Status:** ACCEPTED
> **Data:** 2026-07-03
> **Trigger:** Henrique pediu mais uma validação repassando a validação 13.
> Validação focada em grep patterns dos fixes anteriores (F13.8, F13.6, F13.1)
> e seus efeitos colaterais.
> **Versão:** v1.5.0 (sem bump — fixes in-place aplicados).

## 🎯 Resumo executivo

**Estado pós-validação 14:**
- ✅ 5 fixes aplicados:
  - **F14.1** 🔴 — DSN com password logado em cmd/worker + cmd/radar (CRÍTICO)
  - **F14.2** 🟡 — cmd/api::go func() sem recover
  - **F14.3** 🟡 — `indexOf` reinventa `strings.Index` em crossdoc/rules
  - **F14.6** 🟢 — Self-inconsistency no doc FOLLOWUP (FIXED)
  - **F14.9** 🟡 — `cmd/seed` consistencia: DATABASE_URL support
- ⏸️ 2 documentados Sprint 7 backlog (F14.4, F14.8)

**Tamanho da validação 14:** menor que 13 (5 fixes vs 4), mas
**F14.1 só** já justifica o tempo investido — outro secret leak
em cmd/* que afetaria produção real.

---

## 🔴 CRÍTICO (P0) corrigido in-place

### F14.1 — SECRET LEAK #2: DSN inteira (com password Postgres) em logs

**Severidade:** 🔴 CRÍTICO (security disclosure — same class as F13.8)

**Diagnóstico:**
```go
// cmd/worker/main.go (ANTES)
logger.Info("worker started",
    "db", resolvedDB,  // <-- DSN INTEIRA com password Postgres!
    "backend", db.Backend(resolvedDB),
    ...
)

// cmd/radar/main.go (ANTES)
logger.Info("radar worker started",
    "db", resolvedDB,  // <-- mesma leak
    "backend", db.Backend(resolvedDB),
    ...
)
```

DSN Postgres típico:
`postgres://user:PA$$W0RD@db.example.com:5432/radiant?sslmode=require`

Logado em plain text no startup. Mesma classe de vulnerability do F13.8
(token prefix em logs), agora em vetor **muito mais sério** porque:

1. **DSN sempre tem password**. Token admin pode ser empty/missing
   (FAIL CLOSED), mas Postgres DATABASE_URL sempre tem password ou
   auth-via-cert (cenário comum).
2. **Disclosed em log de startup**: visível em CI logs, systemd
   journal, CloudWatch.
3. **Cross-referencing com logs de erros**: stack traces de DB connection
   frequentemente incluem DSN na mensagem.

**Por que escapou de F12.2 + F13.8:** ambos focaram em token prefix.
DSN logging é paralelo, ficou intocado.

**Fix aplicado:**
```go
logger.Info("worker started",
    // Validação 14 (F14.1): NÃO logar DSN (pode conter password).
    // Logger apenas backend name (sqlite/postgres) — não loga path/host.
    "backend", db.Backend(resolvedDB),
    ...
)
```

E o mesmo em cmd/radar. F14.9 aplica o mesmo padrão em cmd/seed
(consistência cross-cmd).

**Lição (cross-project):** Token em logs é óbvio. DSN em logs é
menos óbvio mas MAIS provável de vazar password (porque DSN sempre
tem credenciais em produção). Toda string que vem de env var deve
ser tratada como potencial secret até provado contrário.

---

## 🟡 MÉDIOS (P1) corrigidos

### F14.2 — cmd/api::go func() sem recover

**Severidade:** 🟡 MÉDIO (defense in depth, memory pattern consistência)

**Diagnóstico:**
```go
// cmd/api/main.go::ListenAndServe goroutine (ANTES)
go func() {
    logger.Info("api listening", "addr", addr)
    if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        serverErr <- err
    }
    close(serverErr)
}()
```

`ListenAndServe` raramente panica (stdlib muito bem testada), MAS
middleware custom ou handlers com deps externas podem. Sem recover,
panic → main goroutine morre → exit code 1 MAS sem log de erro.

**Fix aplicado:**
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            logger.Error("server goroutine panic", "panic", r)
            os.Exit(1)
        }
    }()
    // ...
}()
```

Memory pattern: "Goroutine de long-running em cmd/* precisa panic recover"
aplicado consistentemente em:
- cmd/radar/main.go::scan (validação 13)
- cmd/worker/main.go::runLeaseSweeperLoop (validação 13)
- cmd/api/main.go::ListenAndServe (validação 14 — esta)

---

### F14.3 — `indexOf` reinventa `strings.Index`

**Severidade:** 🟡 MÉDIO (memory pattern "reinventar stdlib" #2)

**Diagnóstico:**
```go
// crossdoc/rules/3040_4111.go::indexOf (ANTES)
func indexOf(sub, s string) int {
    for i := 0; i+len(sub) <= len(s); i++ {
        if s[i:i+len(sub)] == sub {
            return i
        }
    }
    return -1
}
```

É literalmente `strings.Index`. Byte-by-byte comparison custom.

**Fix aplicado:**
```go
// Removido.
// indexFrom agora usa strings.Index diretamente:
func indexFrom(s, sub string, from int) int {
    if from >= len(s) {
        return -1
    }
    idx := strings.Index(s[from:], sub)
    if idx == -1 {
        return -1
    }
    return idx + from
}
```

**Bonus:** simplifica de 9 linhas para 4. Removido wrapper redundante
— exatamente o memory pattern (mesma racional do itoa v1.4.4 e min() v1.5.0).

---

### F14.9 — cmd/seed consistency: DATABASE_URL support

**Severidade:** 🟡 MÉDIO (consistency — toda a stack agora fala Postgres)

**Diagnóstico:**
```go
// cmd/seed/main.go (ANTES)
dbPath := flag.String("db", "radiant.db", "caminho do banco SQLite")
// Sem suporte a DATABASE_URL → seed não roda em Postgres
```

`cmd/seed` é usado para popular DB com críticas/leiautes. Em prod,
teria que usar Postgres. Sem suporte, **não tem como seedar Postgres** —
operador precisa converter Postgres→SQLite temporariamente, seedar, dump,
reconvert. Hack.

**Fix aplicado:**
- `dbPath` agora default `""`; se vazio, lê `DATABASE_URL` env; fallback
  `radiant.db` (dev).
- Mesma função `db.Open` detecta driver.
- `cmd/seed` agora consistente com `cmd/api/worker/radar`.

Padrão: **toda entrypoint em cmd/* usa mesmo DSN resolution**.

---

## 🟢 BAIXO corrigido

### F14.6 — Self-inconsistency no VALIDATION_v1.5.0_FOLLOWUP.md

**Severidade:** 🟢 BAIXA (doc accuracy, mesmo pattern F12.17)

**Diagnóstico:**
- Sumário executivo dizia "4 críticos (P0) + 3 médios (P1) corrigidos"
- Tabela detalhada dizia "**3 P0 + 2 P1 + 7 🟢**" (F12.17 listado como 🟢)
- Self-inconsistency entre seções do mesmo doc.

**Fix aplicado:**
Sumário alinhou com tabela:
- 3 P0 (F12.2, F12.5, F12.19) corrigidos
- 3 P1 (F12.6, F12.8, F12.11) corrigidos
- 1 🟢 corrigido (F12.17 self-inconsistency)
- 5 🟢 Sprint 7 backlog (F12.1, F12.10, F12.14, F12.21, F12.22)

Lição: memory pattern "self-referência em docs é armadilha" aplicado.

---

## ⏸️ Sprint 7 backlog (documentados, não bloqueiam)

### F14.4 — `slog.Default()` direto em handlers

**Severidade:** 🟢 BAIXA (acoplamento implícito)

**Detalhes:**
```go
// internal/api/server.go
if s.AdminAuth == nil {
    logger := slog.Default()  // fallback implícito
    logger.Error(...)
}
```

Server struct não tem field `Logger *slog.Logger`. Test setup não tem
como trocar logger (slog.Default já está configurado por cmd/*). Engine
crossdoc tem `SetLogger` (validação 13) — Server poderia ter mesma API.

**Sprint 7:** adicionar `srv.Logger *slog.Logger` field + `SetLogger()`.

---

### F14.8 — `http.Error(w, err.Error(), 500)` em server.go leaks

**Severidade:** 🟢 BAIXA (information disclosure)

**Detalhes:**
Múltiplos handlers usam `http.Error(w, err.Error(), 500)` — erro
interno do DB é exposto na response HTTP. Em produção, isso vaza
SQL fragments, table names, etc.

**Sprint 7:** helper `internalServerError(w, err, logger)` que log
interno + retorna 500 genérico + correlation ID.

---

## 📊 Findings por categoria (validação 14)

| Categoria | Críticos (P0) | Médios (P1) | Baixos (🟢) |
|-----------|---------------|--------------|-------------|
| Secret leak em logs | 1 (F14.1) | 0 | 0 |
| Panic recover missing | 0 | 1 (F14.2) | 0 |
| Reinveting stdlib | 0 | 1 (F14.3) | 0 |
| Consistency cross-cmd | 0 | 1 (F14.9) | 0 |
| Self-doc consistency | 0 | 0 | 1 (F14.6) |
| Logger injection | 0 | 0 | 1 (F14.4 backlog) |
| Error message disclosure | 0 | 0 | 1 (F14.8 backlog) |
| **TOTAL** | **1** | **3** | **1 corrigido + 2 backlog** |

---

## 🎯 Padrão memory consolidado

**Duas validações consecutivas (13 + 14) com secret leaks em logs.**
F13.8 era token prefix; F14.1 era DSN password. Mesma classe.

Memory pattern atualizado para:
**"NUNCA logar conteúdo de qualquer string que veio de env var.
Logar metadados apenas (length, presence, hash truncated). Aplica-se
a tokens, passwords, DSNs, API keys, certificates — tudo que
envolve credencial."**

---

## ✅ Acceptance da validação 14

- ✅ F14.1 DSN secret leak corrigido (CRÍTICO)
- ✅ F14.2 panic recover cmd/api (consistência memory pattern)
- ✅ F14.3 reinvent-stdlib corrigido
- ✅ F14.9 cross-cmd consistency
- ✅ F14.6 self-doc consistency corrigido
- ✅ 213 tests passing, race detector clean, vet clean

---

## 🚀 Estado final pré-Sprint 7

```
Backend:    estável, race-clean, vet-clean, 213 tests passing
Tag:        v1.5.0 local
Push:       25 commits ahead of origin (pendente)
Frontier:   Sprint 7 (ver VALIDATION_v1.5.0_DEEP.md backlog)
Memory:     4 patterns novos (cmd/entrypoint, secret logs, panic recover,
            reinvent-stdlib) — todos cross-project
```
