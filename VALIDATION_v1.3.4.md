# Validação Profunda v1.3.4 — Silent Errors + Case-Insensitive + Healthz Uptime

> **Data:** 2026-07-03
> **Validação:** 4ª passada (v1.3.3 → v1.3.4)
> **Método:** grep em `_ = ...` + manual code review + E2E regression

## 🎯 Escopo

Caçar bugs latentes escondidos em **erros silenciados** (`_ = func()`),
endurecer o endpoint `/v1/rules/{cadoc}`, e melhorar UX do healthz.

## 🐛 Bugs detectados (6 achados)

### 🔴 Bug #1 — Radar duplica alertas após falha de baseline

**Arquivo:** `internal/radar/radar.go:178`

**Sintoma silencioso:**
```go
// Mudança detectada!
id, err := s.insertAlert(ctx, alert)
if err != nil { return nil, fmt.Errorf("insert alert: %w", err) }
alert.ID = id

// Atualiza baseline
_ = s.recordBaseline(ctx, src, hash)   // ← silenciado
```

**Falha:** Se `recordBaseline` falha (DB cheio, lock, qualquer erro SQL), o alerta
foi gravado mas a baseline não foi atualizada. Próximo scan:
1. `lastKnownHash` retorna hash antiga
2. `hash != lastHash` → **MESMO alerta disparado de novo**
3. Tabela `radar_alerts` incha com duplicações

**Fix:**
```go
if err := s.recordBaseline(ctx, src, hash); err != nil {
    s.logger.Warn("recordBaseline failed after alert — próximo scan pode duplicar",
        "alert_id", id, "cadoc", src.CadocCode, "err", err)
}
```

**Severidade:** 🔴 Alta — bug latente em produção, indetectável sem testes.

---

### 🟡 Bug #2 — Radar first scan silencia erro de baseline

**Arquivo:** `internal/radar/radar.go:151`

**Sintoma silencioso:**
```go
if lastHash == "" {
    s.logger.Info("first scan, recording baseline", ...)
    _ = s.recordBaseline(ctx, src, hash)   // ← silenciado
    return nil, nil
}
```

**Falha:** Se DB tem problema no first scan, retorna `(nil, nil)` — operador
não vê erro, mas toda scan subsequente também vai cair em "first scan" e
silenciar. Loop silencioso.

**Fix:**
```go
if err := s.recordBaseline(ctx, src, hash); err != nil {
    s.logger.Error("recordBaseline failed (first scan)",
        "cadoc", src.CadocCode, "err", err)
}
```

**Severidade:** 🟡 Média — operador não vê erro, mas cada scan vai logar
"first scan" + erro do DB até DB voltar.

---

### 🔴 Bug #3 — Worker loop infinito após STA submit failed

**Arquivo:** `cmd/worker/main.go:165`

**Sintoma silencioso:**
```go
result, err := staClient.Submit(ctx, sub)
if err != nil {
    logger.Error("sta submit failed", "envio_id", e.ID, "err", err)
    _, _ = d.ExecContext(ctx,
        "UPDATE envios SET status='error', error_message=? WHERE id=?",
        err.Error(), e.ID)
    continue
}
```

**Falha:** Se STA submit falha E o UPDATE subsequente falha (mesma transação
do DB, mesmo motivo — provavelmente DB indisponível), o envio fica em
`status='pending'` no DB. O worker volta a claimar esse envio a cada iteração
e falha de novo — **loop infinito de retries** até DB voltar.

**Fix:**
```go
if _, uerr := d.ExecContext(ctx,
    "UPDATE envios SET status='error', error_message=? WHERE id=?",
    err.Error(), e.ID); uerr != nil {
    logger.Error("failed to mark envio as error — possible loop",
        "envio_id", e.ID, "err", uerr)
}
```

**Severidade:** 🔴 Alta — loop infinito é o pior tipo de bug operacional.

---

### 🟡 Bug #4 — `?enabled=true|false` case-sensitive

**Arquivo:** `internal/api/server.go:listRulesByCadoc`

**Sintoma:**
```
?enabled=true   → 320   ✓
?enabled=TRUE   → 349   ✗ (cai no default, retorna TODAS)
?enabled=True   → 349   ✗ (mesmo problema)
```

Cliente que envia `enabled=TRUE` (maiúsculo, comum em JS query string) recebe
TODAS as regras em vez das habilitadas. Quebra expectativa.

**Fix:**
```go
enabledFilter := strings.ToLower(r.URL.Query().Get("enabled"))
```

**Severidade:** 🟡 Média — inconsistência cross-platform (HTTP header case-insensitive,
mas query string era case-sensitive).

---

### ⚪ Bug #5 — Lógica redundante em listRulesByCadoc

**Arquivo:** `internal/api/server.go:listRulesByCadoc`

**Sintoma:** Antes do fix, o switch tinha 3 branches conflitantes:
```go
switch enabledFilter {
case "true":   // filtra habilitadas
case "false":  // filtra desabilitadas
default:       // sobrescreve filtered = criticas (todas)
}
if enabledFilter == "" {
    filtered = criticas   // redundante com default
}
```

**Fix:** simplificado para um só path:
```go
filtered := make([]audit.Critica, 0, len(criticas))
for _, c := range criticas {
    switch enabledFilter {
    case "true":   if c.Enabled { filtered = append(filtered, c) }
    case "false":  if !c.Enabled { filtered = append(filtered, c) }
    default:       filtered = append(filtered, c)
    }
}
```

**Severidade:** ⚪ Baixa — não é bug funcional, é code smell.

---

### ⚪ Bug #6 — 14 arquivos com drift gofmt

**Sintoma:** `gofmt -l .` retorna 14 arquivos com formatação não-canônica
(espaços, tabs, alinhamentos). Não quebra nada, mas inconsistência.

**Fix:** `gofmt -w .`

**Severidade:** ⚪ Cosmética.

---

## ✅ Melhorias UX

### Healthz com uptime

**Antes:**
```json
{ "status": "ok", "time": "...", "version": "1.3.1" }
```

**Depois:**
```json
{
  "status": "ok",
  "time": "2026-07-03T20:22:54Z",
  "uptime_seconds": 2,
  "version": "1.3.4"
}
```

**Razão:** Operações precisa saber há quanto tempo processo está rodando
(restart recente, hang, etc). Implementação: `Server.startedAt = time.Now()`
no construtor + `time.Since(s.startedAt).Seconds()` no handler.

---

## 📊 Suite de regressão E2E (v1.3.4)

```
✓ POST /v1/validate XML oficial              → passed=true, 0 erros, 0ms
✓ GET  /v1/rules/3040?enabled=true|True|TRUE  → 320 (case-insensitive)
✓ GET  /v1/rules/3040?enabled=false|False|FALSE → 29
✓ GET  /v1/rules/3040?enabled=                → 349 (default all)
✓ GET  /v1/rules/3050                         → 51
✓ GET  /v1/rules/9999                         → total=0, HTTP 200 (cadoc vazio)
✓ S04 detecta 6 modalidades BACEN (0204, 0210, 1304, 0201, 0213, 0214)
✓ S04 NÃO detecta modalidades fora (0202, 0215, 0218, 0501, 0900, 19)
✓ S04 ignora v110/v120/v165 (só V150/V160)
✓ S04 pula quando v150=0 e v160=0
✓ S04 detecta múltiplos Agregs (um por vez)
✓ Audit chain válida após 5 validações (12 entries)
✓ Radar recordBaseline idempotente (3 scans → 1 row)
✓ Worker processa envio pending → status=accepted, protocolo gerado
✓ Healthz uptime_seconds cresce corretamente (2 → 5 após sleep 3s)
```

---

## 🏗️ Lição aprendida (cross-project)

**Silent errors (`_ =`) são hollow stubs de log.**

A 4ª validação encontrou 3 lugares onde `_ = func()` escondia bugs de produção
(loops infinitos, duplicação de alertas, first-scan loops). Padrão geral:

```go
// ✗ ANTI-PATTERN: silencia erro
_ = criticalOperation()

// ✓ ACEITÁVEL: best-effort (audit log, métrica secundária)
_ = json.Marshal(metadata)  // log entry ainda é gravado sem metadata

// ✓ CORRETO: erro crítico deve propagar OU logar
if err := criticalOperation(); err != nil {
    logger.Error("...", "err", err)
}
```

**Regra:** se a chamada pode falhar E o erro importa pra operação, log ou
propague. Silêncio só é OK em best-effort secundário (metadata cosmético,
cache, etc).

Aplicável a qualquer projeto Go.

---

## 📂 Arquivos modificados

```
internal/api/server.go       — healthz uptime + case-insensitive enabled filter
internal/radar/radar.go      — silent errors → logger.Error/Warn
cmd/worker/main.go           — silent error → logger.Error
+ 14 arquivos formatados com gofmt -w
```

## 📂 Commits

- v1.3.3 (commit `921ee8c`): S04 fix + LoadCriticas + filtro enabled
- v1.3.4 (commit local, este): silent errors + case-insensitive + healthz uptime

---

**Próxima validação (v5):** quando introduzirmos testes unitários em Sprint 5,
rodar `go test -race` + `go test -cover` pra detectar padrões de race conditions
que 4 passadas manuais não conseguem pegar.