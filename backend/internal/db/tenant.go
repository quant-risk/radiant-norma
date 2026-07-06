// Package db — tenant.go: helper centralizado para multi-tenant isolation.
//
// Sprint 30 (v3.33.0): com migration 014 ativa (FORCE RLS), TODA query a
// tabela tenant-scoped PRECISA estar em transação com `SET LOCAL app.if_id`
// aplicado. Sem isso, Postgres retorna 0 rows (policy USING falha porque
// app.if_id é NULL).
//
// Helper WithTenantTx encapsula BeginTx + SET LOCAL + Commit/Rollback,
// garantindo que toda transação tenha o contexto de tenant correto.
//
// Trade-offs:
//   - Em SQLite: SET LOCAL é no-op (SQLite não tem FORCE RLS nem custom
//     settings). Helper funciona como wrapper de BeginTx normal.
//   - Em Postgres: SET LOCAL tem escopo de transação (rollback-safe).
//     Não vaza entre conexões do pool.
//
// Uso:
//
//	if rows, err := db.WithTenantTx(ctx, db, ifID, func(tx *sql.Tx) ([]Row, error) {
//	    return tx.QueryContext(ctx, "SELECT ...")
//	}); err != nil { ... }
package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// driverCache memoiza detecção de driver por *sql.DB (evita QueryRow extra
// em cada WithTenantTx). Postgres detection faz QueryRow("SELECT
// sqlite_version()") — em high-throughput (50+ goroutines Log concorrentes),
// isso vira contenção adicional. Cache resolve.
//
// Trade-off: cache vive pela vida do *sql.DB. Como DSN não muda após Open,
// é seguro.
//
// Sprint 30 (v3.33.0): adicionado após F-54-H debugging — 50 goroutines
// testes falhando por overhead de driver detection em cada call.
var driverCache sync.Map // map[*sql.DB]bool

func isPostgresCached(d *sql.DB) bool {
	if cached, ok := driverCache.Load(d); ok {
		return cached.(bool)
	}
	isPG := isPostgresDB(d)
	driverCache.Store(d, isPG)
	return isPG
}

// WithTenantTx executa fn dentro de uma transação com `SET LOCAL app.if_id`
// aplicado no início. Garante que toda query a tabela com FORCE RLS veja
// apenas rows do tenant correto.
//
// Parâmetros:
//   - ctx: contexto (cancelamento propaga para tx).
//   - d: conexão do pool. Helper pega uma do pool internamente.
//   - ifID: identificador da Instituição Financeira. Vazio ("") = sem
//     contexto de tenant. Em Postgres, faz app.if_id = ” — policy falha
//     porque if_id (string vazia) != if_id da row (valor real).
//   - fn: função callback que recebe *sql.Tx. Erros retornados são
//     preservados.
//
// Retorno:
//   - Erro do BeginTx, SET LOCAL, fn ou Commit. Rollback sempre executado
//     se algo falhar (no-op se Commit rodar primeiro).
//
// Importante: SET LOCAL só funciona DENTRO de uma transação. Helper
// garante que isso é respeitado.
func WithTenantTx(
	ctx context.Context,
	d *sql.DB,
	ifID string,
	fn func(tx *sql.Tx) error,
) error {
	// Sprint 30 (v3.33.0) Validação 55 [F-55-X]: nil check defensivo.
	// Sem isso, nil d causa nil pointer panic em d.BeginTx. Caller error
	// deve ser tratado graciosamente (log + early return), não crash.
	if d == nil {
		return fmt.Errorf("with tenant tx: nil *sql.DB")
	}
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op se Commit rodar

	// SET LOCAL app.if_id = <ifID>.
	// Funciona em Postgres (escopo de tx). Em SQLite é syntax error,
	// então detectamos driver antes.
	if isPostgresCached(d) {
		// Postgres aceita SET LOCAL como Exec dentro de tx. Não usa
		// parameterized query porque SET não aceita placeholders.
		// Validação: ifID não pode conter aspas simples (SQL injection).
		// Chamadores devem validar input; aqui defendemos em profundidade.
		if err := validateIFID(ifID); err != nil {
			return fmt.Errorf("invalid ifID: %w", err)
		}
		setSQL := fmt.Sprintf("SET LOCAL app.if_id = '%s'", escapeSingleQuote(ifID))
		if _, err := tx.ExecContext(ctx, setSQL); err != nil {
			return fmt.Errorf("set local app.if_id: %w", err)
		}
	}

	// Executa callback.
	if err := fn(tx); err != nil {
		return fmt.Errorf("tx fn: %w", err)
	}

	// Commit.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// validateIFID garante que ifID é seguro para usar em SET LOCAL.
// Aceita apenas alfanumérico + hífen + underscore (UUID-like).
// Bloqueia qualquer caractere que poderia ser SQL injection.
//
// Sprint 30 (v3.33.0) Validação 55 [F-55-F]: empty string é aceita
// (admin escape valve). Caller pode passar "" intencionalmente para
// ações admin/system onde if_id no DB é NULL. Política 012 permite
// `if_id IS NULL OR if_id = current_setting('app.if_id', true)` —
// admin actions têm `if_id IS NULL`.
func validateIFID(ifID string) error {
	if ifID == "" {
		return nil // admin escape valve
	}
	if len(ifID) > 64 {
		return fmt.Errorf("ifID too long (max 64 chars)")
	}
	for i, c := range ifID {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '-' || c == '_':
		default:
			return fmt.Errorf("ifID[%d]: invalid char %q", i, c)
		}
	}
	return nil
}

// escapeSingleQuote duplica aspas simples — defesa adicional se
// validateIFID for removido ou bypassado no futuro.
func escapeSingleQuote(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			result = append(result, '\'', '\'')
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}
