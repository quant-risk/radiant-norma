// Auth handlers (Sprint 8a v2.1.0).
//
// Sprint 8a introduziu bridge JWT real frontend↔backend. Frontend
// /api/login chama POST /v1/auth/dev-token para gerar JWT RS256 válido.
//
// Em produção: tokens emitidos por IdP externo (Keycloak/Okta/etc) —
// /v1/auth/dev-token retorna 404.
//
// Em dev: tokens emitidos in-process usando chave privada carregada de
// `RADIANT_DEV_JWT_PRIVATE_KEY` (env var path ou PEM inline). Endpoint
// ativado por `RADIANT_DEV_TOKEN=1` env flag.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
)

// envDevTokenFlag habilita o endpoint /v1/auth/dev-token.
// "1" ou "true" → habilitado. Default off (404).
const envDevTokenFlag = "RADIANT_DEV_TOKEN"

// envDevSignerPath é o env var com path da chave privada RSA em PEM.
// Ou `RADIANT_DEV_JWT_PRIVATE_KEY_PEM` para inline.
const envDevSignerPath = "RADIANT_DEV_JWT_PRIVATE_KEY"
const envDevSignerPEM = "RADIANT_DEV_JWT_PRIVATE_KEY_PEM"

// devTokenRequest é o payload aceito por /v1/auth/dev-token.
//
// Campos:
//   - if_id: tenant identifier (max 64 chars).
//   - role: "if" (default), "admin", "readonly".
//   - ttl_seconds: opcional, default 86400 (24h). Max 30 dias
//     (TTLCap) para evitar tokens de vida excessiva acidental.
//
// Frontend envia exatamente esta forma; cmd/jwt-mint usa --if, --role,
// --ttl flags equivalentes.
type devTokenRequest struct {
	IFID       string `json:"if_id"`
	Role       string `json:"role"`
	TTLSeconds int    `json:"ttl_seconds"`
}

// devTokenResponse é payload retornado.
//
// Campos:
//   - token: JWT RS256 string. Frontend armazena em cookie httpOnly.
//   - if_id, role: echo do pedido (sanitizados).
//   - expires_at: RFC3339 timestamp.
//   - ttl_seconds: TTL aplicado (pode diferir do pedido se clamped).
type devTokenResponse struct {
	Token      string `json:"token"`
	IFID       string `json:"if_id"`
	Role       string `json:"role"`
	ExpiresAt  string `json:"expires_at"`
	TTLSeconds int    `json:"ttl_seconds"`
}

// devTokenHandler emite JWT RS256 in-process para IF logado.
//
// Por design: NO FUTURO esta rota emite tokens via IdP. Hoje retorna
// 404 se `RADIANT_DEV_TOKEN` != "1" (defesa contra exposição acidental
// em prod). Se flag on mas DevSigner nil: 503 (configuração quebrada).
//
// Auditoria: dev-token emission logada com action="auth.dev_token.minted"
// (target = if_id) para forensic trail. Quem emitiu tokens em prod é
// rastreável mesmo que endpoint seja dev-only.
func (s *Server) devTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Defense: 404 se dev mode off — esconde endpoint em prod sem
	// revelar existência.
	if !isDevTokenEnabled() {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if s.DevSigner == nil {
		http.Error(w, "dev-token endpoint enabled but no signer configured (Server misconfigured)", http.StatusServiceUnavailable)
		return
	}

	// Decode body com cap de tamanho (MaxBytesReader defesa contra DOS).
	body, err := readBoundedJSON(r)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "devToken.readBody", err)
		return
	}
	var req devTokenRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.userError(w, http.StatusBadRequest, "devToken.parse", err)
		return
	}

	// Validate if_id.
	if req.IFID == "" {
		http.Error(w, `{"error":"if_id required"}`, http.StatusBadRequest)
		return
	}

	// Sprint 17 — v3.7.0 [S17.6 fix]: cross-tenant guard via lint.
	// Em dev mode sem JWT (não há claims no request), alinha req.IFID
	// com X-IF-ID header. Se header presente e diferente, 403 — impede
	// emitir JWT pra IF arbitrária.
	//
	// NOTA: endpoint é dev-only (404 em prod via RADIANT_DEV_TOKEN!=1).
	// Fail-closed gate no main.go já bloqueia em RADIANT_ENV=production.
	// Defense in depth: este guard garante que mesmo em dev multi-tenant,
	// um client não minta JWT pra outro IF via header X-IF-ID spoofing.
	if !s.enforceSameIF(w, r, req.IFID) {
		return
	}

	// Default role.
	if req.Role == "" {
		req.Role = string(auth.RoleIF)
	}

	// Validate role.
	switch auth.Role(req.Role) {
	case auth.RoleIF, auth.RoleAdmin, auth.RoleReadOnly:
	default:
		http.Error(w, `{"error":"role inválido (esperado if|admin|readonly)"}`, http.StatusBadRequest)
		return
	}

	// TTL: default TTLDefault, clamp TTLCap.
	ttl := auth.TTLDefault
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}
	if ttl > auth.TTLCap {
		ttl = auth.TTLCap
	}
	if ttl <= 0 {
		http.Error(w, `{"error":"ttl_seconds inválido"}`, http.StatusBadRequest)
		return
	}

	// Mint.
	signed, exp, err := s.DevSigner.MintSimple(req.IFID, auth.Role(req.Role), ttl)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "devToken.mint", err)
		return
	}

	// Audit emission (logs who asked for dev token).
	if s.AuditLog != nil {
		// Use remote IP — não temos ifID garantido até token minted.
		_, _ = s.AuditLog.Log("system", r.RemoteAddr, "auth.dev_token.minted", req.IFID,
			nil, map[string]any{
				"role": req.Role,
				"ttl":  ttl.String(),
			})
	}

	writeJSON(w, http.StatusOK, devTokenResponse{
		Token:      signed,
		IFID:       req.IFID,
		Role:       req.Role,
		ExpiresAt:  exp.UTC().Format(time.RFC3339),
		TTLSeconds: int(ttl / time.Second),
	})
}

// isDevTokenEnabled lê RADIANT_DEV_TOKEN env flag.
// Returns true se "1" ou "true" (case-insensitive).
func isDevTokenEnabled() bool {
	v := os.Getenv(envDevTokenFlag)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		// aceitar "1" também
		return v == "1"
	}
	return b
}

// readBoundedJSON lê body com cap de tamanho razoável.
//
// Defesa contra DOS-via-large-body (mesmo padrão de maxBodyBytesMiddleware,
// mas aplicado a JSON parse aqui para evitar parsing de body gigantes
// antes de validar). 1 MiB é mais que suficiente para /auth/dev-token
// (payload é trivial: if_id + role + ttl).
func readBoundedJSON(r *http.Request) ([]byte, error) {
	const maxBytes = 1 << 20 // 1 MiB
	if r.Body != nil && r.ContentLength > maxBytes {
		return nil, errors.New("body too large")
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	defer r.Body.Close()
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 1024)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if errors.Is(err, http.ErrBodyReadAfterClose) {
				break
			}
			// EOF ou error esperado em fim de stream
			break
		}
	}
	return buf, nil
}
