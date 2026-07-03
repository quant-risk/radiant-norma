// Package sta implementa o cliente STA (Sistema de Transmissão de Arquivos) do BACEN.
//
// Em Sprint 3, temos apenas um stub que simula o envio. Sprint 4 terá:
//   - Playwright (V1) pra STA Web
//   - WS nativo (V1.5) com cert A1/A3
package sta

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// Submission representa uma submissão ao STA.
type Submission struct {
	CadocCode string `json:"cadoc_code"`
	DataBase  string `json:"data_base"`
	XML       string `json:"xml,omitempty"`       // XML cru (string) — JSON tag não decodifica como []byte (base64)
	Zip       []byte `json:"zip,omitempty"`       // ZIP binário (base64 no JSON)
	Cert      []byte `json:"-"`                   // A1 (PEM) ou nil pra A3 (token) — nunca via JSON
	CNPJ      string `json:"cnpj"`
}

// Result é o resultado de uma submissão.
type Result struct {
	ProtocolSTA string     `json:"protocol_sta"`
	Accepted    bool       `json:"accepted"`
	Rejection   *Rejection `json:"rejection,omitempty"`
}

// Rejection contém o motivo da rejeição.
type Rejection struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Client é a interface do cliente STA.
type Client interface {
	Submit(ctx context.Context, sub *Submission) (*Result, error)
}

// StubClient é a implementação mock pra testes e desenvolvimento.
// Em produção, será substituída por WebClient (Playwright) ou WSClient.
type StubClient struct {
	AlwaysAccept bool
}

// NewStubClient cria um cliente mock.
func NewStubClient() *StubClient {
	return &StubClient{AlwaysAccept: true}
}

// Submit simula o envio. Calcula hash, gera protocolo fake, retorna aceito.
func (c *StubClient) Submit(ctx context.Context, sub *Submission) (*Result, error) {
	if sub.Zip == nil && len(sub.XML) == 0 {
		return nil, errors.New("submission sem XML nem ZIP")
	}

	payload := sub.Zip
	if payload == nil {
		payload = []byte(sub.XML)
	}
	hash := sha256.Sum256(payload)
	hashHex := hex.EncodeToString(hash[:8])

	// Gera protocolo STA fake (até 18 dígitos numéricos, conforme BACEN)
	proto := fmt.Sprintf("2026%02d%02d%05d%s",
		time.Now().Month(), time.Now().Day(),
		time.Now().Second()*1000+int(time.Now().UnixMilli()%1000),
		hashHex[:8],
	)

	if c.AlwaysAccept {
		return &Result{
			ProtocolSTA: proto,
			Accepted:    true,
		}, nil
	}
	return &Result{
		ProtocolSTA: proto,
		Accepted:    false,
		Rejection: &Rejection{
			Code:    "T01",
			Message: "dataHoraRemessa anterior a dataSaldoDevedor (stub)",
		},
	}, nil
}