# Phase 1.4 — Whitelist de versão no generator

> **Status**: shipped
> **Branch**: `remediation/gates-1-14`
> **Closes gate**: gate #5 from `RELATORIO_FINAL.md` ("generator deve rejeitar versões não whitelist")
> **Benchmark coverage**: `GEN-VERSION-*`.

## Problema

Antes desta fase, o handler `POST /v1/generate/{cadoc}` aceitava qualquer `versao_layout` no body e usava-a sem validação. Um cliente poderia pedir `"versao_layout": "99.9"` e o generator tentaria gerar com essa versão inválida, potencialmente produzindo XML inconsistente.

## Solução

Phase 1.4 adiciona validação de versão no handler antes de chamar `Generate()`:

1. **`isVersionSupported(g, version)`** — função helper que verifica se a versão está na whitelist do generator
2. **Handler `generateCadoc`** — valida versão antes de criar o documento
3. **Handler `generateBatch`** — valida versão para cada CADOC no batch

Se a versão não é suportada, retorna `HTTP 400` com:
```json
{"error": "INVALID_VERSION", "message": "versão \"99.9\" não suportada para CADOC 3040 (suportadas: [3.0, 3.1, 3.2])"}
```

## Mudanças de código

### `internal/api/generate.go`

```go
// isVersionSupported checks if version is in the generator's supported list.
func isVersionSupported(g generator.CADOCGenerator, version string) bool {
    for _, v := range g.SupportedVersions() {
        if v == version {
            return true
        }
    }
    return false
}
```

Handler `generateCadoc`:
```go
if req.VersaoLayout != "" {
    // Phase 1.4: valida que a versão é whitelist.
    if !isVersionSupported(g, req.VersaoLayout) {
        writeError(w, http.StatusBadRequest, "INVALID_VERSION", ...)
        return
    }
    doc.VersaoLayout = req.VersaoLayout
}
```

## O que não está em Phase 1.4

- Phase 1.5: required fields enforced + data_base obrigatório
- Phase 1.6: `/v1/validate` exige `data_base` e `versao_layout`
