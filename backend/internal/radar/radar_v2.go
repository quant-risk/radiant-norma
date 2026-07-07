// Package radar implementa o Radar v2 com diff semântico e Auto-PR.
//
// Radar v2 estende o Radar v1 (hash-based) com:
//   - Diff semântico: parseia XLSX e detecta o que especificamente mudou
//   - Auto-PR: cria GitHub PR automaticamente com as regras atualizadas
//
// Este arquivo define o service RadarV2 que integra diff + autopr.
package radar

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/radar/autopr"
	"github.com/fortvna/radiant-norma/backend/internal/radar/diff"
	"github.com/fortvna/radiant-norma/backend/internal/version"
)

// SourceV2 é uma fonte que o RadarV2 monitora (suporta XLSX).
type SourceV2 struct {
	Source
	// SheetName é o nome da aba do XLSX a parsear (opcional).
	SheetName string
	// ParserType: "xlsx" | "xsd" | "json" | "html"
	ParserType string
}

// RadarV2 é o service Radar com diff semântico.
type RadarV2 struct {
	db     *sql.DB
	hc     *http.Client
	differ *diff.Differ
	pr     *autopr.Client
}

// NewRadarV2 cria um novo RadarV2.
func NewRadarV2(db *sql.DB, prConfig autopr.Config) *RadarV2 {
	return &RadarV2{
		db: db,
		hc: &http.Client{
			Timeout: 60 * time.Second, // XLSX maiores precisam de mais tempo
		},
		differ: diff.NewDiffer(),
		pr:     autopr.NewClient(prConfig),
	}
}

// ScanResult é o resultado de uma scan V2.
type ScanResult struct {
	CadocCode string
	OldHash   string
	NewHash   string
	Diff      *diff.DiffResult
	PR        *autopr.PRResult
}

// ScanOnceXLSX executa um ciclo de detecção com diff semântico para XLSX.
func (s *RadarV2) ScanOnceXLSX(ctx context.Context, src SourceV2) (*ScanResult, error) {
	// 1. Fetch do conteúdo novo.
	newData, newHash, err := s.fetchContent(ctx, src.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer newData.Close()

	// 2. Última hash conhecida (do DB).
	oldHash, err := s.lastKnownHash(ctx, src.CadocCode, src.Label)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("last hash: %w", err)
	}

	result := &ScanResult{
		CadocCode: src.CadocCode,
		OldHash:   oldHash,
		NewHash:   newHash,
	}

	// Primeira vez: só registra baseline.
	if oldHash == "" {
		if err := s.recordBaseline(ctx, src, newHash); err != nil {
			return nil, fmt.Errorf("recordBaseline: %w", err)
		}
		return result, nil
	}

	// Hash igual: sem mudança.
	if newHash == oldHash {
		return result, nil
	}

	// 3. Hash mudou — diff estruturado requer old body (não disponível na baseline MVP).
	// Registramos que houve mudança sem identificar regras específicas.
	diffResult := diff.NewResult(src.CadocCode, src.URL, oldHash, newHash)
	// TODO: quando tivermos old body no cache, usar CompareRowMaps(oldMap, newMap, ...)

	result.Diff = diffResult

	// 4. Atualiza baseline.
	if err := s.recordBaseline(ctx, src, newHash); err != nil {
		return nil, fmt.Errorf("recordBaseline: %w", err)
	}

	return result, nil
}

// ScanAndCreatePR executa scan V2 e cria Auto-PR se houver mudanças em regras.
func (s *RadarV2) ScanAndCreatePR(ctx context.Context, src SourceV2) (*ScanResult, error) {
	result, err := s.ScanOnceXLSX(ctx, src)
	if err != nil {
		return nil, err
	}

	if result.Diff == nil || len(result.Diff.Entries) == 0 {
		return result, nil // sem diff, sem PR
	}

	// Extrai rule codes do diff.
	var ruleCodes []string
	for _, e := range result.Diff.Entries {
		if e.RuleCode != "" {
			ruleCodes = append(ruleCodes, e.RuleCode)
		}
	}

	if len(ruleCodes) == 0 {
		return result, nil
	}

	prResult, prErr := s.pr.CreateRuleUpdatePR(ctx, autopr.RuleUpdatePRInput{
		CadocCode:   src.CadocCode,
		RuleCodes:   ruleCodes,
		DiffSummary: result.Diff.Summary,
		BranchName:  fmt.Sprintf("radar/update/%s-%s", src.CadocCode, time.Now().Format("20060102")),
	})
	if prErr != nil {
		// PR falhou — não falha o scan. Logado pelo client.
		return result, nil
	}

	result.PR = prResult
	return result, nil
}

// fetchContent baixa conteúdo da URL e retorna Reader + hash.
func (s *RadarV2) fetchContent(ctx context.Context, url string) (io.ReadCloser, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("Mozilla/5.0 (Radiant-Norma-Radar/%s; +https://fortvna.com.br)", version.Version))
	req.Header.Set("Accept", "*/*")

	resp, err := s.hc.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("status %d", resp.StatusCode)
	}

	// Limita tamanho para 50 MB.
	lr := &io.LimitedReader{R: resp.Body, N: 50 * 1024 * 1024}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, "", err
	}

	hash := sha256Hash(data)
	return io.NopCloser(bytes.NewReader(data)), hash, nil
}

// recordBaseline persiste hash no DB.
func (s *RadarV2) recordBaseline(ctx context.Context, src SourceV2, hash string) error {
	baselineType := baselineTypeFor(src.Label)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO radar_baselines (cadoc_code, alert_type, hash, source_url, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(cadoc_code, alert_type) DO UPDATE SET
			hash = excluded.hash,
			source_url = excluded.source_url,
			updated_at = CURRENT_TIMESTAMP
	`, src.CadocCode, baselineType, hash, src.URL)
	return err
}

// lastKnownHash retorna a última hash registrada.
func (s *RadarV2) lastKnownHash(ctx context.Context, cadocCode, label string) (string, error) {
	baselineType := baselineTypeFor(label)
	var hash string
	err := s.db.QueryRowContext(ctx, `
		SELECT hash FROM radar_baselines
		WHERE cadoc_code = ? AND alert_type = ?
	`, cadocCode, baselineType).Scan(&hash)
	return hash, err
}

// sha256Hash calcula SHA-256 hex de bytes.
func sha256Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
