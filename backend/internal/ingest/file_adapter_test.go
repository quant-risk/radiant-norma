package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
)

func TestFileAdapter_Fetch_CSV(t *testing.T) {
	content := `cnpj,nome_if,modalidade,valor,uf,tipo_pessoa,classificacao_if,contrato
12345678000123,Banco Teste S.A.,1000,50000.00,SP,PJ,A,C001
12345678000123,Banco Teste S.A.,1000,30000.00,RJ,PJ,B,C002
12345678000123,Banco Teste S.A.,2000,100000.00,SP,PF,C,C003
`
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(csvPath, []byte(content), 0600); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}

	adapter := &FileAdapter{}
	cfg := SourceConfig{
		File: &FileConfig{
			Path:      csvPath,
			Format:    "csv",
			HasHeader: true,
		},
	}

	doc, fetchErr := adapter.Fetch(context.Background(), cfg, "3040", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if fetchErr != nil {
		t.Fatalf("Fetch() error = %v", fetchErr)
	}

	if doc.Header.CNPJ != "12345678000123" {
		t.Errorf("CNPJ = %q, want %q", doc.Header.CNPJ, "12345678000123")
	}
	if doc.Header.NomeIF != "Banco Teste S.A." {
		t.Errorf("NomeIF = %q, want %q", doc.Header.NomeIF, "Banco Teste S.A.")
	}
	if len(doc.Operacoes) != 3 {
		t.Errorf("len(Operacoes) = %d, want 3", len(doc.Operacoes))
	}

	op := doc.Operacoes[0]
	if op.Modalidade != "1000" {
		t.Errorf("op[0].Modalidade = %q, want %q", op.Modalidade, "1000")
	}
	if op.UF != "SP" {
		t.Errorf("op[0].UF = %q, want %q", op.UF, "SP")
	}

	// Assert canonical type is used.
	var _ *canonical.CanonicalDocument = doc
}

func TestFileAdapter_Fetch_CSV_NoHeader(t *testing.T) {
	content := `12345678000123,Banco Teste,1000,50000.00
`
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "noheader.csv")
	if err := os.WriteFile(csvPath, []byte(content), 0600); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}

	adapter := &FileAdapter{}
	cfg := SourceConfig{
		File: &FileConfig{
			Path:      csvPath,
			Format:    "csv",
			HasHeader: false,
		},
	}

	doc, fetchErr := adapter.Fetch(context.Background(), cfg, "3040", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if fetchErr != nil {
		t.Fatalf("Fetch() error = %v", fetchErr)
	}
	if len(doc.Operacoes) != 1 {
		t.Errorf("len(Operacoes) = %d, want 1", len(doc.Operacoes))
	}

	var _ *canonical.CanonicalDocument = doc
}

func TestFileAdapter_ValidateConfig(t *testing.T) {
	adapter := &FileAdapter{}

	cfg := SourceConfig{File: &FileConfig{Path: "/tmp/test.csv", Format: "csv"}}
	if err := adapter.ValidateConfig(cfg); err != nil {
		t.Errorf("ValidateConfig valid: got error %v", err)
	}

	cfg = SourceConfig{File: &FileConfig{Format: "csv"}}
	if err := adapter.ValidateConfig(cfg); err == nil {
		t.Error("ValidateConfig missing path: expected error, got nil")
	}
}

func TestFileAdapter_HealthCheck(t *testing.T) {
	adapter := &FileAdapter{}
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "exists.csv")
	os.WriteFile(csvPath, []byte("a,b,c"), 0600)

	cfg := SourceConfig{File: &FileConfig{Path: csvPath}}
	if err := adapter.HealthCheck(context.Background(), cfg); err != nil {
		t.Errorf("HealthCheck existing file: got error %v", err)
	}

	cfg = SourceConfig{File: &FileConfig{Path: "/tmp/nonexistent.csv"}}
	if err := adapter.HealthCheck(context.Background(), cfg); err == nil {
		t.Error("HealthCheck missing file: expected error, got nil")
	}
}

func TestMoney(t *testing.T) {
	cases := []struct {
		input   string
		wantVal float64
	}{
		{"50000.00", 50000.00},
		{"50.000,00", 50000.00},
		{"R$ 50.000,00", 50000.00},
		{"$50000.00", 50000.00},
		{"50000", 50000.00},
	}
	for _, c := range cases {
		m := money(c.input)
		got, _ := m.Valor.Float64()
		if got != c.wantVal {
			t.Errorf("money(%q) = %v, want %v", c.input, got, c.wantVal)
		}
	}
}

func TestNormalizeHeader(t *testing.T) {
	cases := []struct{ input, want string }{
		{"CNPJ", "cnpj"},
		{"Nome da IF", "nome_da_if"},
		{"Valor Principal", "valor_principal"},
		{"Taxa de Juros", "taxa_de_juros"},
	}
	for _, c := range cases {
		got := normalizeHeader(c.input)
		if got != c.want {
			t.Errorf("normalizeHeader(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestParseDate(t *testing.T) {
	cases := []struct{ input, want string }{
		{"2026-07-01", "2026-07-01"},
		{"01/07/2026", "2026-07-01"},
		{"20260701", "2026-07-01"},
	}
	for _, c := range cases {
		got := parseDate(c.input)
		if got.Format("2006-01-02") != c.want {
			t.Errorf("parseDate(%q) = %v, want %v", c.input, got.Format("2006-01-02"), c.want)
		}
	}
}

func TestCleanCNPJ(t *testing.T) {
	cases := []struct{ input, want string }{
		{"123.456.789/0001-23", "123456789000123"},
		{"12345678000123", "12345678000123"},
	}
	for _, c := range cases {
		got := cleanCNPJ(c.input)
		if got != c.want {
			t.Errorf("cleanCNPJ(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestCoalesce(t *testing.T) {
	if got := coalesce("a", "b"); got != "a" {
		t.Errorf("coalesce(a,b) = %q, want a", got)
	}
	if got := coalesce("", "b"); got != "b" {
		t.Errorf("coalesce('',b) = %q, want b", got)
	}
	if got := coalesce("", "", "c"); got != "c" {
		t.Errorf("coalesce('','',c) = %q, want c", got)
	}
}
