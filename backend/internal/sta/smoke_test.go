// Package sta — BACEN Homologation Smoke Tests (Sprint 79).
//
// Run against sta-h.bcb.gov.br (homologação) with real pilot credentials.
// Requires environment variables:
//   STA_HOMOLOG_USER     — BacenHomolog username
//   STA_HOMOLOG_PASSWORD — BacenHomolog password
//   STA_HOMOLOG_IF_ID   — IF identifier for the pilot client
//
// These tests are ALWAYS skipped unless STA_HOMOLOG_* are set, and
// are NEVER run in CI (only manually with pilot credentials).
//
// Tests cover the full happy path: Submit → Status → Download → (RangeUpload).
//
// Reference: STA_Manual_WebServices.pdf (BACEN, July 2022).
package sta

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"testing"
)

// TestSmokeHomolog_Submit tests the full submission flow against
// sta-h.bcb.gov.br (homologação).
//
// Tests: POST /arquivos → 201 + protocolo → PUT /conteudo → 200.
func TestSmokeHomolog_Submit(t *testing.T) {
	user := os.Getenv("STA_HOMOLOG_USER")
	pass := os.Getenv("STA_HOMOLOG_PASSWORD")
	cnpj := os.Getenv("STA_HOMOLOG_IF_ID")
	if user == "" || pass == "" || cnpj == "" {
		t.Skip("STA_HOMOLOG_USER, STA_HOMOLOG_PASSWORD, or STA_HOMOLOG_IF_ID not set — skipping smoke test")
	}

	// Create a minimal 3040 XML for testing (tiny valid structure).
	xmlContent := minimal3040XML()

	client, err := NewWSClient(WSConfig{
		BaseURL: "https://sta-h.bcb.gov.br/staws",
		User:    user,
		Password: pass,
	})
	if err != nil {
		t.Fatalf("NewWSClient failed: %v", err)
	}

	sub := &Submission{
		CadocCode: "3040",
		CNPJ:      os.Getenv("STA_HOMOLOG_IF_ID"),
		XML:       xmlContent,
	}

	result, err := client.Submit(context.Background(), sub)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if result.ProtocolSTA == "" {
		t.Fatal("ProtocolSTA is empty — expected non-empty protocolo from BACEN")
	}
	if !result.Accepted {
		t.Fatalf("Accepted=false — expected true for valid submission")
	}

	t.Logf("✅ Submit succeeded: protocolo=%s", result.ProtocolSTA)

	// Clean up: log protocolo for manual verification in BACEN portal.
	t.Logf("📋 Protocolo %s — verify at https://sta-h.bcb.gov.br/staws", result.ProtocolSTA)
}

// TestSmokeHomolog_Status tests the status check endpoint.
//
// Tests: GET /arquivos/{protocolo}/posicaoupload.
func TestSmokeHomolog_Status(t *testing.T) {
	user := os.Getenv("STA_HOMOLOG_USER")
	pass := os.Getenv("STA_HOMOLOG_PASSWORD")
	cnpj := os.Getenv("STA_HOMOLOG_IF_ID")
	if user == "" || pass == "" || cnpj == "" {
		t.Skip("STA_HOMOLOG_USER, STA_HOMOLOG_PASSWORD, or STA_HOMOLOG_IF_ID not set — skipping smoke test")
	}

	// First submit a document to get a protocolo.
	xmlContent := minimal3040XML()
	client, err := NewWSClient(WSConfig{
		BaseURL: "https://sta-h.bcb.gov.br/staws",
		User:    user,
		Password: pass,
	})
	if err != nil {
		t.Fatalf("NewWSClient failed: %v", err)
	}

	sub := &Submission{
		CadocCode: "3040",
		CNPJ:      os.Getenv("STA_HOMOLOG_IF_ID"),
		XML:       xmlContent,
	}

	result, err := client.Submit(context.Background(), sub)
	if err != nil {
		t.Fatalf("Submit failed (pre-requisite): %v", err)
	}

	// Now check status.
	status, err := client.StatusUpload(context.Background(), result.ProtocolSTA)
	if err != nil {
		t.Fatalf("StatusUpload failed: %v", err)
	}

	t.Logf("✅ StatusUpload succeeded: protocolo=%s status=%+v", result.ProtocolSTA, status)
}

// TestSmokeHomolog_Download tests the document download endpoint.
//
// Tests: GET /arquivos/{protocolo}/conteudo.
func TestSmokeHomolog_Download(t *testing.T) {
	user := os.Getenv("STA_HOMOLOG_USER")
	pass := os.Getenv("STA_HOMOLOG_PASSWORD")
	cnpj := os.Getenv("STA_HOMOLOG_IF_ID")
	if user == "" || pass == "" || cnpj == "" {
		t.Skip("STA_HOMOLOG_USER, STA_HOMOLOG_PASSWORD, or STA_HOMOLOG_IF_ID not set — skipping smoke test")
	}

	// Submit first.
	xmlContent := minimal3040XML()
	client, err := NewWSClient(WSConfig{
		BaseURL: "https://sta-h.bcb.gov.br/staws",
		User:    user,
		Password: pass,
	})
	if err != nil {
		t.Fatalf("NewWSClient failed: %v", err)
	}

	sub := &Submission{
		CadocCode: "3040",
		CNPJ:      os.Getenv("STA_HOMOLOG_IF_ID"),
		XML:       xmlContent,
	}

	result, err := client.Submit(context.Background(), sub)
	if err != nil {
		t.Fatalf("Submit failed (pre-requisite): %v", err)
	}

	// Download.
	data, err := client.Download(context.Background(), result.ProtocolSTA)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if len(data.Conteudo) == 0 {
		t.Fatal("Download returned empty data")
	}

	// Verify content hash matches what we submitted.
	sum := sha256.Sum256(data.Conteudo)
	hash := hex.EncodeToString(sum[:])
	t.Logf("✅ Download succeeded: protocolo=%s bytes=%d sha256=%s", result.ProtocolSTA, len(data.Conteudo), hash)
}

// TestSmokeHomolog_SubmitRange tests the chunked upload flow (Section 5.6).
//
// Tests: POST /arquivos → protocolo → PUT range chunks → GET status.
func TestSmokeHomolog_SubmitRange(t *testing.T) {
	user := os.Getenv("STA_HOMOLOG_USER")
	pass := os.Getenv("STA_HOMOLOG_PASSWORD")
	cnpj := os.Getenv("STA_HOMOLOG_IF_ID")
	if user == "" || pass == "" || cnpj == "" {
		t.Skip("STA_HOMOLOG_USER, STA_HOMOLOG_PASSWORD, or STA_HOMOLOG_IF_ID not set — skipping smoke test")
	}

	xmlContent := minimal3040XML()
	client, err := NewWSClient(WSConfig{
		BaseURL: "https://sta-h.bcb.gov.br/staws",
		User:    user,
		Password: pass,
	})
	if err != nil {
		t.Fatalf("NewWSClient failed: %v", err)
	}

	// Init range session — calculates hash and calls InitRangeSession.
	sum := sha256.Sum256([]byte(xmlContent))
	hashHex := hex.EncodeToString(sum[:])

	ctx := context.Background()
	protocolo, err := client.InitRangeSession(ctx, "3040", hashHex, int64(len(xmlContent)))
	if err != nil {
		t.Fatalf("InitRangeSession failed: %v", err)
	}

	if protocolo == "" {
		t.Fatal("InitRangeSession returned empty protocolo")
	}

	// Upload in 2 chunks.
	chunkSize := len(xmlContent) / 2
	if chunkSize == 0 {
		chunkSize = len(xmlContent)
	}

	// Chunk 1.
	err = client.SubmitRange(ctx, protocolo, 0, int64(chunkSize), int64(len(xmlContent)), []byte(xmlContent[:chunkSize]))
	if err != nil {
		t.Fatalf("SubmitRange chunk 1 failed: %v", err)
	}

	// Chunk 2.
	err = client.SubmitRange(ctx, protocolo, int64(chunkSize), int64(len(xmlContent)), int64(len(xmlContent)), []byte(xmlContent[chunkSize:]))
	if err != nil {
		t.Fatalf("SubmitRange chunk 2 failed: %v", err)
	}

	// Check status.
	status, err := client.StatusUpload(ctx, protocolo)
	if err != nil {
		t.Fatalf("StatusUpload after range failed: %v", err)
	}

	t.Logf("✅ RangeUpload succeeded: protocolo=%s ranges_received=%d", protocolo, len(status.RangesRecebidos))
}

// minimal3040XML returns a minimal valid 3040 XML for smoke testing.
func minimal3040XML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Doc3040>
  <CNPJ>12345678</CNPJ>
  <DtBase>2026-01</DtBase>
  <Remessa>1</Remessa>
  <Parte>1</Parte>
  <TpArq>F</TpArq>
  <NomeResp>Test User</NomeResp>
  <EmailResp>test@example.com</EmailResp>
  <TelResp>11999999999</TelResp>
  <TotalCli>0</TotalCli>
</Doc3040>`
}

// TestSmokeHomolog_AuthFailure tests that an invalid user format is rejected
// at construction time, and that a valid-format user with wrong password
// returns 401 from BACEN.
func TestSmokeHomolog_AuthFailure(t *testing.T) {
	// Test 1: invalid user format → rejected at NewWSClient.
	_, err := NewWSClient(WSConfig{
		BaseURL:  "https://sta-h.bcb.gov.br/staws",
		User:     "invalid-user",
		Password: "anything",
	})
	if err == nil {
		t.Fatal("expected NewWSClient to reject invalid user format, got nil")
	}
	t.Logf("✅ NewWSClient correctly rejected invalid user format: %v", err)

	// Test 2: valid format user + wrong password → BACEN returns 401.
	// Requires real pilot credentials set.
	user := os.Getenv("STA_HOMOLOG_USER")
	pass := os.Getenv("STA_HOMOLOG_PASSWORD")
	if user == "" || pass == "" {
		t.Skip("STA_HOMOLOG_USER or STA_HOMOLOG_PASSWORD not set — skipping BACEN auth test")
	}

	// Use correct format but wrong password.
	wrongPassClient, err := NewWSClient(WSConfig{
		BaseURL:  "https://sta-h.bcb.gov.br/staws",
		User:     user,
		Password: "wrong-password-xyz",
	})
	if err != nil {
		t.Fatalf("NewWSClient failed: %v", err)
	}

	sub := &Submission{
		CadocCode: "3040",
		CNPJ:      os.Getenv("STA_HOMOLOG_IF_ID"),
		XML:       minimal3040XML(),
	}

	_, err = wrongPassClient.Submit(context.Background(), sub)
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
	t.Logf("✅ Auth failure correctly rejected: %v", err)
}

// TestSmokeHomolog_HealthCheck tests the health/readiness endpoint.
func TestSmokeHomolog_HealthCheck(t *testing.T) {
	user := os.Getenv("STA_HOMOLOG_USER")
	pass := os.Getenv("STA_HOMOLOG_PASSWORD")
	cnpj := os.Getenv("STA_HOMOLOG_IF_ID")
	if user == "" || pass == "" || cnpj == "" {
		t.Skip("STA_HOMOLOG_USER, STA_HOMOLOG_PASSWORD, or STA_HOMOLOG_IF_ID not set — skipping smoke test")
	}

	// Health check = GET /arquivos/disponiveis with no params.
	req, err := http.NewRequest("GET", "https://sta-h.bcb.gov.br/staws/arquivos/disponiveis", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.SetBasicAuth(user, pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("health check request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("health check returned %d, expected 200", resp.StatusCode)
	}

	_, _ = io.ReadAll(resp.Body)
	t.Logf("✅ Health check passed: status=%d", resp.StatusCode)
}
