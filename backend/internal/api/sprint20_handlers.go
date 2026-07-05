// Package api — handlers Sprint 20 (v3.10.0): STA WS read side REST.
//
// Endpoints:
//
//	GET  /v1/sta/disponiveis?dataHoraInicio=YYYY-MM-DDTHH:MM:SS.SSS
//	POST /v1/sta/situacao  body {"protocolos":["1","2"],"situacao":"REC"}
//
// Requer WSClient (RADIANT_STA_BACKEND=ws). Se s.STAClient for *StubClient,
// retorna 503 (interface segregation — ver SPRINT_20_RESEARCH.md §4.6).
//
// Auth: JWT RS256 + enforceSameIF (caller não pode listar/alterar arquivos
// de outra IF). Rate limiting: middleware global (rateLimitMiddleware) +
// rate por rota não implementado nesta sprint (Sprint 22+ se virar problema).
//
// Audit: toda chamada emite sta.disponiveis.listed ou sta.situacao.changed
// em sucesso. Em erro 4xx BACEN, emite sta.{op}.rejected. Em erro de
// transporte, sta.{op}.failed com err.Error() sanitizado via SafeError.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
)

// staDisponiveisHandler — GET /v1/sta/disponiveis.
//
// Query params (todos opcionais exceto dataHoraInicio):
//   - dataHoraInicio: obrigatório, formato "yyyy-MM-ddTHH:mm:ss.SSS"
//   - identificadorDocumento: opcional (ex.: "3040")
//   - sistemas: opcional, até 100 separados por ";"
//   - dependencia: opcional (default = if_id do tenant autenticado)
//
// Response 200: JSON com lista de arquivos + paginação.
func (s *Server) staDisponiveisHandler(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	dataHoraInicio := q.Get("dataHoraInicio")
	if dataHoraInicio == "" {
		http.Error(w, "dataHoraInicio obrigatório (formato yyyy-MM-ddTHH:mm:ss.SSS)",
			http.StatusBadRequest)
		return
	}

	rc, ok := s.STAClient.(sta.ReadClient)
	if !ok {
		// StubClient ou outro backend sem read side. Sem retry — caller
		// precisa setar RADIANT_STA_BACKEND=ws.
		_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "sta.disponiveis.stub_backend",
			"disponiveis", nil, map[string]any{
				"err": "STA backend não suporta read side (RADIANT_STA_BACKEND deve ser ws)",
			})
		http.Error(w, "read side do STA não disponível neste backend", http.StatusServiceUnavailable)
		return
	}

	opts := sta.ListDisponiveisOpts{
		DataHoraInicio:         dataHoraInicio,
		IdentificadorDocumento: q.Get("identificadorDocumento"),
		Sistemas:               q.Get("sistemas"),
		Dependencia:            q.Get("dependencia"),
	}
	// Default dependencia = tenant do JWT (se caller não informou).
	if opts.Dependencia == "" {
		opts.Dependencia = ifID
	}

	res, err := rc.ListDisponiveis(r.Context(), opts)
	if err != nil {
		s.handleSTAReadError(w, r, ifID, "sta.disponiveis", "disponiveis", err)
		return
	}

	// Audit sucesso.
	_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "sta.disponiveis.listed",
		"disponiveis", nil, map[string]any{
			"data_hora_inicio":   dataHoraInicio,
			"qtde_arquivos":      len(res.Arquivos),
			"tem_proxima_pagina": res.TemProximaPagina,
		})

	// JSON response — converte tipos para snake_case + tipos públicos.
	writeJSON(w, http.StatusOK, map[string]any{
		"arquivos":                   convertArquivosJSON(res.Arquivos),
		"data_hora_proxima_consulta": res.DataHoraProximaConsulta,
		"proxima_pagina_url":         res.ProximaPaginaURL,
		"tem_proxima_pagina":         res.TemProximaPagina,
	})
}

// staSituacaoHandler — POST /v1/sta/situacao.
//
// Body JSON: {"protocolos": ["1", "2"], "situacao": "REC"}
//
// situacao aceita: "A_REC" | "REC" (case-sensitive).
//
// Response 204 No Content em sucesso.
func (s *Server) staSituacaoHandler(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "staSituacao.readBody", err)
		return
	}

	var req struct {
		Protocolos []string `json:"protocolos"`
		Situacao   string   `json:"situacao"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		s.userError(w, http.StatusBadRequest, "staSituacao.jsonUnmarshal", err)
		return
	}
	if len(req.Protocolos) == 0 {
		http.Error(w, "protocolos obrigatório (lista não vazia)", http.StatusBadRequest)
		return
	}
	if req.Situacao != "A_REC" && req.Situacao != "REC" {
		http.Error(w, `situacao deve ser "A_REC" ou "REC"`, http.StatusBadRequest)
		return
	}

	rc, ok := s.STAClient.(sta.ReadClient)
	if !ok {
		_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "sta.situacao.stub_backend",
			"situacao", body, map[string]any{
				"err": "STA backend não suporta read side (RADIANT_STA_BACKEND deve ser ws)",
			})
		http.Error(w, "read side do STA não disponível neste backend", http.StatusServiceUnavailable)
		return
	}

	alterReq := sta.AlterarSituacaoReq{
		Protocolos: req.Protocolos,
		Situacao:   staParseSituacaoTransferencia(req.Situacao),
	}
	if err := rc.AlterarSituacao(r.Context(), alterReq); err != nil {
		s.handleSTAReadError(w, r, ifID, "sta.situacao", "situacao", err)
		return
	}

	_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "sta.situacao.changed",
		"situacao", body, map[string]any{
			"qtde_protocolos": len(req.Protocolos),
			"situacao":        req.Situacao,
		})

	// 204 No Content (per spec manual Seção 7.1).
	w.WriteHeader(http.StatusNoContent)
}

// handleSTAReadError mapeia erros do WSClient read side para HTTP status + audit.
//   - *STAError: BACEN rejeitou formalmente (4xx). Retorna status correspondente.
//   - Outros: erro de transporte. Retorna 502 Bad Gateway (proxy error).
func (s *Server) handleSTAReadError(w http.ResponseWriter, r *http.Request, ifID, auditPrefix, ref string, err error) {
	var staErr *sta.STAError
	if errors.As(err, &staErr) {
		// Mapeia status BACEN → HTTP status. BACEN usa 400/403/404/410.
		httpStatus := staErr.StatusCode
		if httpStatus == 0 {
			httpStatus = http.StatusBadGateway
		}
		// Audit: rejeitado formalmente.
		_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, auditPrefix+".rejected",
			ref, nil, map[string]any{
				"status_code": staErr.StatusCode,
				"message":     loggerutil.SafeError(staErr),
			})
		// Cliente vê mensagem genérica (não vaza err cru — F18.1).
		publicMsg := fmt.Sprintf("BACEN rejeitou requisição (status %d)", staErr.StatusCode)
		http.Error(w, publicMsg, httpStatus)
		return
	}
	// Erro de transporte / parse / config.
	_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, auditPrefix+".failed",
		ref, nil, map[string]any{"err": loggerutil.SafeError(err)})
	http.Error(w, "erro ao contatar BACEN", http.StatusBadGateway)
}

// convertArquivosJSON converte []ArquivoDisponivel (Go) → JSON-friendly map slice
// com snake_case keys. Mantém tipos públicos JSON-safe.
func convertArquivosJSON(arquivos []sta.ArquivoDisponivel) []map[string]any {
	out := make([]map[string]any, 0, len(arquivos))
	for _, a := range arquivos {
		out = append(out, map[string]any{
			"protocolo":                  a.Protocolo,
			"tipo_arquivo":               a.TipoArquivo,
			"codigo_documento":           a.CodigoDocumento,
			"sistema":                    a.Sistema,
			"tamanho_arquivo":            a.TamanhoArquivo,
			"hash":                       a.Hash,
			"situacao_atual":             a.SituacaoAtual.String(),
			"situacao_atual_raw":         a.SituacaoAtualRaw,
			"data_hora_disponibilizacao": a.DataHoraDisponibilizacao,
		})
	}
	return out
}

// staParseSituacaoTransferencia converte string ("A_REC" | "REC") → enum.
// Já validado no handler antes de chamar (400 se não for um dos 2).
func staParseSituacaoTransferencia(s string) sta.SituacaoTransferencia {
	switch s {
	case "A_REC":
		return sta.SituacaoTransferenciaAReceber
	case "REC":
		return sta.SituacaoTransferenciaRecebido
	default:
		return sta.SituacaoTransferenciaUnknown
	}
}
