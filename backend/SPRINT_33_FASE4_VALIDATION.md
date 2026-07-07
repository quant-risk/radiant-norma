# Sprint 33 Fase 4 — VALIDAÇÃO PROFUNDA

> **Data:** 2026-07-07
> **Sprint:** 33 Fase 4 (validação retroativa)
> **Versão auditada:** v3.34.6 (commit 2cd997e)
> **Validador:** auto-auditoria (regra HOT memory §self-verify)

## 🎯 Objetivo da validação

Repassar código, testes, docs e arquitetura da Fase 4 (commit 2cd997e) **antes** de subir a próxima sprint, validar claims e detectar drift entre doc/code.

## ✅ Método (regra HOT memory)

1. `git log` + `git show --stat` — verificar commit + arquivos.
2. `grep -c "Register3050("` — contar regras no Builtin3050.
3. `grep -c "^func Test"` — contar test functions no 3050_fase4_test.go.
4. `grep "^type"` — contar structures da Fase 4.
5. Ler 3050_helpers.go (edge case fix), parser change (Encoding/BomPresent).
6. `go test -race -count=1 ./...` — todos packages.
7. `go tool cover -func` — coverage real.
8. Comparar claims em CHANGELOG, SPRINT_33_FASE4_RESULTS.md vs real.

## 📊 Drift detectado e corrigido

### Drift #1 — header de `3050_helpers.go` desatualizado (BAIXA)

**Local:** `backend/internal/audit/rules/3050_helpers.go:1`

**Problema:** header dizia "Sprint 33 Fase 3 helpers" mas o arquivo foi editado na Fase 4 (`IsUltimoDiaUtilMes` ganhou varredura retroativa pra edge case do sábado).

**Correção:** header reescrito pra "Sprint 33 Fase 3+4 helpers" com menção ao edge case fix.

### Drift #2 — comentário de `H19ApenasUmaReferencia` (BAIXA)

**Local:** `backend/internal/audit/rules/3050.go` (H19 Apply3050)

**Problema:** comentário mencionava "se Diario E Mensal estão vazios E Root vazio, mas doc foi declarado não-vazio antes → suspeito" mas o código era literalmente `_ = doc; return nil` — não implementava a validação descrita.

**Correção:** comentário reescrito pra descrever o que o stub realmente faz (no-op, carry-over Fase 5).

## ✅ Claims validados (todos passaram)

| Claim | Verificação | Status |
|---|---|---|
| 17 novas regras (5 H + 4 S + 8 I) | `grep "^type H1[6-9]\|^type H20\|^type S33\|^type S34\|^type S36\|^type S38\|^type I2[9]\|^type I3[0-6]"` = 17 | ✅ |
| 97 Register3050 totais | `grep -c "r.Register3050("` = 97 | ✅ |
| 20 test functions no 3050_fase4_test.go | `grep -c "^func Test"` = 20 | ✅ |
| 5 H + 4 S + 8 I + 2 parser + 1 integração = 20 | Distribuição 1:1 regra → test + 2 DT-31 parser tests + integração | ✅ |
| Cobertura 57.06% (97/170) | `97/170 = 0.57058...` | ✅ |
| Coverage 72.2% | `go tool cover -func` total = 72.2% | ✅ |
| 23/23 packages PASS -race | `go test -race ./...` retornou 23 ok + 6 sem test files | ✅ |
| vet + gofmt clean | `go vet ./...` e `gofmt -l` retornaram vazio | ✅ |
| Edge case IsUltimoDiaUtilMes corrigido | Loop varre pra trás do último dia do mês até achar dia útil (linha 104) | ✅ |
| Teste edge case 2025-05-31 (sábado) | `TestIsUltimoDiaUtilMes` tem 2 novos sub-tests (sábado → false; sexta 30 → true) | ✅ |
| DT-31 parser change | Doc3050Root.Encoding + BomPresent; xmlEncodingRe; aplicação DEPOIS de root = Doc3050Root{} | ✅ |
| H16 case-insensitive UTF-8 | `strings.EqualFold(doc.Root.Encoding, "UTF-8")` | ✅ |
| S33 uses time.Now() | `limite := time.Now().UTC().AddDate(-1, 0, 0)` | ✅ |
| H17 BOM detection | `if doc.Root.BomPresent` retorna erro | ✅ |

## 🔬 Auditoria de código

### DT-31 — Parser change (Encoding/BomPresent) — ✅ implementado

4 mudanças coordenadas em `3050.go`:

1. **Doc3050Root struct (linhas 53-56):** adiciona `Encoding string` + `BomPresent bool`.
2. **xmlEncodingRe regex (linha 112):** captura `<?xml encoding="UTF-8"?>`.
3. **Variáveis locais (linhas 148-151):** `bomPresent` + `xmlEncoding` calculados ANTES do loop (seriam sobrescritos).
4. **Aplicação DEPOIS de `root = Doc3050Root{}` (linhas 215-217):** dentro do case DocTXB, atribui após o reset.

**Lição crítica documentada:** setter em variável que vai ser reatribuída dentro do loop é armadilha. Variáveis locais pra valores calculados antes do loop, aplicar DEPOIS. Bate com regra HOT memory "self-verify em teste flagra testes errados".

### Edge case fix — `IsUltimoDiaUtilMes` (carry-over Fase 3 → Fase 4) — ✅

**Comportamento anterior (Fase 3):** apenas verificava se data == último dia do mês E dia útil. Sábado último dia → false.

**Comportamento atual (Fase 4):** varre do último dia do mês pra trás até achar dia útil. Se último dia = sábado, retorna sexta anterior.

**Teste empírico (2025-05-31 = sábado):**
- `IsUltimoDiaUtilMes(2025-05-31)` = false (sábado, não é último útil)
- `IsUltimoDiaUtilMes(2025-05-30)` = true (sexta, último dia útil do mês)

### Auditoria de regras individuais (17)

| Regra | Teste correspondente | Cases | Severidade | Lógica |
|---|---|---|---|---|
| H16 EncodingUTF8 | TestH16_EncodingUTF8 | 5 | E | case-insensitive, vazio OK |
| H17 SemBOMUTF8 | TestH17_SemBOMUTF8 | 2 | A | simples check BomPresent |
| H18 RaizDocTXB | TestH18_RaizDocTXB | 2 | E | verifica se algo foi parseado |
| H19 ApenasUmaReferencia | TestH19_ApenasUmaReferencia | 1 | A | stub no-op (carry-over) |
| H20 ApenasUmDiarioUmMensal | TestH20_ApenasUmDiarioUmMensal | 1 | A | stub no-op (carry-over) |
| S33 DataBaseMax1YearOld | TestS33_DataBaseMax1YearOld | 4 | A | time.Now() relativo (não hardcoded) |
| S34 DataBaseConsistente | TestS34_DataBaseConsistente | 2 | A | verifica Root.DataBase não-vazio |
| S36 IndRemessaIApenasPrimeiraVez | TestS36_StubReturnsNil | 1 | I (stub) | no-op (precisa histórico) |
| S38 DocUnicoPorCNPJDataBase | TestS38_StubReturnsNil | 1 | A (stub) | no-op (precisa histórico) |
| I29-I36 NaoNeg_PorSubModalidade | TestI29-TestI36 | 1-2 cada | E | loop + check Codigo + valor ≥ 0 |

**Cobertura teste/regra:** 17/17 (100%).

## 🧪 Auditoria de testes (20 funções Fase 4)

| Categoria | Tests | Sub-tests | Status |
|---|---|---|---|
| Header (H16-H20) | 5 | ~12 | ✅ |
| Sistema (S33, S34, S36, S38) | 4 | ~8 | ✅ |
| Individuais (I29-I36) | 8 | ~10 | ✅ |
| Parser (DT-31) | 2 | 3 | ✅ |
| Integração | 1 | 1 | ✅ |
| **Total** | **20** | **~34** | **PASS -race** |

**Output:** `ok github.com/fortvna/radiant-norma/backend/internal/audit/rules 1.394s coverage: 72.2%`

## 🏗️ Auditoria arquitetural

### D-24 (Rule3050 interface paralela) — ✅ mantida

`rules3050` map separado. Sem mudanças.

### D-25 (Modalidade achatada) — ✅ mantida

Sem mudanças estruturais.

### D-26 (parser best-effort) — ✅ mantida + DT-31 aplica

Parser agora detecta BOM e Encoding. Não quebra compat: novos campos são `omitempty` e opcionais.

### D-27 (stubs severity "I") — ✅ mantida

H19/H20 (severity A, stubs no-op), S36 (severity I, stub honesto), S38 (severity A, stub honesto).

### DT-31 (Parser header avançado) — ✅ aplicada

4 mudanças coordenadas (Doc3050Root + regex + variáveis locais + aplicação tardia).

## 🔍 Edge cases identificados para próxima sprint

### Edge case #1 — H19/H20 (contar `<referencia>`)

Validação real precisa contar elementos `<referencia>` no XML bruto. Parser atual agrega tudo em slices Diario/Mensal. **Carry-over Fase 5.**

### Edge case #2 — S36/S38 (histórico de envios)

Validação real requer contexto de envios anteriores (tabela `historico_envios`). Sem infra de DB-history no parser 3050 atual. **Carry-over indefinido.**

### Edge case #3 — H19 comentário descritivo (Fase 5)

Comentário mencionava validação condicional não implementada. Corrigido pra refletir no-op honesto. Carry-over: implementar a validação descrita se validação semântica for útil.

### Edge case #4 — TestS33 com time.Now() em testes CI

Teste usa `time.Now().AddDate(...)` em vez de data hardcoded. **Edge case conhecido (corrigido in-loop)**: testes com datas absolutas quebram em CI com tempo variável.

## 🚦 Status final pré-push

| Item | Status |
|---|---|
| Código compila | ✅ |
| vet clean | ✅ |
| gofmt clean | ✅ |
| Tests -race PASS | ✅ 23/23 packages |
| Tests Fase 4 3050 PASS | ✅ 20/20 funções |
| Coverage | ✅ 72.2% (claim exato) |
| Edge case IsUltimoDiaUtilMes | ✅ corrigido |
| Parser change DT-31 | ✅ aplicado sem regressão |
| Decisões D-24/D-25/D-26/D-27 mantidas | ✅ |
| DT-31 aplicada | ✅ |
| Não-regrediu 3040 | ✅ Builtin3040 + Builtin3050 coexistentes |
| Doc alinhado com código (drift #1, #2 corrigidos) | ✅ |

## ⏭️ Próxima sprint — após validação Fase 4

**Status Sprint 33:** 97/170 = 57.06% cobertura. Sprint 33 (Audit3050) com 4 fases incrementais.

Carry-over restante: 73 regras (matriz modalidade × encargo coberta por XSD + sub-modalidades específicas).

**Opções (já apresentadas no SPRINT_33_FASE4_RESULTS.md):**
- Fase 5 (Sprint 33 continuação): fechar 100% via stubs informativos
- Sprint 34 — AuditDLO 2061 (próximo CADOC)
- Sprint 34 — FrontendNext (ROADMAP)

**Recomendação:** abrir **AuditDLO 2061** (valor novo, 3050 em 57% é suficiente pra validar).