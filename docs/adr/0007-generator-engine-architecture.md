# ADR-0007: Generator Engine Architecture

> **Status:** Proposto (implementação: Sprint 57 — não iniciado)
> **Data:** 2026-07-08
> **Decisor(es):** Henrique Costa

## Contexto

Radiant Norma era um "validador de CADOCs" — função commodity (BCValidador é gratuito, Matera/Dimensa já fazem). O diferenciador real é **gerar** CADOCs a partir de dados brutos de qualquer fonte.

## Problema

IFs gastam 1-3 pessoas 100% do tempo gerando CADOCs manualmente. O mercado oferece:
- BCValidador: só valida (não gera)
- Matera/Dimensa: validação + push, não geração AI-driven
- Planilhas com macros: frágeis, sem validação, quebram em mudança de leiaute

Geração é o que justifica R$ 1.500–12.000/mês.

## Decisão

Motor de geração em **5 camadas**:

```
5. INGEST    [Manual UI | PDF/XLSX/DOCX | API | DB | MCP]
   ↓
4. CANONICAL [CanonicalDocument — JSON tipado, IF-agnóstico]
   ↓
3. MAPPER   [IF-schema → CADOC-field via COSIF + LLM]
   ↓
2. EMITTER  [structs Go → XML no leiaute BACEN vigente]
   ↓
1. VALIDATOR [audit.Service L1→L4 + crossdoc + histórico]
   ↓
0. SUBMITTER [STA/DRRSystem via Autran/SLIM800]
```

**Interface central:**

```go
type CADOCGenerator interface {
    CadocCode() string
    Generate(ctx context.Context, doc *CanonicalDocument, dataBase time.Time) (*GeneratedDoc, error)
    RequiredFields() []schema.Field
}

type GeneratedDoc struct {
    XML      []byte
    ZIP      []byte
    SHA256   string
    Errors   []ValidationError
    FieldMap map[string]FieldMapping
}
```

**Regras de LLM:**
- LLM escreve `CanonicalDocument` (representação tipada)
- Motor de geração (`Emitter`) sempre serializa para XML a partir de structs Go — **nunca LLM gera XML direto**
- Validação L1-L4 downstream sempre roda após a geração

## Consequências

**Positivas:**
- ✅ Diferenciação real vs Matera/Dimensa/BCValidador
- ✅ Geração + validação integrada (moat: 1.099 regras)
- ✅ CanonicalDocument como contrato estável entre conectores e generators
- ✅ Schema Registry versionado alimenta os generators automaticamente

**Negativas:**
- ❌ LLM pode inventar valores no Canonical — mitigado por canonical model tipado + validação downstream
- ❌ Mais abstrações — tradeoff aceito pelo valor de mercado

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| LLM gera XML direto | SchemaBACEN muda 3-5x/ano; LLM hallucina; sem auditabilidade |
| Apenas Form UI (sem conectores) | Não escala; cada IF tem fonte de dados diferente |
| ETL batch (Airflow) | Bom pra DB, péssimo pra inputs manuais/PDFs ad-hoc |
| Tudo via LLM tool-use | Caro em escala, difícil auditar (compliance exige reprodutibilidade) |
