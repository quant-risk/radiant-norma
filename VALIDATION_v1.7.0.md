# VALIDATION v1.7.0 — Sprint 7b (Regras 3040 expandidas)

> **Status:** ACCEPTED
> **Data:** 2026-07-03
> **Trigger:** Henrique pediu fechamento Sprint 7 (regras + frontend).
> Sprint 7b continua execute without pause.
> **Versão:** v1.7.0 (minor — coverage expandida)

## 🎯 Resumo

Backend coverage de regras 3040 expandido **de 30 → 60 regras**.
60 regras: 5 raw + 55 tipadas (B06-B25, F01-F15, C01-C10, S01-S10).

**Stats:**
- 281 → 301 tests passing (+20)
- Fuzz test: 427k execs em 2s, zero panics/deadlocks
- vet-clean, race-clean, build-clean

## Mudanças

### 30 regras novas (Sprint 7b)

**B16-B25 (10) — Básicas expandidas:**
- B16 Totalizadores coerentes (TotalCli == soma(QtdCli))
- B17 DtBase formato YYYY-MM-DD
- B18 TpArq deve ser F ou S
- B19 Email formato
- B20 Telefone formato (XX) XXXXX-XXXX
- B21 CNPJ raiz 8 dígitos
- B22 NomeResp não vazio
- B23 Mínimo 1 Agreg
- B24 DtBase não futura (até 2030)
- B25 QtdOp >= 1 por Agreg

**F06-F15 (10) — Formato expandido:**
- F06 ClassOp A-H
- F07 Mod 2-4 dígitos
- F08 NatuOp 01/02
- F09 UF válida (27 siglas brasileiras)
- F10 VincME S/N
- F11 PrzProvm S/N
- F12 TpCli 1=PF / 2=PJ
- F13 DesempOp numérico
- F14 FaixaVlr numérico
- F15 OrigemRec 1-3 dígitos

**C06-C10 (5) — Campos Obrigatórios expandidos:**
- C06 ClassOp C-H requer ProvConsttd
- C07 DesempOp != "00" com vencimentos > 0
- C08 Tel preenchido requer Email
- C09 NatuOp=01 requer QtdCli
- C10 QtdOp>0 requer ClassOp

**S06-S10 (5) — Semânticas expandidas:**
- S06 QtdOp zero warning
- S07 Mod=0213 requer ClassOp E-H (cheque especial high risk)
- S08 PF com ClassOp A é suspeito
- S09 Soma V110..V165 ≈ QtdOp (10% tolerance)
- S10 NatuOp=01 com VincME=N (próprias não devem ser moeda estrangeira)

### Fuzz testing

`internal/crossdoc/rules/iter_fuzz_test.go`:
- 427167 execs em 2 segundos
- 1 new interesting case descoberto
- ZERO panics ou deadlocks em:
  - XML vazio
  - CDATA com nested Mod
  - Entities (5 &lt; 10 &amp; ok)
  - Control chars
  - 1.5MB spam
  - Case wrong (`agreg` lowercase)
  - Mixed attrs (Mod + ExtraAttr)

### Catalog documentation

`docs/rules-3040-catalog.md`:
- 60 regras catalogadas (todas com code/severity/sheet/desc/example)
- Resumo por categoria + sprint origem
- Vetor mapeamento ao tests

## Acceptance

- ✅ 30 regras novas implementadas + testadas
- ✅ 301 tests passing
- ✅ Fuzz test 427k execs, no panic
- ✅ Documentação catálogo completa
- ✅ vet/race/build clean

## Próximo: Sprint 7c (Frontend Next.js)
