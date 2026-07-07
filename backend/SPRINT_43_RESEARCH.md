# Sprint 43 — RESEARCH — CrossDoc_v2 (Regras Cross-Documento DRL/DLP/3044)

> **Data:** 2026-07-07
> **Sprint:** 43
> **Domínio:** Cross-Documento (CADOCs 2160, 2170, 3044)
> **Versão atual:** v3.34.23
> **Próxima:** v3.34.24

---

## 1. Contexto

### Cross-Doc Sprint 39 vs Sprint 43

| Sprint | Foco | Pattern |
|---|---|---|
| Sprint 39 Fase 2 | DDR vs DRM/DLO | `SetDRM`/`SetDLO` globais em `2070_crossdoc.go` |
| **Sprint 43** | **DRL/DLP vs 3044** | `Set3044` global + nova interface `Rule3044CrossDoc` |

### Documentos e relações

- **DRL (2160)** — LCR = HQLA / (Outflows - Inflows) × 100 >= 100%
- **DLP (2170)** — NSFR = ASF / RSF × 100 >= 100%
- **3044 (JSON)** — Eventos individuais: saldoDevedor, pagamentos, concessões

**Relação:** 3044 é a granularidade que compõe os totais de DRL/DLP. Se 3044 reports `saldoDevedor` de todas as operações, a soma deveria ser consistente com os valores agregados de DRL/DLP (HQLA, Outflows, etc.).

---

## 2. Regras Cross-Doc Planejadas

### Fase 1 — 5 regras DRL/DLP × 3044

| Cod | Sev | Descrição | Lógica |
|---|---|---|---|
| **XD01** | E | CNPJ DRL == CNPJ 3044 | `DRL.CNPJ == DLP.CNPJ == 3044.cnpjIF` |
| **XD02** | E | DtBase DRL == DtBase DLP == dataSaldoDevedor 3044 | Consistência temporal |
| **XD03** | A | Soma saldoDevedor 3044 >= HQLA DRL | HQLA deve cobrir saídas |
| **XD04** | A | DLP NSFR Ratio consistente com DRL LCR Ratio | Se LCR < 100%, NSFR também deve estar sob pressão |
| **XD05** | A | Soma pagamentos 3044 consistente com Outflows DRL | Pagamentos são parte das saídas |

### Fase 2 — 3 regras carry-over

| Cod | Sev | Descrição | Status |
|---|---|---|---|
| **XD06** | E | IPOC em 3044 existe no histórico DDR/DLO | carry-over |
| **XD07** | E | Atraso em 3044 consistente com classificação DRL/DLP | carry-over |
| **XD08** | A | Consistência de prazos entre 3044 e DRL/DLP | carry-over |

---

## 3. Arquitetura

### Novos arquivos

```
backend/internal/audit/rules/
  crossdoc_liquidity.go  (NOVO — XD01-XD08 + Set3044 + globals)
  crossdoc_liquidity_test.go  (NOVO — testes)
```

### Pattern

```go
// Globals para cross-doc (set via service layer)
var (
    parsedDRL  *DocDRL
    parsedDLP  *DocDLP
    parsed3044 *Doc3044
)

func Set3044(doc *Doc3044) { parsed3044 = doc }
```

### Interface

Regras cross-doc vivem em `crossdoc_liquidity.go` e usam globals `parsedDRL`/`parsedDLP`/`parsed3044`. Não precisam de nova interface de registry — são chamadas diretamente pelo service layer.

---

## 4. Critérios de aceitação

- [ ] 5 regras XD01-XD05 com lógica real
- [ ] 0 stubs disfarçados
- [ ] `go test -race ./...` 23/23 PASS
- [ ] `go vet ./...` clean
- [ ] `gofmt -l ./...` clean
- [ ] Testes Sprint 43 PASS (≥5 subtests)
- [ ] CHANGELOG entry v3.34.24

---

## 5. Riscos

| Risco | Mitigação |
|---|---|
| XD06/XD07/XD08 requerem DB lookup | Carry-over documentado |
| DRL/DLP não têm IPOC individual (são agregados) | Validar só no nível que faz sentido |
| Domínio de class3050 não disponível para XD07 | Carry-over ou validação de prefixo |
