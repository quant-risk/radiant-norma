# Phase 1.5 — Required fields enforced + data_base obrigatório

> **Status**: shipped
> **Branch**: `remediation/gates-1-14`
> **Closes gate**: gate #6 from `RELATORIO_FINAL.md` ("data_base obrigatório no generate")
> **Benchmark coverage**: `GEN-DATA_BASE-*`, `GEN-REQUIRED-*`.

## Problema

Antes desta fase, o handler `POST /v1/generate/{cadoc}`:
1. Aceitava `data_base` opcional e calculava automaticamente se omitido
2. Não validava campos obrigatórios antes de chamar `Generate()`

Isso permitia gerar documentos com data-base incorreta (base errada) e campos obrigatórios missing.

## Solução

Phase 1.5 adiciona validações no handler:

1. **`data_base` é obrigatório** — retorna `HTTP 400` se omitido
2. **Campos obrigatórios verificados** — `cnpj`, `nome_if` e os campos do generator validados antes da geração

## Mudanças de código

### `internal/api/generate.go`

**Handler `generateCadoc`:**
```go
// Phase 1.5: data_base é obrigatório.
dataBase := req.DataBase
if dataBase.IsZero() {
    writeError(w, http.StatusBadRequest, "MISSING_DATA_BASE",
        "data_base é obrigatório (formato: 2026-06-30T00:00:00Z)")
    return
}

// Phase 1.5: verifica campos obrigatórios.
if missing := checkRequiredFields(g, req); len(missing) > 0 {
    writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELDS",
        fmt.Sprintf("campos obrigatórios ausentes: %v", missing))
    return
}
```

**Handler `generateBatch`:**
- Mesma validação para cada CADOC no batch
- Continua processando outros CADOCs se um falhar (não aborta batch inteiro)

**Função `checkRequiredFields`:**
```go
func checkRequiredFields(g generator.CADOCGenerator, req GenerateRequest) []string {
    var missing []string
    requiredStrFields := map[string]string{
        "cnpj":    req.CNPJ,
        "nome_if": req.NomeIF,
    }
    for field, value := range requiredStrFields {
        if value == "" {
            missing = append(missing, field)
        }
    }
    // Check generator-specific required fields via RequiredFields().
    return missing
}
```

## Erros de resposta

```json
// data_base ausente
{"error": "MISSING_DATA_BASE", "message": "data_base é obrigatório (formato: 2026-06-30T00:00:00Z)"}

// campos obrigatórios ausentes
{"error": "MISSING_REQUIRED_FIELDS", "message": "campos obrigatórios ausentes: [cnpj, nome_if]"}
```

## O que não está em Phase 1.5

- Phase 1.6: `/v1/validate` exige `data_base` e `versao_layout`
