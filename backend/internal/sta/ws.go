// Package sta — WSClient (cliente nativo BACEN STA Web Services v1.5).
//
// Sprint 18 (v3.8.0): cliente REST contra https://sta-h.bcb.gov.br/staws (homologação)
// ou https://sta.bcb.gov.br/staws (produção).
//
// Sprint 19 (v3.9.0): read side — WSClient.StatusUpload (Seção 5.3.1) e
// WSClient.Download (Seção 6.1.1). Validação X-Content-Hash obrigatória.
//
// Sprint 20 (v3.10.0): listagem / disponiveis (Seção 8.1.1) + alteração /situacao
// (Seção 7.1). ReadClient interface segregation permite handlers checarem
// capability em runtime (StubClient não implementa read side).
//
// Sprint 21 (v3.11.0): chunked transfer — WSClient.SubmitRange (Seção 5.6) +
// WSClient.DownloadRange (Seção 6.4). ChunkedClient interface segregation
// (StubClient não implementa — capability de range/chunk).
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
//  5. GET /arquivos/disponiveis?dataHoraInicio=... → listagem de arquivos a receber
//  6. PUT /arquivos/situacao → alterar A_REC ↔ REC
//
// E chunked transfer (Sprint 21):
//  7. PUT /arquivos/{protocolo}/conteudo com Content-Range → chunk upload
//  8. GET /arquivos/{protocolo}/conteudo com Range + If-Match → chunk download
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
//   - Compara SHA-256 do body do Download/DownloadRange com X-Content-Hash
//     do header (Seção 6.1.1 / 6.4, validação obrigatória de integridade —
//     manual linha 641-643)
//   - Respeita timeout configurado
//   - Sanitiza error bodies (não vaza err.Error() cru — F18.1 defense)
//   - Emite audit emission em failure (S17.6 pattern — mas ainda
//     sem audit_log wiring aqui; feito no handler do /v1/sta/submit)
//
// Limitações V1 (cobertas em Sprint 22+):
//   - Sem retry exponencial
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
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
//
// Sprint 36 — v3.36.0: OTel tracing instrumentation em todas as operações.
type WSClient struct {
	cfg    WSConfig
	logger *slog.Logger
}

// getTracer returns the global OTel tracer for BACEN STA operations.
// OTel's TracerProvider caches tracers internally, so calling otel.Tracer
// on every use is cheap and thread-safe.
func getTracer() trace.Tracer {
	return otel.Tracer("bacen-sta",
		trace.WithInstrumentationVersion("1.0"),
	)
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
//   - (*Result{ProtocolSTA: "X", Accepted: true}, nil) em sucesso.
//   - (*Result{ProtocolSTA: "X", Accepted: false, Rejection: &Rejection{...}}, nil)
//     em rejeição conhecida do BACEN. ProtocolSTA é preservado mesmo quando
//     o upload falha (fase 1 OK, fase 2 falhou) — útil pra forensic trail.
//   - (nil, err) em erro de transporte (timeout, malformed XML, malformed config).
//
// Audit emission é deferido para o handler HTTP (Sprint 8c / Sprint 20+).
// WSClient emite logs estruturados aqui (N1.4-debug), mas não emite audit_log
// diretamente (single responsibility: cliente só fala com BACEN, handler
// decide o que auditar).
func (c *WSClient) Submit(ctx context.Context, sub *Submission) (*Result, error) {
	ctx, span := getTracer().Start(ctx, "sta.Submit",
		trace.WithAttributes(
			attribute.String("sta.cadoc", sub.CadocCode),
			attribute.String("sta.if_id", sub.CNPJ),
			attribute.Int64("sta.payload_bytes", int64(len(sub.XML))),
		))
	defer span.End()

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "POST /arquivos failed")
		return nil, fmt.Errorf("STA POST /arquivos falhou: %w", err)
	}
	if protocolo == "" {
		// Não deveria acontecer — requestProtocol retorna erro antes de
		// string vazia. Defensive log.
		c.logger.Warn("STA requestProtocol retornou protocolo vazio sem erro",
			"if_id", sub.CNPJ, "cadoc", sub.CadocCode)
		return &Result{ProtocolSTA: "", Accepted: false, Rejection: &Rejection{Code: "INTERNAL", Message: "protocolo vazio"}}, nil
	}

	span.SetAttributes(attribute.String("sta.protocolo", protocolo))

	// Phase 2: PUT /arquivos/{protocolo}/conteudo (binário).
	if err := c.uploadContent(ctx, protocolo, payload); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "PUT conteudo failed")
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

	span.SetStatus(codes.Ok, "")

	observability.AddBreadcrumb(ctx, "sta.submit",
		fmt.Sprintf("STA submission %s succeeded, protocolo=%s", sub.CadocCode, protocolo),
		map[string]any{
			"cadoc":     sub.CadocCode,
			"protocolo": protocolo,
			"bytes":     len(payload),
		})

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

// parseSTAError mapeia resposta de erro XML do BACEN para *STAError tipado.
//
// O BACEN sempre responde 4xx/5xx com body XML no formato:
//
//	<Resultado><Erro><Codigo>{STATUS}</Codigo><Descricao>{MSG}</Descricao></Erro></Resultado>
//
// (Listagem 4 do manual)
//
// Validação 42 (Sprint 22) finding F-S22-1: RetryingClient precisa fazer
// errors.As(err, &staErr) para classificar 5xx vs 4xx. parseSTAError
// anterior retornava fmt.Errorf opaco — quebrava o wrapping. Agora retorna
// *STAError direto, wrappeado pelo caller (Submit) com %w para preservar
// a cadeia de erros original.
//
// Caller-side use:
//   - errors.As(err, &staErr) para inspecionar status code tipado
//   - err.Error() para mensagem legível (formato preservado)
func (c *WSClient) parseSTAError(status int, body []byte) error {
	var xe xmlError
	if err := xml.Unmarshal(body, &xe); err == nil && xe.Erro.Codigo != 0 {
		return &STAError{
			StatusCode: status,
			Code:       strconv.Itoa(xe.Erro.Codigo),
			Message:    xe.Erro.Descricao,
		}
	}
	// Body não parseou — retorna STAError com body cru truncado.
	return &STAError{
		StatusCode: status,
		Code:       fmt.Sprintf("HTTP_%d", status),
		Message:    truncate(body, 200),
	}
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
	ctx, span := getTracer().Start(ctx, "sta.StatusUpload",
		trace.WithAttributes(
			attribute.String("sta.protocolo", protocolo),
		))
	defer span.End()

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "HTTP request failed")
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.StatusCode != http.StatusOK {
		err := c.parseSTAErrorTyped(resp.StatusCode, respBody, protocolo)
		span.RecordError(err)
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		return nil, err
	}

	var p posicaoUploadResponse
	if err := xml.Unmarshal(respBody, &p); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "XML parse failed")
		return nil, fmt.Errorf("parse posicaoupload XML: %w (body=%s)", err, truncate(respBody, 200))
	}

	span.SetStatus(codes.Ok, "")
	return &UploadStatus{
		Protocolo:       p.Protocolo,
		RangesRecebidos: parseRanges(p.RangesRecebidos),
		Situacao:        parseUploadSituacao(p.Situacao),
		SituacaoRaw:     p.Situacao,
	}, nil
}

// InitRangeSession implementa RangeUploader.InitRangeSession — Sprint 31.
//
// POST /arquivos com os metadados do arquivo (não o conteúdo) para obter
// um protocolo de upload chunkado (Seção 5.6 do manual BACEN).
//
// cadocCode: IdentificadorDocumento (ex: "3040").
// hashHex: SHA-256 hex do arquivo completo (pode ser vazio se desconhecido).
// totalBytes: tamanho total do arquivo em bytes (pode ser 0 se desconhecido).
//
// Retorna o protocolo BACEN em sucesso.
func (c *WSClient) InitRangeSession(ctx context.Context, cadocCode, hashHex string, totalBytes int64) (string, error) {
	ctx, span := getTracer().Start(ctx, "sta.InitRangeSession",
		trace.WithAttributes(
			attribute.String("sta.cadoc", cadocCode),
			attribute.Int64("sta.total_bytes", totalBytes),
		))
	defer span.End()

	params := requestProtocolParams{
		IdentificadorDocumento: cadocCode,
		Hash:                   hashHex,
		Tamanho:                totalBytes,
		NomeArquivo:            fmt.Sprintf("%s.zip", cadocCode),
	}
	body, err := xml.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("marshal XML: %w", err)
	}
	body = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" + string(body))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/arquivos", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Accept", "application/xml")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "HTTP request failed")
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.StatusCode != http.StatusCreated {
		err := c.parseSTAError(resp.StatusCode, respBody)
		span.RecordError(err)
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		return "", err
	}

	var rp responseProtocol
	if err := xml.Unmarshal(respBody, &rp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "XML parse failed")
		return "", fmt.Errorf("parse XML response: %w", err)
	}
	if rp.Protocolo == "" {
		err := fmt.Errorf("BACEN retornou 201 mas protocolo vazio")
		span.RecordError(err)
		span.SetStatus(codes.Error, "empty protocolo")
		return "", err
	}

	span.SetAttributes(attribute.String("sta.protocolo", rp.Protocolo))
	span.SetStatus(codes.Ok, "")
	return rp.Protocolo, nil
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
	ctx, span := getTracer().Start(ctx, "sta.Download",
		trace.WithAttributes(
			attribute.String("sta.protocolo", protocolo),
		))
	defer span.End()

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "HTTP request failed")
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Lê body com cap (defense in depth). Se exceder maxDownloadBodyBytes,
	// retornamos *STAError 413 sem alocar 500 MB indevidamente.
	limited := io.LimitReader(resp.Body, maxDownloadBodyBytes+1) // +1 pra detectar overflow
	body, err := io.ReadAll(limited)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "read body failed")
		return nil, fmt.Errorf("read download body: %w", err)
	}
	if int64(len(body)) > maxDownloadBodyBytes {
		err := &STAError{
			StatusCode: http.StatusRequestEntityTooLarge,
			Code:       fmt.Sprintf("HTTP_%d", http.StatusRequestEntityTooLarge),
			Message:    fmt.Sprintf("body excede %d bytes (cap defensivo)", maxDownloadBodyBytes),
			Protocolo:  protocolo,
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "body too large")
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err := c.parseSTAErrorTyped(resp.StatusCode, body, protocolo)
		span.RecordError(err)
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		return nil, err
	}

	// Validação X-Content-Hash — obrigatória conforme manual §6.1.1.
	hashHeader := resp.Header.Get("X-Content-Hash")
	if hashHeader == "" {
		err := &STAError{
			StatusCode: http.StatusBadGateway,
			Code:       "MISSING_X_CONTENT_HASH",
			Message:    "BACEN não retornou header X-Content-Hash (esperado conforme manual §6.1.1)",
			Protocolo:  protocolo,
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "missing X-Content-Hash")
		return nil, err
	}
	wantHash, malformedErr := parseXContentHash(hashHeader)
	if malformedErr != nil {
		err := fmt.Errorf("%w: %v (header=%q)", ErrContentHashHeaderMalformed, malformedErr, hashHeader)
		span.RecordError(err)
		span.SetStatus(codes.Error, "malformed X-Content-Hash")
		return nil, err
	}

	// SHA-256 do body (Seção 2.4 do manual — mesmo algoritmo do upload).
	sum := sha256.Sum256(body)
	gotHash := hex.EncodeToString(sum[:])

	if gotHash != wantHash {
		err := fmt.Errorf("%w: esperado=%s got=%s (body=%d bytes)",
			ErrContentHashMismatch, wantHash, gotHash, len(body))
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash mismatch")
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("sta.content_bytes", int64(len(body))),
		attribute.String("sta.content_hash", gotHash),
	)
	span.SetStatus(codes.Ok, "")
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
//
// Validação de hex usa encoding/hex.DecodeString (stdlib) — defesa em
// profundidade contra BACEN bugado mandar chars não-hex.
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
	// SHA-256 hex = 64 chars [0-9a-fA-F]. hex.DecodeString valida ambos
	// (comprimento + chars) e retorna erro descritivo em input inválido.
	if _, err := hex.DecodeString(hash); err != nil {
		return "", fmt.Errorf("hash não-hex (%d chars): %v", len(hash), err)
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

// ============================================================
// Sprint 20 (v3.10.0) — ReadClient interface segregation
// ============================================================

// ReadClient é o subset de operações de leitura do BACEN STA WS. Apenas
// *WSClient implementa (porque StubClient não tem como listar/alterar contra
// BACEN real — seria hollow stub).
//
// Handler pattern: type-assert s.STAClient.(ReadClient). Se ok, usa; senão
// retorna 503 com mensagem clara (backend=stub não suporta read side).
//
// Interface segregation (vs estender Client interface): forçar StubClient
// a implementar ListDisponiveis/AlterarSituacao com zero-values seria
// hollow stub piorado — caller acharia que funcionou mas BACEN nunca foi
// chamado. Falhar explícito é melhor que mentir.
type ReadClient interface {
	ListDisponiveis(ctx context.Context, opts ListDisponiveisOpts) (*ListDisponiveisResult, error)
	AlterarSituacao(ctx context.Context, req AlterarSituacaoReq) error
}

// ListDisponiveis consulta arquivos que BACEN disponibilizou a partir de uma
// data-hora (Seção 8.1.1 do manual).
//
// Endpoint: GET /arquivos/disponiveis?dataHoraInicio={YYYY-MM-DDTHH:MM:SS.SSS}&...
// Parâmetros opcionais: dependencia, identificadorDocumento, sistemas.
//
// Paginação: até 1000 protocolos por chamada. Se >1000, BACEN retorna
// <atom:link href="..." rel="disponiveis"/> com URL da próxima página —
// exposta em ListDisponiveisResult.ProximaPaginaURL.
//
// Caller polling pattern: usar DataHoraProximaConsulta no próximo call.
// Caller pagination pattern: usar ProximaPaginaURL no próximo call.
//
// Retorna:
//   - (*ListDisponiveisResult, nil) em sucesso — mesmo se lista vazia.
//   - (nil, *STAError) em rejeição formal BACEN (400).
//   - (nil, err) em erro de transporte (rede, timeout, parse falho).
func (c *WSClient) ListDisponiveis(ctx context.Context, opts ListDisponiveisOpts) (*ListDisponiveisResult, error) {
	ctx, span := getTracer().Start(ctx, "sta.ListDisponiveis",
		trace.WithAttributes(
			attribute.String("sta.data_hora_inicio", opts.DataHoraInicio),
			attribute.String("sta.identificador_documento", opts.IdentificadorDocumento),
		))
	defer span.End()

	if opts.DataHoraInicio == "" {
		return nil, errors.New("DataHoraInicio obrigatório (Tabela 4 do manual §8.1.1)")
	}

	// Constrói query string com URL encoding manual (manual linha 874).
	q := url.Values{}
	q.Set("dataHoraInicio", opts.DataHoraInicio)
	if opts.IdentificadorDocumento != "" {
		q.Set("identificadorDocumento", opts.IdentificadorDocumento)
	}
	if opts.Sistemas != "" {
		q.Set("sistemas", opts.Sistemas)
	}
	if opts.Dependencia != "" {
		q.Set("dependencia", opts.Dependencia)
	}
	endpoint := c.cfg.BaseURL + "/arquivos/disponiveis?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	// Content-Type intencionalmente omitido — manual §8.1.1 linha 878.

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "HTTP request failed")
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.StatusCode != http.StatusOK {
		err := c.parseSTAErrorTyped(resp.StatusCode, respBody, "")
		span.RecordError(err)
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		return nil, err
	}

	var p arquivosDisponiveisResponse
	if err := xml.Unmarshal(respBody, &p); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "XML parse failed")
		return nil, fmt.Errorf("parse disponiveis XML: %w (body=%s)", err, truncate(respBody, 200))
	}

	span.SetAttributes(attribute.Int("sta.arquivos_count", len(p.Arquivos)))
	span.SetStatus(codes.Ok, "")

	result := &ListDisponiveisResult{
		DataHoraProximaConsulta: p.DataHoraProximaConsulta,
		ProximaPaginaURL:        p.Link.HRef,
		TemProximaPagina:        p.Link.HRef != "",
		Arquivos:                make([]ArquivoDisponivel, 0, len(p.Arquivos)),
	}
	for _, a := range p.Arquivos {
		result.Arquivos = append(result.Arquivos, ArquivoDisponivel{
			Protocolo:                a.Protocolo,
			TipoArquivo:              a.TipoArquivo,
			CodigoDocumento:          a.CodigoDocumento,
			Sistema:                  a.Sistema,
			TamanhoArquivo:           a.TamanhoArquivo,
			Hash:                     a.Hash,
			SituacaoAtual:            parseSituacaoArquivo(a.SituacaoAtual.Codigo),
			SituacaoAtualRaw:         a.SituacaoAtual.Descricao,
			DataHoraDisponibilizacao: a.DataHoraDisponibilizacao,
		})
	}
	return result, nil
}

// AlterarSituacao muda a situação de N protocolos para A_REC (a receber) ou
// REC (recebido) — Seção 7.1 do manual.
//
// Endpoint: PUT /arquivos/situacao com body XML <Parametros>.
//
// Content-Type OBRIGATÓRIO "application/xml" (manual linha 792) — único
// endpoint do manual que exige Content-Type. Diferente dos outros que dizem
// "não enviar".
//
// BACEN responde 204 No Content em sucesso (sem body).
//
// Validações client-side:
//   - Protocolos vazio → erro imediato.
//   - Situacao fora de {A_REC, REC} → erro imediato.
//
// Retorna:
//   - (nil) em sucesso.
//   - (*STAError) em rejeição formal BACEN (400 com XML Listagem 4).
//   - err opaco em erro de transporte.
func (c *WSClient) AlterarSituacao(ctx context.Context, req AlterarSituacaoReq) error {
	ctx, span := getTracer().Start(ctx, "sta.AlterarSituacao",
		trace.WithAttributes(
			attribute.Int("sta.protocolos_count", len(req.Protocolos)),
			attribute.String("sta.situacao", req.Situacao.String()),
		))
	defer span.End()

	if len(req.Protocolos) == 0 {
		return errors.New("Protocolos não pode ser vazio (Seção 7.1)")
	}
	situacaoStr := req.Situacao.String()
	if situacaoStr == "" {
		return fmt.Errorf("Situacao inválida (esperado A_REC|REC, got %v)", req.Situacao)
	}

	params := situacaoParams{
		Protocolos: strings.Join(req.Protocolos, ";"),
		Situacao:   situacaoStr,
	}
	body, err := xml.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal XML request: %w", err)
	}
	body = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" + string(body))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.cfg.BaseURL+"/arquivos/situacao", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", c.basicAuthHeader())
	httpReq.Header.Set("Content-Type", "application/xml")
	httpReq.Header.Set("Accept", "application/xml")

	resp, err := c.cfg.HTTPClient.Do(httpReq)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "HTTP request failed")
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.StatusCode != http.StatusNoContent {
		err := c.parseSTAErrorTyped(resp.StatusCode, respBody, "")
		span.RecordError(err)
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// ============================================================
// Sprint 21 (v3.11.0) — chunked transfer (range upload + download)
// ============================================================

// ChunkedClient é o subset de operações chunked (range-based) do BACEN STA WS.
// Apenas *WSClient implementa (porque StubClient não tem como fazer chunked
// contra BACEN real — seria hollow stub).
//
// Operações chunked:
//   - SubmitRange: PUT com Content-Range (manual §5.6)
//   - DownloadRange: GET com Range + opcional If-Match/If-Unmodified-Since
//     (manual §6.4)
//
// Caller pattern: type-assert s.STAClient.(ChunkedClient). Se ok, usa.
// Senão retorna erro explícito ("chunked transfer não disponível neste backend").
//
// Interface segregation (mesmo padrão de ReadClient da Sprint 20): forçar
// StubClient a implementar com zero-values seria hollow stub piorado.
type ChunkedClient interface {
	SubmitRange(ctx context.Context, protocolo string, inicio, fim, total int64, chunk []byte) error
	DownloadRange(ctx context.Context, protocolo string, inicio, fim int64, expectedTotalHash, ifMatch, ifUnmodifiedSince string) (*DownloadResult, error)
}

// RangeUploader é a interface estendida para RangeUploadAPI (Sprint 31).
// Inclui InitRangeSession + StatusUpload além de SubmitRange.
type RangeUploader interface {
	ChunkedClient
	// InitRangeSession pede um protocolo ao BACEN (POST /arquivos) e retorna
	// o protocolo gerado. Opcionalmente recebe hash e tamanho total do arquivo.
	InitRangeSession(ctx context.Context, cadocCode, hashHex string, totalBytes int64) (protocolo string, err error)
	// StatusUpload retorna o status de upload de um protocolo (RangesRecebidos).
	StatusUpload(ctx context.Context, protocolo string) (*UploadStatus, error)
}

// SubmitRange envia 1 chunk de arquivo (parte de upload chunked) — Seção 5.6
// do manual BACEN.
//
// Endpoint: PUT /arquivos/{protocolo}/conteudo com Content-Range header.
// protocolo: obtido via POST /arquivos (Fase 1 — Submit).
// inicio, fim: byte range [inicio, fim] **inclusivo** (RFC 7233 §2.1).
// total: tamanho total do arquivo completo (>= fim+1).
// chunk: bytes do chunk (esperado len(chunk) == fim-inicio+1).
//
// Content-Type omitido (manual §5.6 linha 538-539 — mesmo que §5.2 single-call).
//
// Retorna:
//   - nil em sucesso (BACEN responde 200 OK).
//   - *STAError em rejeição formal BACEN (400/403/404/410/416/501).
//   - err em transporte ou validação client-side.
//
// Validações client-side (defense in depth — BACEN também valida 416):
//   - protocolo não vazio
//   - inicio >= 0
//   - fim >= inicio
//   - total > 0 e total >= fim+1
//   - len(chunk) == fim-inicio+1 (chunks devem ter tamanho exato)
func (c *WSClient) SubmitRange(ctx context.Context, protocolo string, inicio, fim, total int64, chunk []byte) error {
	ctx, span := getTracer().Start(ctx, "sta.SubmitRange",
		trace.WithAttributes(
			attribute.String("sta.protocolo", protocolo),
			attribute.Int64("sta.range_start", inicio),
			attribute.Int64("sta.range_end", fim),
			attribute.Int64("sta.total_bytes", total),
			attribute.Int64("sta.chunk_bytes", int64(len(chunk))),
		))
	defer span.End()

	if protocolo == "" {
		return errors.New("protocolo requerido")
	}
	if inicio < 0 {
		return fmt.Errorf("inicio deve ser >= 0 (got %d)", inicio)
	}
	if fim < inicio {
		return fmt.Errorf("fim deve ser >= inicio (inicio=%d, fim=%d)", inicio, fim)
	}
	if total <= 0 || total < fim+1 {
		return fmt.Errorf("total deve ser > 0 e >= fim+1 (got total=%d, fim=%d)", total, fim)
	}
	expectedLen := fim - inicio + 1
	if int64(len(chunk)) != expectedLen {
		return fmt.Errorf("len(chunk) deve ser fim-inicio+1=%d (got %d)", expectedLen, len(chunk))
	}

	// Content-Range: bytes inicio-fim/total (RFC 7233 §4.2)
	contentRange := fmt.Sprintf("bytes %d-%d/%d", inicio, fim, total)

	url := c.cfg.BaseURL + "/arquivos/" + protocolo + "/conteudo"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(chunk))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Content-Range", contentRange)
	req.ContentLength = expectedLen
	// Content-Type intencionalmente omitido — manual §5.6 linha 538-539.

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "HTTP request failed")
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.StatusCode != http.StatusOK {
		err := c.parseSTAErrorTyped(resp.StatusCode, respBody, protocolo)
		span.RecordError(err)
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// DownloadRange baixa 1 chunk de arquivo (parte de download chunked) — Seção
// 6.4 do manual BACEN.
//
// Endpoint: GET /arquivos/{protocolo}/conteudo com Range header.
// protocolo: protocolo do arquivo.
// inicio, fim: byte range [inicio, fim] **inclusivo** (RFC 7233).
// expectedTotalHash: SHA-256 hex do arquivo COMPLETO (de ListDisponiveis.Hash).
//
//	Se vazio, sem validação de hash (cliente confia no BACEN).
//
// ifMatch: valor do header If-Match (opcional, RFC 7232).
// ifUnmodifiedSince: valor do header If-Unmodified-Since (opcional, RFC 7232).
//
// Content-Type omitido (manual §6.4 linha 701).
//
// **Detalhe crítico:** X-Content-Hash do BACEN é do **arquivo completo**, não
// do chunk. Cliente valida comparando contra expectedTotalHash. Se match, chunk
// veio de arquivo íntegro. Se mismatch → ErrContentHashMismatch (sentinel da
// Sprint 19).
//
// BACEN responde 206 Partial Content (não 200 OK) em sucesso.
//
// Retorna:
//   - (*DownloadResult, nil) em sucesso — Conteudo contém apenas o chunk.
//   - (nil, *STAError) em rejeição formal BACEN (400/404/410/412/416/501).
//   - (nil, ErrContentHashMismatch) se expectedTotalHash fornecido e difere.
//   - (nil, ErrContentHashHeaderMalformed) se X-Content-Hash malformado.
//   - (nil, err) em transporte.
//
// Validações client-side:
//   - protocolo não vazio
//   - inicio >= 0, fim >= inicio
//   - (fim-inicio+1) <= maxDownloadBodyBytes (defesa DoS)
func (c *WSClient) DownloadRange(ctx context.Context, protocolo string, inicio, fim int64, expectedTotalHash, ifMatch, ifUnmodifiedSince string) (*DownloadResult, error) {
	ctx, span := getTracer().Start(ctx, "sta.DownloadRange",
		trace.WithAttributes(
			attribute.String("sta.protocolo", protocolo),
			attribute.Int64("sta.range_start", inicio),
			attribute.Int64("sta.range_end", fim),
		))
	defer span.End()

	if protocolo == "" {
		return nil, errors.New("protocolo requerido")
	}
	if inicio < 0 {
		return nil, fmt.Errorf("inicio deve ser >= 0 (got %d)", inicio)
	}
	if fim < inicio {
		return nil, fmt.Errorf("fim deve ser >= inicio (inicio=%d, fim=%d)", inicio, fim)
	}
	requestedLen := fim - inicio + 1
	if requestedLen > int64(maxDownloadBodyBytes) {
		return nil, &STAError{
			StatusCode: http.StatusRequestEntityTooLarge,
			Code:       fmt.Sprintf("HTTP_%d", http.StatusRequestEntityTooLarge),
			Message:    fmt.Sprintf("range solicitado %d bytes excede cap defensivo %d", requestedLen, maxDownloadBodyBytes),
			Protocolo:  protocolo,
		}
	}

	// Range: bytes=inicio-fim (RFC 7233 §3.1, sem /total — diferente de Content-Range)
	rangeHeader := fmt.Sprintf("bytes=%d-%d", inicio, fim)

	url := c.cfg.BaseURL + "/arquivos/" + protocolo + "/conteudo"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Range", rangeHeader)
	// If-Match e If-Unmodified-Since opcionais (manual linha 703).
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	if ifUnmodifiedSince != "" {
		req.Header.Set("If-Unmodified-Since", ifUnmodifiedSince)
	}
	// Content-Type intencionalmente omitido — manual §6.4 linha 701.

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "HTTP request failed")
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Cap no body.
	limited := io.LimitReader(resp.Body, maxDownloadBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "read body failed")
		return nil, fmt.Errorf("read download range body: %w", err)
	}
	if int64(len(body)) > maxDownloadBodyBytes {
		err := &STAError{
			StatusCode: http.StatusRequestEntityTooLarge,
			Code:       fmt.Sprintf("HTTP_%d", http.StatusRequestEntityTooLarge),
			Message:    fmt.Sprintf("body excede %d bytes (cap defensivo)", maxDownloadBodyBytes),
			Protocolo:  protocolo,
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "body too large")
		return nil, err
	}

	// 206 Partial Content esperado em sucesso. Outros 2xx viram erro.
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		err := c.parseSTAErrorTyped(resp.StatusCode, body, protocolo)
		span.RecordError(err)
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		return nil, err
	}
	// 200 OK (em vez de 206) é improvável mas tolerável — BACEN pode
	// retornar 200 com range respeitado (single-call completo). Caller
	// que recebe 200 deve checar Content-Range no response. Para V1,
	// aceitamos 200 e 206 sem distinção.

	// Validação X-Content-Hash (Sprint 19 pattern).
	hashHeader := resp.Header.Get("X-Content-Hash")
	if hashHeader == "" {
		err := &STAError{
			StatusCode: http.StatusBadGateway,
			Code:       "MISSING_X_CONTENT_HASH",
			Message:    "BACEN não retornou header X-Content-Hash (esperado conforme manual §6.4)",
			Protocolo:  protocolo,
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "missing X-Content-Hash")
		return nil, err
	}
	gotHash, malformedErr := parseXContentHash(hashHeader)
	if malformedErr != nil {
		err := fmt.Errorf("%w: %v (header=%q)", ErrContentHashHeaderMalformed, malformedErr, hashHeader)
		span.RecordError(err)
		span.SetStatus(codes.Error, "malformed X-Content-Hash")
		return nil, err
	}

	// Validação contra expectedTotalHash do caller.
	if expectedTotalHash != "" && gotHash != expectedTotalHash {
		err := fmt.Errorf("%w: esperado=%s got=%s (chunk=%d bytes)",
			ErrContentHashMismatch, expectedTotalHash, gotHash, len(body))
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash mismatch")
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("sta.chunk_bytes", int64(len(body))),
		attribute.String("sta.content_hash", gotHash),
	)
	span.SetStatus(codes.Ok, "")
	return &DownloadResult{
		Conteudo:          body,
		ContentHash:       gotHash,
		ETag:              resp.Header.Get("ETag"),
		LastModified:      resp.Header.Get("Last-Modified"),
		ContentHashHeader: hashHeader,
	}, nil
}

// Compile-time guarantees: *WSClient implementa as interfaces sta.Client,
// ReadClient e ChunkedClient. Permite interface segregation sem erro de
// compilação silencioso se assinatura de qualquer interface mudar.
//
// Validação 44 (Sprint 25 follow-up): assertions movidas de ws_test.go
// para cá (production source) — idiomático e catching imediato se método
// mudar.
var (
	_ Client        = (*WSClient)(nil)
	_ ReadClient    = (*WSClient)(nil)
	_ ChunkedClient = (*WSClient)(nil)
)
