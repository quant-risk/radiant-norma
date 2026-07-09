package canonical

import (
	"testing"
	"time"
)

func TestNewCanonical(t *testing.T) {
	now := time.Now()
	doc := NewCanonical("if-123", now, "3040")

	if doc.IFID != "if-123" {
		t.Errorf("IFID: got %q, want %q", doc.IFID, "if-123")
	}
	if time.Time(doc.DataBase) != now {
		t.Errorf("DataBase: got %v, want %v", time.Time(doc.DataBase), now)
	}
	if doc.CadocCode != "3040" {
		t.Errorf("CadocCode: got %q, want %q", doc.CadocCode, "3040")
	}
	if doc.Extra == nil {
		t.Error("Extra should be initialized")
	}
}

func TestCanonicalDocument_Validate(t *testing.T) {
	tests := []struct {
		name   string
		doc    *CanonicalDocument
		wantErr int // expected number of errors
	}{
		{
			name:   "empty document",
			doc:    &CanonicalDocument{},
			wantErr: 5, // IFID, CadocCode, DataBase, CNPJ, NomeIF
		},
		{
			name: "missing CNPJ",
			doc: &CanonicalDocument{
				IFID:      "if-123",
				CadocCode: "3040",
				DataBase:  DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
				Header:    DocumentHeader{NomeIF: "Banco Teste"},
			},
			wantErr: 1, // only CNPJ
		},
		{
			name: "missing NomeIF",
			doc: &CanonicalDocument{
				IFID:      "if-123",
				CadocCode: "3040",
				DataBase:  DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
				Header:    DocumentHeader{CNPJ: "12345678000123"},
			},
			wantErr: 1, // only NomeIF
		},
		{
			name: "all required fields present",
			doc: &CanonicalDocument{
				IFID:      "if-123",
				CadocCode: "3040",
				DataBase:  DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
				Header: DocumentHeader{
					CNPJ:   "12345678000123",
					NomeIF: "Banco Teste S.A.",
				},
			},
			wantErr: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.doc.Validate()
			if len(errs) != tt.wantErr {
				t.Errorf("Validate() = %v, want %d errors", errs, tt.wantErr)
			}
		})
	}
}

func TestDataBase_Format(t *testing.T) {
	db := DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	got := time.Time(db).Format("200601")
	if got != "202607" {
		t.Errorf("DataBase.Format: got %q, want %q", got, "202607")
	}
}
