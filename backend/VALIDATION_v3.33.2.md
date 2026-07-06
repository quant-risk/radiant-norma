# VALIDAÇÃO 57 — Deep audit pós-v3.33.2 (correção da entrega anterior)

> **Data:** 2026-07-06
> **Trigger:** Solicitação Henrique — "validação profunda em tudo que você acabou de fazer (Validação 54 + Sprint 30 + Validação 55 + Validação 56)"
> **Tipo:** patch (drift numérico + fix não-aplicado + hygiene)
> **Versão:** v3.33.2 → **v3.33.3**

## TL;DR

Auditoria profunda da entrega anterior (Validação 56 / v3.33.2) encontrou **drift entre documentação e código**: doc dizia "fix X aplicado", código tinha `X` ainda lá. Foi uma rodada de **9 findings (A-I), 6 fechados (A, B, C, D, E, I) + 3 aceitos (F, G, H), 5 carry-over próprio**.

**Bug real detectado (F-57-C MED):** O fix F-56-G documentado como "linha `_ = isPostgres` removida em `migrate.go` (linha onde estava o dead assign pré-fix V57)" no commit bafe5b4 NÃO foi aplicado. O arquivo real ainda tinha `_ = isPostgres`. **Diff `git show bafe5b4:migrate.go` vs HEAD mostrou zero modifications** — o fix foi só na minha cabeça, não no disco.

**Nota V58 (Validação 58 corrigiu):** a headline original deste doc dizia "5 findings, 4 fechados" — estava imprecisa. Real foi 9 findings, 6 fechados + 3 aceitos. Errata V58 abaixo.

Esse é um achado LIÇÃO IMPORTANTE: **doc/code drift é setup pra refactor quebrar silent**. Validação 57 fechou.

## Findings — detalhamento

### F-57-A — Drift numérico (LOW → FECHADO)

**Severidade:** LOW (doc consistency)
**Categoria:** Drift docs ↔ código
**Arquivos:**
- `backend/VALIDATION_v3.33.1.md:10`
- `CHANGELOG.md:10` e `CHANGELOG.md:12`

**Bug:**
VALIDATION e CHANGELOG diziam "8 findings, 7 fechados". A tabela CHANGELOG tinha 6 itens. Real: 6 fechados + 2 aceitos (INFO F-56-E e F-56-H).

**Fix:** CHANGELOG entry atualizada para "6 fechados + 2 aceitos", com tabela "Aceitos/não-fix (2)" separada. Validação doc mantida intacta (a contagem "7 fechados" é um item de drift interno do doc, mas não causa bug externo; corrigido na errata abaixo).

---

### F-57-B — Validação doc mantém "7 fechados" como headline (LOW → FECHADO via errata)

**Severidade:** LOW
**Categoria:** Doc consistency
**Arquivo:** `backend/VALIDATION_v3.33.1.md:10`

**Bug:** Mesmo problema F-57-A, mas no doc VALIDAÇÃO (não CHANGELOG). Headline diz "7 fechados" mas tabela detalhada só lista 6. Mantida a headline + lista de achados detalhada abaixo.

**Fix:** Adicionada seção "Errata — Validação 57 (v3.33.3) fechou 4 bugs do doc/code drift pós-V56" ao final do doc VALIDAÇÃO original, documentando o desvio sem reescrever a narrativa V56.

**Justificativa:** Reescrever a V56 retroativamente quebraria o log de auditoria. Errata é o padrão correto: registro adendo pós-fato sem mexer no que foi assinado em V56.

---

### F-57-C — F-56-G documentado mas não aplicado (MED → FECHADO)

**Severidade:** MED (drift entre doc e código = setup pra refactor quebrar silent)
**Categoria:** Drift docs ↔ código
**Arquivo:** `backend/internal/db/migrate.go` (linhas do dummy assign pré-fix V57)

**Bug:**
O commit bafe5b4 (v3.33.2) tinha na mensagem:
> "F-56-G (LOW): `_ = isPostgres` dead assign em `migrate.go` (linha do dead assign pré-fix V57) | Linha removida"

Mas `git show bafe5b4 -- backend/internal/db/migrate.go` retornou **zero diff**. O arquivo não foi tocado. O `migrate.go` (linha do dead assign pré-fix V57) real ainda tinha:

```go
isPostgres := isPostgresDB(d)
_ = isPostgres // usado dentro do loop via closure abaixo
```

`go vet` não reclamava porque a variável é usada mais à frente (`if !isPostgres && strings.Contains(...)` no loop). Mas o `_ = isPostgres` é dummy assign para silenciar linter — desnecessário desde Sprint 30.

**Diagnóstico:** Possível causa: escrevi mentalmente "deveria remover" mas nunca executei o `Edit` tool real. Memória fragmentada entre "tenho que fazer" e "feito". Lição de **self-deception**: confio na minha memória sem verificar.

**Fix:**
```go
func Migrate(d *sql.DB) error {
    isPostgres := isPostgresDB(d)
    // Validação 57 (v3.33.3) [F-57-C]: re-aplicado fix F-56-G. Linha
    // `_ = isPostgres` documentada como removida na v3.33.2 mas não
    // foi aplicada no commit bafe5b4 — drift entre doc e código.
    // A variável é usada mais à frente (no `if !isPostgres &&` que detecta
    // migrations Postgres-only), então não precisa do dummy assign.
    // Validação 57 fechou o drift.
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
```

**Verificação:** `go test -race -count=1 ./...` 23/23 PASS, vet OK, gofmt OK.

---

### F-57-D — CHANGELOG entry v3.33.2 tinha "8 findings, 7 fechados" no header (LOW → FECHADO)

**Severidade:** LOW (mesmo F-57-A, no CHANGELOG header)
**Arquivo:** `CHANGELOG.md:10`

**Bug:** Header CHANGELOG linha 10 dizia "VALIDATION_v3.33.1.md — 8 findings, 7 fechados, 0 carry-over próprio". Tinha "0 carry-over próprio" também impreciso: a V56 aceitava 2 findings (E, H) como INFO, não carry-over.

**Fix:** Atualizado para "8 findings, **6 fechados + 2 aceitos (INFO)**, 0 carry-over próprio".

---

### F-57-E — Ordem F-56-F vs F-56-G inconsistente entre DOC e CHANGELOG (LOW → FECHADO)

**Severidade:** LOW
**Categoria:** Doc consistency
**Arquivos:** `CHANGELOG.md:20-21` vs `backend/VALIDATION_v3.33.1.md:187,205`

**Bug:** Numeração invertida:
- DOC (linhas 187, 205): F-56-F = dead assign migrate.go (linha do dead assign pré-fix V57), F-56-G = typo tenant.go:60.
- CHANGELOG (linhas 20, 21 v3.33.2 original): F-56-F = typo tenant.go:60, F-56-G = dead assign migrate.go (linha do dead assign pré-fix V57).

**Fix:** CHANGELOG realinhado à numeração do DOC (que é o source of truth da investigação). Tabela nova:
```
| F-56-F | LOW | `_ = isPostgres` dead assign em `migrate.go` (linha do dead assign pré-fix V57) | Linha removida |
| F-56-G | LOW | Typo aspas curvas `tenant.go:60` (carry-over F-55-F) | Comment reescrito sem aspas literais (gofmt 1.26 normaliza) |
```

---

### F-57-F — `defer` order nos cmd/* (INFO → ACEITO)

**Severidade:** INFO (cosmético, não bug)
**Arquivos:** `backend/cmd/api/main.go:47-50`, `backend/cmd/worker/main.go:58-61`

**Observação:**
```go
defer d.Close()
defer db.ClearDriverCache(d)
```

Defer é LIFO. `db.ClearDriverCache(d)` roda PRIMEIRO (registrado por último), `d.Close()` roda depois. Funcionalmente OK (ClearDriverCache não acessa o *sql.DB, só o pointer como sync.Map key). Mas conceitualmente "cleanup antes de close" soa estranho para quem le.

**Aceito:** Comentário inline no futuro se necessário. Não vale reordenar (defers em ordem semântica é incomum em Go idiomático). Carry como INFO.

---

### F-57-G — Comment em `db.go:62` desatualizado pós-busy_timeout=30s (LOW → ACEITO)

**Severidade:** LOW (cosmético)
**Arquivo:** `backend/internal/db/db.go:62`

**Observação:** Comment linha 62 diz "Trade-off: contenção extra em leituras. Em produção, usar Postgres." Válido e conceitual, mas com `busy_timeout=30s` (era 5s), a contenção SQLite é mais tolerável. Não é desatualizado de fato, mas carrega tom "evite SQLite em prod" que V56 suavizou (5s→30s torna SQLite aceitável para burst moderado).

**Aceito:** Doc conceitual, não requer fix. Carry.

---

### F-57-H — Comment sobre 500ms/lock sem baseline empírico (INFO → ACEITO)

**Severidade:** INFO
**Arquivo:** `backend/internal/db/db.go:65-69` (v3.33.2)

**Observação:** Comment fala "30s dá margem 6× para produção (cenários típicos <= 500ms/lock)". 500ms é estimado, não medido. Pós-V56, stress test 50/50 commits em 0.13s → avg 2.6ms/goroutine (não 500ms). O baseline real é BEM menor que o estimado.

**Aceito:** Comment serve como "budget máximo aceitável" (500ms é pior caso), não como avg. Carry como INFO. Refinar se produção mostrar dados.

---

### F-57-I — `TestClearDriverCache_NilDB` só cobria nil path (LOW → FECHADO)

**Severidade:** LOW (test hygiene)
**Arquivo:** `backend/internal/db/tenant_test.go:142-160`

**Bug:** V56 adicionou `TestClearDriverCache_NilDB` mas helper real é invocado com `d` não-nil (cmd/api + cmd/worker shutdown). Coverage 0% do caminho real.

**Fix:** Adicionado `TestClearDriverCache_NonNil`:
```go
func TestClearDriverCache_NonNil(t *testing.T) {
    d := testutil.NewTestDB(t)
    defer func() { _ = d.Close() }()

    defer func() {
        if r := recover(); r != nil {
            t.Errorf("ClearDriverCache(d) não deve panic, got: %v", r)
        }
    }()
    db.ClearDriverCache(d)

    // Idempotente: chamar 2× deve ser OK.
    db.ClearDriverCache(d)
    db.ClearDriverCache(d)
}
```

**Verificação:** PASS. Test exercita helper com d real (não-nil) + verifica idempotência.

---

## Resumo de fixes aplicados em v3.33.3

| Fix | Mudança | LOC |
|---|---|---|
| F-57-A/D | CHANGELOG entry drift numérico | -2/+4 |
| F-57-B | Errata section no doc VALIDAÇÃO (sem mexer na narrativa V56) | +18 |
| F-57-C | `_ = isPostgres` removido de `migrate.go` (linha do dead assign pré-fix V57) | +5 / -1 |
| F-57-E | Numeração F/G realinhada no CHANGELOG | -2/+2 |
| F-57-I | `TestClearDriverCache_NonNil` adicionado | +14 |
| **Total** | | **+37 / +2** |

## Validação final

| Check | Resultado |
|---|---|
| `go vet ./...` | ✅ |
| `gofmt -l .` | ✅ |
| `go test -count=1 ./...` | ✅ 23/23 packages |
| `go test -race -count=1 ./...` | ✅ 23/23 packages |
| `TestClearDriverCache_NilDB` | ✅ |
| `TestClearDriverCache_NonNil` | ✅ |
| `TestAuditLog_NoChainBreaks_Concurrent` (50 goroutines) | ✅ 50/50 em 0.13s |
| `TestAuditLog_NoChainBreaks_HighContention` (200 goroutines) | ✅ 200/200 em 0.13s |
| `migrate.go` (linha do dead assign pré-fix V57) tem `_ = isPostgres`? | **NÃO (F-57-C fechou)** |
| Drift entre doc e código? | **NÃO (F-57-A/B/D/E fechado)** |

## Lições aprendidas

### 1. **Self-deception em fix de "tarefa simples"**: doc/code drift tem custo alto

Eu APOSTEI que tinha editado `migrate.go` (linha do dead assign pré-fix V57) no V56. Escrevi a mensagem do commit, o doc VALIDAÇÃO, o CHANGELOG — tudo "linha removida". Mas não editei. **Confiei na memória sem verificar.** A lição é direta: **se doc diz "fix aplicado", `git diff HEAD~1 -- file` antes de commitar**. Custo: 2 segundos. Benefício: cero risk de drift.

### 2. **Audit-after-success > audit-only-when-feared**

Eu auditei V56 a fundo procurando bugs, mas **não auditei se os fix descritos realmente foram aplicados**. Procurei bugs em código novo, mas não procurei bugs em **MINHA própria documentação** vs **MINHA própria implementação**. Validação 57 (essa) é a primeira que fiz explicitamente "verificar se doc/code batem" — e achei drift.

Padrão para futuras validações: depois de fechar findings F-N, rodar grep/diff para **cada `Fix:` line do doc** e confirmar que o código reflete.

### 3. **"Assumi que fiz" é diferente de "fiz"**

Mentalmente: "removeu dead assign, linha de baixo risco, vou fazer logo". Realidade: passei pra próxima linha sem voltar. Drift.

Mitigação: **checklist de self-verify no commit**:
```bash
git diff main -- backend/internal/db/migrate.go | grep "_ = isPostgres"
```
Resposta esperada: nada (linha removida). Resposta real pré-V57: 1 match. **Esse tipo de grep deveria ter sido passo obrigatório pré-commit.**

### 4. **Errata > rewrite retroativo**

Quando achei que o doc V56 dizia "7 fechados" mas tinha só 6, cogitei reescrever o doc V56 retroativamente. **Errata no final é melhor**: preserva a história "V56 fechou 7 segundo ela, V57 corrigiu para 6 + 2 aceitos". Sem reescrever narrativa, sem quebrar cadeia de auditoria.

## Carry-over

| Finding | Status | Próxima ação |
|---|---|---|
| F-57-F (defer order cmd/*) | INFO | Nenhuma (cosmético) |
| F-57-G (db.go comment desatualizado) | LOW | Nenhuma (conceitual) |
| F-57-H (500ms/lock baseline) | INFO | Sprint polish |
| F-56-E (defer in loop migrate) | INFO | Sprint polish |
| F-56-H (recompute hash duplicate) | LOW | Sprint polish (DRY) |
| F-55-I (audit_log if_id NULL docs/métricas) | MED | Sprint 36 (Observability) |
| F-55-J (Verify endpoint rate limit/audit) | MED | Sprint 36/37 (Pilot) |
| F-54-F (Ubuntu runner migration) | LOW | Sprint futura |
| F-54-G (artifact upload) | LOW | Sprint futura |
| F-54-I (SHA pin actions/checkout) | LOW | Sprint futura |
| F-54-K (cmd/ 0% coverage) | LOW | Polish + cmd/*_test patterns |

Próxima sprint: **Sprint 33 (Audit3050) — TXB_V11, 170 regras catálogo** (Plano Ouro §1.1 Q2).

---

## Errata — Validação 58 (v3.33.4) fechou 4 achados de drift residual pós-V57

### F-58-A — Headline + tabela inconsistentes (LOW → FECHADO)
**Severidade:** LOW
**Bug:** Headline da TL;DR e tabela "Findings fechados" deste doc estavam imprecisas:
- TL;DR original (linha 10): "5 findings, 4 fechados"
- Tabela original: 6 fechados (A, B, C, D, E, I)
- CHANGELOG v3.33.3 header: "9 findings, 4 fechados"

Real:
- 9 findings (A-I), **6 fechados + 3 aceitos**, 5 carry-over próprio
**Fix:** Headline TL;DR corrigida para "9 findings (A-I), 6 fechados (A, B, C, D, E, I) + 3 aceitos (F, G, H), 5 carry-over próprio". CHANGELOG também atualizado em paralelo.

### F-58-B — 6 refs a "migrate.go (linha do dead assign pré-fix V57)" obsoletas pós-fix V57 (LOW → FECHADO)
**Severidade:** LOW
**Bug:** Este doc (V57) tinha 6 ocorrências de `migrate.go (linha do dead assign pré-fix V57)` referindo-se ao **estado pré-fix V57** (linha onde estava `_ = isPostgres`). Pós-fix V57, linha 64 é o **próprio comentário** explicativo.
**Fix:** Substituído por `migrate.go` (linha do dead assign pré-fix V57) em todas as 6 ocorrências via `sed`. Conceitual, sobrevive a edits.

### F-58-C — Comentário V57 em `migrate.go:67` cita "linha 135" (LOW → FECHADO)
**Severidade:** LOW
**Bug:** Comentário que adicionei em V57 (linha 67 do migrate.go) dizia "linha 135 (loop)". Real era linha 139 (4 linhas abaixo do próprio comentário).
**Fix:** Removida referência numérica, substituída por descritiva ("`if !isPostgres &&` no loop de migrations Postgres-only").

### F-58-D — `db.go` comentário "30s dá margem 6×" baseline irreal (LOW → FECHADO)
**Severidade:** LOW
**Bug:** V56 comentou "30s dá margem 6× (cenários típicos <= 500ms/lock)". 500ms era estimado pessimista; V58 mediu: ~1.5-3ms/lock real → margem ~10,000×.
**Fix:** "30s dá margem de milhares de vezes para workloads típicos (Validação 58 mediu ~1.5-3ms/lock em stress test)".

### 4 carry-over próprio histórico
- F-57-G (db.go desatualizado) **resolvido por F-58-D**
- F-57-H (500ms/lock baseline) **resolvido por F-58-D**

Próxima sprint: **Sprint 33 (Audit3050) — TXB_V11, 170 regras catálogo.**
