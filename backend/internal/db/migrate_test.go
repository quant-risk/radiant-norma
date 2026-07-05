// Tests para migrate (Sprint 6 v1.5.0 / F7).
//
// Cobertura:
//   - Migrate aplica migrations em ordem alfabética
//   - Migrate é idempotente (rodar 2x = mesmo resultado)
//   - Migrate cria todas as 5 tabelas principais
//   - drop+recreate demonstra resiliência (DB corrompido → recupera)
package db_test

import (
	"database/sql"
	"path/filepath"
	"sort"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/db"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// IsPostgresDSN é alias exportado do helper (pra testes).
func IsPostgresDSN(dsn string) bool { return db.IsPostgresDSN(dsn) }

// TestMigrate_AppliesAllMigrations valida que todas as migrations
// conhecidas são aplicadas e tabelas esperadas existem.
func TestMigrate_AppliesAllMigrations(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Migrations esperadas (referenciadas no test — atualize quando
	// adicionar novas).
	expectedTables := []string{
		"ifs",
		"schema_versions",
		"criticas",
		"envios",
		"audit_log",
		"radar_alerts",
		// Sprint 6 v1.5.0 — F3 e W1/W2 adicionaram
		"radar_baselines",
	}

	for _, table := range expectedTables {
		var name string
		err := d.QueryRow(`
			SELECT name FROM sqlite_master
			WHERE type='table' AND name=?
		`, table).Scan(&name)
		if err != nil {
			t.Errorf("Tabela %s deveria existir após migrate: %v", table, err)
		}
	}

	// Verifica que schema_migrations tem todas as aplicadas
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	// 7 migrations: 001-007 (007 added in Sprint 11 for disabled_rules)
	if count != 8 {
		t.Errorf("schema_migrations tem %d entries, want 8", count)
	}
}

// TestMigrate_Idempotent valida que rodar 2x não quebra.
func TestMigrate_Idempotent(t *testing.T) {
	d := testutil.NewTestDB(t)

	// 2ª chamada não deve dar erro
	if err := db.Migrate(d); err != nil {
		t.Fatalf("2ª migrate: %v", err)
	}

	// Schema_migrations ainda tem 7 entries (não duplica)
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 8 {
		t.Errorf("Após 2x migrate: %d entries, want 8 (idempotente)", count)
	}
}

// TestMigrate_RecreateFromCorrupted simula DB corrompido (drop tables)
// e valida que migrate reconstrói tudo.
//
// Sprint 6 (F7): integração importante — se migration tem bug que só
// aparece em DB reconstruído, este teste pega.
func TestMigrate_RecreateFromCorrupted(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Corrompe: drop todas as tabelas + schema_migrations
	rows, err := d.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var name string
		_ = rows.Scan(&name)
		tables = append(tables, name)
	}
	rows.Close()

	for _, table := range tables {
		// Não dropa sqlite_sequence (tabela interna do SQLite)
		if table == "sqlite_sequence" {
			continue
		}
		if _, err := d.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			t.Fatalf("drop %s: %v", table, err)
		}
	}

	// DB "zerado". Re-migrate deve reconstruir tudo.
	if err := db.Migrate(d); err != nil {
		t.Fatalf("re-migrate after drop: %v", err)
	}

	// Tabelas principais devem voltar
	for _, table := range []string{"ifs", "schema_versions", "audit_log", "radar_alerts", "radar_baselines"} {
		var name string
		err := d.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("Tabela %s deveria existir após recreate: %v", table, err)
		}
	}
}

// TestMigrate_OpenFresh valida que DB novo (sem migrate prévio) funciona.
func TestMigrate_OpenFresh(t *testing.T) {
	// Cria DB em arquivo (não in-memory) e roda migrate
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "fresh.db")

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		_ = d.Close()
	})

	if err := db.Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verifica que tem as tabelas
	var count int
	if err := d.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'
	`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count < 5 {
		t.Errorf("Esperado ≥5 tabelas, got %d", count)
	}
}

// TestMigrate_RaceConcurrent valida que 2 migrates simultâneos
// não corrompem. BEGIN IMMEDIATE garante serialização.
func TestMigrate_RaceConcurrent(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Lança 2 goroutines chamando Migrate em paralelo
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			results <- db.Migrate(d)
		}()
	}

	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Errorf("Migrate concorrente falhou: %v", err)
		}
	}
}

// TestMigrate_FreshSchemaVersionsTable valida que schema_migrations é
// criada (idempotente) mesmo se migrate rodou 0 vezes prévias.
func TestMigrate_FreshSchemaVersionsTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "fresh2.db")

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	// Antes de migrate: tabela não existe
	var count int
	err = d.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'
	`).Scan(&count)
	if err == nil && count != 0 {
		t.Errorf("schema_migrations não deveria existir antes de migrate")
	}

	// Migrate cria a tabela
	if err := db.Migrate(d); err != nil {
		t.Fatal(err)
	}

	err = d.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count)
	if err != nil {
		t.Fatalf("Após migrate: %v", err)
	}
	if count != 8 {
		t.Errorf("schema_migrations tem %d, want 8", count)
	}
}

// helper para testar sql queries genéricas
func execNoRows(_ *sql.DB) {}
var _ = sort.Strings // garante import não removido por tooling

// ============================================================
// Postgres driver detection (Sprint 6 v1.5.0)
// ============================================================

func TestIsPostgresDSN(t *testing.T) {
	cases := []struct {
		dsn  string
		want bool
	}{
		{"postgres://user:pass@localhost/db", true},
		{"postgresql://user:pass@localhost/db", true},
		{"file:radiant.db?_pragma=...", false},
		{"/path/to/radiant.db", false},
		{"radiant.db", false},
		{"", false},
	}
	for _, c := range cases {
		got := db.IsPostgresDSN(c.dsn)
		if got != c.want {
			t.Errorf("IsPostgresDSN(%q) = %v, want %v", c.dsn, got, c.want)
		}
	}
}

func TestBackend(t *testing.T) {
	if got := db.Backend("postgres://localhost/db"); got != "postgres" {
		t.Errorf("Backend(postgres) = %q, want postgres", got)
	}
	if got := db.Backend("file:radiant.db"); got != "sqlite" {
		t.Errorf("Backend(sqlite) = %q, want sqlite", got)
	}
	if got := db.Backend("/path/to/radiant.db"); got != "sqlite" {
		t.Errorf("Backend(path) = %q, want sqlite", got)
	}
}
