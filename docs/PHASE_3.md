# Phase 3 — RBAC readonly middleware

> **Status**: shipped
> **Branch**: `remediation/gates-1-14`
> **Closes gate**: gate #9 from `RELATORIO_FINAL.md` ("readonly role deve ser impedida de mutar")
> **Benchmark coverage**: `AUTH-ROLE-*`.

## Problema

Antes desta fase, tokens com role `readonly` podiam fazer POST/PUT/DELETE em endpoints como `/validate`, `/generate`, `/rules/{code}/toggle`, etc. A role `readonly` existia no JWT mas não era enforces.

## Solução

Phase 3 adiciona `readonlyMiddleware` que:

1. Permite GET/HEAD/OPTIONS livremente
2. Bloqueia POST/PUT/DELETE/PATCH para tokens com role `readonly`
3. Retorna `403 Forbidden` com `{"error":"readonly: mutation not allowed"}`

## Mudanças de código

### `internal/api/server.go`

```go
// Phase 3: RBAC readonly middleware — bloqueia POST/PUT/DELETE/PATCH
// para tokens com role "readonly".
r.Use(readonlyMiddleware)
```

```go
// readonlyMiddleware bloqueia requests mutativas (POST/PUT/DELETE/PATCH) para
// tokens com role "readonly". Phase 3 implementa RBAC coarse-grained.
func readonlyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
            next.ServeHTTP(w, r)
            return
        }
        claims, err := auth.ClaimsFromContext(r.Context())
        if err != nil || claims == nil {
            next.ServeHTTP(w, r)
            return
        }
        if claims.Role == auth.RoleReadOnly {
            http.Error(w, `{"error":"readonly: mutation not allowed"}`, http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

## Roles existentes

| Role | Acesso |
|------|---------|
| `admin` | Todas as operações |
| `if` | Apenas seu tenant (read + write) |
| `readonly` | Apenas leitura (GET) |

## O que não está em Phase 3

- Phase 4: STA persist + dedupe + retry + DLQ
- Phase 5: Webhook inicializar + assinatura + idempotência
