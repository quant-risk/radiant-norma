// Package audit — tests for the Phase 1.1 fail-closed contract.
//
// These tests verify the L1 validator's new behavior, which is the central
// change of Phase 1.1 of the remediation plan. The benchmark black-box
// (e2e/run_benchmarks.py) sends empty root XMLs to /v1/validate and expects
// rejection; these unit tests are the in-process mirror of that contract so
// regressions show up in `go test ./...` instead of only after a full E2E run.
package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// TestPhase1_1_RejectsEmptyRootCadoc2030 verifies that the canonical empty
// document for CADOC 2030 (<DocumentoDRSAC/>) is rejected by L1.
//
// Before Phase 1.1: this document was approved silently by the root-tag
// fallback (no schema registered for 2030, root matched "DocumentoDRSAC").
// After Phase 1.1: schema unavailability is ErrSchemaUnavailable and the
// strict-fallback rejects the empty root element.
//
// Note: the response Message is intentionally generic ("documento XML/JSON
// inválido") because Validation 19 (F19.11) sanitizes L1 parse errors to
// avoid leaking XML element names, attribute paths, or SQL fragments
// through the public API. The detailed reason lives in the server log.
// We assert behavior (rejection with passed=false and a non-empty errors
// list), not the user-facing message.
func TestPhase1_1_RejectsEmptyRootCadoc2030(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := New(d)

	empty := `<?xml version="1.0"?><DocumentoDRSAC/>`
	resp, err := svc.Validate(context.Background(), &ValidationRequest{
		CadocCode: "2030",
		DataBase:  "2026-06",
		XML:       empty,
	})
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if resp.Passed {
		t.Errorf("expected empty 2030 document to be rejected; passed=true, errors=%v", resp.Errors)
	}
	if len(resp.Errors) == 0 {
		t.Error("expected at least one error explaining the rejection")
	}
	if len(resp.Errors) > 0 && resp.Errors[0].Critica.Codigo != "L1-PARSE" {
		t.Errorf("expected rejection to be classified as L1-PARSE, got %q", resp.Errors[0].Critica.Codigo)
	}
}

// TestPhase1_1_RejectsEmptyRootCadoc4111 covers the CADOC whose root
// tag in the benchmark is "Documento" (legacy) but whose generator
// emits "Documento4111" (canonical). The validator should reject
// either empty variant.
func TestPhase1_1_RejectsEmptyRootCadoc4111(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := New(d)

	cases := []string{
		`<?xml version="1.0"?><Documento/>`,
		`<?xml version="1.0"?><Documento4111/>`,
	}
	for _, xml := range cases {
		t.Run(xml, func(t *testing.T) {
			resp, err := svc.Validate(context.Background(), &ValidationRequest{
				CadocCode: "4111",
				DataBase:  "2026-06",
				XML:       xml,
			})
			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
			if resp.Passed {
				t.Errorf("expected empty 4111 document to be rejected; passed=true, errors=%v", resp.Errors)
			}
		})
	}
}

// TestPhase1_1_AcceptsNonEmpty3040 verifies that a 3040 document with
// attributes (even with no children) is accepted by L1 — the strict
// fallback distinguishes "empty" (no attrs, no children) from "stub"
// (has attrs or has children). This preserves the ability to validate
// L2/L3 rules on documents that pass L1.
func TestPhase1_1_AcceptsNonEmpty3040(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := New(d)

	cases := []string{
		// Has attributes — non-empty.
		`<?xml version="1.0"?><Doc3040 DtBase="2026-06" CNPJ="12345678"/>`,
		// Has children — non-empty.
		`<?xml version="1.0"?><Doc3040><X/></Doc3040>`,
	}
	for _, xml := range cases {
		t.Run(xml, func(t *testing.T) {
			_, err := svc.Validate(context.Background(), &ValidationRequest{
				CadocCode: "3040",
				DataBase:  "2026-06",
				XML:       xml,
			})
			if err != nil {
				t.Errorf("expected non-empty 3040 document to pass L1; got error: %v", err)
			}
		})
	}
}

// TestPhase1_1_RejectsUnknownCadoc verifies that a CADOC not in
// expectedRootTag is rejected at L1 (fail-closed for unknown CADOCs).
func TestPhase1_1_RejectsUnknownCadoc(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := New(d)

	resp, err := svc.Validate(context.Background(), &ValidationRequest{
		CadocCode: "9999",
		DataBase:  "2026-06",
		XML:       `<?xml version="1.0"?><Algo stub="x"/>`,
	})
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if resp.Passed {
		t.Errorf("expected unknown CADOC 9999 to be rejected; passed=true, errors=%v", resp.Errors)
	}
}

// TestPhase1_1_ErrSchemaUnavailable_Type verifies the typed error.
// Callers can use errors.As to distinguish "schema not registered"
// from other validation failures.
func TestPhase1_1_ErrSchemaUnavailable_Type(t *testing.T) {
	_, err := ValidateXSD("2030", "<Doc/>")
	if err == nil {
		t.Fatal("expected ErrSchemaUnavailable for CADOC 2030, got nil")
	}
	var unavailable *ErrSchemaUnavailable
	if !errors.As(err, &unavailable) {
		t.Errorf("expected *ErrSchemaUnavailable, got %T: %v", err, err)
	}
	if unavailable != nil && unavailable.Cadoc != "2030" {
		t.Errorf("ErrSchemaUnavailable.Cadoc = %q, want %q", unavailable.Cadoc, "2030")
	}
}

// TestPhase1_1_SupportedCADOCs_NotEmpty guards against accidentally
// emptying the schema map (which would make Validate refuse everything).
func TestPhase1_1_SupportedCADOCs_NotEmpty(t *testing.T) {
	supported := SupportedCADOCs()
	if len(supported) == 0 {
		t.Fatal("SupportedCADOCs returned empty list — validator would refuse everything")
	}
	// We expect at least the three schemas that ship in this build.
	required := []string{"3050", "3045", "3040"}
	have := make(map[string]bool, len(supported))
	for _, s := range supported {
		have[s] = true
	}
	for _, r := range required {
		if !have[r] {
			t.Errorf("SupportedCADOCs missing %q (have: %v)", r, supported)
		}
	}
}
