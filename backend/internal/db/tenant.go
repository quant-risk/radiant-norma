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

// ClearDriverCache remove uma entrada específica do driverCache. Útil
// quando o caller sabe que um *sql.DB será fechado definitivamente
// (shutdown, test cleanup dinâmico). Não afeta entries de outros DBs.
//
// Validação 56 (v3.33.2) [F-56-C]: anteriormente driverCache era
// unbounded — entries de DBs fechados permaneciam para sempre. Em
// prática, leak é negligenciável (app tem 1 *sql.DB por processo), mas
// adicionado cleanup explícito para higiene e para suportar testes que
// criam/fecham múltiplos DBs.
//
// Uso recomendado: chamar após `d.Close()` se o caller criar/fechar
// DBs dinamicamente. Em produção (cmd/api, cmd/worker), chamar no
// shutdown handler. No-op se *sql.DB nunca foi registrado.
func ClearDriverCache(d *sql.DB) {
	if d == nil {
		return
	}
	driverCache.Delete(d)
}

// WithTenantTx executa fn dentro de uma transação com `SET LOCAL app.if_id`
// aplicado no início. Garante que toda query a tabela com FORCE RLS veja
// apenas rows do tenant correto.
//
// Parâmetros:
//   - ctx: contexto (cancelamento propaga para tx).
//   - d: conexão do pool. Helper pega uma do pool internamente.
//   - ifID: identificador da Instituição Financeira. Vazio ("") = sem
//     contexto de tenant. Em Postgres, faz o setting ficar com string
//     vazia para app.if_id — política audit_log/audit_events (migration
//     012) usa `if_id IS NULL OR if_id = current_setting('app.if_id', true)`,
//     então string vazia no setting só vê rows com if_id NULL
//     (admin/system escape valve).
//     Validação 55 (v3.33.1) F-55-C: empty string AGORA aceita (era
//     divergência SQLite vs Postgres pré-fix).
//   - fn: função callback que recebe *sql.Tx. Erros retornados são
//     preservados.
//
// Retorno:
//   - Erro do BeginTx, SET LOCAL, fn ou Commit. Rollback sempre executado
//     se algo falhar (no-op se Commit rodar primeiro).
//
// Importante: SET LOCAL só funciona DENTRO de uma transação. Helper
// garante que isso é respeitado.
//
// Validação 59 (v3.33.5) [F-59-A]: tentativa de retry-on-SQLITE_BUSY
// (3 attempts, backoff 5/10/20ms) REVERTIDA em V59 — empírica em 15
// runs pós-implementação: 5/15 PASS (33%), regressão vs 11/15 (73%)
// pré-retry. Root cause: retries amplificam contenção (cada retry pega
// nova conn → in-flight count sobe → próxima iteração contenção maior).
// Carry-over para Sprint polish; estudar alternativa (retry só no auditlog.Log,
// não no helper central).
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

	if err := fn(tx); err != nil {
		return fmt.Errorf("tx fn: %w", err)
	}

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
