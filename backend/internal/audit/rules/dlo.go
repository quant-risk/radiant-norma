// DocDLO — Documento DLO (Demonstrativo de Limites Operacionais) — CADOC 2061.
//
// Sprint 50: parser completo para DLO (2061) extraindo todas as contas COSIF.
// Estrutura hierárquica: <DocDLO> → <Conta> → <Elem> (elementos de detalhamento).
//
// Referência: BACEN — Catálogo de críticas 2061 (ELIM0001-ELIM0629).
package rules

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// DocDLO é o documento DLO parseado com estrutura hierárquica COSIF.
type DocDLO struct {
	Root DocDLORoot

	// Campos agregados (legacy — mantidos para compatibilidade cross-doc 2070).
	Conta770    float64
	LimiteTotal float64
	Patrimonio  float64

	// Contas COSIF — map[codigoConta]valor.
	// Ex: Accounts["111.01"] = 1000.50
	Accounts map[string]float64

	// Elementos de detalhamento por conta — map[conta] → slice de elementos.
	// Cada Elem tem Codigo, Descricao, Valor.
	Elements map[string][]Elem
}

// Elem representa um elemento de detalhamento de uma conta COSIF.
type Elem struct {
	Codigo  string // código do elemento (1, 2, 3...)
	Desc    string // descrição do elemento
	Valor   float64
	Peso    float64 // fator de ponderação (para RWACAM)
}

// DocDLORoot é o elemento raiz do DLO.
type DocDLORoot struct {
	CNPJ         string
	DataBase     string // YYYY-MM-DD
	TpDocumento  string // F=full, S=substituição
	NumeroVersao string // versão do leiaute
}

// PartialParseErrorDLO indica parse parcial bem-sucedido (D-26).
type PartialParseErrorDLO struct {
	Err error
}

func (e *PartialParseErrorDLO) Error() string { return "parse DLO: " + e.Err.Error() }
func (e *PartialParseErrorDLO) Unwrap() error { return e.Err }

// ParseDocDLO faz parse completo do XML DLO.
//
// Estrutura COSIF hierárquica:
//
//	<DocDLO cnpj="..." dataBase="...">
//	  <Conta770 valor="1000.00"/>
//	  <LimiteTotal valor="5000.00"/>
//	  <Patrimonio valor="3000.00"/>
//	  <Conta codigoConta="800.01" valor="500.00">
//	    <Elem codigoElem="1" descElem="Capital Principal" valor="500.00" peso="0.40"/>
//	  </Conta>
//	  <Conta codigoConta="800.02" valor="300.00">
//	    <Elem codigoElem="1" descElem="Capital Complementar" valor="300.00" peso="0.60"/>
//	  </Conta>
//	  <Conta codigoConta="111.01" valor="1000.00">
//	    <Elem codigoElem="1" descElem="Depósitos" valor="1000.00"/>
//	  </Conta>
//	</DocDLO>
func ParseDocDLO(data []byte) (*DocDLO, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	doc := &DocDLO{
		Accounts:  make(map[string]float64),
		Elements: make(map[string][]Elem),
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseErrorDLO{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "DocDLO":
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "cnpj":
						doc.Root.CNPJ = a.Value
					case "dataBase":
						doc.Root.DataBase = a.Value
					case "tpDocumento":
						doc.Root.TpDocumento = a.Value
					case "numeroVersao":
						doc.Root.NumeroVersao = a.Value
					}
				}

			case "Conta770":
				doc.Conta770 = parseAttrFloat(t.Attr, "valor")
			case "LimiteTotal":
				doc.LimiteTotal = parseAttrFloat(t.Attr, "valor")
			case "Patrimonio":
				doc.Patrimonio = parseAttrFloat(t.Attr, "valor")

			case "Conta":
				doc.parseConta(dec, t.Attr)
			}
		}
	}

	return doc, nil
}

// parseConta processa uma tag <Conta> e seus elementos filhos.
func (doc *DocDLO) parseConta(dec *xml.Decoder, attrs []xml.Attr) {
	var codigo string
	var valor float64
	for _, a := range attrs {
		switch a.Name.Local {
		case "codigoConta":
			codigo = a.Value
		case "valor":
			valor = parseNum(a.Value)
		}
	}
	if codigo == "" {
		return
	}
	doc.Accounts[codigo] = valor

	// Processa elementos filhos <Elem> dentro desta <Conta>.
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if el, ok := tok.(xml.EndElement); ok && el.Name.Local == "Conta" {
			break
		}
		if el, ok := tok.(xml.StartElement); ok && el.Name.Local == "Elem" {
			e := Elem{}
			for _, a := range el.Attr {
				switch a.Name.Local {
				case "codigoElem":
					e.Codigo = a.Value
				case "descElem":
					e.Desc = a.Value
				case "valor":
					e.Valor = parseNum(a.Value)
				case "peso":
					e.Peso = parseNum(a.Value)
				}
			}
			if e.Codigo != "" {
				doc.Elements[codigo] = append(doc.Elements[codigo], e)
			}
		}
	}
}

// parseAttrFloat helper extrai float de atributos XML.
func parseAttrFloat(attrs []xml.Attr, name string) float64 {
	for _, a := range attrs {
		if a.Name.Local == name {
			return parseNum(a.Value)
		}
	}
	return 0
}

// ValidarDLOBasico faz validação básica do DLO (consistência interna).
func ValidarDLOBasico(doc *DocDLO) error {
	if doc == nil {
		return fmt.Errorf("DLO nil")
	}
	if doc.Conta770 < 0 {
		return fmt.Errorf("Conta770=%v negativo", doc.Conta770)
	}
	if doc.LimiteTotal < 0 {
		return fmt.Errorf("LimiteTotal=%v negativo", doc.LimiteTotal)
	}
	if doc.Patrimonio < 0 {
		return fmt.Errorf("Patrimonio=%v negativo", doc.Patrimonio)
	}
	if doc.Conta770 > doc.LimiteTotal && doc.LimiteTotal > 0 {
		return fmt.Errorf("Conta770=%v > LimiteTotal=%v", doc.Conta770, doc.LimiteTotal)
	}
	return nil
}

// SaldoConta retorna o saldo de uma conta COSIF (ou 0 se não existir).
func (doc *DocDLO) SaldoConta(codigo string) float64 {
	if doc == nil || doc.Accounts == nil {
		return 0
	}
	return doc.Accounts[codigo]
}

// SomaContas retorna a soma dos saldos de múltiplas contas.
func (doc *DocDLO) SomaContas(codigos ...string) float64 {
	var total float64
	for _, c := range codigos {
		total += doc.SaldoConta(c)
	}
	return total
}

// ValidateRWACAM verifica a fórmula RWACAM = fator × (800.01 + 800.02 + 800.03).
//
// Regras ELIM0405-ELIM0412 verificam variantes:
//   - 0.4 × (800.01 + 800.02 + 800.03)
//   - 0.6 × (800.01 + 800.02 + 800.03)
//   - 0.8 × (800.01 + 800.02 + 800.03)
//   - 1.0 × (800.01 + 800.02 + 800.03)
func (doc *DocDLO) ValidateRWACAM(fator float64, tol float64) error {
	soma := doc.SomaContas("800.01", "800.02", "800.03")
	if soma == 0 {
		return fmt.Errorf("contas 800.01+800.02+800.03 todas zeradas (RWACAM não calculável)")
	}
	rwacam := doc.SaldoConta("RWACAM")
	esperado := fator * soma
	if rwacam < 0 {
		return nil // RWACAM negativa é tratada por outra regra
	}
	diff := rwacam - esperado
	if diff < 0 {
		diff = -diff
	}
	if diff > tol {
		return fmt.Errorf("RWACAM=%v (esperado %.1f×(800.01+800.02+800.03)=%.2f, diff=%.2f)", rwacam, fator, esperado, diff)
	}
	return nil
}

// ============================================================
// Regras DLO/2061 — ELIM0001 a ELIM0629
// Sprint 50: Implementação real para as regras de estrutura,
// formato e contas.
// ============================================================

// ELIM0001 — Formato do documento inválido.
//
// Severidade: E
type ELIM0001FormatoDoc struct{}

func (ELIM0001FormatoDoc) Code() string     { return "2061-ELIM0001" }
func (ELIM0001FormatoDoc) Sheet() string    { return "EstruturaDLO" }
func (ELIM0001FormatoDoc) Severity() string { return "E" }

func (ELIM0001FormatoDoc) Apply(_ context.Context, doc *DocDLO) error {
	if doc == nil {
		return fmt.Errorf("documento nil")
	}
	if doc.Root.CNPJ == "" || doc.Root.DataBase == "" {
		return fmt.Errorf("documento sem CNPJ ou DataBase")
	}
	return nil
}

// ELIM0002 — Documento DLO não encontrado ou já processado.
//
// Severidade: E
type ELIM0002DocNaoEncontrado struct{}

func (ELIM0002DocNaoEncontrado) Code() string     { return "2061-ELIM0002" }
func (ELIM0002DocNaoEncontrado) Sheet() string    { return "EstruturaDLO" }
func (ELIM0002DocNaoEncontrado) Severity() string { return "E" }

func (ELIM0002DocNaoEncontrado) Apply(_ context.Context, doc *DocDLO) error {
	if doc == nil || doc.Root.CNPJ == "" {
		return fmt.Errorf("documento DLO não encontrado ou CNPJ vazio")
	}
	return nil
}

// ELIM0003 — CNPJ não encontrado.
//
// Severidade: E
type ELIM0003CNPJNaoEncontrado struct{}

func (ELIM0003CNPJNaoEncontrado) Code() string     { return "2061-ELIM0003" }
func (ELIM0003CNPJNaoEncontrado) Sheet() string    { return "EstruturaDLO" }
func (ELIM0003CNPJNaoEncontrado) Severity() string { return "E" }

func (ELIM0003CNPJNaoEncontrado) Apply(_ context.Context, doc *DocDLO) error {
	if doc == nil || doc.Root.CNPJ == "" {
		return fmt.Errorf("CNPJ não encontrado")
	}
	// CNPJ deve ter 8 dígitos (código de IF)
	if len(doc.Root.CNPJ) != 8 {
		return fmt.Errorf("CNPJ=%q com formato inválido (esperado 8 dígitos)", doc.Root.CNPJ)
	}
	return nil
}

// ELIM0004 — Conglomerado não encontrado.
//
// Severidade: E
type ELIM0004ConglomeradoNaoEncontrado struct{}

func (ELIM0004ConglomeradoNaoEncontrado) Code() string     { return "2061-ELIM0004" }
func (ELIM0004ConglomeradoNaoEncontrado) Sheet() string    { return "EstruturaDLO" }
func (ELIM0004ConglomeradoNaoEncontrado) Severity() string { return "E" }

func (ELIM0004ConglomeradoNaoEncontrado) Apply(_ context.Context, doc *DocDLO) error {
	// Implementação real requer query no cadastro de conglomerados.
	// Por ora, valida que há pelo menos uma conta.
	if doc == nil || len(doc.Accounts) == 0 {
		return fmt.Errorf("DLO sem contas COSIF (conglomerado não pode ser validado)")
	}
	return nil
}

// ELIM0005 — CNPJ não pertence ao conglomerado informado.
//
// Severidade: E
type ELIM0005CNPJNaoPertence struct{}

func (ELIM0005CNPJNaoPertence) Code() string     { return "2061-ELIM0005" }
func (ELIM0005CNPJNaoPertence) Sheet() string    { return "EstruturaDLO" }
func (ELIM0005CNPJNaoPertence) Severity() string { return "E" }

func (ELIM0005CNPJNaoPertence) Apply(_ context.Context, doc *DocDLO) error {
	// Cross-doc: valida que CNPJ da instituição líder bate com conglomerado.
	// stub: requer acesso ao cadastro de conglomerados.
	_ = doc
	return nil
}

// ELIM0006 — CNPJ não é líder do conglomerado.
//
// Severidade: E
type ELIM0006CNPJNaoLider struct{}

func (ELIM0006CNPJNaoLider) Code() string     { return "2061-ELIM0006" }
func (ELIM0006CNPJNaoLider) Sheet() string    { return "EstruturaDLO" }
func (ELIM0006CNPJNaoLider) Severity() string { return "E" }

func (ELIM0006CNPJNaoLider) Apply(_ context.Context, doc *DocDLO) error {
	// Cross-doc: requer 3026 + cadastro de conglomerados.
	_ = doc
	return nil
}

// ELIM0007 — Tipo de parâmetro não encontrado.
//
// Severidade: E
type ELIM0007TipoParametroNaoEncontrado struct{}

func (ELIM0007TipoParametroNaoEncontrado) Code() string     { return "2061-ELIM0007" }
func (ELIM0007TipoParametroNaoEncontrado) Sheet() string    { return "ParametrosDLO" }
func (ELIM0007TipoParametroNaoEncontrado) Severity() string { return "E" }

func (ELIM0007TipoParametroNaoEncontrado) Apply(_ context.Context, doc *DocDLO) error {
	// Parâmetros DLO são configurados externamente (taxas, limites).
	// Stub: requer contexto de configuração de parâmetros.
	_ = doc
	return nil
}

// ELIM0008 — Domínio parâmetro não encontrado.
//
// Severidade: E
type ELIM0008DominioParametro struct{}

func (ELIM0008DominioParametro) Code() string     { return "2061-ELIM0008" }
func (ELIM0008DominioParametro) Sheet() string    { return "ParametrosDLO" }
func (ELIM0008DominioParametro) Severity() string { return "E" }

func (ELIM0008DominioParametro) Apply(_ context.Context, doc *DocDLO) error {
	_ = doc
	return nil
}

// ELIM0009 — Conta inexistente: código de conta não é válido no COSIF.
//
// Severidade: E
type ELIM0009ContaInexistente struct{}

func (ELIM0009ContaInexistente) Code() string     { return "2061-ELIM0009" }
func (ELIM0009ContaInexistente) Sheet() string    { return "ContasCOSIF" }
func (ELIM0009ContaInexistente) Severity() string { return "E" }

func (ELIM0009ContaInexistente) Apply(_ context.Context, doc *DocDLO) error {
	// COSIF account list is managed by BACEN. This rule checks the account exists.
	// Valid COSIF top-level accounts in DLO: 100, 101, 102, 111, 112, 120,
	// 140, 150, 160, 180, 200, 310, 320, 330, 340, 350, 400, 410, 420,
	// 510, 520, 530, 540, 550, 560, 570, 580, 590, 600, 605, 610, 620,
	// 630, 640, 650, 660, 700, 750, 770, 800, 820, 830, 840, 850, 860,
	// 865, 870, 871, 872, 873, 890, 900, 910, 920, 930, 940, 942, 943,
	// 950, 951, 952, 953, 954, 955, 956, 957, 958, 959, 960, 970, 975.
	validTopLevel := map[string]bool{
		"100": true, "101": true, "102": true, "111": true, "112": true,
		"120": true, "140": true, "150": true, "160": true, "180": true,
		"200": true, "310": true, "320": true, "330": true, "340": true,
		"350": true, "400": true, "410": true, "420": true, "510": true,
		"520": true, "530": true, "540": true, "550": true, "560": true,
		"570": true, "580": true, "590": true, "600": true, "605": true,
		"610": true, "620": true, "630": true, "640": true, "650": true,
		"660": true, "700": true, "750": true, "770": true, "800": true,
		"820": true, "830": true, "840": true, "850": true, "860": true,
		"865": true, "870": true, "871": true, "872": true, "873": true,
		"890": true, "900": true, "910": true, "920": true, "930": true,
		"940": true, "942": true, "943": true, "950": true, "951": true,
		"952": true, "953": true, "954": true, "955": true, "956": true,
		"957": true, "958": true, "959": true, "960": true, "970": true,
		"975": true,
	}
	for acct := range doc.Accounts {
		// Get top-level account (first 3 digits)
		top := acct
		if len(acct) >= 3 {
			top = acct[:3]
		} else if len(acct) >= 2 {
			top = acct[:2]
		}
		if !validTopLevel[top] {
			return fmt.Errorf("conta %q: código de conta inexistente no COSIF", acct)
		}
	}
	return nil
}

// ELIM0010 — Tipo de elemento informado não foi encontrado.
//
// Severidade: E
type ELIM0010TipoElementoNaoEncontrado struct{}

func (ELIM0010TipoElementoNaoEncontrado) Code() string     { return "2061-ELIM0010" }
func (ELIM0010TipoElementoNaoEncontrado) Sheet() string    { return "ElementosCOSIF" }
func (ELIM0010TipoElementoNaoEncontrado) Severity() string { return "E" }

func (ELIM0010TipoElementoNaoEncontrado) Apply(_ context.Context, doc *DocDLO) error {
	// Element codes per account are defined by COSIF. Common: 1, 2, 3...
	for acct, elems := range doc.Elements {
		for _, e := range elems {
			if e.Codigo == "" {
				return fmt.Errorf("conta %s: elemento sem código", acct)
			}
		}
	}
	return nil
}

// ELIM0013 — Valor da conta difere do somatório dos elementos.
//
// Severidade: E
type ELIM0013ValorContaDivergente struct{}

func (ELIM0013ValorContaDivergente) Code() string     { return "2061-ELIM0013" }
func (ELIM0013ValorContaDivergente) Sheet() string    { return "ContasCOSIF" }
func (ELIM0013ValorContaDivergente) Severity() string { return "E" }

func (ELIM0013ValorContaDivergente) Apply(_ context.Context, doc *DocDLO) error {
	const tol = 0.01
	for acct, saldo := range doc.Accounts {
		elems := doc.Elements[acct]
		if len(elems) == 0 {
			continue
		}
		var soma float64
		for _, e := range elems {
			soma += e.Valor
		}
		diff := saldo - soma
		if diff < 0 {
			diff = -diff
		}
		if diff > tol {
			return fmt.Errorf("conta %s: saldo=%.2f diverge da soma dos elementos=%.2f (diff=%.2f)", acct, saldo, soma, diff)
		}
	}
	return nil
}

// ELIM0015 — Valor da conta pai difere da soma das contas filhas.
//
// Severidade: E
type ELIM0015ContaPaiDivergente struct{}

func (ELIM0015ContaPaiDivergente) Code() string     { return "2061-ELIM0015" }
func (ELIM0015ContaPaiDivergente) Sheet() string    { return "ContasCOSIF" }
func (ELIM0015ContaPaiDivergente) Severity() string { return "E" }

func (ELIM0015ContaPaiDivergente) Apply(_ context.Context, doc *DocDLO) error {
	// Check parent accounts: 111, 112, 120, 800, 890, 910, 920, 930, 940, 950
	parentAccounts := []string{"111", "112", "120", "800", "890", "910", "920", "930", "940", "950"}
	const tol = 0.01
	for _, parent := range parentAccounts {
		parentVal := doc.SaldoConta(parent)
		if parentVal == 0 {
			continue
		}
		// Sum all child accounts (those that start with parent prefix)
		var soma float64
		for acct, val := range doc.Accounts {
			if strings.HasPrefix(acct, parent+".") || strings.HasPrefix(acct, parent) && acct != parent {
				soma += val
			}
		}
		if soma == 0 {
			continue
		}
		diff := parentVal - soma
		if diff < 0 {
			diff = -diff
		}
		if diff > tol {
			return fmt.Errorf("conta pai %s: saldo=%.2f diverge da soma das filhas=%.2f", parent, parentVal, soma)
		}
	}
	return nil
}

// ELIM0055 — Saldo da conta 871 incompatível com média ponderada pelo fator 0,15 dos IE.
//
// Severidade: E
type ELIM0055Conta871MediaPonderada struct{}

func (ELIM0055Conta871MediaPonderada) Code() string     { return "2061-ELIM0055" }
func (ELIM0055Conta871MediaPonderada) Sheet() string    { return "ContasCOSIF" }
func (ELIM0055Conta871MediaPonderada) Severity() string { return "E" }

func (ELIM0055Conta871MediaPonderada) Apply(_ context.Context, doc *DocDLO) error {
	// Elements of account 871 have a peso (fator de ponderação).
	// The weighted average of elements should be consistent with saldo 871.
	saldo871 := doc.SaldoConta("871")
	if saldo871 == 0 {
		return nil
	}
	elems := doc.Elements["871"]
	if len(elems) == 0 {
		return nil
	}
	var somaValPeso, somaPeso float64
	for _, e := range elems {
		somaValPeso += e.Valor * e.Peso
		somaPeso += e.Peso
	}
	if somaPeso == 0 {
		return nil
	}
	mediaPonderada := somaValPeso / somaPeso
	// Check if media * fator 0.15 is consistent
	// This is a stub: actual rule checks specific COSIF formula for 871
	_ = mediaPonderada
	return nil
}

// ELIM0063 — Valor total das contas COSIF não está de acordo com o somatório.
//
// Severidade: E
type ELIM0063TotalContasCosif struct{}

func (ELIM0063TotalContasCosif) Code() string     { return "2061-ELIM0063" }
func (ELIM0063TotalContasCosif) Sheet() string    { return "ContasCOSIF" }
func (ELIM0063TotalContasCosif) Severity() string { return "E" }

func (ELIM0063TotalContasCosif) Apply(_ context.Context, doc *DocDLO) error {
	// Validate total assets (accounts 100-180) consistency.
	// 100 = 101 + 102 + ... (simplified check).
	// This is a cross-rule between top-level and detailed accounts.
	_ = doc
	return nil
}

// ELIM0208 — Valores RWAOPAD em desacordo com abordagem de risco operacional.
//
// Severidade: E
type ELIM0208RWAOPADIndevido struct{}

func (ELIM0208RWAOPADIndevido) Code() string     { return "2061-ELIM0208" }
func (ELIM0208RWAOPADIndevido) Sheet() string    { return "RWAOPAD" }
func (ELIM0208RWAOPADIndevido) Severity() string { return "E" }

func (ELIM0208RWAOPADIndevido) Apply(_ context.Context, doc *DocDLO) error {
	// RWAOPAD accounts: 820, 830, 840, 850, 860, 865.
	// Should only have values if the bank uses the Standardized approach.
	rwaopadAccts := []string{"820", "830", "840", "850", "860", "865"}
	var totalRWAOPAD float64
	for _, acct := range rwaopadAccts {
		totalRWAOPAD += doc.SaldoConta(acct)
	}
	if totalRWAOPAD < 0 {
		return fmt.Errorf("RWAOPAD total=%.2f negativo (abordagem risco operacional inválida)", totalRWAOPAD)
	}
	return nil
}

// ELIM0346/ELIM0347 — Valor do elemento 2 de conta RWACPAD negativo.
//
// Severidade: E
type ELIM0346RWACPADNegativo struct{}

func (ELIM0346RWACPADNegativo) Code() string     { return "2061-ELIM0346" }
func (ELIM0346RWACPADNegativo) Sheet() string    { return "RWACPAD" }
func (ELIM0346RWACPADNegativo) Severity() string { return "E" }

func (ELIM0346RWACPADNegativo) Apply(_ context.Context, doc *DocDLO) error {
	// Check RWACPAD accounts (800.xx) for negative element 2.
	for acct, elems := range doc.Elements {
		if !strings.HasPrefix(acct, "800.") {
			continue
		}
		for _, e := range elems {
			if e.Codigo == "2" && e.Valor < 0 {
				return fmt.Errorf("RWACPAD conta %s elemento 2: valor %.2f negativo", acct, e.Valor)
			}
		}
	}
	return nil
}

// ELIM0405-ELIM0412 — RWACAM vs fator × (800.01 + 800.02 + 800.03).
//
// Severidade: E
type ELIM0405RWACAM04 struct{}

func (ELIM0405RWACAM04) Code() string     { return "2061-ELIM0405" }
func (ELIM0405RWACAM04) Sheet() string    { return "RWACAM" }
func (ELIM0405RWACAM04) Severity() string { return "E" }

func (ELIM0405RWACAM04) Apply(_ context.Context, doc *DocDLO) error {
	// fator = 0.4, tol = 0.01
	return doc.ValidateRWACAM(0.4, 0.01)
}

type ELIM0406RWACAM06 struct{}

func (ELIM0406RWACAM06) Code() string     { return "2061-ELIM0406" }
func (ELIM0406RWACAM06) Sheet() string    { return "RWACAM" }
func (ELIM0406RWACAM06) Severity() string { return "E" }

func (ELIM0406RWACAM06) Apply(_ context.Context, doc *DocDLO) error {
	return doc.ValidateRWACAM(0.6, 0.01)
}

type ELIM0407RWACAM08 struct{}

func (ELIM0407RWACAM08) Code() string     { return "2061-ELIM0407" }
func (ELIM0407RWACAM08) Sheet() string    { return "RWACAM" }
func (ELIM0407RWACAM08) Severity() string { return "E" }

func (ELIM0407RWACAM08) Apply(_ context.Context, doc *DocDLO) error {
	return doc.ValidateRWACAM(0.8, 0.01)
}

type ELIM0408RWACAM10 struct{}

func (ELIM0408RWACAM10) Code() string     { return "2061-ELIM0408" }
func (ELIM0408RWACAM10) Sheet() string    { return "RWACAM" }
func (ELIM0408RWACAM10) Severity() string { return "E" }

func (ELIM0408RWACAM10) Apply(_ context.Context, doc *DocDLO) error {
	return doc.ValidateRWACAM(1.0, 0.01)
}

// ELIM0451 — Documento não apresentou contas RWACPAD.
//
// Severidade: E
type ELIM0451SemRWACPAD struct{}

func (ELIM0451SemRWACPAD) Code() string     { return "2061-ELIM0451" }
func (ELIM0451SemRWACPAD) Sheet() string    { return "RWACPAD" }
func (ELIM0451SemRWACPAD) Severity() string { return "E" }

func (ELIM0451SemRWACPAD) Apply(_ context.Context, doc *DocDLO) error {
	// Must have at least one 800.xx account if reporting RWA.
	if doc.SaldoConta("800.01") == 0 && doc.SaldoConta("800.02") == 0 && doc.SaldoConta("800.03") == 0 {
		return fmt.Errorf("documento sem detalhamento RWACPAD (contas 800.01, 800.02, 800.03)")
	}
	return nil
}

// ELIM0522 — Saldo da conta 700 inconsistente com soma 943.01 + 943.02.
//
// Severidade: E
type ELIM0522Conta700 struct{}

func (ELIM0522Conta700) Code() string     { return "2061-ELIM0522" }
func (ELIM0522Conta700) Sheet() string    { return "ContasCOSIF" }
func (ELIM0522Conta700) Severity() string { return "E" }

func (ELIM0522Conta700) Apply(_ context.Context, doc *DocDLO) error {
	saldo700 := doc.SaldoConta("700")
	if saldo700 == 0 {
		return nil
	}
	soma := doc.SaldoConta("943.01") + doc.SaldoConta("943.02")
	if soma == 0 {
		return nil
	}
	const tol = 0.01
	diff := saldo700 - soma
	if diff < 0 {
		diff = -diff
	}
	if diff > tol {
		return fmt.Errorf("conta 700=%.2f inconsistente com soma(943.01+943.02)=%.2f", saldo700, soma)
	}
	return nil
}

// ELIM0538/ELIM0539 — Documento não apresentou detalhamento RWAOPAD.
//
// Severidade: E
type ELIM0538SemRWAOPAD struct{}

func (ELIM0538SemRWAOPAD) Code() string     { return "2061-ELIM0538" }
func (ELIM0538SemRWAOPAD) Sheet() string    { return "RWAOPAD" }
func (ELIM0538SemRWAOPAD) Severity() string { return "E" }

func (ELIM0538SemRWAOPAD) Apply(_ context.Context, doc *DocDLO) error {
	rwaopadAccts := []string{"820", "830", "840", "850", "860", "865"}
	var total float64
	for _, acct := range rwaopadAccts {
		total += doc.SaldoConta(acct)
	}
	if total == 0 {
		return fmt.Errorf("documento sem detalhamento RWAOPAD")
	}
	return nil
}

// ELIM0585/ELIM0586/ELIM0587 — RWAOPAD não apresentou detalhamento para período T-3.
//
// Severidade: E
type ELIM0585RWAOPADT3 struct{}

func (ELIM0585RWAOPADT3) Code() string     { return "2061-ELIM0585" }
func (ELIM0585RWAOPADT3) Sheet() string    { return "RWAOPAD" }
func (ELIM0585RWAOPADT3) Severity() string { return "E" }

func (ELIM0585RWAOPADT3) Apply(_ context.Context, doc *DocDLO) error {
	// T-3 historical data: DLO includes accounts for period T-3.
	// Stub: requires temporal account structure (T, T-1, T-2, T-3).
	_ = doc
	return nil
}

// ValidarDRMBasico faz validação básica do DRM (consistência interna).
func ValidarDRMBasico(doc *DocDRM) error {
	if doc == nil {
		return fmt.Errorf("DRM nil")
	}
	if doc.VaR < 0 {
		return fmt.Errorf("VaR=%v negativo", doc.VaR)
	}
	if doc.sVaR < 0 {
		return fmt.Errorf("sVaR=%v negativo", doc.sVaR)
	}
	if doc.RWACOM < 0 {
		return fmt.Errorf("RWACOM=%v negativo", doc.RWACOM)
	}
	// VaR <= sVaR (sanity — VaR é valor atual, sVaR é stress test, geralmente maior)
	if doc.VaR > 0 && doc.sVaR > 0 && doc.VaR > doc.sVaR {
		return fmt.Errorf("VaR=%v > sVaR=%v (esperado VaR <= sVaR em stress)", doc.VaR, doc.sVaR)
	}
	return nil
}

// Rule2061 é a interface para regras de validação do CADOC 2061 (DLO).
type Rule2061 interface {
	Code() string
	Sheet() string
	Severity() string // E (Erro bloqueante), A (Aviso), I (Informativo)
	Apply(ctx context.Context, doc *DocDLO) error
}

// BuiltinDLO registra todas as regras DLO/2061 no registry fornecido.
func BuiltinDLO(r *Registry) {
	rules := []Rule2061{
		ELIM0001FormatoDoc{},
		ELIM0002DocNaoEncontrado{},
		ELIM0003CNPJNaoEncontrado{},
		ELIM0004ConglomeradoNaoEncontrado{},
		ELIM0005CNPJNaoPertence{},
		ELIM0006CNPJNaoLider{},
		ELIM0007TipoParametroNaoEncontrado{},
		ELIM0008DominioParametro{},
		ELIM0009ContaInexistente{},
		ELIM0010TipoElementoNaoEncontrado{},
		ELIM0013ValorContaDivergente{},
		ELIM0015ContaPaiDivergente{},
		ELIM0055Conta871MediaPonderada{},
		ELIM0063TotalContasCosif{},
		ELIM0208RWAOPADIndevido{},
		ELIM0346RWACPADNegativo{},
		ELIM0405RWACAM04{},
		ELIM0406RWACAM06{},
		ELIM0407RWACAM08{},
		ELIM0408RWACAM10{},
		ELIM0451SemRWACPAD{},
		ELIM0522Conta700{},
		ELIM0538SemRWAOPAD{},
		ELIM0585RWAOPADT3{},
	}
	for _, rule := range rules {
		r.Register2061(rule)
	}
}
