// Sprint 31 v3.34.31: RangeUploadAPI — chunked upload via STA (manual §5.6).
//
// Endpoints:
//   POST /v1/sta/range/init        — inicia sessão, pede protocolo ao BACEN
//   PUT  /v1/sta/range/{protocolo} — faz upload de 1 chunk (Content-Range)
//   GET  /v1/sta/range/{protocolo} — status dos chunks recebidos (resume)
//
// Fluxo completo (Seção 5.6):
//   1. POST /v1/sta/range/init → BACEN POST /arquivos → protocolo
//   2. PUT  /v1/sta/range/{protocolo} com Content-Range: bytes X-Y/TOTAL
//   3. GET  /v1/sta/range/{protocolo} → UploadStatus (ranges recebidos, situacao)
//
// Auth: JWT (same as other endpoints).
package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/sta"
	"github.com/go-chi/chi/v5"
)

// rangeSession representa uma sessão de upload chunkado.
type rangeSession struct {
	ID             string      `json:"id"`
	IfID           string      `json:"if_id"`
	Protocolo      string      `json:"protocolo"`
	CadocCode      string      `json:"cadoc_code"`
	TotalBytes     int64       `json:"total_bytes"`
	ReceivedBytes  int64       `json:"received_bytes"`
	Ranges         []sta.Range `json:"ranges"`
	Status         string      `json:"status"` // pending|complete|failed|abandoned
	CreatedAt      time.Time   `json:"created_at"`
}

// activeSessions mantém sessões em memória enquanto o upload está ativo.
// key: protocolo (único no BACEN).
// Protegido por sessionsMu (sync.RWMutex) para acesso concorrente.
var (
	activeSessions = make(map[string]*rangeSession)
	sessionsMu     sync.RWMutex
)

var logger = slog.Default()

// staRangeInit — POST /v1/sta/range/init
//
// Body JSON:
//   {
//     "cadoc_code": "3040",
//     "hash_hex": "sha256...",     // opcional
//     "total_bytes": 1048576        // opcional
//   }
//
// Retorna:
//   201 { "protocolo": "12345", "session_id": "uuid", "ranges": [] }
//   400 invalid request
//   401 unauthorized
//   503 STA backend não suporta chunked
func (s *Server) staRangeInit(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	body, _ := io.ReadAll(r.Body)
	if len(body) == 0 {
		s.userError(w, http.StatusBadRequest, "staRangeInit.body", errors.New("body vazio"))
		return
	}

	var req rangeInitRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.userError(w, http.StatusBadRequest, "staRangeInit.jsonUnmarshal", err)
		return
	}
	if req.CadocCode == "" {
		s.userError(w, http.StatusBadRequest, "staRangeInit.cadoc", errors.New("cadoc_code requerido"))
		return
	}

	// BACEN — POST /arquivos para obter protocolo.
	ru, ok := s.STAClient.(sta.RangeUploader)
	if !ok {
		http.Error(w, `{"error":"STA backend não suporta chunked upload"}`, http.StatusServiceUnavailable)
		return
	}

	protocolo, err := ru.InitRangeSession(r.Context(), req.CadocCode, req.HashHex, req.TotalBytes)
	if err != nil {
		logger.Error("staRangeInit: BACEN InitRangeSession failed",
			"err", err, "cadoc", req.CadocCode, "if_id", ifID)
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "BACEN rejeitou init: " + err.Error(),
		})
		return
	}

	// Cria sessão em memória.
	session := &rangeSession{
		ID:             generateID(),
		IfID:           ifID,
		Protocolo:      protocolo,
		CadocCode:      req.CadocCode,
		TotalBytes:     req.TotalBytes,
		ReceivedBytes:  0,
		Ranges:         []sta.Range{},
		Status:         "pending",
		CreatedAt:      time.Now(),
	}
	sessionsMu.Lock()
	activeSessions[protocolo] = session
	sessionsMu.Unlock()

	// Persiste no DB (fire-and-forget).
	go func() {
		ctx := context.Background()
		_, _ = s.DB.ExecContext(ctx, `
			INSERT INTO range_sessions (id, if_id, protocolo, total_bytes, received_bytes, ranges_json, status)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(protocolo) DO UPDATE SET
				updated_at = CURRENT_TIMESTAMP,
				status = excluded.status
		`, session.ID, session.IfID, session.Protocolo, session.TotalBytes, 0, `[]`, "pending")
	}()

	logger.Info("staRangeInit: sessão chunkada iniciada",
		"protocolo", protocolo, "cadoc", req.CadocCode, "if_id", ifID)

	writeJSON(w, http.StatusCreated, map[string]any{
		"protocolo":  protocolo,
		"session_id": session.ID,
		"ranges":     []sta.Range{},
	})
}

// staRangeUpload — PUT /v1/sta/range/{protocolo}
//
// Header Content-Range: bytes X-Y/TOTAL (RFC 7233 §4.2).
// Body: bytes brutos do chunk.
//
// Retorna:
//   200 { "received_bytes": N, "ranges": [...] }
//   400 invalid range / content-range missing
//   401 unauthorized
//   404 sessão não encontrada
//   503 STA error
func (s *Server) staRangeUpload(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	protocolo := chi.URLParam(r, "protocolo")
	if protocolo == "" {
		s.userError(w, http.StatusBadRequest, "staRangeUpload.protocolo", errors.New("protocolo requerido"))
		return
	}

	// Recupera sessão: memória → DB.
	sessionsMu.RLock()
	session := activeSessions[protocolo]
	sessionsMu.RUnlock()
	if session == nil {
		session = s.loadSessionFromDB(r.Context(), protocolo)
		if session == nil {
			http.Error(w, `{"error":"sessão não encontrada"}`, http.StatusNotFound)
			return
		}
	}

	// Owner check.
	if session.IfID != ifID {
		http.Error(w, `{"error":"acesso negado"}`, http.StatusForbidden)
		return
	}

	// Parse Content-Range.
	cr := r.Header.Get("Content-Range")
	if cr == "" {
		s.userError(w, http.StatusBadRequest, "staRangeUpload.contentRange", errors.New("Content-Range requerido"))
		return
	}
	crParsed, err := parseContentRange(cr)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "staRangeUpload.parseContentRange", err)
		return
	}

	// Lê chunk — rejeita ranges absurdamente grandes para evitar DoS via header.
	const maxChunkSize = 8 * 1024 * 1024 // 8 MiB
	chunkLen := crParsed.end - crParsed.start + 1
	if chunkLen > maxChunkSize || chunkLen < 0 {
		s.userError(w, http.StatusBadRequest, "staRangeUpload.chunkSize",
			fmt.Errorf("range size %d excede máximo de %d bytes", chunkLen, maxChunkSize))
		return
	}
	chunk := make([]byte, chunkLen)
	n, err := io.ReadFull(r.Body, chunk)
	if err != nil || int64(n) != chunkLen {
		s.userError(w, http.StatusBadRequest, "staRangeUpload.readChunk",
			fmt.Errorf("chunk size mismatch: leu %d, esperava %d", n, chunkLen))
		return
	}

	// BACEN PUT chunkado.
	ru, ok := s.STAClient.(sta.RangeUploader)
	if !ok {
		http.Error(w, `{"error":"STA backend não suporta chunked upload"}`, http.StatusServiceUnavailable)
		return
	}

	err = ru.SubmitRange(r.Context(), protocolo, crParsed.start, crParsed.end, crParsed.total, chunk)
	if err != nil {
		logger.Error("staRangeUpload: SubmitRange failed",
			"err", err, "protocolo", protocolo, "if_id", ifID)
		s.userError(w, http.StatusBadGateway, "staRangeUpload.bacen", err)
		return
	}

	// Atualiza sessão local (protegido por write lock).
	sessionsMu.Lock()
	session.ReceivedBytes += int64(n)
	session.Ranges = mergeRanges(session.Ranges, sta.Range{Start: crParsed.start, End: crParsed.end})
	if session.TotalBytes > 0 && session.ReceivedBytes >= session.TotalBytes {
		session.Status = "complete"
	}
	sessionsMu.Unlock()

	// Persiste progresso no DB (fire-and-forget).
	go s.persistSession(context.Background(), session)

	logger.Debug("staRangeUpload: chunk registrado",
		"protocolo", protocolo,
		"range", fmt.Sprintf("%d-%d/%d", crParsed.start, crParsed.end, crParsed.total),
		"received", session.ReceivedBytes, "if_id", ifID)

	writeJSON(w, http.StatusOK, map[string]any{
		"received_bytes": session.ReceivedBytes,
		"total_bytes":    session.TotalBytes,
		"ranges":         session.Ranges,
		"status":         session.Status,
	})
}

// staRangeStatus — GET /v1/sta/range/{protocolo}
//
// Retorna status dos chunks recebidos (para resume de upload interrompido).
//
// Retorna:
//   200 { protocolo, received_bytes, total_bytes, ranges, status }
//   401 unauthorized
//   404 não encontrada
func (s *Server) staRangeStatus(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	protocolo := chi.URLParam(r, "protocolo")
	if protocolo == "" {
		s.userError(w, http.StatusBadRequest, "staRangeStatus.protocolo", errors.New("protocolo requerido"))
		return
	}

	// 1. Source of truth: BACEN StatusUpload.
	ru, ok := s.STAClient.(sta.RangeUploader)
	if ok {
		bus, err := ru.StatusUpload(r.Context(), protocolo)
		if err == nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"protocolo":       bus.Protocolo,
				"received_bytes":  computeReceivedBytes(bus.RangesRecebidos),
				"total_bytes":     0,
				"ranges":          bus.RangesRecebidos,
				"status":          bus.Situacao.String(),
				"source":          "bacen",
			})
			return
		}
		// 404 do BACEN → tenta DB. Outro erro → loga e segue para DB.
		if !isSTA404(err) {
			logger.Warn("staRangeStatus: BACEN StatusUpload failed, falling back to DB",
				"err", err, "protocolo", protocolo)
		}
	}

	// 2. DB.
	session := s.loadSessionFromDB(r.Context(), protocolo)
	if session != nil {
		if session.IfID != ifID {
			http.Error(w, `{"error":"acesso negado"}`, http.StatusForbidden)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"protocolo":      session.Protocolo,
			"received_bytes": session.ReceivedBytes,
			"total_bytes":    session.TotalBytes,
			"ranges":         session.Ranges,
			"status":         session.Status,
			"source":         "db",
		})
		return
	}

	// 3. Memória.
	sessionsMu.RLock()
	sess := activeSessions[protocolo]
	sessionsMu.RUnlock()

	if sess != nil {
		if sess.IfID != ifID {
			http.Error(w, `{"error":"acesso negado"}`, http.StatusForbidden)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"protocolo":      sess.Protocolo,
			"received_bytes": sess.ReceivedBytes,
			"total_bytes":    sess.TotalBytes,
			"ranges":         sess.Ranges,
			"status":         sess.Status,
			"source":         "memory",
		})
		return
	}

	http.Error(w, `{"error":"sessão não encontrada"}`, http.StatusNotFound)
}

// --- Helpers ---

type rangeInitRequest struct {
	CadocCode  string `json:"cadoc_code"`
	HashHex    string `json:"hash_hex"`
	TotalBytes int64  `json:"total_bytes"`
}

type contentRange struct {
	start int64
	end   int64
	total int64
}

// parseContentRange parseia "bytes X-Y/TOTAL" (RFC 7233 §4.2).
func parseContentRange(cr string) (*contentRange, error) {
	if !strings.HasPrefix(cr, "bytes ") {
		return nil, fmt.Errorf("Content-Range deve começar com 'bytes ': %s", cr)
	}
	cr = strings.TrimPrefix(cr, "bytes ")
	parts := strings.Split(cr, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Content-Range sem total: %s", cr)
	}
	total, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Content-Range total inválido: %s", parts[1])
	}
	ends := strings.Split(parts[0], "-")
	if len(ends) != 2 {
		return nil, fmt.Errorf("Content-Range range part inválida: %s", parts[0])
	}
	start, err := strconv.ParseInt(ends[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Content-Range start inválido: %s", ends[0])
	}
	end, err := strconv.ParseInt(ends[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Content-Range end inválido: %s", ends[1])
	}
	if end < start {
		return nil, fmt.Errorf("end < start: %d < %d", end, start)
	}
	if total < end+1 {
		return nil, fmt.Errorf("total < end+1: %d < %d", total, end+1)
	}
	return &contentRange{start: start, end: end, total: total}, nil
}

// loadSessionFromDB carrega uma sessão do DB.
func (s *Server) loadSessionFromDB(ctx context.Context, protocolo string) *rangeSession {
	var id, ifID, status, rangesJSON string
	var totalBytes, receivedBytes int64
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, if_id, protocolo, total_bytes, received_bytes, ranges_json, status
		FROM range_sessions WHERE protocolo = ?
	`, protocolo).Scan(&id, &ifID, &protocolo, &totalBytes, &receivedBytes, &rangesJSON, &status)
	if err != nil {
		return nil
	}
	var ranges []sta.Range
	_ = json.Unmarshal([]byte(rangesJSON), &ranges)
	return &rangeSession{
		ID:             id,
		IfID:           ifID,
		Protocolo:      protocolo,
		TotalBytes:     totalBytes,
		ReceivedBytes:  receivedBytes,
		Ranges:         ranges,
		Status:         status,
	}
}

// persistSession persiste o progresso no DB (fire-and-forget).
func (s *Server) persistSession(ctx context.Context, session *rangeSession) {
	rangesJSON, _ := json.Marshal(session.Ranges)
	_, _ = s.DB.ExecContext(ctx, `
		UPDATE range_sessions SET
			received_bytes = ?,
			ranges_json = ?,
			status = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE protocolo = ?
	`, session.ReceivedBytes, rangesJSON, session.Status, session.Protocolo)
}

// computeReceivedBytes calcula total de bytes a partir dos ranges.
func computeReceivedBytes(ranges []sta.Range) int64 {
	var total int64
	for _, r := range ranges {
		total += r.End - r.Start + 1
	}
	return total
}

// mergeRanges adiciona um novo range ao slice, com coalescência de overlaps.
func mergeRanges(existing []sta.Range, newR sta.Range) []sta.Range {
	ranges := append(existing, newR)
	// Bubble sort (n ≤ few hundred chunks, não é hot path).
	for i := 0; i < len(ranges); i++ {
		for j := i + 1; j < len(ranges); j++ {
			if ranges[j].Start < ranges[i].Start {
				ranges[i], ranges[j] = ranges[j], ranges[i]
			}
		}
	}
	// Coalesce overlaps — aloca slice novo para evitar aliasing com ranges.
	merged := make([]sta.Range, 0, len(ranges))
	for _, r := range ranges {
		last := &merged[len(merged)-1]
		if len(merged) > 0 && r.Start <= last.End+1 {
			if r.End > last.End {
				last.End = r.End
			}
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}

// generateID gera um ID único estilo UUID v4 via crypto/rand.
func generateID() string {
	var b [16]byte
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(16))
		b[i] = byte(n.Int64())
		if b[i] >= 10 {
			b[i] = b[i] - 10 + 'a'
		} else {
			b[i] += '0'
		}
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// isSTA404 retorna true se o erro é um STAError com status 404.
func isSTA404(err error) bool {
	var staErr *sta.STAError
	if errors.As(err, &staErr) {
		return staErr.StatusCode == http.StatusNotFound
	}
	return false
}
