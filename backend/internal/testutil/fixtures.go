// Package testutil — fixtures helpers pra Sprint 8c tests.
//
// Validação 30 (C18 fix): os 7 handlers novos do Sprint 8c não tinham
// tests. Aqui criamos SeedTest* helpers + fixtures tipadas pra testá-los
// em happy path + edge cases.
package testutil

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
)

// --- Envio fixtures ---

// EnvioFixture é um envio mockado pra tests.
type EnvioFixture struct {
	ID          string
	Cadoc       string
	Status      string // accepted | rejected | pending | error
	RulesPassed int
	RulesFailed int
	DaysAgo     int // 0 = hoje, 5 = 5 dias atrás (envio antigo)
}

// SeedTestEnvios insere envios no DB de teste.
//
// Garante IF existe. Defaults razoáveis: period = MM/YYYY atual.
func SeedTestEnvios(t *testing.T, d *sql.DB, ifID string, fs []EnvioFixture) {
	t.Helper()
	// Garante IF existe (FK constraint)
	_, err := d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, segmento, plano)
		VALUES (?, ?, ?, ?, ?, ?)`, ifID, "00000000", "Test IF "+ifID, "SCD", "S5", "pro")
	if err != nil {
		t.Fatalf("insert if: %v", err)
	}

	now := time.Now()
	period := fmt.Sprintf("%02d/%04d", int(now.Month()), now.Year())

	stmt, err := d.Prepare(`
		INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
		                    status, rules_passed, rules_failed, period, duration_ms, approver,
		                    sent_at, confirmed_at, xml_content, created_at)
		VALUES (?, ?, ?, ?, 1, 'hash', 'hash', ?, ?, ?, ?, 1500, 'system',
		        ?, ?, '', ?)
	`)
	if err != nil {
		t.Fatalf("prepare envio: %v", err)
	}
	defer stmt.Close()

	for _, f := range fs {
		if f.ID == "" {
			t.Fatalf("envio fixture sem ID")
		}
		daysAgo := f.DaysAgo
		if daysAgo == 0 {
			daysAgo = 1 // sent_at/confirmed_at não pode ser futuro
		}
		sentAt := now.AddDate(0, 0, -daysAgo)
		confirmedAt := sentAt.Add(time.Minute)

		// Sprint 13: data_base deve ser YYYY-MM-DD (CHECK constraint).
		// Antes (gap): fixture usava period (MM/YYYY) aqui. Corrigido.
		dataBase := sentAt.Format("2006-01-02")
		_, err := stmt.Exec(
			f.ID, ifID, f.Cadoc, dataBase,
			f.Status, f.RulesPassed, f.RulesFailed, period,
			sentAt, confirmedAt, sentAt,
		)
		if err != nil {
			t.Fatalf("insert envio %s: %v", f.ID, err)
		}
	}
}

// --- RuleFailure fixtures ---

// RuleFailureFixture é uma falha mockada pra tests.
type RuleFailureFixture struct {
	Cadoc    string
	RuleCode string
	Severity string // E | A | I
	DaysAgo  int
	Count    int // múltiplas inserções se > 1
}

// SeedTestRuleFailures insere rule_failures no DB de teste.
//
// Count > 1 insere múltiplas rows com mesmo code/cadoc (pra testar agregação).
func SeedTestRuleFailures(t *testing.T, d *sql.DB, ifID string, fs []RuleFailureFixture) {
	t.Helper()

	stmt, err := d.Prepare(`
		INSERT INTO rule_failures (envio_id, if_id, cadoc_code, rule_code, rule_severity, failed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		t.Fatalf("prepare failure: %v", err)
	}
	defer stmt.Close()

	for _, f := range fs {
		// Garante envio existe (FK). Usa INSERT OR IGNORE pra idempotência,
		// mas se já existe (por outro test), só usa. O INSERT OR IGNORE
		// ignora conflito de PK — funciona mesmo se id já existe.
		envioID := "TEST-ENV-" + f.RuleCode
		_, err := d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, segmento, plano)
			VALUES (?, '00000000', 'Test', 'SCD', 'S5', 'pro')`, ifID)
		if err != nil {
			t.Fatalf("insert if: %v", err)
		}
		_, err = d.Exec(`INSERT OR IGNORE INTO envios (id, if_id, cadoc_code, data_base, remessa,
			xml_hash, zip_hash, status, period, created_at)
			VALUES (?, ?, ?, '2026-01-01', 1, 'h', 'h', 'accepted', '01/2026', ?)`,
			envioID, ifID, f.Cadoc, time.Now())
		if err != nil {
			t.Fatalf("insert envio: %v", err)
		}

		count := f.Count
		if count == 0 {
			count = 1
		}
		// Formato UTC sem timezone (YYYY-MM-DD HH:MM:SS) — compatível com
		// strftime do SQLite. Time.Time direto (com timezone) retorna NULL
		// silencioso em strftime (mesmo bug que pegamos no seed prod).
		failedAt := time.Now().UTC().AddDate(0, 0, -f.DaysAgo).Format("2006-01-02 15:04:05")
		for i := 0; i < count; i++ {
			_, err := stmt.Exec(envioID, ifID, f.Cadoc, f.RuleCode, f.Severity, failedAt)
			if err != nil {
				t.Fatalf("insert failure: %v", err)
			}
		}
	}
}

// --- AuditEvent fixtures ---

// AuditEventFixture é um evento denormalizado (audit_events).
type AuditEventFixture struct {
	IFID        string
	Action      string
	Description string
}

// SeedTestAuditEvents insere audit_events com audit_log_id válido.
// Pra tests que não precisam de chain (skip chain validation).
//
// Validação 30 (C18 follow-up): audit_events.audit_log_id é NOT NULL com
// FK pra audit_log. Antes tava NULL — gerava constraint failure.
func SeedTestAuditEvents(t *testing.T, d *sql.DB, fs []AuditEventFixture) {
	t.Helper()

	// Sprint 13 [S14.1]: audit_log.if_id agora tem FK → ifs(id).
	// Pre-seed IFs únicos que aparecem nos fixtures (id + cnpj unique).
	for i, f := range fs {
		if f.IFID == "" {
			continue
		}
		_, _ = d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, plano)
			VALUES (?, ?, ?, 'SCD', 'pro')`,
			f.IFID, fmt.Sprintf("aud%07d", i), "Test "+f.IFID)
	}

	// Cria 1 audit_log entry por evento (FK target). Sem payload real —
	// só o suficiente pra satisfazer a constraint.
	stmt, err := d.Prepare(`
		INSERT INTO audit_log (if_id, actor, action, target, payload_hash, prev_hash, entry_hash, metadata, created_at)
		VALUES (?, 'system', ?, '', 'hash', '0', 'hash', '{}', ?)
	`)
	if err != nil {
		t.Fatalf("prepare log: %v", err)
	}
	defer stmt.Close()

	for _, f := range fs {
		res, err := stmt.Exec(f.IFID, f.Action, time.Now())
		if err != nil {
			t.Fatalf("insert log: %v", err)
		}
		logID, _ := res.LastInsertId()
		if logID == 0 {
			t.Fatalf("insert log: got logID=0 (audit_log_id is FK target — needs to be set)")
		}

		_, err = d.Exec(`
			INSERT INTO audit_events (audit_log_id, if_id, actor, action, target, description, payload, created_at)
			VALUES (?, ?, 'system', ?, '', ?, '{}', ?)
		`, logID, f.IFID, f.Action, f.Description, time.Now())
		if err != nil {
			t.Fatalf("insert event (logID=%d ifID=%s): %v", logID, f.IFID, err)
		}
	}
}

// AuditLogFixture é uma entrada do audit_log (chain).
type AuditLogFixture struct {
	Actor  string
	Action string
	Target string
}

// SeedTestAuditLog insere entradas no audit_log + audit_events pra chain testing.
func SeedTestAuditLog(t *testing.T, d *sql.DB, fs []AuditLogFixture) {
	t.Helper()

	// Sprint 13 [S14.1]: audit_log.if_id FK → ifs(id). Pre-seed "demo".
	_, _ = d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, plano)
		VALUES (?, '00000010', 'Demo', 'SCD', 'pro')`, "demo")

	log := auditlog.New(d)
	for _, f := range fs {
		_, err := log.Log("demo", f.Actor, f.Action, f.Target, []byte("test"), map[string]any{})
		if err != nil {
			t.Fatalf("log: %v", err)
		}
	}
}

// NewAuditLogForTest cria um *auditlog.Logger com DB pronto pra Verify().
func NewAuditLogForTest(t *testing.T, d *sql.DB) *auditlog.Logger {
	t.Helper()
	return auditlog.New(d)
}
