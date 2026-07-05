// Package senhaws — cliente para o serviço senhaws do BACEN STA (manual v1.5 §9).
//
// Endpoint separado do STA WS (URLs diferentes: www9.bcb.gov.br/senhaws vs
// sta-h.bcb.gov.br/staws). Propósito: gerenciar credenciais Sisbacen
// programaticamente — alterar senha + consultar vencimento.
//
// Caso de uso típico:
//  1. Admin IF agenda cron job diário.
//  2. Cron chama ConsultarVencimento. Se < 7 dias, chama AlterarSenha
//     com nova senha random.
//  3. Cron atualiza secret manager (env var / vault / AWS Secrets Manager).
//  4. Próxima call STA usa senha nova automaticamente.
//
// Referência: SPRINT_23_RESEARCH.md.
package senhaws

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SenhawsConfig configura o SenhawsClient.
type SenhawsConfig struct {
	// BaseURL é a URL base do senhaws BACEN.
	// Exemplo: "https://www9.bcb.gov.br/senhaws" (homologação)
	//          "https://www3.bcb.gov.br/senhaws" (produção)
	BaseURL string

	// User é o usuário Sisbacen no formato "UUUUUDDDD.operador".
	// Exemplo: "12345/0001.fulano" ou "123450001.fulano".
	User string

	// Password é a senha Sisbacen ATUAL.
	//
	// SPRINT 13 — F13.8: NÃO logar esta struct. Logger emite SafeError.
	// Caller é responsável por atualizar secret manager após AlterarSenha.
	Password string

	// Timeout é o timeout total para cada chamada HTTP.
	// Default 30 segundos.
	Timeout time.Duration

	// HTTPClient é opcional. Se nil, usa http.DefaultClient com timeout
	// acima + TLS 1.2 mínimo + HTTP/1.1 only (mesmo padrão WSClient).
	HTTPClient *http.Client

	// AllowInsecureHTTP desabilita a checagem estrita de HTTPS — útil
	// para testes com httptest.NewServer (que retorna http://127.0.0.1:port).
	// Default false. NUNCA setar true em produção.
	AllowInsecureHTTP bool

	// Logger opcional. Default slog.Default().
	Logger *slog.Logger
}

// SenhawsClient é o cliente para o serviço senhaws do BACEN.
//
// Thread-safe: cfg é read-only após construção. Não há mutex porque
// todas as calls são read-only na config. Caller que rotaciona senha
// concorrentemente com calls STA ativas precisa de mutex externo.
type SenhawsClient struct {
	cfg    SenhawsConfig
	logger *slog.Logger
}

// sisbacenUserRegex é o mesmo padrão do WSClient (validação 39).
// Aceita tanto formato concatenado (123450001.fulano) quanto com slash (12345/0001.fulano).
var sisbacenUserRegex = regexp.MustCompile(`^(\d{5}\d{4}|\d{5}/\d{4})\.[A-Za-z0-9_-]+$`)

// NewSenhawsClient valida config e constrói o cliente. Retorna erro descritivo
// se cfg inválida (sem fazer call de rede).
func NewSenhawsClient(cfg SenhawsConfig) (*SenhawsClient, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("SenhawsConfig.BaseURL requerida (ex.: https://www9.bcb.gov.br/senhaws)")
	}
	if !cfg.AllowInsecureHTTP && !strings.HasPrefix(cfg.BaseURL, "https://") {
		return nil, fmt.Errorf("SenhawsConfig.BaseURL deve usar HTTPS (got %q; use AllowInsecureHTTP=true para testes dev)", cfg.BaseURL)
	}
	if strings.HasSuffix(cfg.BaseURL, "/") {
		return nil, errors.New("SenhawsConfig.BaseURL não deve terminar com /")
	}
	if cfg.User == "" {
		return nil, errors.New("SenhawsConfig.User requerida (formato UUUUUDDDD.operador)")
	}
	if !sisbacenUserRegex.MatchString(cfg.User) {
		return nil, fmt.Errorf("SenhawsConfig.User formato Sisbacen inválido (got %q)", cfg.User)
	}
	if cfg.Password == "" {
		return nil, errors.New("SenhawsConfig.Password requerida")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HTTPClient == nil {
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
	return &SenhawsClient{cfg: cfg, logger: logger}, nil
}

// basicAuthHeader monta Authorization: Basic base64(user:pass).
// F13.8: NÃO logar.
func (c *SenhawsClient) basicAuthHeader() string {
	creds := c.cfg.User + ":" + c.cfg.Password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

// AlterarSenha rotaciona senha Sisbacen no BACEN (manual §9.1).
//
// Endpoint: PUT /senha com body XML <Parametros>.<Senha> + <NovaSenha> +
// <ConfirmacaoNovaSenha>. Content-Type application/xml (manual linha 1121).
//
// IMPORTANTE: após sucesso, cfg.Password está desatualizado. Caller DEVE
// atualizar secret manager antes da próxima call STA, senão todas as calls
// STA retornam 401.
//
// novaSenha: nova senha. Validado 8-128 chars client-side.
//
// Retorna:
//   - nil em sucesso (204 No Content).
//   - *SenhaError em rejeição formal BACEN.
//   - err em transporte / validação client-side.
func (c *SenhawsClient) AlterarSenha(ctx context.Context, novaSenha string) error {
	if novaSenha == "" {
		return errors.New("novaSenha não pode ser vazia")
	}
	if len(novaSenha) < 8 {
		return errors.New("novaSenha deve ter no mínimo 8 chars")
	}
	if len(novaSenha) > 128 {
		return errors.New("novaSenha deve ter no máximo 128 chars")
	}
	if novaSenha == c.cfg.Password {
		return errors.New("novaSenha deve ser diferente da senha atual")
	}

	params := senhaParams{
		Senha:                c.cfg.Password,
		NovaSenha:            novaSenha,
		ConfirmacaoNovaSenha: novaSenha,
	}
	body, err := xml.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal XML request: %w", err)
	}
	body = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" + string(body))

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.cfg.BaseURL+"/senha", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Accept", "application/xml")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.StatusCode != http.StatusNoContent {
		return parseSenhaError(resp.StatusCode, respBody)
	}
	return nil
}

// ConsultarVencimento retorna dias restantes até vencimento da senha
// Sisbacen (manual §9.2).
//
// Endpoint: GET /senha/vencimento. Response 200 + XML <Resultado>.
//
// Retorna:
//   - dias (>= 0) em sucesso.
//   - 0 + *SenhaError em rejeição formal BACEN.
//   - 0 + err em transporte.
func (c *SenhawsClient) ConsultarVencimento(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.cfg.BaseURL+"/senha/vencimento", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.StatusCode != http.StatusOK {
		return 0, parseSenhaError(resp.StatusCode, respBody)
	}

	var v vencimentoResponse
	if err := xml.Unmarshal(respBody, &v); err != nil {
		return 0, fmt.Errorf("parse vencimento XML: %w (body=%s)", err, truncateSenha(respBody, 200))
	}
	if v.DiasVencimentoSenha == "" {
		return 0, errors.New("BACEN retornou 200 mas <DiasVencimentoSenha> vazio")
	}
	dias, err := strconv.Atoi(strings.TrimSpace(v.DiasVencimentoSenha))
	if err != nil {
		return 0, fmt.Errorf("DiasVencimentoSenha não é inteiro válido (got %q)", v.DiasVencimentoSenha)
	}
	if dias < 0 {
		return 0, fmt.Errorf("DiasVencimentoSenha negativo (got %d)", dias)
	}
	return dias, nil
}

// GerarSenhaRandom gera senha aleatória de 16 chars hex (32 hex chars).
// Helper opcional para callers que querem rotação automática.
//
// Não usa crypto/rand (determinismo de testes é importante — caller pode
// passar senha custom). Para produção, caller deve usar crypto/rand.
func GerarSenhaRandom() string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, 16)
	for i := range b {
		b[i] = hexChars[rand.Intn(len(hexChars))]
	}
	return hex.EncodeToString(b)
}

// === tipos internos ===

type senhaParams struct {
	XMLName              struct{} `xml:"Parametros"`
	Senha                string   `xml:"Senha"`
	NovaSenha            string   `xml:"NovaSenha"`
	ConfirmacaoNovaSenha string   `xml:"ConfirmacaoNovaSenha"`
}

type vencimentoResponse struct {
	XMLName             struct{} `xml:"Resultado"`
	DiasVencimentoSenha string   `xml:"DiasVencimentoSenha"`
}

// SenhaError representa rejeição formal do senhaws BACEN (4xx/5xx com XML
// Listagem 4 — mesmo formato do STA WS).
type SenhaError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *SenhaError) Error() string {
	return fmt.Sprintf("BACEN senhaws error %d: %s", e.StatusCode, e.Message)
}

// parseSenhaError extrai erro tipado de body XML.
func parseSenhaError(status int, body []byte) error {
	type xmlError struct {
		XMLName struct{} `xml:"Resultado"`
		Erro    struct {
			Codigo    int    `xml:"Codigo"`
			Descricao string `xml:"Descricao"`
		} `xml:"Erro"`
	}
	var xe xmlError
	if err := xml.Unmarshal(body, &xe); err == nil && xe.Erro.Codigo != 0 {
		return &SenhaError{
			StatusCode: status,
			Code:       strconv.Itoa(xe.Erro.Codigo),
			Message:    xe.Erro.Descricao,
		}
	}
	return &SenhaError{
		StatusCode: status,
		Code:       fmt.Sprintf("HTTP_%d", status),
		Message:    truncateSenha(body, 200),
	}
}

// maxResponseBodyBytes limita tamanho do body de resposta (defesa em profundidade).
const maxResponseBodyBytes = 1 << 20 // 1 MiB — senhaws responses são minúsculas

// truncateSenha é helper local (similar a truncate em sta/ws.go).
func truncateSenha(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
