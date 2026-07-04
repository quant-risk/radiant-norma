// Package api — E2E tests pros handlers Sprint 8c.
//
// Validação 30 (C18 fix): zero coverage dos 7 handlers novos. Aqui
// criamos tests de integração que batem na API real (httptest.Server)
// com DB de teste (testutil). Cada handler é testado em happy path
// + edge cases críticos.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// --- listEnvios ---

func TestListEnvios_HappyPath(t *testing.T) {
	d := testutil.NewTestDB(t)
	testutil.SeedTestEnvios(t, d, "demo", []testutil.EnvioFixture{
		{ID: "E1", Cadoc: "3040", Status: "accepted", RulesPassed: 50, RulesFailed: 2},
		{ID: "E2", Cadoc: "3050", Status: "rejected", RulesPassed: 30, RulesFailed: 8},
		{ID: "E3", Cadoc: "3040", Status: "pending", RulesPassed: 0, RulesFailed: 0},
	})

	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/envios?limit=10", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.listEnvios(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Envios []struct {
			ID, CadocCode, Status string
			RulesPassed, RulesFailed int
		} `json:"envios"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected 3 envios, got %d", resp.Total)
	}
	if resp.Envios[0].CadocCode != "3040" { // ORDER BY DESC, mas todas têm mesmo time
		t.Logf("first cadoc: %s", resp.Envios[0].CadocCode)
	}
}

func TestListEnvios_FilterByCadoc(t *testing.T) {
	d := testutil.NewTestDB(t)
	testutil.SeedTestEnvios(t, d, "demo", []testutil.EnvioFixture{
		{ID: "E1", Cadoc: "3040", Status: "accepted"},
		{ID: "E2", Cadoc: "3050", Status: "accepted"},
	})

	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/envios?cadoc=3040", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.listEnvios(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Envios []struct {
			CadocCode string `json:"cadoc_code"`
		} `json:"envios"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("expected 1 envio (filter 3040), got %d", resp.Total)
	}
	if resp.Total > 0 && resp.Envios[0].CadocCode != "3040" {
		t.Errorf("expected only 3040, got %s", resp.Envios[0].CadocCode)
	}
}

func TestListEnvios_NoAuth(t *testing.T) {
	d := testutil.NewTestDB(t)
	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/envios", nil)
	// Sem X-IF-ID, sem JWT → deve retornar 401
	w := httptest.NewRecorder()
	srv.listEnvios(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", w.Code)
	}
}

// --- enviosStats ---

func TestEnviosStats_Aggregation(t *testing.T) {
	d := testutil.NewTestDB(t)
	testutil.SeedTestEnvios(t, d, "demo", []testutil.EnvioFixture{
		{ID: "E1", Status: "accepted"},
		{ID: "E2", Status: "accepted"},
		{ID: "E3", Status: "accepted"},
		{ID: "E4", Status: "rejected"},
		{ID: "E5", Status: "pending"},
		{ID: "E6", Status: "error"},
	})

	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/envios/stats", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.enviosStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]int
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total"] != 6 {
		t.Errorf("total: got %d, want 6", resp["total"])
	}
	if resp["accepted"] != 3 {
		t.Errorf("accepted: got %d, want 3", resp["accepted"])
	}
	if resp["rejected"] != 1 {
		t.Errorf("rejected: got %d, want 1", resp["rejected"])
	}
	if resp["pending"] != 1 {
		t.Errorf("pending: got %d, want 1", resp["pending"])
	}
	if resp["error"] != 1 {
		t.Errorf("error: got %d, want 1", resp["error"])
	}
}

// --- insightsKPIs ---

func TestInsightsKPIs_CalculatesApprovalRate(t *testing.T) {
	d := testutil.NewTestDB(t)
	testutil.SeedTestEnvios(t, d, "demo", []testutil.EnvioFixture{
		{ID: "E1", Status: "accepted", DaysAgo: 5},
		{ID: "E2", Status: "accepted", DaysAgo: 5},
		{ID: "E3", Status: "accepted", DaysAgo: 5},
		{ID: "E4", Status: "rejected", DaysAgo: 5},
		{ID: "E5", Status: "accepted", DaysAgo: 45}, // período anterior
	})

	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/insights/kpis", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.insightsKPIs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Current struct {
			ApprovalRate float64 `json:"approval_rate"`
			SentTotal    int     `json:"sent_total"`
			Accepted     int     `json:"accepted"`
		} `json:"current"`
		Previous struct {
			SentTotal int `json:"sent_total"`
		} `json:"previous"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Current.SentTotal != 4 {
		t.Errorf("current sent_total: got %d, want 4", resp.Current.SentTotal)
	}
	if resp.Current.Accepted != 3 {
		t.Errorf("current accepted: got %d, want 3", resp.Current.Accepted)
	}
	if resp.Current.ApprovalRate != 75.0 {
		t.Errorf("current approval_rate: got %f, want 75.0", resp.Current.ApprovalRate)
	}
	if resp.Previous.SentTotal != 1 {
		t.Errorf("previous sent_total: got %d, want 1", resp.Previous.SentTotal)
	}
}

// --- insightsHeatmap ---

func TestInsightsHeatmap_GroupsByDay(t *testing.T) {
	d := testutil.NewTestDB(t)
	testutil.SeedTestRuleFailures(t, d, "demo", []testutil.RuleFailureFixture{
		{Cadoc: "3040", RuleCode: "F23", Severity: "E", DaysAgo: 0, Count: 5},
		{Cadoc: "3040", RuleCode: "F23", Severity: "E", DaysAgo: 1, Count: 3},
		{Cadoc: "3050", RuleCode: "B12", Severity: "E", DaysAgo: 0, Count: 2},
	})

	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/insights/heatmap?days=7", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.insightsHeatmap(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []struct {
			Row, Col string
			Value    int
		} `json:"data"`
		Rows []string `json:"rows"`
		Cols []string `json:"cols"`
		Days int     `json:"days"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 heatmap cells, got %d", len(resp.Data))
	}
	if resp.Days != 7 {
		t.Errorf("days: got %d, want 7", resp.Days)
	}
	// Não pode ter "<nil>" como coluna (validação 30 descobrira bug strftime)
	for _, cell := range resp.Data {
		if cell.Col == "<nil>" || cell.Col == "" {
			t.Errorf("cell has empty/nil col: row=%s col=%s", cell.Row, cell.Col)
		}
	}
}

// --- insightsTopFailingRules ---

func TestInsightsTopFailingRules_RankingAndDelta(t *testing.T) {
	d := testutil.NewTestDB(t)
	testutil.SeedTestRuleFailures(t, d, "demo", []testutil.RuleFailureFixture{
		// Top: F23 com 10 falhas atuais, 5 anteriores (delta +100%)
		{Cadoc: "3040", RuleCode: "F23", Severity: "E", DaysAgo: 5, Count: 10},
		{Cadoc: "3040", RuleCode: "F23", Severity: "E", DaysAgo: 45, Count: 5},
		// B12: 3 falhas atuais, 3 anteriores (flat)
		{Cadoc: "3040", RuleCode: "B12", Severity: "E", DaysAgo: 5, Count: 3},
		{Cadoc: "3040", RuleCode: "B12", Severity: "E", DaysAgo: 45, Count: 3},
	})

	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/insights/rules/top-failing?limit=5", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.insightsTopFailingRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Rules []struct {
			Code     string `json:"code"`
			Count    int    `json:"count"`
			DeltaPct int    `json:"delta_pct"`
			TrendDir string `json:"trend_direction"`
		} `json:"rules"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(resp.Rules))
	}
	// F23 deve estar primeiro (count=10)
	if resp.Rules[0].Code != "F23" {
		t.Errorf("expected F23 first, got %s", resp.Rules[0].Code)
	}
	if resp.Rules[0].Count != 10 {
		t.Errorf("F23 count: got %d, want 10", resp.Rules[0].Count)
	}
	if resp.Rules[0].DeltaPct != 100 {
		t.Errorf("F23 delta_pct: got %d, want +100", resp.Rules[0].DeltaPct)
	}
	if resp.Rules[0].TrendDir != "up" {
		t.Errorf("F23 trend: got %s, want up", resp.Rules[0].TrendDir)
	}
	// B12 flat
	if resp.Rules[1].Code != "B12" {
		t.Errorf("expected B12 second, got %s", resp.Rules[1].Code)
	}
	if resp.Rules[1].TrendDir != "flat" {
		t.Errorf("B12 trend: got %s, want flat", resp.Rules[1].TrendDir)
	}
}

// --- insightsRecommendations ---

func TestInsightsRecommendations_ConcentrationRule(t *testing.T) {
	d := testutil.NewTestDB(t)
	testutil.SeedTestRuleFailures(t, d, "demo", []testutil.RuleFailureFixture{
		// F23 com 70% das falhas (>25% threshold)
		{Cadoc: "3040", RuleCode: "F23", Severity: "E", DaysAgo: 5, Count: 70},
		{Cadoc: "3040", RuleCode: "B12", Severity: "E", DaysAgo: 5, Count: 20},
		{Cadoc: "3050", RuleCode: "C04", Severity: "E", DaysAgo: 5, Count: 10},
	})

	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/insights/recommendations", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.insightsRecommendations(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Recommendations []struct {
			ID, Kind, Headline, Impact string
			Confidence                  int
		} `json:"recommendations"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Recommendations) == 0 {
		t.Fatal("expected at least 1 recommendation")
	}
	// Deve haver a recomendação de concentração em F23
	found := false
	for _, r := range resp.Recommendations {
		if strings.Contains(r.Headline, "F23") {
			found = true
			if r.ID == "" {
				t.Error("Validação 30 (C6 fix): recommendation ID vazio — quebra React keys")
			}
			if r.Kind != "recommendation" {
				t.Errorf("kind: got %s, want recommendation", r.Kind)
			}
		}
	}
	if !found {
		t.Errorf("expected F23 concentration recommendation, got: %+v", resp.Recommendations)
	}
}

// --- listAuditLog ---

func TestListAuditLog_NonAdminSeesOnlyOwnIF(t *testing.T) {
	d := testutil.NewTestDB(t)
	testutil.SeedTestAuditEvents(t, d, []testutil.AuditEventFixture{
		{IFID: "demo", Action: "envio.approved", Description: "demo event"},
		{IFID: "other", Action: "envio.approved", Description: "other event"},
	})

	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/audit_log", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.listAuditLog(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Events []struct {
			IFID   string `json:"if_id"`
			Action string `json:"action"`
		} `json:"events"`
		Total      int  `json:"total"`
		ChainValid bool `json:"chain_valid"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v body=%s", err, w.Body.String())
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 event (filter by IF), got %d. Body=%s", resp.Total, w.Body.String())
	}
	if resp.Total > 0 && resp.Events[0].IFID != "demo" {
		t.Errorf("expected only demo events, got '%s' (body=%s)", resp.Events[0].IFID, w.Body.String())
	}
}

func TestListAuditLog_NoAuthFails401(t *testing.T) {
	d := testutil.NewTestDB(t)
	srv := &Server{DB: d}
	req := httptest.NewRequest("GET", "/v1/audit_log", nil)
	// Sem X-IF-ID, sem JWT
	w := httptest.NewRecorder()
	srv.listAuditLog(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// --- Validação 30 (C2 fix): X-Role header NÃO promove a admin ---

func TestGetRole_DoesNotTrustXRoleHeader(t *testing.T) {
	d := testutil.NewTestDB(t)
	srv := &Server{DB: d}

	// Attacker manda X-Role: admin. Sem JWT, deveria default pra 'if'.
	req := httptest.NewRequest("GET", "/v1/audit_log", nil)
	req.Header.Set("X-Role", "admin") // tentativa de privilege escalation
	w := httptest.NewRecorder()
	srv.listAuditLog(w, req)

	// Sem X-IF-ID OU JWT, o middleware bloqueia ANTES de chegar no handler.
	// Aqui testamos que getRole() retorna 'if' (não 'admin').
	if got := getRole(req); got == "admin" {
		t.Errorf("Validação 30 (C2): getRole() ainda confia em X-Role header — privilege escalation!")
	}
}

// --- Validação 30 (C7 fix): chain_valid usa Verify() real ---

func TestListAuditLog_ChainValidReflectsRealVerify(t *testing.T) {
	d := testutil.NewTestDB(t)

	// AuditLog Logger criado com DB de teste (sem chain → primeiro entry tem
	// prevHash = 0×64 conforme genesis).
	auditLog := testutil.NewAuditLogForTest(t, d)
	srv := &Server{DB: d, AuditLog: auditLog}

	testutil.SeedTestAuditLog(t, d, []testutil.AuditLogFixture{
		{Actor: "system", Action: "envio.approved", Target: "E1"},
	})

	req := httptest.NewRequest("GET", "/v1/audit_log", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.listAuditLog(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		ChainValid bool `json:"chain_valid"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.ChainValid {
		t.Error("Validação 30 (C7): chain_valid=false com chain real intacta — Verify() não foi chamado")
	}
}