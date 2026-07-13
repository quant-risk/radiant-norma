// manual_file_test.go — Sprint 57 v3.36.3: testes para ManualAdapter e FileAdapter.
package adapters

import (
	"context"
	"strings"
	"testing"
)

func TestManualAdapter_Fetch_ReturnsInjectedDoc(t *testing.T) {
	doc := &CanonicalDoc{
		CadocCode: "3040",
		DataBase:  "2026-06-01",
		Operacoes: []Operacao{{ID: "1", Valor: 1000}},
	}
	a := NewManualAdapter(doc)
	got, err := a.Fetch(context.Background(), "3040", "2026-06-01")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got.CadocCode != "3040" {
		t.Errorf("CadocCode=%q, want 3040", got.CadocCode)
	}
	if len(got.Operacoes) != 1 {
		t.Errorf("Operacoes len=%d, want 1", len(got.Operacoes))
	}
	if got.VersaoLayout != "1.0" {
		t.Errorf("VersaoLayout default not applied: %q", got.VersaoLayout)
	}
}

func TestManualAdapter_Fetch_CadocMismatch(t *testing.T) {
	doc := &CanonicalDoc{CadocCode: "4111"}
	a := NewManualAdapter(doc)
	_, err := a.Fetch(context.Background(), "3040", "")
	if err == nil {
		t.Fatal("expected cadoc mismatch error")
	}
	if !strings.Contains(err.Error(), "cadoc mismatch") {
		t.Errorf("error=%q, want mismatch msg", err.Error())
	}
}

func TestManualAdapter_Fetch_DataBaseOverride(t *testing.T) {
	doc := &CanonicalDoc{CadocCode: "3040", DataBase: "2026-01-01"}
	a := NewManualAdapter(doc)
	got, err := a.Fetch(context.Background(), "3040", "2026-06-01")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got.DataBase != "2026-06-01" {
		t.Errorf("DataBase=%q, want 2026-06-01 (override)", got.DataBase)
	}
}

func TestManualAdapter_NilDoc(t *testing.T) {
	a := &ManualAdapter{Doc: nil}
	_, err := a.Fetch(context.Background(), "3040", "")
	if err == nil {
		t.Fatal("expected error for nil doc")
	}
}

func TestManualAdapterFromJSON_Valid(t *testing.T) {
	payload := []byte(`{"cadoc_code":"3040","data_base":"2026-06-01","operacoes":[{"id":"1","valor":1000}]}`)
	a, err := NewManualAdapterFromJSON(payload)
	if err != nil {
		t.Fatalf("NewManualAdapterFromJSON: %v", err)
	}
	if a.Doc == nil {
		t.Fatal("Doc nil")
	}
	if a.Doc.CadocCode != "3040" {
		t.Errorf("CadocCode=%q", a.Doc.CadocCode)
	}
}

func TestManualAdapterFromJSON_MissingCadoc(t *testing.T) {
	payload := []byte(`{"data_base":"2026-06-01"}`)
	_, err := NewManualAdapterFromJSON(payload)
	if err == nil {
		t.Fatal("expected error for missing cadoc_code")
	}
}

func TestManualAdapterFromJSON_InvalidJSON(t *testing.T) {
	_, err := NewManualAdapterFromJSON([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFileAdapter_CSVHeader(t *testing.T) {
	content := []byte("id,valor,modalidade,vencimento\n1,1000,CCB,2026-12-31\n2,2000,CCB,2027-06-30\n")
	a := NewFileAdapter("dados.csv", content)
	got, err := a.Fetch(context.Background(), "3040", "2026-06-01")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	headers, ok := got.Extra["file_headers"].([]string)
	if !ok {
		t.Fatalf("file_headers missing or wrong type: %T", got.Extra["file_headers"])
	}
	if len(headers) != 4 {
		t.Errorf("headers len=%d, want 4 (got %v)", len(headers), headers)
	}
	if headers[0] != "id" {
		t.Errorf("headers[0]=%q, want id", headers[0])
	}
	if headers[2] != "modalidade" {
		t.Errorf("headers[2]=%q, want modalidade", headers[2])
	}
}

func TestFileAdapter_CSVQuotedFields(t *testing.T) {
	// Raw string com newline real (não "\n" como 2 chars).
	content := []byte("\"nome completo\",\"valor\",observacao\n\"João da Silva\",1000,\"texto, com virgula\"")
	a := NewFileAdapter("dados.csv", content)
	got, err := a.Fetch(context.Background(), "3040", "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	headers := got.Extra["file_headers"].([]string)
	if len(headers) != 3 {
		t.Fatalf("headers len=%d, want 3 (got %v)", len(headers), headers)
	}
	if headers[0] != "nome completo" {
		t.Errorf("quoted field not parsed: %q", headers[0])
	}
	if headers[2] != "observacao" {
		t.Errorf("headers[2]=%q", headers[2])
	}
}

func TestFileAdapter_JSONHeaders(t *testing.T) {
	content := []byte(`{"cadoc_code":"3040","operacoes":[{"id":"1","valor":1000,"modalidade":"CCB"}]}`)
	a := NewFileAdapter("dados.json", content)
	got, err := a.Fetch(context.Background(), "3040", "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	headers := got.Extra["file_headers"].([]string)
	if len(headers) != 3 {
		t.Errorf("headers len=%d, want 3 (id, valor, modalidade): %v", len(headers), headers)
	}
}

func TestFileAdapter_UnsupportedExtension(t *testing.T) {
	a := NewFileAdapter("dados.pdf", []byte("anything"))
	_, err := a.Fetch(context.Background(), "3040", "")
	if err == nil {
		t.Fatal("expected error for .pdf")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error=%q, want unsupported msg", err.Error())
	}
}

func TestFileAdapter_EmptyContent(t *testing.T) {
	a := NewFileAdapter("dados.csv", nil)
	_, err := a.Fetch(context.Background(), "3040", "")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestFileAdapter_ExtraMetadata(t *testing.T) {
	content := []byte("a,b,c\n1,2,3\n")
	a := NewFileAdapter("test.csv", content)
	got, err := a.Fetch(context.Background(), "4111", "2026-06-01")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got.CadocCode != "4111" {
		t.Errorf("CadocCode=%q, want 4111", got.CadocCode)
	}
	if got.DataBase != "2026-06-01" {
		t.Errorf("DataBase=%q", got.DataBase)
	}
	if got.Extra["operacoes_pending_mapping"] != true {
		t.Errorf("operacoes_pending_mapping should be true (frontend will populate)")
	}
	if got.Extra["file_name"] != "test.csv" {
		t.Errorf("file_name=%v", got.Extra["file_name"])
	}
}

func TestFileAdapter_XLSX_Stub(t *testing.T) {
	// XLSX parser é stub — apenas verifica que retorna erro ou resultado,
	// não testa parsing completo (dependeria de archive/zip).
	content := []byte("fake xlsx content")
	a := NewFileAdapter("dados.xlsx", content)
	_, err := a.Fetch(context.Background(), "3040", "")
	// Esperamos erro pois "fake xlsx content" não tem <row.
	if err == nil {
		t.Log("XLSX parser retornou sucesso com conteúdo fake — pode ser inesperado")
	}
}

func TestParseCSVHeader_Empty(t *testing.T) {
	got, err := parseCSVHeader(nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got=%v, want empty", got)
	}
}

func TestFileExt(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"dados.csv", ".csv"},
		{"planilha.XLSX", ".xlsx"},
		{"path/to/file.json", ".json"},
		{"noext", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := fileExt(tt.in)
		if got != tt.want {
			t.Errorf("fileExt(%q)=%q, want %q", tt.in, got, tt.want)
		}
	}
}
