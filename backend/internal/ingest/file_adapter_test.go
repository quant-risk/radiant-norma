package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/xuri/excelize/v2"
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

func TestFileAdapter_Fetch_XLSX(t *testing.T) {
	// Create a minimal XLSX in memory using excelize.
	f := excelize.NewFile()
	sheet := "Sheet1"
	// Row 1: header
	f.SetCellValue(sheet, "A1", "cnpj")
	f.SetCellValue(sheet, "B1", "nome_if")
	f.SetCellValue(sheet, "C1", "modalidade")
	f.SetCellValue(sheet, "D1", "valor")
	f.SetCellValue(sheet, "E1", "uf")
	f.SetCellValue(sheet, "F1", "tipo_pessoa")
	f.SetCellValue(sheet, "G1", "classificacao_if")
	f.SetCellValue(sheet, "H1", "contrato")
	// Row 2: IF metadata + first operation
	f.SetCellValue(sheet, "A2", "12345678000123")
	f.SetCellValue(sheet, "B2", "Banco Teste S.A.")
	f.SetCellValue(sheet, "C2", "1000")
	f.SetCellValue(sheet, "D2", "50000.00")
	f.SetCellValue(sheet, "E2", "SP")
	f.SetCellValue(sheet, "F2", "PJ")
	f.SetCellValue(sheet, "G2", "A")
	f.SetCellValue(sheet, "H2", "C001")
	// Row 3: second operation
	f.SetCellValue(sheet, "A3", "12345678000123")
	f.SetCellValue(sheet, "B3", "Banco Teste S.A.")
	f.SetCellValue(sheet, "C3", "1000")
	f.SetCellValue(sheet, "D3", "30000.00")
	f.SetCellValue(sheet, "E3", "RJ")
	f.SetCellValue(sheet, "F3", "PJ")
	f.SetCellValue(sheet, "G3", "B")
	f.SetCellValue(sheet, "H3", "C002")
	// Row 4: third operation
	f.SetCellValue(sheet, "A4", "12345678000123")
	f.SetCellValue(sheet, "B4", "Banco Teste S.A.")
	f.SetCellValue(sheet, "C4", "2000")
	f.SetCellValue(sheet, "D4", "100000.00")
	f.SetCellValue(sheet, "E4", "SP")
	f.SetCellValue(sheet, "F4", "PF")
	f.SetCellValue(sheet, "G4", "C")
	f.SetCellValue(sheet, "H4", "C003")

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}

	tmpDir := t.TempDir()
	xlsxPath := filepath.Join(tmpDir, "test.xlsx")
	if err := os.WriteFile(xlsxPath, buf.Bytes(), 0600); err != nil {
		t.Fatalf("write temp xlsx: %v", err)
	}

	adapter := &FileAdapter{}
	cfg := SourceConfig{
		File: &FileConfig{
			Path:      xlsxPath,
			Format:    "xlsx",
			HasHeader: true,
		},
	}

	doc, err := adapter.Fetch(context.Background(), cfg, "3040", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
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
		{"Nome da IF", "nomedaif"},
		{"Valor Principal", "valorprincipal"},
		{"Taxa de Juros", "taxadejuros"},
		{"modalidade", "modalidade"},
		{"valor_principal", "valorprincipal"},
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
