// Doc3044 — Documento 3044 (Eventos de Operações de Crédito) — CADOC JSON.
//
// Sprint 42: parser para 3044 (formato JSON, não XML). Valida eventos de
// operações de crédito: pagamentos, concessões, cessões, aquisições.
//
// Referência: IN BCB 530/2024 (vigência novembro/2025).
package rules

import (
	"encoding/json"
	"fmt"
	"time"
)

// Doc3044 é o documento 3044 parseado (JSON).
//
// Os tags `json:` nos campos string (CNPJ, Envia3050, etc.) são usados pelo
// json.Unmarshal no ParseDoc3044 (via struct raw intermediária). Os campos
// time.Time NÃO têm tags json — o parsing é feito manualmente com time.Parse
// para suportar o formato customizado "YYYY-MM-DD HH:mm:ss".
type Doc3044 struct {
	CNPJ            string
	DataHoraRemessa time.Time
	Envia3050       string
	Operacoes       []Operacao3044
}

// Operacao3044 representa uma operação individual no 3044.
type Operacao3044 struct {
	Acao             int
	IPOC             string
	Class3050        string
	SaldoDevedor     float64
	DataSaldoDevedor time.Time
	Atraso           string
	Pagamentos       []Pagamento3044
	Concessoes       []Concessao3044
	Cessoes          []Cessao3044
	Aquisicoes       []Aquisicao3044
}

// Pagamento3044 representa um evento de pagamento.
type Pagamento3044 struct {
	Acao     int
	TpMotivo string
	Data     time.Time
	Valor    float64
}

// Concessao3044 representa um evento de concessão de crédito.
type Concessao3044 struct {
	Acao     int
	TpMotivo string
	Data     time.Time
	Valor    float64
}

// Cessao3044 representa um evento de cessão de operação.
type Cessao3044 struct {
	Acao          int
	Data          time.Time
	CdCessionario string
	Valor         float64
}

// Aquisicao3044 representa um evento de aquisição de operação.
type Aquisicao3044 struct {
	Acao      int
	Data      time.Time
	CdCedente string
	Valor     float64
}

// PartialParseError3044 indica parse parcial bem-sucedido (D-26 pattern).
type PartialParseError3044 struct {
	Err error
}

func (e *PartialParseError3044) Error() string { return "parse 3044: " + e.Err.Error() }
func (e *PartialParseError3044) Unwrap() error { return e.Err }

// ParseDoc3044 faz parse do JSON 3044.
//
// Estrutura esperada:
//
//	{
//	  "cnpjIF": "12345678",
//	  "dataHoraRemessa": "2026-07-03 14:30:00",
//	  "envia3050": "S",
//	  "operacoes": [
//	    {
//	      "acao": 1,
//	      "ipoc": "876543210216210020716C1234",
//	      "class3050": "112212101",
//	      "saldoDevedor": 5000.00,
//	      "dataSaldoDevedor": "2026-07-03",
//	      "atraso": "N",
//	      "pagamentos": [...],
//	      "concessoes": [...],
//	      "cessoes": [...],
//	      "aquisicoes": [...]
//	    }
//	  ]
//	}
//
// Usa custom time formats para campos de data/hora.
func ParseDoc3044(data []byte) (*Doc3044, error) {
	doc := &Doc3044{}

	// Mapa de decoders parcial para capturar campos com formato de data customizado.
	var raw struct {
		CNPJ            string `json:"cnpjIF"`
		DataHoraRemessa string `json:"dataHoraRemessa"`
		Envia3050       string `json:"envia3050"`
		Operacoes       []struct {
			Acao             int     `json:"acao"`
			IPOC             string  `json:"ipoc"`
			Class3050        string  `json:"class3050"`
			SaldoDevedor     float64 `json:"saldoDevedor"`
			DataSaldoDevedor string  `json:"dataSaldoDevedor"`
			Atraso           string  `json:"atraso"`
			Pagamentos       []struct {
				Acao     int     `json:"acao"`
				TpMotivo string  `json:"tpMotivo,omitempty"`
				Data     string  `json:"data"`
				Valor    float64 `json:"valor"`
			} `json:"pagamentos"`
			Concessoes []struct {
				Acao     int     `json:"acao"`
				TpMotivo string  `json:"tpMotivo,omitempty"`
				Data     string  `json:"data"`
				Valor    float64 `json:"valor"`
			} `json:"concessoes"`
			Cessoes []struct {
				Acao          int     `json:"acao"`
				Data          string  `json:"data"`
				CdCessionario string  `json:"cdCessionario"`
				Valor         float64 `json:"valor"`
			} `json:"cessoes"`
			Aquisicoes []struct {
				Acao      int     `json:"acao"`
				Data      string  `json:"data"`
				CdCedente string  `json:"cdCedente"`
				Valor     float64 `json:"valor"`
			} `json:"aquisicoes"`
		} `json:"operacoes"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return doc, &PartialParseError3044{Err: fmt.Errorf("json unmarshal: %w", err)}
	}

	doc.CNPJ = raw.CNPJ
	doc.Envia3050 = raw.Envia3050

	// Parse dataHoraRemessa (formato: "YYYY-MM-DD HH:mm:ss")
	dtr, err := time.Parse("2006-01-02 15:04:05", raw.DataHoraRemessa)
	if err != nil {
		return doc, &PartialParseError3044{Err: fmt.Errorf("dataHoraRemessa: %w", err)}
	}
	doc.DataHoraRemessa = dtr

	// Parse operações e sub-eventos
	for _, op := range raw.Operacoes {
		o := Operacao3044{
			Acao:         op.Acao,
			IPOC:         op.IPOC,
			Class3050:    op.Class3050,
			SaldoDevedor: op.SaldoDevedor,
			Atraso:       op.Atraso,
		}

		dsd, err := time.Parse("2006-01-02", op.DataSaldoDevedor)
		if err != nil {
			return doc, &PartialParseError3044{Err: fmt.Errorf("dataSaldoDevedor IPOC %s: %w", op.IPOC, err)}
		}
		o.DataSaldoDevedor = dsd

		for _, p := range op.Pagamentos {
			dataP, err := time.Parse("2006-01-02", p.Data)
			if err != nil {
				return doc, &PartialParseError3044{Err: fmt.Errorf("pagamento data IPOC %s: %w", op.IPOC, err)}
			}
			o.Pagamentos = append(o.Pagamentos, Pagamento3044{
				Acao: p.Acao, TpMotivo: p.TpMotivo, Data: dataP, Valor: p.Valor,
			})
		}

		for _, c := range op.Concessoes {
			dataC, err := time.Parse("2006-01-02", c.Data)
			if err != nil {
				return doc, &PartialParseError3044{Err: fmt.Errorf("concessao data IPOC %s: %w", op.IPOC, err)}
			}
			o.Concessoes = append(o.Concessoes, Concessao3044{
				Acao: c.Acao, TpMotivo: c.TpMotivo, Data: dataC, Valor: c.Valor,
			})
		}

		for _, cs := range op.Cessoes {
			dataCs, err := time.Parse("2006-01-02", cs.Data)
			if err != nil {
				return doc, &PartialParseError3044{Err: fmt.Errorf("cessao data IPOC %s: %w", op.IPOC, err)}
			}
			o.Cessoes = append(o.Cessoes, Cessao3044{
				Acao: cs.Acao, Data: dataCs, CdCessionario: cs.CdCessionario, Valor: cs.Valor,
			})
		}

		for _, aq := range op.Aquisicoes {
			dataAq, err := time.Parse("2006-01-02", aq.Data)
			if err != nil {
				return doc, &PartialParseError3044{Err: fmt.Errorf("aquisicao data IPOC %s: %w", op.IPOC, err)}
			}
			o.Aquisicoes = append(o.Aquisicoes, Aquisicao3044{
				Acao: aq.Acao, Data: dataAq, CdCedente: aq.CdCedente, Valor: aq.Valor,
			})
		}

		doc.Operacoes = append(doc.Operacoes, o)
	}

	return doc, nil
}
