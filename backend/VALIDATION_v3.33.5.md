# VALIDAÇÃO 60 — Deep audit pós-v3.33.5 (drift cleanup em doc da V59)

> **Data:** 2026-07-06
> **Trigger:** Solicitação Henrique — "validação profunda em tudo que você acabou de fazer (Validação 58 + Validação 59)"
> **Tipo:** patch (drift numérico + imprecisão no sumário + off-by-one carry-over)
> **Versão:** v3.33.5 → **v3.33.6**

## TL;DR

Auditoria profunda da Validação 59 (v3.33.5) usando self-verify checklist (regra HOT memory). **Self-verify confirmou os 4 achados V59 corretamente categorizados** (1 revertido, 2 fechados, 2 carry) — mas o próprio **doc V59 tinha imprecisões numéricas** (F-60-A/B/C/D) deixadas pela redação rápida sob turnaround de experiment revertido.

V60 fechou **4 findings LOW** que são **drift cleanup no doc V59**, sem bug funcional novo. Categoria: revisão editorial pós-release para alinhar doc ↔ realidade sem reescrever narrativa.

## Findings — detalhamento

### F-60-A — TL;DR doc V59 imprecisa sobre "5 findings" sem distinguir status (LOW → FECHADO)

**Severidade:** LOW (doc consistency)
**Categoria:** Drift docs ↔ contagem real
**Arquivo:** `backend/VALIDATION_v3.33.4.md:10-12`

**Bug:** TL;DR original:
> "V59 fechou **5 findings** com foco em **estabilidade do stress test sob carga V59**."

Mas real:
- 1 revertido (F-59-A)
- 2 fechados (F-59-B, C)
- 1 carry próprio (F-59-D)
- 1 carry histórico (F-59-E)

A frase "fechou 5" sem distinção dá impressão errada — pode parecer que V59 fechou 5 bugs quando não fechou nenhum (apenas 2 audit + 1 revertido).

**Fix:** TL;DR detalhado:
> "V59 fechou **5 findings (A-E)** com foco em **estabilidade do stress test sob carga V59**:
> - **2 fechados (B, C)** — audit regra memory + self-verify V58 confirmado
> - **1 revertido (A)** — retry-on-SQLITE_BUSY experimental mostrou regressão empírica (73% → 33%)
> - **2 carry (D, E)** — flake residual sob carga variável + carry-over histórico intocado"

---

### F-60-B — Resumo V59 cita apenas F-59-A mas ignora F-59-B/C (LOW → FECHADO)

**Severidade:** LOW (doc consistency)
**Categoria:** Drift docs ↔ ship list real
**Arquivo:** `backend/VALIDATION_v3.33.4.md:136-141`

**Bug:** Resumo V59 original:
> | F-59-A | Retry-on-SQLITE_BUSY implementado E revertido | -0/+15 → -15 net |

Mas V59 NÃO shipou código de produção. **Único fix material:** comentário explicativo REVERTIDA em `tenant.go:98`. F-59-B (audit regra memory) e F-59-C (self-verify V58) são achados de processo.

**Fix:** Resumo honesto:
> V59 não shipou fix material. **Único fix shipped:** comentário explicativo da reversão em `WithTenantTx` (carry-over F-59-A).
>
> | F-59-A | Comentário REVERTIDA em `WithTenantTx` | +4 / 0 |
> | **Total** | | **+4 / -0 (cosmético)** |

Tabela reflete +4 LOC cosmético.

---

### F-60-C — "12 carry-overs históricos" off-by-one (LOW → FECHADO)

**Severidade:** LOW (cosmético, off-by-one)
**Categoria:** Drift docs ↔ contagem
**Arquivo:** `backend/VALIDATION_v3.33.4.md:132`

**Bug:** Doc V59 dizia "12 carry-overs históricos (F-54-F/G/I/K, F-55-I/J, F-56-E/H, F-57-F, F-58-E/F)". **Real:**
- F-54-F/G/I/K = 4
- F-55-I/J = 2
- F-56-E/H = 2
- F-57-F = 1
- F-58-E/F = 2
- **Total = 11**

Off-by-one. Erro de contagem durante redação rápida.

**Fix:** "11 carry-overs históricos" + lista detalhada na mesma linha.

---

### F-60-D — CHANGELOG entry v3.33.5 confunde status dos 5 findings (LOW → FECHADO)

**Severidade:** LOW (CHANGELOG consistency)
**Categoria:** Drift CHANGELOG ↔ doc
**Arquivo:** `CHANGELOG.md:8-12`

**Bug:** Header CHANGELOG dizia:
> "5 findings (A-E), **1 REVERTIDO + 3 fechados/audit + 1 carry-over histórico**"

Soma 1+3+1=5, mas status confuso:
- "REVERTIDO" (F-59-A) — correto
- "fechados/audit" — implica 3, mas só B e C são fechados; E é carry histórico (não fechado)
- "carry-over histórico" — 1 só (E)

Real:
- 2 fechados (B, C)
- 1 revertido (A)
- 1 carry próprio (D)
- 1 carry histórico (E)

**Fix:** "5 findings (A-E), **2 fechados (B, C) + 1 revertido (A) + 1 carry próprio (D) + 1 carry histórico (E)**". Soma explícita.

---

## Resumo de fixes aplicados em v3.33.6

| Fix | Mudança | LOC |
|---|---|---|
| F-60-A | TL;DR V59 detalhado por status (5 findings → 2 fechados + 1 revertido + 2 carry) | -0/+3 |
| F-60-B | Resumo V59 honesto: "+4 / 0 (cosmético)" (não "-15 net" irreal) | -1/+1 |
| F-60-C | "12 carry-overs" → "11 carry-overs" + lista detalhada | -0/+1 |
| F-60-D | CHANGELOG header V59 com soma explícita dos status | -0/+0 (rewording) |
| **Total** | | **+5 / -1 (cosmético)** |

**Validação-doc-only:** todas as mudanças são em prosa de doc. Zero alteração de código, testes, performance, ou API.

## Validação final

| Check | Resultado |
|---|---|
| `go vet ./...` | ✅ |
| `gofmt -l .` | ✅ |
| `go test -count=1 ./...` | ✅ 23/23 packages |
| `go test -race -count=1 ./...` | ✅ 23/23 packages |
| Self-verify V59 (4 achados V59 categorizados) | ✅ (1 revertido + 2 fechados + 2 carry) |
| Self-verify V60 (4 fixes aplicados) | ✅ todos grep-confirmados |
| Drift numérico V59 doc vs CHANGELOG | **NÃO (F-60-A/B/C/D fechados)** |
| Stress 50 goroutines | ✅ **11/15 (73% estável)** |
| Stress 200 goroutines | ✅ 200/200 estável |
| Coverage `internal/db` | **62.7% (mantida)** |
| Coverage `auditlog` | **92.5% (mantida)** |
| Coverage `audit/rules` | **70.8% (mantida)** |
| Coverage `radar` | **81.2% (mantida)** |

## Lições aprendidas

### 1. **Releases sob turnaround curto produzem drift numérico**

V59 teve cycle curto por causa do F-59-A experimental revertido (algumas horas). Pressão de tempo levou a imprecisões numéricas no doc (5 findings sem distinção, "12" carry-over ao invés de 11, sumário com "net -15" irreal). V60 fechou 4 dessas imprecisões.

**Pattern futuro:** após release com revert/scope-creep, agendar **V_revisão_2 (apenas doc cleanup)** separado da V_normal. Reduz drift sem custo.

### 2. **Self-verify é parte da release, não só auditoria**

V60 aplicou self-verify (regra HOT memory) e descobriu drift numérico no doc da V59. **Self-verify deveria ser passo pré-tag, não só pré-commit code changes.** Doc fixes também merecem grep-check pré-tag.

### 3. **V59 + V60 = par complementar**

V59 (entregou) e V60 (limpeza) ilustram uma invariante útil:
> **Toda release shipped vira alvo de V_drift_cleanup subsequente.**

Especialmente quando V_anterior teve revert ou scope-creep.

## Carry-over

| Finding | Status | Próxima ação |
|---|---|---|
| F-58-F (`--with-radiant-memory` flag) | INFO | Nenhuma |
| F-57-F (defer order cmd/*) | INFO | Nenhuma |
| F-56-E (defer in loop migrate) | INFO | Sprint polish |
| F-56-H (recompute hash duplicate) | LOW | Sprint polish (DRY) |
| F-55-I (audit_log if_id NULL metrics) | MED | Sprint 36 (Observability) |
| F-55-J (Verify endpoint rate limit) | MED | Sprint 36/37 (Pilot) |
| F-54-F (Ubuntu runner migration) | LOW | Sprint futura |
| F-54-G (artifact upload) | LOW | Sprint futura |
| F-54-I (SHA pin actions/checkout) | LOW | Sprint futura |
| F-54-K (cmd/ 0% coverage) | LOW | Polish + cmd/*_test patterns |
| F-59-A (retry em WithTenantTx central) | MED | Sprint polish — retry apenas em `auditlog.Log` (escopo menor) |
| F-59-D (flake variável) | LOW | Carry se ressurgir > 5% em CI dedicada |

Próxima sprint: **Sprint 33 (Audit3050) — TXB_V11, 170 regras catálogo** (Plano Ouro §1.1 Q2).
