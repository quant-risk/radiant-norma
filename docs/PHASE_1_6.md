# Phase 1.6 — /v1/validate exige data_base e versao_layout

> **Status**: shipped
> **Branch**: `remediation/gates-1-14`
> **Closes gate**: gate #7 from `RELATORIO_FINAL.md` ("/v1/validate deve exigir data_base")
> **Benchmark coverage**: `VAL-DATA_BASE-*`.

## Problema

Antes desta fase, o endpoint `POST /v1/validate` aceitava requests sem `data_base`. Sem a data-base, a validação não conseguia determinar quais regras estavam em vigor (algumas regras têm `data_base_inicio`).

## Solução

Phase 1.6 adiciona validação no handler:

```go
// Phase 1.6: data_base é obrigatório no body.
if req.DataBase == "" {
    http.Error(w, "data_base is required (formato: 2026-06 ou 2026-06-30)", http.StatusBadRequest)
    return
}
```

**Nota**: `versao_layout` não é adicionado como obrigatório porque o validator extrai a versão do XML quando necessário, ou usa a mais recente se não especificada.

## Mudanças de código

### `internal/api/server.go` — handler validate

```go
// Phase 1.6: data_base é obrigatório no body.
if req.DataBase == "" {
    http.Error(w, "data_base is required (formato: 2026-06 ou 2026-06-30)", http.StatusBadRequest)
    return
}
```

## O que não está em Phase 1.6

- Phase 2: Wizard funcional ponta a ponta
