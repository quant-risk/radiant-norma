# Sprint 51 — RESEARCH.md

## Audit4111 — 30+ Regras Iniciais

**Sprint:** 51
**Tema:** Audit4111 — 30+ regras iniciais para CADOC 4111
**Período:** 2026-07-07
**Versão alvo:** v3.34.32

---

## 1. O que é o CADOC 4111

**Documento:** CADOC 4111 — Registro de Clientes e Modalidades
**Sistema:** SCR (Sistema de Informações de Crédito do BACEN)
**Base legal:** Resolução CMN 4.557/17 (GIR) e derivadas

**Finalidade:** O 4111 lista os clientes e modalidades de crédito contratados. É usado pelo DRSAC (2030) e pelo SCR (3040) para validação cruzada de clientes e operações.

**Contexto no Radiant Norma:**
- O CADOC 4111 é referenciado nas regras cross-doc XD-001 e XD-002 em `internal/crossdoc/rules/3040_4111.go`
- A tag `<Cliente>` com `<QtdCli>` é usada para contar clientes
- O 4111 é comparado com o 3040 para validar consistência de operações

---

## 2. Lacuna: Spec 4111 Não Disponível

**Problema:** O leiaute oficial do 4111 (XSD ou críticas) **não existe** no repositório.

A regra cross-doc XD-001 (3040↔4111) faz:
```go
clients := crossdoc.ExtractSumOfTag(xml4111, "Cliente", "QtdCli")
```

Isso indica que o 4111 tem:
- Tag `<Cliente>` no nível root
- Tag `<QtdCli>` dentro de Cliente (quantidade de clientes)

**Mas sem o leiaute oficial, não é possível:**
1. Criar um parser tipado (como fiz para DRSAC)
2. Definir 30+ regras de validação precisas
3. Integrar ao schema registry

---

## 3. Ação Necessária

Solicitar ao BACEN:
1. XSD do leiaute 4111
2. Documento de críticas e validações

---

## 4. Alternativa: Implementar com Base no Existing Cross-Doc

Mesmo sem o spec completo, posso implementar:

### 4.1 Parser XML Genérico 4111

```go
// Parse4111 genérico (sem struct tipado)
func Parse4111(data []byte) (*Doc4111Generic, error)
```

### 4.2 Regras de Estrutura (genéricas, sem spec)

- 4111-01: XML bem formado
- 4111-02: Tag `<Cliente>` presente
- 4111-03: CNPJ de cliente válido (8 ou 14 dígitos)
- 4111-04: Data-base no formato AAAA-MM
- 4111-05: Quantidade de clientes > 0

Essas regras são genéricas demais — não são as "30+ regras reais" prometidas no ROADMAP.

---

## 5. Dependências

| Item | Status | Ação |
|---|---|---|
| XSD 4111 | ❌ FALTANDO | Solicitar ao BACEN |
| Críticas 4111 | ❌ FALTANDO | Solicitar ao BACEN |
| Parser 4111 | ⏳ Aguardando XSD | Implementar após receber |
| Regras 4111 | ⏳ Aguardando críticas | Implementar após receber |

---

## 6. Riscos

| Risco | Mitigação |
|---|---|
| BACEN não responde com spec | Implementar com parser genérico + validações estruturais mínimas |
| 30+ regras não implementáveis sem spec | Priorizar as 5 regras estruturais + cross-doc com DRSAC |
| XSD 4111 difere do esperado | Flexibilidade no parser com fallback para genérico |

---

## 7. Plano Revised

Dado que o spec 4111 não está disponível, vou:

1. Criar parser genérico para 4111 (só validação estrutural)
2. Implementar 5 regras estruturais mínimas
3. Marcar como "parcial" no CHANGELOG
4. Cruzar com DRSAC (Sprint 52) para usar o que temos

**Aviso:** as 30+ regras reais dependem do documento de críticas do BACEN.
