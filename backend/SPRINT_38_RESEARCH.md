# Sprint 38 — Audit3040 Fase 4 (última) — RESEARCH

> **Data:** 2026-07-07
> **Sprint:** 38 (Audit3040 Fase 4 — fechamento do 3040 até teto de cobertura)
> **Pré-requisito:** v3.34.16 (V68 fechou drift do Sprint 37)
> **Marco esperado:** 221 → ~275+ regras 3040 (61.2% → 76%+ cobertura)
> **Esta é a última sprint do CADOC 3040.** Após Sprint 38, 3040 entra em "manutenção" e workstreams futuras focam em outros CADOCs.

## 🎯 Escopo Sprint 38 Fase 4

**+54 regras 3040** baseadas em:
1. **Campos Opcionais C71-C90** (20 regras — C-level expandido).
2. **Substituição Parcial SUB01-SUB15** (15 regras — substituição de remessas).
3. **Cross-doc básico X01-X10** (10 regras — validações cross-IF).
4. **Carry-over destravadas** (~9 stubs que agora têm parser suficiente).

Total: **+54 regras**. **221 → ~275** (cobertura 61.2% → **76.2%**).

## 📋 Regras prioritárias

### C71-C90 — Campos Opcionais expandidos (20 regras)

C71-C80 — C-level subset:
| Cod | Descrição |
|---|---|
| C71 | Inf 1301 (Comissão) obrigatória quando tipo operação = corretagem |
| C72 | Inf 1302 (Tarifa) obrigatória quando operação tem tarifa |
| C73 | Inf 1401 (Seguro) vinculada a operação habitacional |
| C74 | Inf 1501 (IOF) obrigatória quando aplica |
| C75 | Inf 1601 (Custo aquisição) quando cessão |
| C76 | Inf 1701-1799 (Garantias específicas) por tipo garantia |
| C77 | Inf 1801-1899 (Coobrigação específica) por tipo |
| C78 | Inf 1901-1999 (Reestruturação) por tipo |
| C79 | Inf 2001+ (Novos códigos) |
| C80 | Inf cross-ref (0307 ↔ 1201) — parcial |

C81-C90 — Operacao-specific:
| Cod | Descrição |
|---|---|
| C81 | DtContr <= DtBase (operação não pode ser no futuro) |
| C82 | DtVencOp >= DtContr (saneamento) |
| C83 | Valor positivo para operação ativa |
| C84 | Perc = 100 quando NatuOp = 01 (operação própria, sem coobrigação) |
| C85 | QtdParcelas >= 1 quando operação parcelada |
| C86 | Perc coobrigação <= 100 (saneamento) |
| C87 | DtVencOp - DtContr = prazo operação (consistência) |
| C88 | Valor principal + juros = valor contratado (sanity) |
| C89 | Garantia fidejussória exige avalista com CPF/CNPJ |
| C90 | Cessão (Inf=0307) tem cedente com CNPJ/CPF |

### SUB01-SUB15 — Substituição Parcial (15 regras)

| Cod | Descrição |
|---|---|
| SUB01 | TpArq=S (substituição) tem Remessa > 1 |
| SUB02 | TpArq=S tem Parte != última aceita |
| SUB03 | Documentos a substituir referenciados explicitamente |
| SUB04 | Substituição preserva operações não-listadas |
| SUB05 | Substituição só permite Inf=I03XX (substituível) |
| SUB06 | Substituição parcial tem no mínimo 1 operação |
| SUB07 | Substituição total (todas operações) marcada como TpArq=F |
| SUB08 | Histórico de substituições por remessa |
| SUB09 | Substituição não pode referenciar documento do mesmo período |
| SUB10 | Substituição tem CNPJ raiz = header |
| SUB11 | Substituição parcial preserva Cli não-listados |
| SUB12 | Substituição tem data <= DtBase + 30 dias |
| SUB13 | Substituição múltipla (Parte > 1) tem ordem preservada |
| SUB14 | Substituição de agregados cruzados |
| SUB15 | Substituição consolida histórico cross-IF |

### X01-X10 — Cross-doc básico (10 regras)

| Cod | Descrição |
|---|---|
| X01 | CNPJ raiz header = CNPJ raiz 3040 cross-doc |
| X02 | DtBase header coerente com DtBase 3040 |
| X03 | Operações 3040 individuais têm contraparte em 3042 (cross-doc) |
| X04 | Operações 3040 agregadas têm somatório em 3042 |
| X05 | Cli Cd único cross-doc 3040 + 3042 |
| X06 | IPOC único cross-doc 3040 + 3042 |
| X07 | Vencimentos 3040 <= Vencimentos 3042 (consistência) |
| X08 | ProvConsttd 3040 >= ProvConsttd 3042 (consistência) |
| X09 | Operações 3040 + 3042 = Operações 3050 (consolidação) |
| X10 | Modalidade 3040 = Modalidade 3042 (consistência cross-doc) |

### Carry-over destravadas (~9 stubs Sprint 36-37)

- **I15** — implementar limites PF com tabela default.
- **S78** — implementar ClassOp × Mod usando tabela default.
- **S84** — implementar CNPJ consolidado (versão simples).
- **S85** — implementar cedente quando cessão.
- **S86** — implementar DtVenc = DtContr + prazo default (12 meses).
- **S90** — implementar Remessa única (assume sequencial).
- **N05** — implementar Limite Basileia (versão simplificada).
- **N07** — implementar Prazo Max (default 60 meses).
- **N08** — implementar Carência Min (default 30 dias).

## 🏗️ Decisões técnicas

### DT-44 — Tabelas default para destravadas

Para destravar stubs que precisam de tabelas regulatórias, vou usar tabelas default conservadoras:
- Limites PF: R$ 500k (limite PF regulamentar default).
- Prazo max: 60 meses (5 anos).
- Carência: 30 dias.
- ClassOp × Mod: tabela simplificada (Mod 02XX aceita A-H, outros só A-D).

### DT-45 — SUB-prefix para Substituição Parcial

Regras de substituição parcial recebem prefixo SUB01-SUB15 (em vez de CXX) para distinguir de Campos Obrigatórios. Isso é uma extensão da nomenclatura — alinhada com o catálogo BACEN que tem seção própria.

### DT-46 — X-prefix para Cross-doc

Regras cross-doc recebem prefixo X01-X10. Já existe crossdoc/rules (Sprint 35), mas com escopo limitado. Sprint 38 adiciona cross-doc 3040-3042.

## 🎯 Métricas alvo

| Métrica | Pré (v3.34.16) | Pós esperado |
|---|---|---|
| Regras 3040 | 221 | **~275** |
| Cobertura catálogo | 61.2% | **~76.2%** |
| Coverage `internal/audit/rules` | 68.2% | **66-68%** |
| Test functions Sprint 38 | 0 | **3 (54 subtests)** |
| Packages PASS -race | 23/23 | **23/23** |

## 📁 Arquivos a criar/modificar

```
backend/internal/audit/rules/3040_sprint38.go         (NOVO — 54 regras)
backend/internal/audit/rules/3040_sprint38_test.go    (NOVO — 54 subtests)
backend/internal/audit/rules/3040_helpers.go          (atualizar — tabelas default)
backend/internal/audit/rules/registry.go              (atualizar Builtin3040)
backend/internal/audit/rules/3040_test.go            (atualizar expectedCodigos)
backend/internal/audit/rules/raw_rules_test.go       (atualizar total = 275)
backend/internal/audit/rules/3040_sprint36_test.go   (atualizar stubs Sprint 36 destravadas)
backend/internal/audit/rules/3040_sprint37_test.go   (atualizar stubs Sprint 37 destravadas)
CHANGELOG.md                                         (entry v3.34.17)
backend/SPRINT_38_RESEARCH.md                        (NOVO — este arquivo)
backend/SPRINT_38_RESULTS.md                         (NOVO — após implementação)
```

## 🎯 Self-verify

- [ ] `grep -c "r.Register(" no Builtin3040 confere com soma esperada (275).
- [ ] `go test -race ./...` 23/23 PASS.
- [ ] `gofmt -l ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] V68-style: cada regra declarada como real tem body que detecta violação.
- [ ] Stubs Sprint 36-37 destravadas são removidas das listas de stubs.

## ⏭️ Após Sprint 38

3040 entra em manutenção. Próximas workstreams:
- **Sprint 39:** AuditDDR Fase 2 (parser DRM/DLO cross-doc).
- **Sprint 40:** AuditDRL (2160 LCR).
- **Sprint 41:** AuditDLP (2170 NSFR).
- **Sprint 42:** Audit3044 (engine JSON eventos).

**Carry-over permanente 3040 (estimado):** ~50 regras (~14%) que dependem de cross-doc DRM/DLO ou parser de catálogos específicos.

---

**Esta é a ÚLTIMA sprint do CADOC 3040.** Próximas focarão em outros CADOCs.