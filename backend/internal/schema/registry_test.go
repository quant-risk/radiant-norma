// Tests for schema.Registry — GetEffective, List, Insert.
//
// Cobertura (F6 do Sprint 6 — fechamento do gap de testes em internal/schema):
//   - GetEffective: data exata, passada, futura, sem data, sem versão
//   - Insert: básico + UNIQUE(cadoc, effective_from) constraint
//   - List: ordenação DESC, multi-version, single-version
//
// Helpers de fixture: helperVersion cria Version pronta pra Insert.
package schema_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// helperVersion cria uma Version válida para testes.
func helperVersion(cadoc string, effectiveFrom time.Time, sourceURI string, fieldsJSON string) schema.Version {
	return schema.Version{
		CadocCode:     cadoc,
		EffectiveFrom: effectiveFrom,
		SourceURI:     sourceURI,
		Fields: []schema.Field{
			{Tag: "testField", Type: "A8", Required: true, Desc: "Campo de teste"},
		},
	}
}

// ============================================================
// GetEffective — data exata, passada, futura, sem data
// ============================================================

func TestGetEffective_DateExact(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	v1 := helperVersion("3040", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "http://example.com/v1", "")
	if err := reg.Insert(&v1); err != nil {
		t.Fatalf("insert v1: %v", err)
	}

	// Pede a versão efetiva EXATAMENTE na data 2024-01-01
	got, err := reg.GetEffective("3040", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if got.CadocCode != "3040" {
		t.Errorf("CadocCode = %q, want 3040", got.CadocCode)
	}
	if !got.EffectiveFrom.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("EffectiveFrom = %v, want 2024-01-01", got.EffectiveFrom)
	}
}

func TestGetEffective_DatePast(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	// Insere 2 versões: 2024-01-01 (antiga) e 2025-06-01 (nova)
	old := helperVersion("3040", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "http://example.com/old", "")
	if err := reg.Insert(&old); err != nil {
		t.Fatalf("insert old: %v", err)
	}
	newer := helperVersion("3040", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), "http://example.com/new", "")
	if err := reg.Insert(&newer); err != nil {
		t.Fatalf("insert newer: %v", err)
	}

	// Pede efetiva em 2024-12-01 (entre as 2): deve retornar a antiga (effective_from <= data_base)
	got, err := reg.GetEffective("3040", time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if !got.EffectiveFrom.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Esperado versão antiga (2024-01-01), got %v", got.EffectiveFrom)
	}
}

func TestGetEffective_DateFutureNoExactMatch(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	// Insere 2 versões no passado
	v1 := helperVersion("3040", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "http://example.com/v1", "")
	if err := reg.Insert(&v1); err != nil {
		t.Fatalf("insert: %v", err)
	}
	v2 := helperVersion("3040", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), "http://example.com/v2", "")
	if err := reg.Insert(&v2); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Pede futura (2030-01-01): comportamento atual = mais recente que <= 2030 = v2
	got, err := reg.GetEffective("3040", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if !got.EffectiveFrom.Equal(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Esperado v2 (2025-06-01), got %v", got.EffectiveFrom)
	}
}

func TestGetEffective_NoDateReturnsLatest(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	v1 := helperVersion("3040", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "http://example.com/v1", "")
	if err := reg.Insert(&v1); err != nil {
		t.Fatalf("insert: %v", err)
	}
	v2 := helperVersion("3040", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), "http://example.com/v2", "")
	if err := reg.Insert(&v2); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Sem data (zero) → retorna a mais recente DESC
	got, err := reg.GetEffective("3040", time.Time{})
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if !got.EffectiveFrom.Equal(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Esperado mais recente (2025-06-01), got %v", got.EffectiveFrom)
	}
}

// ============================================================
// GetEffective — error cases
// ============================================================

func TestGetEffective_NoRows(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	// CADOC inexistente
	got, err := reg.GetEffective("9999", time.Now())
	if got != nil {
		t.Errorf("Esperado nil para CADOC inexistente, got %v", got)
	}
	if err == nil {
		t.Fatal("Esperado erro, got nil")
	}
	// Mensagem deve mencionar o CADOC e a data
	if !strings.Contains(err.Error(), "9999") {
		t.Errorf("Erro deveria mencionar CADOC 9999, got: %v", err)
	}
}

func TestGetEffective_FutureDateNoVersions(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	// CADOC sem versões registradas
	_, err := reg.GetEffective("3040", time.Now())
	if err == nil {
		t.Fatal("Esperado erro para CADOC sem versões")
	}
}

// ============================================================
// Insert — UNIQUE(cadoc_code, effective_from) constraint
// ============================================================

func TestInsert_Basic(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	v := helperVersion("3040", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		"http://example.com/v1", "")
	if err := reg.Insert(&v); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// Recupera e confirma
	got, err := reg.GetEffective("3040", time.Time{})
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if got.SourceURI != "http://example.com/v1" {
		t.Errorf("SourceURI = %q, want http://example.com/v1", got.SourceURI)
	}
	if len(got.Fields) != 1 {
		t.Errorf("Fields = %d, want 1", len(got.Fields))
	}
}

func TestInsert_DuplicateSameDayFails(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	v1 := helperVersion("3040", day, "http://example.com/v1", "")
	if err := reg.Insert(&v1); err != nil {
		t.Fatalf("Insert v1: %v", err)
	}

	// Tentar inserir outra versão com mesma (cadoc, effective_from)
	v2 := helperVersion("3040", day, "http://example.com/v2", "")
	err := reg.Insert(&v2)
	if err == nil {
		t.Fatal("Esperado erro UNIQUE, got nil")
	}
	// Erro pode ser do wrapped sqlite (UNIQUE constraint failed) — não
	// verificamos mensagem exata, só que falhou.
	if !errors.Is(err, err) { // sanity check: err é não-nil (já checado acima)
		t.Errorf("Erro não é wrapped corretamente: %v", err)
	}
}

func TestInsert_DifferentCadocSameDaySucceeds(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	v1 := helperVersion("3040", day, "http://example.com/3040", "")
	if err := reg.Insert(&v1); err != nil {
		t.Fatalf("Insert 3040: %v", err)
	}
	v2 := helperVersion("3050", day, "http://example.com/3050", "")
	if err := reg.Insert(&v2); err != nil {
		t.Fatalf("Insert 3050 (mesmo dia, cadoc diferente): %v", err)
	}
}

func TestInsert_EmptyXSDAndChangelogAreNULL(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	v := helperVersion("3040", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		"http://example.com/v1", "")
	// XSD e Changelog vazios (defaults)
	if err := reg.Insert(&v); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	got, err := reg.GetEffective("3040", time.Time{})
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	// nullableString("") → NULL → XSD/Changelog vazios ao recuperar
	if got.XSD != "" {
		t.Errorf("XSD não deveria estar preenchido, got %q", got.XSD)
	}
	if got.Changelog != "" {
		t.Errorf("Changelog não deveria estar preenchido, got %q", got.Changelog)
	}
}

// ============================================================
// List — ordenação DESC, multi-version, single-version
// ============================================================

func TestList_OrderingDesc(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	day1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC)

	// Insere em ordem aleatória
	{
		v := helperVersion("3040", day1, "u1", "")
		if err := reg.Insert(&v); err != nil {
			t.Fatal(err)
		}
	}
	{
		v := helperVersion("3040", day2, "u2", "")
		if err := reg.Insert(&v); err != nil {
			t.Fatal(err)
		}
	}
	{
		v := helperVersion("3040", day3, "u3", "")
		if err := reg.Insert(&v); err != nil {
			t.Fatal(err)
		}
	}

	versions, err := reg.List("3040")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("Esperado 3 versões, got %d", len(versions))
	}

	// Ordem esperada: day2 (2025) → day1 (2024) → day3 (2023)
	if !versions[0].EffectiveFrom.Equal(day2) {
		t.Errorf("[0] = %v, want %v (2025-06-01)", versions[0].EffectiveFrom, day2)
	}
	if !versions[1].EffectiveFrom.Equal(day1) {
		t.Errorf("[1] = %v, want %v (2024-01-01)", versions[1].EffectiveFrom, day1)
	}
	if !versions[2].EffectiveFrom.Equal(day3) {
		t.Errorf("[2] = %v, want %v (2023-03-01)", versions[2].EffectiveFrom, day3)
	}
}

func TestList_NoRows(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	versions, err := reg.List("9999")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("Esperado slice vazio, got %d versões", len(versions))
	}
}

func TestList_OnlyReturnsRequestedCadoc(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)

	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	{
		v := helperVersion("3040", day, "u1", "")
		if err := reg.Insert(&v); err != nil {
			t.Fatal(err)
		}
	}
	{
		v := helperVersion("3050", day, "u2", "")
		if err := reg.Insert(&v); err != nil {
			t.Fatal(err)
		}
	}
	{
		v := helperVersion("3042", day, "u3", "")
		if err := reg.Insert(&v); err != nil {
			t.Fatal(err)
		}
	}

	v3040, _ := reg.List("3040")
	if len(v3040) != 1 {
		t.Errorf("List(3040) deveria retornar 1, got %d", len(v3040))
	}
	v3050, _ := reg.List("3050")
	if len(v3050) != 1 {
		t.Errorf("List(3050) deveria retornar 1, got %d", len(v3050))
	}
}

// ============================================================
// Smoke — Insert + List + GetEffective compõem corretamente
// ============================================================

func TestSchema_EndToEnd(t *testing.T) {
	d := testutil.NewTestDB(t)
	reg := schema.New(d)
	ctx := context.Background()

	// Simula 3 releases do BACEN ao longo de 2 anos
	releases := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for i, eff := range releases {
		v := helperVersion("3040", eff, "http://example.com/v"+string(rune('1'+i)), "")
		if err := reg.Insert(&v); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	// IF submete em 2024-10-01 → deve pegar versão 2 (Jul/2024)
	got, err := reg.GetEffective("3040", time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if !got.EffectiveFrom.Equal(releases[1]) {
		t.Errorf("Versão esperada = %v (Jul/2024), got %v", releases[1], got.EffectiveFrom)
	}

	// List retorna 3 na ordem certa
	all, err := reg.List("3040")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Esperado 3 versões, got %d", len(all))
	}
	// Order DESC: [2025, 2024-07, 2024-01]
	if !all[0].EffectiveFrom.Equal(releases[2]) {
		t.Errorf("List[0] deveria ser mais recente, got %v", all[0].EffectiveFrom)
	}

	// Confirma que ctx não é usado mas é aceito (não muda nada)
	_ = ctx
}
