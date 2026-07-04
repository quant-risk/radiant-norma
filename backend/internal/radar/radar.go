// Package radar implementa o worker de detecção de mudanças regulatórias.
//
// Por design: periodicamente faz fetch de URLs BACEN conhecidas (XSDs,
// planilhas de críticas, instruções), calcula SHA-256 e compara com a
// última hash conhecida. Quando detecta mudança, insere em radar_alerts.
//
// Diferencial de marketing: "first-mover" — IFs são notificadas ANTES
// de o leiaute mudar oficialmente (vs. descobrir quando o STA rejeita).
package radar

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
	"github.com/fortvna/radiant-norma/backend/internal/version"
)

// Source é uma URL que o Radar monitora.
type Source struct {
	CadocCode string // 3040, 3050, ...
	Label     string // "XSD", "Críticas", "Instruções"
	URL       string
	AlertType string // 'leiaute_changed', 'criticas_changed', 'normativo_published'
	Severity  string // info, warn, critical
}

// DefaultSources é a lista de URLs BACEN monitoradas.
//
// NOTA: muitas URLs BACEN são protegidas por login ou mudam frequentemente.
// O Radar é resiliente: falha de fetch é logada mas não quebra o pipeline.
var DefaultSources = []Source{
	{
		CadocCode: "3040",
		Label:     "Críticas 3040",
		URL:       "https://www.bcb.gov.br/content/estabilidadefinanceira/SCR/SCR3040_Criticas.xls",
		AlertType: "criticas_changed",
		Severity:  "critical",
	},
	{
		CadocCode: "3050",
		Label:     "Críticas 3050 V11",
		URL:       "https://www.bcb.gov.br/content/estabilidadefinanceira/estatisticas/EstatisticasAgregadas/Criticas_TXB_V11.xlsx",
		AlertType: "criticas_changed",
		Severity:  "critical",
	},
	{
		CadocCode: "2030",
		Label:     "DRSAC FAQ",
		URL:       "https://www.bcb.gov.br/estabilidadefinanceira/drsac",
		AlertType: "normativo_published",
		Severity:  "info",
	},
}

// Service é o serviço de Radar.
type Service struct {
	db       *sql.DB
	hc       *http.Client
	interval time.Duration
	logger   *slog.Logger
}

// New cria um novo Service.
func New(db *sql.DB, interval time.Duration) *Service {
	return &Service{
		db: db,
		hc: &http.Client{
			Timeout: 30 * time.Second,
			// User-Agent decente pra não levar 403 do BACEN
		},
		interval: interval,
		logger:   slog.Default(),
	}
}

// SetLogger permite injetar logger customizado.
func (s *Service) SetLogger(l *slog.Logger) {
	s.logger = l
}

// Alert representa um alerta de mudança detectado.
type Alert struct {
	ID          int64     `json:"id"`
	CadocCode   string    `json:"cadoc_code"`
	AlertType   string    `json:"alert_type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	SourceURL   string    `json:"source_url"`
	DetectedAt  time.Time `json:"detected_at"`
	Resolved    bool      `json:"resolved"`
}

// ScanOnce executa 1 ciclo de detecção para todas as fontes.
//
// Retorna alertas NOVOS detectados neste ciclo (não inclui alertas
// já existentes).
func (s *Service) ScanOnce(ctx context.Context, sources []Source) ([]Alert, error) {
	if sources == nil {
		sources = DefaultSources
	}

	var newAlerts []Alert
	for _, src := range sources {
		alert, err := s.scanSource(ctx, src)
		if err != nil {
			// Validação 18 (F18.9): sanitizar err (pode incluir DSN
			// caso fonte seja re-escrita com auth params no futuro, ou
			// detalhes de pgx se storage for migrado).
			s.logger.Warn("scan source failed",
				"cadoc", src.CadocCode,
				"err", loggerutil.SafeError(err),
			)
			continue
		}
		if alert != nil {
			newAlerts = append(newAlerts, *alert)
		}
	}

	if len(newAlerts) > 0 {
		s.logger.Info("scan cycle complete", "new_alerts", len(newAlerts))
	}
	return newAlerts, nil
}

// scanSource faz fetch + hash + diff de UMA fonte.
// Retorna Alert novo (se mudança detectada) ou nil (se igual/primeira vez).
func (s *Service) scanSource(ctx context.Context, src Source) (*Alert, error) {
	// Fetch
	hash, err := s.fetchHash(ctx, src.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", src.URL, err)
	}

	// Última hash conhecida
	lastHash, err := s.lastKnownHash(ctx, src)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("last hash: %w", err)
	}

	// Primeira vez ou hash igual → não é mudança
	if lastHash == "" {
		s.logger.Info("first scan, recording baseline",
			"cadoc", src.CadocCode, "hash", ShortHash(hash))
		if err := s.recordBaseline(ctx, src, hash); err != nil {
			// Não silenciar: se baseline não gravou, próxima scan tenta de novo
			// (loop de log "first scan" até DB voltar). Erro aqui é operacional.
			// Validação 18 (F18.11): sanitizar err (pode incluir DSN).
			s.logger.Error("recordBaseline failed (first scan)",
				"cadoc", src.CadocCode, "err", loggerutil.SafeError(err))
		}
		return nil, nil
	}

	if hash == lastHash {
		s.logger.Debug("hash unchanged", "cadoc", src.CadocCode)
		return nil, nil
	}

	// Mudança detectada!
	alert := Alert{
		CadocCode:   src.CadocCode,
		AlertType:   src.AlertType,
		Severity:    src.Severity,
		Title:       fmt.Sprintf("%s mudou: %s", src.CadocCode, src.Label),
		Description: fmt.Sprintf("Hash anterior: %s\nHash novo: %s", ShortHash(lastHash), ShortHash(hash)),
		SourceURL:   src.URL,
		DetectedAt:  time.Now(),
	}

	id, err := s.insertAlert(ctx, alert)
	if err != nil {
		return nil, fmt.Errorf("insert alert: %w", err)
	}
	alert.ID = id

	// Atualiza baseline — se falhar, próximo scan dispara alerta duplicado.
	// Logamos warning para investigar; não silenciar.
	if err := s.recordBaseline(ctx, src, hash); err != nil {
		// Validação 18 (F18.12): sanitizar err.
		s.logger.Warn("recordBaseline failed after alert — próximo scan pode duplicar",
			"alert_id", id, "cadoc", src.CadocCode,
			"err", loggerutil.SafeError(err))
	}

	s.logger.Info("CHANGE DETECTED",
		"cadoc", src.CadocCode,
		"label", src.Label,
		"old", ShortHash(lastHash),
		"new", ShortHash(hash),
	)
	return &alert, nil
}

// fetchHash faz GET na URL e retorna SHA-256 do conteúdo.
func (s *Service) fetchHash(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("Mozilla/5.0 (Radiant-Norma-Radar/%s; +https://fortvna.com.br)", version.Version))
	req.Header.Set("Accept", "*/*")

	resp, err := s.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	// Limita tamanho (50 MB) pra evitar explosão
	const maxSize = 50 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

// lastKnownHash retorna a última hash registrada para esta source.
// Usa tabela dedicada radar_baselines (Sprint 6 v1.5.0 / F3) com PK
// composta (cadoc_code, alert_type) — UNIQUE garante atomicidade.
//
// Antes (v1.4.x): usava radar_alerts com alert_type='_baseline_<label>',
// que tinha race window entre concurrent scans (ambos faziam UPDATE/INSERT
// sem constraint UNIQUE → múltiplas baselines ou INSERT race).
//
// Performance: query é O(1) (PK lookup) vs antes (ORDER BY + LIMIT 1 sobre
// radar_alerts com filtro LIKE).
func (s *Service) lastKnownHash(ctx context.Context, src Source) (string, error) {
	baselineType := baselineTypeFor(src.Label)
	var hash string
	err := s.db.QueryRowContext(ctx, `
		SELECT hash FROM radar_baselines
		WHERE cadoc_code = ? AND alert_type = ?
	`, src.CadocCode, baselineType).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", err
	}
	return hash, err
}

// ShortHash retorna os primeiros N caracteres de um hash, mas nunca
// panica se a string for menor que N. Defesa contra o mesmo padrão que
// `auditlog.Verify` (v1.4.0 bug #1): SHA-256 hex tem 64 chars sempre,
// mas se alguém inserir hash mal-formado no DB, [:N] panica.
//
// Default: 12 chars (mesmo padrão que logs anteriores usavam hardcoded).
//
// Exportada (Sprint 5 v1.4.1) pra ser testável — ver radar_test.go.
func ShortHash(s string) string {
	const max = 12
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// baselineTypeFor retorna o alert_type usado internamente para baseline.
// Normaliza espaços e caracteres não-ASCII para underscore (consistência
// entre labels com múltiplas palavras como "DRSAC FAQ" → "_baseline_drsac_faq").
func baselineTypeFor(label string) string {
	lower := strings.ToLower(label)
	// Substitui qualquer não-alfanumérico (espaço, hífen, etc) por underscore
	replacer := strings.NewReplacer(" ", "_", "-", "_", ".", "_")
	return "_baseline_" + replacer.Replace(lower)
}

// RecordBaseline grava a hash como baseline (após scan ou mudança).
//
// Atomicidade (Sprint 6 v1.5.0 / F3): usa INSERT ... ON CONFLICT no
// tabela radar_baselines. UNIQUE constraint (PK) garante que 2 scans
// concorrentes escrevam o mesmo baseline sem race window.
//
// Antes (v1.4.x): usava UPDATE-then-INSERT fallback em radar_alerts.
// Janela: 2 goroutines pegam o mesmo prev_hash via SELECT, depois ambos
// tentam UPDATE (0 rows afetadas por ser primeira vez), depois ambos
// tentam INSERT — 1 falha com UNIQUE constraint (ou ambos inserem em
// algumas condições de corrida com SELECT FOR UPDATE não usado).
//
// O NOW: 1 única operação atômica — DB serializa via PK UNIQUE.
//
// Exportado (Sprint 6) para permitir testes de regressão F3 diretamente
// sem precisar passar pelo ScanOnce (mais simples de testar concorrência).
// Em produção, é chamado internamente por scanSource.
func (s *Service) RecordBaseline(ctx context.Context, src Source, hash string) error {
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

// recordBaseline é o alias interno (alias pra RecordBaseline).
// Mantido por compat — internamente scanSource chama direto.
// Em código novo, prefira RecordBaseline.
func (s *Service) recordBaseline(ctx context.Context, src Source, hash string) error {
	return s.RecordBaseline(ctx, src, hash)
}

// insertAlert persiste um alerta novo.
func (s *Service) insertAlert(ctx context.Context, a Alert) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO radar_alerts (cadoc_code, alert_type, severity, title, description, source_url)
		VALUES (?, ?, ?, ?, ?, ?)
	`, a.CadocCode, a.AlertType, a.Severity, a.Title, a.Description, a.SourceURL)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListAlerts retorna alertas (todos ou só unresolved).
func (s *Service) ListAlerts(ctx context.Context, unresolvedOnly bool, limit int) ([]Alert, error) {
	if limit <= 0 {
		limit = 100
	}
	q := `
		SELECT id, cadoc_code, alert_type, severity, title, description,
		       COALESCE(source_url, ''), detected_at, resolved_at IS NOT NULL
		FROM radar_alerts
		WHERE alert_type NOT LIKE '_baseline_%'
	`
	if unresolvedOnly {
		q += ` AND resolved_at IS NULL `
	}
	q += ` ORDER BY detected_at DESC LIMIT ?`

	rows, err := s.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Sempre retorna slice (não nil) quando vazio, pra JSON serializar [] em vez de null.
	alerts := []Alert{}
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.CadocCode, &a.AlertType, &a.Severity,
			&a.Title, &a.Description, &a.SourceURL, &a.DetectedAt, &a.Resolved); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// GetAlertByID retorna um alerta específico por ID.
// Query direta (O(1)) em vez de ListAlerts + filtro linear (O(N)).
func (s *Service) GetAlertByID(ctx context.Context, id int64) (*Alert, error) {
	var a Alert
	err := s.db.QueryRowContext(ctx, `
		SELECT id, cadoc_code, alert_type, severity, title, description,
		       COALESCE(source_url, ''), detected_at, resolved_at IS NOT NULL
		FROM radar_alerts
		WHERE id = ? AND alert_type NOT LIKE '_baseline_%'
	`, id).Scan(&a.ID, &a.CadocCode, &a.AlertType, &a.Severity,
		&a.Title, &a.Description, &a.SourceURL, &a.DetectedAt, &a.Resolved)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Close libera recursos do Radar (HTTP client).
func (s *Service) Close() {
	if s.hc != nil {
		s.hc.CloseIdleConnections()
	}
}

// ResolveAlert marca um alerta como resolvido.
// Retorna erro se nenhum registro foi atualizado (id inexistente).
func (s *Service) ResolveAlert(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE radar_alerts SET resolved_at = CURRENT_TIMESTAMP WHERE id = ? AND resolved_at IS NULL", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("alert id=%d não encontrado ou já resolvido", id)
	}
	return nil
}
