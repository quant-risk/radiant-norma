# Sprint 42 — RESEARCH — Audit3044 (Engine JSON — Eventos de Operações de Crédito)

> **Data:** 2026-07-07
> **Sprint:** 42
> **Domínio:** CADOC 3044 (JSON)
> **Versão atual:** v3.34.22
> **Próxima:** v3.34.23

---

## 1. Contexto

### O que é o 3044?

O **CADOC 3044** é o documento de **eventos de operações de crédito** do SCR (Sistema de Informações de Crédito do Banco Central). Diferente dos demais CADOCs do Radiant Norma (3040, 3050, etc. que usam **XML**), o 3044 usa **JSON** como formato de transporte.

- **Normativo:** IN BCB 530/2024 (vigência: novembro/2025)
- **Formato:** JSON (não XML)
- **Frequência:** evento a evento (não mensal como 3040/3050)
- **Função:** registrar eventos de pagamento, concessão, cessão e aquisição de operações de crédito

### Fontes de dados

| Fonte | Descrição |
|---|---|
| `_catalogos/leiautes_3044.json` | Schema JSON extraído do manual (v2, jul/2025) |
| `_catalogos/extract_3044.py` | Script de extração do schema |
| `_concorrentes/Matera_CADOC_3044.html` | Referência Matera |
| `3044/SCR_InstrucoesDePreenchimento_Doc3044.pdf` | Manual BACEN |

---

## 2. Estrutura do Documento 3044

### Estrutura JSON

```json
{
  "cnpjIF": "12345678",
  "dataHoraRemessa": "2026-07-03 14:30:00",
  "envia3050": "S",
  "operacoes": [
    {
      "acao": 1,
      "ipoc": "876543210216210020716C1234",
      "class3050": "112212101",
      "saldoDevedor": 5000.00,
      "dataSaldoDevedor": "2026-07-03",
      "atraso": "N",
      "pagamentos": [
        { "acao": 1, "data": "2026-06-15", "valor": 1000.00 }
      ],
      "concessoes": [],
      "cessoes": [],
      "aquisicoes": []
    }
  ]
}
```

### Objetos

| Objeto | Descrição |
|---|---|
| `documento` | Raiz: cnpjIF, dataHoraRemessa, envia3050, operacoes[] |
| `Operacao` | IPOC individual com saldo devedor e eventos |
| `Pagamento` | Evento de pagamento (ação: 1=Incluir, 2=Excluir, 3=Alterar) |
| `Concessao` | Evento de concessão de crédito |
| `Cessao` | Evento de cessão de operação |
| `Aquisicao` | Evento de aquisição de operação |

### Ações (campo `acao`)

| Valor | Descrição |
|---|---|
| 1 | Incluir |
| 2 | Excluir (cancelar/estornar) |
| 3 | Alterar |

---

## 3. Regras de Validação (14 regras T01-T19)

### Fase 1 — Regras auto-contidas (sem dependência externa)

| Cod | Sev | Descrição | Implementação |
|---|---|---|---|
| **T01** | E | dataHoraRemessa >= dataSaldoDevedor | Comparar timestamps |
| **T02** | E | Pagamentos: data <= dataSaldoDevedor | Iterar pagamentos |
| **T03** | E | Concessões: data <= dataSaldoDevedor | Iterar concessões |
| **T04** | E | dataHoraRemessa não futura, não >21 dias antiga | Validar janela |
| **T05** | E | Sem pagamentos duplicados (mesmo IPOC + data) | Map de dedup |
| **T06** | E | Sem concessões duplicadas (mesmo IPOC + data) | Map de dedup |
| **T07** | E | class3050 proibido se envia3050='N' | Verificar envios |
| **T08** | A | class3050 deve pertencer ao domínio se envia3050='S' | Domínio + prefijo |

### Fase 2 — Validações de janela temporal

| Cod | Sev | Descrição | Implementação |
|---|---|---|---|
| **T11** | E | Data pagamento dentro dos últimos 6 meses | Comparar com 180 dias |
| **T12** | E | Data concessão dentro dos últimos 6 meses | Comparar com 180 dias |
| **T13** | E | Data cessão dentro dos últimos 6 meses | Comparar com 180 dias |
| **T14** | E | Data aquisição dentro dos últimos 6 meses | Comparar com 180 dias |

### Fase 3 — Consistência de valor e dedup

| Cod | Sev | Descrição | Implementação |
|---|---|---|---|
| **T15** | E | Valores não podem ser negativos | Iterar valores |
| **T16** | E | saldoDevedor não negativo (exceto anuidade/cashback) | Campo específico |
| **T17** | E | IPOC não pode repetir no mesmo documento | Set de dedup |

### Carry-over (requer DB ou cross-doc)

| Cod | Sev | Descrição | Status |
|---|---|---|---|
| **T18** | E | acao=2 (Excluir) requer IPOC existente na base | carry-over |
| **T19** | E | acao=3 (Alterar) requer IPOC existente na base | carry-over |

---

## 4. Arquitetura

### Novos arquivos

```
backend/internal/audit/rules/
  doc3044.go         (NOVO — Doc3044 struct + JSON parser)
  rule3044.go       (NOVO — interface Rule3040 + 14 regras T01-T19)
  rule3044_test.go  (NOVO — testes)
```

### Interface

```go
type Rule3044 interface {
    Code() string   // "T01", "T17", etc.
    Severity() string // "E" | "A" | "I"
    Apply(ctx context.Context, doc *Doc3044) error
}
```

### Doc3044 struct

```go
type Doc3044 struct {
    CNPJ           string
    DataHoraRemessa time.Time
    Envia3050      string // "S" | "N"
    Operacoes      []Operacao3044
}

type Operacao3044 struct {
    Acao             int     // 1, 2
    IPOC             string
    Class3050        string
    SaldoDevedor     float64
    DataSaldoDevedor time.Time
    Atraso           string // "S" | "N"
    Pagamentos       []Pagamento3044
    Concessoes       []Concessao3044
    Cessoes          []Cessao3044
    Aquisicoes       []Aquisicao3044
}
```

---

## 5. Critérios de aceitação

- [ ] Parser JSON 3044 (`encoding/json`)
- [ ] 12+ regras T01-T17 implementadas com lógica real
- [ ] 0 stubs disfarçados
- [ ] `go test -race ./...` 23/23 PASS
- [ ] `go vet ./...` clean
- [ ] `gofmt -l ./...` clean
- [ ] Testes Sprint 42 PASS (≥14 subtests)
- [ ] CHANGELOG entry v3.34.23
- [ ] ROADMAP.md atualizado

---

## 6. Riscos

| Risco | Mitigação |
|---|---|
| T18/T19 requerem DB lookup (não temos ainda) | Carry-over; implementar stubs reais depois |
| Domínio class3050 não disponível | Varrer lista 3050 como referência (aproximação) |
| dataHoraRemessa em formato heterogêneo | Parser flexível com tryparse (YYYY-MM-DD HH:mm:ss) |
