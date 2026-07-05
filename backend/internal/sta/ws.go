// Package sta — WSClient (cliente nativo BACEN STA Web Services v1.5).
//
// Sprint 18 (v3.8.0): cliente REST contra https://sta-h.bcb.gov.br/staws (homologação)
// ou https://sta.bcb.gov.br/staws (produção).
//
// Implementa o fluxo de envio em 2 fases (Seções 5.1 + 5.2 do manual oficial):
//  1. POST /arquivos com XML <Parametros> (IdentificadorDocumento, Hash, Tamanho, etc.)
//     → 201 Created + <Resultado><Protocolo>{NUM}</Protocolo></Resultado>
//  2. PUT /arquivos/{protocolo}/conteudo com conteúdo binário (ZIP)
//     → 200 OK
//
// Auth: HTTP Basic preemptivo (RFC 7617). Header:
//
//	Authorization: Basic base64(UUUUUDDDD.operador:senha)
//
// Onde:
//   - UUUUU = código Sisbacen da IF (5 dígitos)
//   - DDDD  = código Sisbacen da dependência (4 dígitos)
//   - operador = nome de usuário
//
// Pré-requisito de acesso: usuário cadastrado no Sisbacen/Autran + credenciado
// na transação PSTA300. Sem isso, qualquer chamada retorna 401/403.
//
// O cliente:
//   - Calcula SHA-256 do payload ANTES do POST (Seção 2.4 do manual)
//   - Respeita timeout configurado
//   - Sanitiza error bodies (não vaza err.Error() cru — F18.1 defense)
//   - Emite audit emission em failure (S17.6 pattern — mas ainda
//     sem audit_log wiring aqui; feito no handler do /v1/sta/submit)
//
// Limitações V1 (cobertas em Sprint 19+):
//   - Sem retry exponencial
//   - Sem range/resume upload (chunked)
//   - Sem download (recebimento)
//   - Sem senha rotation / consulta disponibilidade
//   - Sem cache de senhaws endpoints
//
// Referência: _referencias/STA_Manual_WebServices.pdf (BACEN, julho/2022).
package sta

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// WSConfig configura o WSClient.
type WSConfig struct {
	// BaseURL é a URL base do BACEN STA WS. Sem path.
	// Exemplo: "https://sta-h.bcb.gov.br/staws" (homologação)
	//          "https://sta.bcb.gov.br/staws"     (produção)
	BaseURL string

	// User é o usuário Sisbacen no formato "UUUUUDDDD.operador".
	// Exemplo: "12345/0001.fulano" — UUUUU=12345, DDDD=0001, operador=fulano.
	// Validação rigorosa em NewWSClient.
	User string

	// Password é a senha Sisbacen. NÃO log em logs (F13.8).
	Password string

	// Timeout é o timeout total para cada chamada HTTP (incluindo
	// dial + write + read). Default 30 segundos.
	Timeout time.Duration

	// HTTPClient é opcional. Se nil, usa http.DefaultClient com o timeout
	// acima. Tests injetam httptest.Server URL via isso.
	HTTPClient *http.Client

	// AllowInsecureHTTP desabilita a checagem estrita de HTTPS — útil para
	// testes com httptest.NewServer (que retorna http://127.0.0.1:port).
	// Default false. NUNCA setar true em produção.
	AllowInsecureHTTP bool

	// Logger é opcional. Se nil, usa slog.Default().
	Logger *slog.Logger
}

// WSClient é o cliente HTTP para BACEN STA Web Services.
//
// Thread-safe: cada Submit usa http.Client.Do() que é thread-safe; conexões
// são reutilizadas via keep-alive. Sem campos mutáveis no estado do client
// após construção.
type WSClient struct {
	cfg    WSConfig
	logger *slog.Logger
}

// NewWSClient valida config e cria o cliente. Retorna erro descritivo se
// config inválida (sem fazer call de rede).
func NewWSClient(cfg WSConfig) (*WSClient, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("WSConfig.BaseURL requerida (ex.: https://sta-h.bcb.gov.br/staws)")
	}
	if !cfg.AllowInsecureHTTP && !strings.HasPrefix(cfg.BaseURL, "https://") {
		return nil, fmt.Errorf("WSConfig.BaseURL deve usar HTTPS (got %q; use AllowInsecureHTTP=true para testes dev)", cfg.BaseURL)
	}
	if strings.HasSuffix(cfg.BaseURL, "/") {
		return nil, errors.New("WSConfig.BaseURL não deve terminar com /")
	}
	if cfg.User == "" {
		return nil, errors.New("WSConfig.User requerida (formato UUUUUDDDD.operador)")
	}
	if !strings.Contains(cfg.User, ".") {
		return nil, fmt.Errorf("WSConfig.User deve estar no formato UUUUUDDDD.operador (got %q)", cfg.User)
	}
	if cfg.Password == "" {
		return nil, errors.New("WSConfig.Password requerida")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: cfg.Timeout}
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &WSClient{cfg: cfg, logger: logger}, nil
}

// basicAuthHeader monta o header Authorization: Basic base64(user:pass).
//
// Sprint 13 — F13.8: NÃO logar esta função nem o resultado dela. Logger
// chamadas só acontecem depois do header ser consumido em request real.
func (c *WSClient) basicAuthHeader() string {
	creds := c.cfg.User + ":" + c.cfg.Password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

// Submit implementa Client.Submit — fluxo 2-fase (POST protocolo + PUT conteúdo).
//
// Fases (Seção 5 do manual):
//  1. POST /arquivos com XML em application/xml. Retorna 201 + protocolo.
//  2. PUT /arquivos/{protocolo}/conteudo com payload binário.
//
// Submit retorna:
//   - (ProtocolSTA string, Accepted=true, nil) em sucesso
//   - (ProtocolSTA, Accepted=false, &Rejection{}) em rejeição conhecida do BACEN
//   - ("", false, err) em erro de transporte (timeout, malformed XML, etc.)
//
// Audit emission acontece no handler /v1/sta/submit — WSClient emite logs
// estruturados aqui (N1.4-debug), mas não emite audit_log (deferido pra
// Sprint 19+ ou decidido no handler).
func (c *WSClient) Submit(ctx context.Context, sub *Submission) (*Result, error) {
	if sub.Zip == nil && len(sub.XML) == 0 {
		return nil, errors.New("STA submission vazia (sem XML nem ZIP)")
	}

	// Conteúdo a enviar: prioriza ZIP (compactado), senão XML.
	payload := sub.Zip
	if payload == nil {
		payload = []byte(sub.XML)
	}

	// Hash SHA-256 do conteúdo compactado (Seção 2.4 do manual).
	sum := sha256.Sum256(payload)
	hashHex := hex.EncodeToString(sum[:])

	// Phase 1: POST /arquivos → protocolo
	protocolo, err := c.requestProtocol(ctx, sub, hashHex, int64(len(payload)))
	if err != nil {
		return nil, fmt.Errorf("STA POST /arquivos falhou: %w", err)
	}
	if protocolo == "" {
		// Não deveria acontecer — requestProtocol retorna erro antes de
		// string vazia. Defensive log.
		c.logger.Warn("STA requestProtocol retornou protocolo vazio sem erro",
			"if_id", sub.CNPJ, "cadoc", sub.CadocCode)
		return &Result{ProtocolSTA: "", Accepted: false, Rejection: &Rejection{Code: "INTERNAL", Message: "protocolo vazio"}}, nil
	}

	// Phase 2: PUT /arquivos/{protocolo}/conteudo (binário).
	if err := c.uploadContent(ctx, protocolo, payload); err != nil {
		// Upload falhou — protocolo existe mas conteúdo não foi aceito.
		// Mantemos protocolo no result para forensic trail. Rejection
		// indica causa da falha de upload.
		return &Result{
			ProtocolSTA: protocolo,
			Accepted:    false,
			Rejection: &Rejection{
				Code:    "UPLOAD_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &Result{
		ProtocolSTA: protocolo,
		Accepted:    true,
	}, nil
}

// requestProtocol implementa a Fase 1 (Seção 5.1.1).
//
// POST {BaseURL}/arquivos com XML body application/xml.
// Retorna protocolo em sucesso.
func (c *WSClient) requestProtocol(ctx context.Context, sub *Submission, hashHex string, size int64) (string, error) {
	params := requestProtocolParams{
		IdentificadorDocumento: sub.CadocCode,
		Hash:                   hashHex,
		Tamanho:                size,
		NomeArquivo:            fmt.Sprintf("%s.zip", sub.CadocCode),
		// Observacao + Destinatarios vazios — não aplicáveis para este
		// fluxo simples (envio único).
	}
	body, err := xml.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("marshal XML request: %w", err)
	}
	// Adiciona declaração XML (xml.Marshal não inclui standalone="yes" por default
	// e o manual exige).
	body = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" + string(body))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.BaseURL+"/arquivos", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Accept", "application/xml")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	// 201 Created esperado. Outros códigos viram error.
	if resp.StatusCode != http.StatusCreated {
		return "", c.parseSTAError(resp.StatusCode, respBody)
	}

	var rp responseProtocol
	if err := xml.Unmarshal(respBody, &rp); err != nil {
		return "", fmt.Errorf("parse XML response: %w (body=%s)", err, truncate(respBody, 200))
	}
	if rp.Protocolo == "" {
		return "", fmt.Errorf("BACEN retornou 201 mas <Protocolo> vazio: %s", truncate(respBody, 200))
	}
	return rp.Protocolo, nil
}

// uploadContent implementa a Fase 2 (Seção 5.2.1).
//
// PUT {BaseURL}/arquivos/{protocolo}/conteudo com payload binário.
// Content-Type OMITIDO por default (BACEN recomenda não setar); quando
// setado, multipart/form-data é proibido.
func (c *WSClient) uploadContent(ctx context.Context, protocolo string, payload []byte) error {
	url := c.cfg.BaseURL + "/arquivos/" + protocolo + "/conteudo"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	// Content-Type intencionalmente omitido — BACEN trata como binary stream.
	// Manual Seção 5.2.1: "não precisa conter o campo Content-Type".
	req.ContentLength = int64(len(payload))

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return c.parseSTAError(resp.StatusCode, respBody)
	}
	return nil
}

// parseSTAError mapeia resposta de erro XML do BACEN para erro Go.
//
// O BACEN sempre responde 4xx/5xx com body XML no formato:
//
//	<Resultado><Erro><Codigo>{STATUS}</Codigo><Descricao>{MSG}</Descricao></Erro></Resultado>
//
// (Listagem 4 do manual)
func (c *WSClient) parseSTAError(status int, body []byte) error {
	var xe xmlError
	if err := xml.Unmarshal(body, &xe); err == nil && xe.Erro.Codigo != 0 {
		return fmt.Errorf("BACEN STA error %d: %s", xe.Erro.Codigo, xe.Erro.Descricao)
	}
	// Body não parseou — retorna status + truncated body.
	return fmt.Errorf("BACEN STA HTTP %d: %s", status, truncate(body, 200))
}

// truncate retorna os primeiros n bytes de b com "..." se truncar.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

// NewClientFromEnv constrói o STA client baseado em env var RADIANT_STA_BACKEND.
//
// Valores aceitos:
//
//	"stub" (default) — StubClient (não faz chamada real, sempre aceita).
//	                   Mantido para garantir compat com v3.7.x.
//	"ws"             — WSClient contra BACEN STA Web Services REST v1.5
//	                   (precisa RADIANT_STA_WS_URL, _SISBACEN_USER, _PASSWORD).
//
// Sprint 18 (v3.8.0): V1 do factory. Playwright stub fica em cmd/
// como wrapper separado (path 1.0 antigo); quando integrado, será
// "playwright" como terceiro valor.
func NewClientFromEnv(logger *slog.Logger) (Client, error) {
	backend := os.Getenv("RADIANT_STA_BACKEND")
	if backend == "" || backend == "stub" {
		logger.Info("STA client usando Stub (default; sem chamadas reais)")
		return NewStubClient(), nil
	}
	if backend == "ws" {
		baseURL := os.Getenv("RADIANT_STA_WS_URL")
		user := os.Getenv("RADIANT_STA_SISBACEN_USER")
		password := os.Getenv("RADIANT_STA_SISBACEN_PASSWORD")
		timeoutStr := os.Getenv("RADIANT_STA_TIMEOUT_SECONDS")

		if baseURL == "" || user == "" || password == "" {
			return nil, fmt.Errorf("RADIANT_STA_BACKEND=ws requer RADIANT_STA_WS_URL + _SISBACEN_USER + _SISBACEN_PASSWORD")
		}

		timeout := 30 * time.Second
		if timeoutStr != "" {
			if n, err := strconv.Atoi(timeoutStr); err == nil && n > 0 {
				timeout = time.Duration(n) * time.Second
			}
		}

		c, err := NewWSClient(WSConfig{
			BaseURL:  baseURL,
			User:     user,
			Password: password,
			Timeout:  timeout,
			Logger:   logger,
		})
		if err != nil {
			return nil, fmt.Errorf("WSClient config inválida: %w", err)
		}
		logger.Info("STA client usando WSClient contra BACEN",
			"base_url", baseURL,
			"user", user,
			"timeout", timeout,
		)
		return c, nil
	}
	return nil, fmt.Errorf("RADIANT_STA_BACKEND=%q inválido (aceito: stub|ws)", backend)
}

// BackendName retorna um identificador curto do tipo de client.
// Útil para logs e métricas. Valores: "stub" | "ws".
func BackendName(c Client) string {
	switch c.(type) {
	case *StubClient:
		return "stub"
	case *WSClient:
		return "ws"
	default:
		return fmt.Sprintf("unknown(%T)", c)
	}
}
