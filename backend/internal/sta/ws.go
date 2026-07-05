// Package sta — WSClient (cliente nativo BACEN STA Web Services v1.5).
//
// Sprint 18 (v3.8.0): cliente REST contra https://sta-h.bcb.gov.br/staws (homologação)
// ou https://sta.bcb.gov.br/staws (produção).
//
// Sprint 19 (v3.9.0): read side — WSClient.StatusUpload (Seção 5.3.1) e
// WSClient.Download (Seção 6.1.1). Validação X-Content-Hash obrigatória.
//
// Implementa o fluxo de envio em 2 fases (Seções 5.1 + 5.2 do manual oficial):
//  1. POST /arquivos com XML <Parametros> (IdentificadorDocumento, Hash, Tamanho, etc.)
//     → 201 Created + <Resultado><Protocolo>{NUM}</Protocolo></Resultado>
//  2. PUT /arquivos/{protocolo}/conteudo com conteúdo binário (ZIP)
//     → 200 OK
//
// E o fluxo de leitura:
//  3. GET /arquivos/{protocolo}/posicaoupload → status + ranges recebidos
//  4. GET /arquivos/{protocolo}/conteudo → arquivo binário + X-Content-Hash
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
//   - Compara SHA-256 do body do Download com X-Content-Hash do header
//     (Seção 6.1.1, validação obrigatória de integridade — manual linha 641-643)
//   - Respeita timeout configurado
//   - Sanitiza error bodies (não vaza err.Error() cru — F18.1 defense)
//   - Emite audit emission em failure (S17.6 pattern — mas ainda
//     sem audit_log wiring aqui; feito no handler do /v1/sta/submit)
//
// Limitações V1 (cobertas em Sprint 20+):
//   - Sem retry exponencial
//   - Sem range/resume upload (chunked)
//   - Sem range/conditional download (single-call apenas)
//   - Sem listagem /arquivos/disponiveis (Sprint 20)
//   - Sem alteração situacao /arquivos/situacao (Sprint 20)
//   - Sem cache de senhaws endpoints
//
// Referência: _referencias/STA_Manual_WebServices.pdf (BACEN, julho/2022).
package sta

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// maxResponseBodyBytes limita o tamanho da response BACEN que vamos ler
// para responses pequenas (XML de protocolo, posicaoupload, erros).
// Defense-in-depth contra BACEN mal-comportado ou proxy transparente que
// retorne body gigante (DoS via memory). Manual Seção 5.1.1 / 5.2.1 / 5.3.1
// não limita tamanho do body, mas responses esperadas são pequenas
// (~few KB para XML de protocolo/posicaoupload, vazio para PUT sucesso,
// <10 KB para erros).
const maxResponseBodyBytes = 10 << 20 // 10 MiB

// maxDownloadBodyBytes limita o tamanho do body do Download.
// CADOC real raramente >10 MB (ZIP com poucas dezenas de milhares de linhas
// de relatório). 100 MiB é folgado mas prudente — se BACEN algum dia enviar
// arquivo de 500 MB (improvável), cliente recebe erro claro em vez de
// estouro de memória silencioso.
//
// Decisão (Sprint 19): não truncar silenciosamente. Body excedente → erro
// *STAError{StatusCode: 413} para o caller decidir (retry? chunked download
// via range? falhar graciosamente?).
const maxDownloadBodyBytes = 100 << 20 // 100 MiB

// Sentinel errors — callers usam errors.Is() para classificar.
var (
	// ErrContentHashMismatch: SHA-256 do body não bate com header X-Content-Hash.
	// Erro fatal — não adianta retry, BACEN mandou dado corrompido. Caller deve
	// abortar e abrir ticket com BACEN.
	ErrContentHashMismatch = errors.New("STA: X-Content-Hash do BACEN não bate com SHA-256 do body")

	// ErrContentHashHeaderMalformed: header X-Content-Hash existe mas não segue
	// formato esperado "SHA-256 {64-hex}". Defense contra BACEN mudar formato
	// sem atualizar IF — caller recebe sentinel distinto pra diferenciar
	// "BACEN mudou header" de "BACEN mandou lixo".
	ErrContentHashHeaderMalformed = errors.New("STA: header X-Content-Hash malformado (esperado: SHA-256 {64-hex})")
)

// STAError representa rejeição formal do BACEN STA WS (4xx/5xx com XML
// formato Listagem 4 do manual). Distinct de erros de transporte (rede,
// parse, timeout) que retornam err tipado diferente.
//
// StatusCode: HTTP status cru (404, 403, 410, 400, ...).
// Code: código extraído de <Erro><Codigo> (mesmo valor de StatusCode hoje,
// mantido para evolução futura se BACEN mudar).
// Message: <Erro><Descricao> cru.
// Protocolo: se conhecido, eco do protocolo que originou o erro (útil pra
// audit correlacionar com submission).
type STAError struct {
	StatusCode int
	Code       string
	Message    string
	Protocolo  string
}

func (e *STAError) Error() string {
	if e.Protocolo != "" {
		return fmt.Sprintf("BACEN STA error %d (protocolo=%s): %s", e.StatusCode, e.Protocolo, e.Message)
	}
	return fmt.Sprintf("BACEN STA error %d: %s", e.StatusCode, e.Message)
}

// sisbacenUserRegex is the canonical format Sisbacen: 5 dígitos (IF code)
// + 4 dígitos (dependência) + "." + operador alfanumérico/underscore/dash.
//
// Exemplos aceitos: "123450001.fulano" (concatenado) ou
// "12345/0001.fulano" (com slash, forma comum em scripts BACEN).
// Rationale: a documentação BACEN é inconsistente entre manuais (alguns
// dizem UUUUUDDDD sem separador; outros usam UUUUU/0001). Aceitamos
// ambos para ergonomia — o BACEN rejeita formalmente se inválido.
var sisbacenUserRegex = regexp.MustCompile(`^(\d{5}\d{4}|\d{5}/\d{4})\.[A-Za-z0-9_-]+$`)

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
	if !sisbacenUserRegex.MatchString(cfg.User) {
		return nil, fmt.Errorf("WSConfig.User deve seguir formato Sisbacen exato (5 dígitos + 4 dígitos + . + operador alfanumérico), got %q", cfg.User)
	}
	if cfg.Password == "" {
		return nil, errors.New("WSConfig.Password requerida")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HTTPClient == nil {
		// Manual BACEN STA WS v1.5 Seção 2.5: "A plataforma de
		// desenvolvimento do cliente dos Web Services deve ter suporte a
		// HTTP 1.1". Default http.Transport do Go (Go 1.18+) tenta
		// HTTP/2 primeiro via ALPN; forçamos HTTP/1.1 aqui pra alinhar
		// com spec e evitar bugs sutis quando BACEN rejeitar upgrade.
		cfg.HTTPClient = &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
				ForceAttemptHTTP2: false,
			},
		}
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

	// Defense-in-depth (Validação 39): cap response body para evitar
	// DoS via BACEN misbehaving ou proxy transparente inflando body.
	// Responses esperadas: protocolo ~few KB; PUT sucesso vazio;
	// erros <10 KB. Cap em 10 MiB é folgado.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))

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

// parseSTAErrorTyped é a versão tipada de parseSTAError — retorna *STAError
// em vez de error opaco. Usada por StatusUpload e Download onde o caller
// precisa inspecionar StatusCode (ex: 404 protocolo inexistente, 410 arquivo
// não disponível).
//
// protocolo é eco do protocolo que originou o erro (string vazia se N/A).
func (c *WSClient) parseSTAErrorTyped(status int, body []byte, protocolo string) error {
	var xe xmlError
	if err := xml.Unmarshal(body, &xe); err == nil && xe.Erro.Codigo != 0 {
		return &STAError{
			StatusCode: status,
			Code:       strconv.Itoa(xe.Erro.Codigo),
			Message:    xe.Erro.Descricao,
			Protocolo:  protocolo,
		}
	}
	// Body não parseou — fallback STAError com body cru truncado.
	return &STAError{
		StatusCode: status,
		Code:       fmt.Sprintf("HTTP_%d", status),
		Message:    truncate(body, 200),
		Protocolo:  protocolo,
	}
}

// ============================================================
// Sprint 19 (v3.9.0) — read side: StatusUpload + Download
// ============================================================

// StatusUpload consulta a situação de um envio em andamento no BACEN.
//
// Endpoint: GET /arquivos/{protocolo}/posicaoupload (Seção 5.3.1 do manual).
//
// Retorna:
//   - (*UploadStatus, nil) em sucesso — RangesRecebidos parseado, Situacao
//     tipado como enum (3 valores oficiais do manual).
//   - (nil, *STAError) em rejeição formal BACEN (400/403/404/410).
//     Caller pode errors.As(err, &staErr) pra inspecionar StatusCode.
//   - (nil, err) em erro de transporte (rede, timeout, parse falho).
//
// Use case típico: cliente tem protocolo "12345" e quer retomar upload
// interrompido — chama StatusUpload pra saber que bytes BACEN já recebeu
// (RangesRecebidos) antes de fazer PUT só da parte que falta.
func (c *WSClient) StatusUpload(ctx context.Context, protocolo string) (*UploadStatus, error) {
	if protocolo == "" {
		return nil, errors.New("protocolo requerido")
	}

	url := c.cfg.BaseURL + "/arquivos/" + protocolo + "/posicaoupload"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	// Content-Type intencionalmente omitido — manual §5.3.1 linha 451:
	// "O cabeçalho HTTP da requisição não deve conter o campo Content-Type".

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseSTAErrorTyped(resp.StatusCode, respBody, protocolo)
	}

	var p posicaoUploadResponse
	if err := xml.Unmarshal(respBody, &p); err != nil {
		return nil, fmt.Errorf("parse posicaoupload XML: %w (body=%s)", err, truncate(respBody, 200))
	}

	return &UploadStatus{
		Protocolo:       p.Protocolo,
		RangesRecebidos: parseRanges(p.RangesRecebidos),
		Situacao:        parseUploadSituacao(p.Situacao),
		SituacaoRaw:     p.Situacao,
	}, nil
}

// Download baixa o arquivo binário de um protocolo do BACEN (Seção 6.1.1).
//
// Endpoint: GET /arquivos/{protocolo}/conteudo (recebimento completo).
//
// Validação obrigatória (manual linha 641-643): o cliente computa SHA-256
// do body e compara com header X-Content-Hash do BACEN. Mismatch →
// ErrContentHashMismatch (sentinel). Não-recuperável.
//
// Retorna:
//   - (*DownloadResult, nil) em sucesso com integridade validada.
//   - (nil, *STAError) em rejeição formal BACEN:
//   - 400: erro genérico XML Listagem 4
//   - 404: protocolo inexistente
//   - 410: arquivo não disponível para download
//   - (nil, ErrContentHashMismatch) se BACEN mandou body com hash divergente.
//   - (nil, ErrContentHashHeaderMalformed) se X-Content-Hash existe mas formato mudou.
//   - (nil, *STAError{StatusCode: 413}) se body excede 100 MiB (cap).
//   - (nil, err) em erro de transporte (rede, timeout).
func (c *WSClient) Download(ctx context.Context, protocolo string) (*DownloadResult, error) {
	if protocolo == "" {
		return nil, errors.New("protocolo requerido")
	}

	url := c.cfg.BaseURL + "/arquivos/" + protocolo + "/conteudo"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	// Content-Type intencionalmente omitido — manual §6.1.1 linha 620.

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Lê body com cap (defense in depth). Se exceder maxDownloadBodyBytes,
	// retornamos *STAError 413 sem alocar 500 MB indevidamente.
	limited := io.LimitReader(resp.Body, maxDownloadBodyBytes+1) // +1 pra detectar overflow
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read download body: %w", err)
	}
	if int64(len(body)) > maxDownloadBodyBytes {
		return nil, &STAError{
			StatusCode: http.StatusRequestEntityTooLarge,
			Code:       fmt.Sprintf("HTTP_%d", http.StatusRequestEntityTooLarge),
			Message:    fmt.Sprintf("body excede %d bytes (cap defensivo)", maxDownloadBodyBytes),
			Protocolo:  protocolo,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseSTAErrorTyped(resp.StatusCode, body, protocolo)
	}

	// Validação X-Content-Hash — obrigatória conforme manual §6.1.1.
	hashHeader := resp.Header.Get("X-Content-Hash")
	if hashHeader == "" {
		return nil, &STAError{
			StatusCode: http.StatusBadGateway,
			Code:       "MISSING_X_CONTENT_HASH",
			Message:    "BACEN não retornou header X-Content-Hash (esperado conforme manual §6.1.1)",
			Protocolo:  protocolo,
		}
	}
	wantHash, malformedErr := parseXContentHash(hashHeader)
	if malformedErr != nil {
		return nil, fmt.Errorf("%w: %v (header=%q)", ErrContentHashHeaderMalformed, malformedErr, hashHeader)
	}

	// SHA-256 do body (Seção 2.4 do manual — mesmo algoritmo do upload).
	sum := sha256.Sum256(body)
	gotHash := hex.EncodeToString(sum[:])

	if gotHash != wantHash {
		return nil, fmt.Errorf("%w: esperado=%s got=%s (body=%d bytes)",
			ErrContentHashMismatch, wantHash, gotHash, len(body))
	}

	return &DownloadResult{
		Conteudo:          body,
		ContentHash:       gotHash,
		ETag:              resp.Header.Get("ETag"),
		LastModified:      resp.Header.Get("Last-Modified"),
		ContentHashHeader: hashHeader,
	}, nil
}

// parseXContentHash extrai o SHA-256 hex de header X-Content-Hash no formato
// "SHA-256 {64-hex}". Manual §6.1.1: "X-Content-Hash: SHA-256 {hash_arquivo}".
//
// Retorna (hash, nil) em sucesso ou ("", err) se header malformado. Erro
// é wrappeado pelo caller como ErrContentHashHeaderMalformed.
func parseXContentHash(header string) (string, error) {
	header = strings.TrimSpace(header)
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("formato esperado 'SHA-256 <hash>', got %q", header)
	}
	algo := strings.TrimSpace(parts[0])
	hash := strings.TrimSpace(parts[1])
	if !strings.EqualFold(algo, "SHA-256") {
		return "", fmt.Errorf("algoritmo esperado SHA-256, got %q", algo)
	}
	// SHA-256 hex = 64 chars [0-9a-fA-F].
	if len(hash) != 64 {
		return "", fmt.Errorf("hash esperado 64 chars hex, got %d chars", len(hash))
	}
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", fmt.Errorf("hash contém char não-hex em %q", hash)
		}
	}
	return strings.ToLower(hash), nil
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
