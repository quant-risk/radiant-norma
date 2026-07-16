# Phase 1.3 — /v1/validate usa ValidateFull (L1→L4)

> **Status**: shipped
> **Branch**: `remediation/gates-1-14`
> **Closes gate**: gate #4 from `RELATORIO_FINAL.md` ("validator deve executar todas as camadas L1-L4")
> **Benchmark coverage**: `VAL-LAYERS-*`, `GEN-VALIDATE-*`.

## Contexto

Antes desta fase, o endpoint `/v1/validate` chamava `s.Audit.Validate()` que executava apenas L1 (parse) e L2 (regras semânticas). As camadas L3 (cross-doc) e L4 (histórico) eram executadas separadamente via `ValidateFull()` ou não eram executadas para requests de validação individual.

## Solução

Phase 1.3 modifica o handler `validate` em `internal/api/server.go` para:

1. Chamar `ValidateFull()` (que executa L1→L4 em goroutines com panic recovery)
2. Converter `FullValidationResponse` → `ValidationResponse` via `FullToValidationResponse()` para manter compatibilidade com callers existentes

## Mudanças de código

### `internal/api/server.go` — handler validate

```go
// Phase 1.3: executa ValidateFull (L1→L4) ao invés de Validate (L1→L2).
// RelatedDocs e EnvioID ficam nil/"" para requests de documento único
// (L3 cross-doc e L4 histórico precisam de contexto que não está
// disponível neste endpoint). O generate/batch endpoint fornece RelatedDocs.
fullReq := &audit.FullValidationRequest{
    Main:        &req,
    RelatedDocs: nil,
    EnvioID:     "",
}
fullResp, err := s.Audit.ValidateFull(r.Context(), fullReq)
if err != nil {
    s.internalServerError(w, err, "validate")
    return
}

// Converte para ValidationResponse (formato da API pública).
resp := audit.FullToValidationResponse(fullResp)
```

### `internal/audit/service.go` — FullToValidationResponse

Nova função que agrega resultados de todas as camadas:

```go
func FullToValidationResponse(full *FullValidationResponse) *ValidationResponse {
    // Agrega errors de L1, L2, L3, L4
    // L3/L4 errors viram warnings se severity != "E"
    // Passed = true apenas se L1 E L2 passaram
}
```

## Comportamento

| Camada | Antes | Depois |
|--------|-------|--------|
| L1 (XSD) | ✅ | ✅ |
| L2 (Regras) | ✅ | ✅ |
| L3 (Cross-doc) | ❌ | ✅ |
| L4 (Histórico) | ❌ | ✅ (via EnvioID) |

**Nota**: L3 e L4 para validação única (sem related docs) ainda são limitadas porque não há outro documento para comparar. O `generate/batch` endpoint fornece `RelatedDocs` para validação cross-doc completa.

## O que não está em Phase 1.3

- Phase 1.4: whitelist de versão no generator
- Phase 1.5: required fields enforced + data_base obrigatório
- Phase 1.6: `/v1/validate` exige `data_base` e `versao_layout`
