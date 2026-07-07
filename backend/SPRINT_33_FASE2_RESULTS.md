# Sprint 33 Fase 2 — RESULTS

> **Data:** 2026-07-06
> **Sprint:** 33 Fase 2 de N
> **Tipo:** minor (+28 regras 3050 — 14 S + 14 I)
> **Versão:** v3.34.0 → **v3.34.1**

## ✅ Status

Shipped. 56 regras 3050 totais (28 A/S Fase 1 + 28 S/I Fase 2). Cobertura 32.9%.

## 📦 Entregas vs planejado

| Item | Planejado | Entregue |
|---|---|---|
| 14 Sistemáticas S15-S28 | ✅ | ✅ (14/14) |
| 14 Individuais/Cruzadas I01-I14 | ✅ | ✅ (14/14) |
| `Builtin3050` atualizado | ✅ | ✅ (56 total) |
| Testes table-driven | ✅ | ✅ (17 funções) |
| Self-verify em testes | ✅ | ✅ (1 bug fix in-loop) |

## 📊 Métricas finais

| Métrica | Pré | Pós |
|---|---|---|
| Regras 3050 | 28 | **56** |
| Cobertura catálogo 3050 | 16.5% | **32.9%** |
| Coverage `internal/audit/rules` | 72.9% | **72.1%** (-0.8pp) |
| Test functions total 3050 | 17 | **34** |
| Files novos | 0 | **1** (3050_fase2_test.go) |
| LOC Go adicionados | 0 | ~470 (3050.go delta + 3050_fase2_test.go) |
| Packages PASS -race | 23/23 | **23/23** |
| Stress 50 goroutines | mantida | **3/3 PASS** |
| vet + gofmt | clean | **clean** |

## 🧪 Testes (17 funções novas, ~50 sub-tests)

### Sistemáticas (8)

- TestS15_DataBaseValida (5 cases: 2009-2030 range)
- TestS16_NmContatoLength (3 cases: ≤100 chars)
- TestS17_TelContatoFormato (5 cases: 10-11 dígitos)
- TestS18_VlrConcessoesZeroTxJurosZero (3 cases: ambos zero OU ambos preenchidos)
- TestS19_TxJurosZeroVlrConcessoesPos (2 cases)
- TestS20-22_TxEncOperZeroVlrConcessoesPos / PrzDecZero / PrzDecPos (smoke)
- TestS23_PrzMedCondicional (3 cases: condicional sldCarAtiva)
- TestS25_CNPJNaoZero (smoke: 00000000 placeholder)
- TestS26_CodigoEncargoTipoCliUnico (2 cases: dedup)
- TestS27-28_NaoNeg (smoke: sldBaiPrejuizo, qtdNovContratos)

### Individuais/Cruzadas (6)

- TestI01-02_CapGir (6 cases: ≤365, >365 boundaries)
- TestI03-06_CredPesNaoConsignado (helper doc3050ComCredPes, 4 regras)
- TestI07-10_PrzLimites (smoke: <30, >5000, <1)
- TestI11-14_SldVlrLimites (smoke: <R$1000, >R$1T)

### Stubs + Integration (3)

- TestS24_StubReturnsNil (severity "I" check)
- TestBuiltin3050_Fase2TotalRulesIs (56 regras)
- (Fase 1 TestBuiltin3050_TotalRulesIs atualizado para 56)

## 🐛 Bugs encontrados pelos próprios tests (2, ambos fechados in-loop)

1. **S24 compile error: composite literal em if statement.** `if err := X{}.Apply(); err != nil` — Go parser confunde. Fix: extrair `rule := X{}` antes do if.
2. **I03-I06 semântica errada: subMods incluía crdPesNaoConsignado.** Soma esperada 700k, calculada 1.4M (incluindo a si mesma). Fix: `subMods` = apenas `aquVeiculos/aquOutBens/arrMerVeiculos/arrMerOutBens`. crdPesNaoConsignado é AGREGADA, não se auto-inclui.

## 🎯 Conformidade vs plano

| Decisão | Status |
|---|---|
| D-24 (Rule3050 interface paralela) | ✅ mantida |
| D-25 (Modalidade achatada) | ✅ mantida |
| D-26 (parser best-effort) | ✅ mantida |
| D-27 (stubs severity "I") | ✅ mantida |

Todas decisões Fase 1 carregadas sem mudança. Fase 2 = pure regra-add sobre mesma arquitetura.

## ⏭️ Próxima sprint (Fase 3)

**Sprint 33 Fase 3 — Audit3050 Header avançado + cruzadas complexas:**

Carry-over (stubs a implementar):
- S09 (DiasUteis) — calendário BACEN
- S13 (Último dia útil) — periodicidade
- S29-S44 — Sistema adicionais

Novas (I15-I28, H10-H15):
- 14 Individuais adicionais (sub-modalidades específicas: desDuplicatas, desCheques, vendor, compror, etc)
- 6 Header (encoding, espaços, length max)

**Alvo:** 56 → 90+ regras 3050 (cobertura 53%+).

## 📁 Arquivos

```
backend/internal/audit/rules/3050.go                 (+S15-S28 +I01-I14 +Builtin3050 56 regras)
backend/internal/audit/rules/3050_fase2_test.go      (NOVO — 17 testes table-driven)
CHANGELOG.md                                          (entry v3.34.1)
```