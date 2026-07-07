# Sprint 37 — Audit3040 Fase 3 — RESEARCH

> **Data:** 2026-07-07
> **Sprint:** 37 (Audit3040 Fase 3 — fechar 3040 ~85%)
> **Pré-requisito:** v3.34.14 (V67 fechou drift do Sprint 36)
> **Marco esperado:** 177 → ~227+ regras 3040 (49.0% → 62.6%+ cobertura)

## 🎯 Escopo Sprint 37 Fase 3

**+50 regras 3040** baseadas em:
1. **Individualizadas I06-I15** (10 regras — destrava com parser expandido).
2. **Agregadas A16-A30** (15 regras — cobertura Tier 3 expandida).
3. **Semântica S71-S90** (20 regras — validações semânticas).
4. **Carry-over destravadas** (~5 stubs Sprint 36 que agora têm parser).

Total: **+50 regras**. **177 → ~227** (cobertura 49.0% → **62.8%**).

## 📋 Regras prioritárias

### Individualizadas I06-I15 (10 regras)

| Cod | Descrição |
|---|---|
| I06 | ContratoModalidadePJ vs PF — separar por TpCli |
| I07 | IPOC + Cliente únicos por combinação |
| I08 | ProvConsttd individualizada vs provision required por ClassOp |
| I09 | Vencimentos individualizados zerados quando ClassOp = A |
| I10 | Cliente IPOC único (não duplicar entre remessas) |
| I12 | Operacao.Cli.IPOC = Operacao.IPOC quando ambos presentes |
| I13 | DtVencOp dentro janela de 5 anos da DtBase |
| I14 | IPOC bem-formado (alfanumérico, 8-20 chars) |
| I15 | Operacoes PF: soma Vencimentos <= limite PF regulamentar |

### Agregadas A16-A30 (15 regras)

| Cod | Descrição |
|---|---|
| A16 | ClassOp + FaixaVlr combinação válida |
| A17 | Soma QtdOp agregado = soma QtdOp operações individuais |
| A18 | Soma QtdCli agregado = soma Cli únicos em Operacoes |
| A19 | Mod + NatuOp combinação regulamentar |
| A20 | PrzProvm = S requer ClassOp E-H |
| A21 | Localiz (UF) válida (27 UFs brasileiras) |
| A22 | TpCli = 1 (PF) tem Localiz (UF) |
| A23 | TpCli = 2 (PJ) tem Localiz (UF) |
| A24 | DesempOp 01-08 mapeado para faixas vencimento |
| A25 | ClassOp agregado == moda ClassOp individual |
| A26 | NatuOp 02 (cobrados) tem OrigemRec específica |
| A27 | VincME = S requer Modalidade ME |
| A28 | FaixaVlr 01-13 sequencial sem gaps |
| A29 | QtdCli > 0 implica NatuOp + Mod + ClassOp presente |
| A30 | ProvConsttd agregado = soma ProvConsttd individuais (por chave) |

### Semântica S71-S90 (20 regras)

| Cod | Descrição |
|---|---|
| S71 | Operacao.Valor > 0 quando QtdOp > 0 |
| S72 | Operacao.Perc = 100 quando ClassOp = A |
| S73 | DtContr não pode ser futuro distante (> DtBase + 1 ano) |
| S74 | Vencimentos não-negativos |
| S75 | TotalCli header = soma QtdCli agregados (alias H09) |
| S76 | Parte sequencial (1, 2, 3...) sem gaps |
| S77 | Substituição (TpArq=S) tem Remessa maior que última aceita |
| S78 | Cada agregado tem ClassOp dentro faixa permitida por Mod |
| S79 | DtBase não pode ser > 2 meses no passado (atraso envio) |
| S80 | QtdOp >= 0 sempre (não negativo) |
| S81 | Vencimentos parcelados (V110 < V120 < V150 < V160 < V165) |
| S82 | Operacao.Valor >= Vencimentos soma (saldo devedor) |
| S83 | QtdCli inteiro positivo |
| S84 | CNPJ raiz cliente = CNPJ raiz header (consolidado) |
| S85 | Operacao sem cliente + Inf 0303 (cessão) tem cedente |
| S86 | DtVencOp = DtContr + prazo operação |
| S87 | QtdOp inteiro positivo |
| S88 | Vencimentos total = V110 + V120 + V150 + V160 + V165 (sanity) |
| S89 | ClassOp cruzada com VincME (não combinação inválida) |
| S90 | Remessa única por DtBase + CNPJ raiz |

### Carry-over destravadas (~5 stubs Sprint 36)

- C44 LocalizPF: agora tem lógica (Localiz 27 UFs).
- C46 OrigemRecBNDES: BNDES modalidades (0271, 0272).
- C57 Inf0307Rel1201: parser cruzado.
- C62 ClassOpIndAg: agora pode comparar.
- C68 CliIPOCEqual: agora pode verificar.

## 🏗️ Decisões técnicas

### DT-41 — Helpers de validação

- `validarUF(string) bool` — 27 UFs brasileiras + exterior.
- `validarIPOC(string) bool` — alfanumérico 8-20 chars.
- `validarModNatuOp(mod, natuOp string) bool` — combinação regulamentar.
- `validarPerc(float64) bool` — 0 <= p <= 100.

### DT-42 — Severity patterns

- **E (erro):** violações que bloqueiam (formato, unicidade, range obrigatório).
- **A (aviso):** violações que sinalizam (combinações suspeitas, prazos).
- **I (info):** stubs ou verificações parciais (não bloqueiam).

### DT-43 — Stubs honestos em Carry-over

Stubs que continuam stub após Sprint 37:
- Cross-doc pesado (precisa DRM/DLO parser).
- Catálogo modalidades específico (Rural, Habitacional, Leasing).
- Limites regulatórios dinâmicos (Basileia, CMN 4.966 tabela).

## 🎯 Métricas alvo

| Métrica | Pré (v3.34.14) | Pós esperado |
|---|---|---|
| Regras 3040 | 177 | **~227** |
| Cobertura catálogo | 49.0% | **~62.8%** |
| Coverage `internal/audit/rules` | 71.0% | **70-72%** |
| Test functions Sprint 37 | 0 | **3 (50 subtests)** |
| Packages PASS -race | 23/23 | **23/23** |

## 📁 Arquivos a criar/modificar

```
backend/internal/audit/rules/3040_sprint37.go         (NOVO — 50 regras)
backend/internal/audit/rules/3040_sprint37_test.go    (NOVO — 50 subtests)
backend/internal/audit/rules/registry.go              (atualizar Builtin3040)
backend/internal/audit/rules/3040_test.go            (atualizar expectedCodigos)
backend/internal/audit/rules/raw_rules_test.go       (atualizar total)
backend/internal/audit/rules/3040_helpers.go         (NOVO — helpers UF/IPOC/Mod)
CHANGELOG.md                                         (entry v3.34.15)
backend/SPRINT_37_RESEARCH.md                        (NOVO — este arquivo)
backend/SPRINT_37_RESULTS.md                         (NOVO — após implementação)
```

## 🎯 Self-verify

- [ ] `grep -c "r.Register(" no Builtin3040 confere com soma esperada (227).
- [ ] `go test -race ./...` 23/23 PASS.
- [ ] `gofmt -l ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Cada regra nova tem pelo menos 1 teste (real ou stub).
- [ ] V67-style: claims em docs refletem classificação real (real/híbrida/stub).

## ⏭️ Após Sprint 37

Sprint 38 Fase 4 (última do 3040): 227 → ~280 (78%) com stubs documentados para carry-over permanente (~80 regras que dependem de cross-doc DRM/DLO ou parser de catálogos específicos).

**Não vou pular para outro CADOC antes de fechar 3040.**

---

**Pré-validação Sprint 37:** antes de implementar, validar o que já existe em 3040_sprint36.go e checar drift similar ao V67. Aplicar protocolo V67 a cada commit.