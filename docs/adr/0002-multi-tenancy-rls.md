# ADR-0002: Multi-tenancy — Postgres RLS, não schema-per-tenant

> **Status:** Aceito
> **Data:** 2026-07-05
> **Decisor(es):** Henrique Costa · Mavis

## Contexto

Modelo de isolamento de dados entre IFs (Instituições Financeiras) clientes. Radiant Norma vai hospedar múltiplos tenants no mesmo cluster Postgres. Precisamos decidir entre:

1. **Database-per-tenant** (isolamento físico)
2. **Schema-per-tenant** (isolamento lógico + administrativo)
3. **Single schema com RLS** (Row-Level Security)
4. **Single schema com WHERE-if_id na app** (sem DB-level isolation)

## Decisão

**Single schema + Postgres RLS + `SET LOCAL app.current_if_id`** em TODA transação.

```sql
-- Habilitar RLS
ALTER TABLE tenant_data.envios ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_data.envios FORCE ROW LEVEL SECURITY;  -- até superuser respeita

-- Policy
CREATE POLICY envios_tenant_isolation ON tenant_data.envios
    USING (if_id = current_setting('app.current_if_id', true));
```

```go
// App-side wiring (obrigatório em toda query)
func (s *Server) withTenantContext(ifID string, fn func(tx *sql.Tx) error) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()

    if !isValidIfID(ifID) {
        return ErrInvalidIfID
    }

    // SET LOCAL aplica só na transação
    if _, err := tx.ExecContext(ctx, "SET LOCAL app.current_if_id = $1", ifID); err != nil {
        return err
    }

    if err := fn(tx); err != nil {
        return err
    }

    return tx.Commit()
}
```

## Consequências

**Positivas:**
- ✅ Migrations simples (single schema, single source of truth).
- ✅ **Defense-in-depth:** mesmo bug na app (esquecer WHERE if_id =) é bloqueado pelo DB.
- ✅ Connection pool compartilhado entre tenants (eficiência).
- ✅ Cross-tenant queries (admin) possíveis via bypass explícito (superuser / role separada).
- ✅ LGPD-friendly: deletar tenant = DELETE FROM tenants WHERE id = $1 + cascade.

**Negativas:**
- ❌ Toda query precisa estar em transação com `SET LOCAL`. Helper `withTenantContext` obrigatório.
- ❌ Cross-tenant queries (admin Radar, suporte) precisam de role separada com `BYPASSRLS`.
- ❌ RLS adiciona ~5% overhead (mensurável em benchmarks, irrelevante em prática).

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| **Database-per-tenant** | Connection pool explosion em 100+ tenants. Migrations nightmare. Backup/restore granular caro. |
| **Schema-per-tenant** | Migrations viram O(N) operations em 100+ schemas. Naming collisions. Rollback complexo. |
| **WHERE if_id = na app** | Vetor de bug — qualquer novo dev esquece WHERE e vaza dados. Auditoria SOC 2 reprova. |

## Notas de implementação

- Usar `pgxpool.Pool` com `AfterAcquire` que seta `SET LOCAL` automaticamente.
- Manter lista de tabelas tenant-scoped em constante + test que valida RLS em todas.
- Admin endpoints (Radar, suporte) usam role `radiant_admin` com `BYPASSRLS`.
- Auditoria externa (PwC/Deloitte) valida isolation: tentar SELECT sem `SET LOCAL` deve retornar 0 rows ou erro.