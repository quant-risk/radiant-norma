// Package audit — tests da integração com ruleprefs (Sprint 12 C32.23).
//
// Verifica que regras desabilitadas via ruleprefs são puladas no engine
// de validação E aparecem em ValidationResponse.DisabledRules.

package audit

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// fakePrefs é stub da interface RulePrefs pra tests.
type fakePrefs struct {
	disabled []string
}

func (f *fakePrefs) ListDisabledCodes(_ context.Context, _ string) ([]string, error) {
	return f.disabled, nil
}

func setupValidateTest(t *testing.T) (*Service, *sql.DB, func()) {
	t.Helper()
	tmpDir, _ := os.MkdirTemp("", "audit-ruleprefs-*")
	defer t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	d, err := sql.Open("sqlite", filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Cria tabelas mínimas (criticas + disabled_rules).
	for _, stmt := range []string{
		`CREATE TABLE criticas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cadoc_code TEXT NOT NULL,
			sheet TEXT,
			codigo TEXT NOT NULL,
			regra TEXT,
			descricao TEXT,
			gravidade TEXT,
			data_base_inicio DATETIME,
			mensagem_erro TEXT,
			enabled INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE disabled_rules (
			if_id TEXT NOT NULL,
			rule_code TEXT NOT NULL,
			disabled_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			disabled_by TEXT NOT NULL,
			PRIMARY KEY (if_id, rule_code)
		)`,
	} {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	// Insere 3 regras (todas habilitadas). Sheet como empty string
	// pra evitar NULL scan error em LoadCriticas.
	for _, code := range []string{"B12", "F23", "S05"} {
		_, err := d.Exec(`INSERT INTO criticas (cadoc_code, sheet, codigo, regra, descricao, gravidade, enabled) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"4060", "", code, code, "rule "+code, "E", 1)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	svc := New(d)
	return svc, d, func() { _ = d.Close() }
}

func TestValidate_FiltersDisabledRules(t *testing.T) {
	svc, _, cleanup := setupValidateTest(t)
	defer cleanup()

	// Stub prefs: F23 desabilitada
	svc.SetRulePrefs(&fakePrefs{disabled: []string{"F23"}})

	// Request: XML válido pra passar L1 (parser exige <Documento> em 4060).
	// Conteúdo irrelevante — L2 não vai aplicar regra nenhuma (registry
	// vazio pra regras dummy). Mas vamos só verificar que F23 é pulada
	// e DisabledRules é populada.
	req := &ValidationRequest{
		CadocCode:   "4060",
		XML:         `<Documento></Documento>`,
		ContentType: "application/xml",
		IfID:        "demo",
	}

	resp, err := svc.Validate(context.Background(), req)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// F23 deve estar pulada (não conta como erro)
	for _, e := range resp.Errors {
		if e.Critica.Codigo == "F23" {
			t.Errorf("F23 should be skipped (disabled), but appears in errors: %+v", e)
		}
	}

	// DisabledRules deve listar F23
	if !contains(resp.DisabledRules, "F23") {
		t.Errorf("expected DisabledRules to contain F23, got %v", resp.DisabledRules)
	}
}

func TestValidate_NoPrefsRunsAllRules(t *testing.T) {
	svc, _, cleanup := setupValidateTest(t)
	defer cleanup()

	// Sem SetRulePrefs — comportamento legacy: todas rodam
	req := &ValidationRequest{
		CadocCode:   "4060",
		XML:         `<Documento></Documento>`,
		ContentType: "application/xml",
		IfID:        "demo",
	}

	resp, err := svc.Validate(context.Background(), req)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// DisabledRules deve ser vazio (sem prefs injetado)
	if len(resp.DisabledRules) != 0 {
		t.Errorf("expected empty DisabledRules without prefs, got %v", resp.DisabledRules)
	}
}

func TestValidate_NoIfIDSkipsFilter(t *testing.T) {
	svc, _, cleanup := setupValidateTest(t)
	defer cleanup()

	svc.SetRulePrefs(&fakePrefs{disabled: []string{"F23"}})

	// Sem IfID no request — sem filtro
	req := &ValidationRequest{
		CadocCode:   "4060",
		XML:         `<Documento></Documento>`,
		ContentType: "application/xml",
		IfID:        "", // empty
	}

	resp, err := svc.Validate(context.Background(), req)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if len(resp.DisabledRules) != 0 {
		t.Errorf("expected empty DisabledRules when IfID is empty, got %v", resp.DisabledRules)
	}
}

func contains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// keep "strings" used (we'll use it elsewhere)
var _ = strings.Contains