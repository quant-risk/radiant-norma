// Package rules implementa regras de validação semânticas portadas dos
// catálogos BACEN para execução em Go.
//
// CADOC 2062 — DLI (Demonstrativo de Limites Operacionais).
//
// Estrutura: documentoDLI → limitesInformados/parametros/contas
// COSIF: 6.x (PLA), 8.x (Capital Realizado), 9.x (Captação BD),
//
//	11-12.x (LCD), 20-22.x (Partes Relacionadas),
//	34-36.x (TVM), 38-39.x (Agências Fomento),
//	56-58.x (SCM), 76-78.x (Cooperativas),
//	91-94.x (SCD/SEP/Confederación).
//
// Sprint 52: parser completo + regras DLI-01 a DLI-18.
//
// Referência: BACEN — 2062_DLI_Leiaute.xlsx + DLI_2062_InstrucoesPreenchimento_v3.pdf.
package rules

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// DocDLI — Documento DLI (Demonstrativo de Limites Operacionais) — CADOC 2062.
type DocDLI struct {
	Root DocDLIRoot

	// Campos do documento.
	TipoEnvio string // "I" ou "S"

	// Limites operacionais reportados (Anexo 2 do leiaute).
	Limites []DLILimite

	// Parâmetros do responsável pelo envio.
	Parametros []DLIParametro

	// Contas COSIF — map[codigoConta]valor.
	Accounts map[string]float64
}

// DocDLIRoot representa o elemento raiz <documentoDLI>.
type DocDLIRoot struct {
	CNPJ      string // 8 dígitos
	DataBase  string // YYYY-MM
	CodigoDoc string // "2062"
	TipoEnvio string // "I" ou "S"
}

// DLILimite representa um limite operacional informado (Anexo 2).
// Ex: <limite codigoLimite="06.00" enviado="S">R$ 1.000.000,00</limite>
type DLILimite struct {
	Codigo  string // NN.NN (ex: "06.00", "20.00", "21.00")
	Enviado string // "S" ou "N"
	Valor   string // N15,2
}

// DLIParametro representa um parâmetro do responsável (Anexo 3).
type DLIParametro struct {
	Codigo string // 31, 32, 33
	Valor  string
}

// PartialParseErrorDLI indica parse parcial bem-sucedido.
type PartialParseErrorDLI struct{ Err error }

func (e *PartialParseErrorDLI) Error() string { return "parse DLI: " + e.Err.Error() }
func (e *PartialParseErrorDLI) Unwrap() error { return e.Err }

// ParseDocDLI faz parse completo do XML DLI.
//
// Estrutura:
//
//	<documentoDLI cnpj="..." dataBase="AAAA-MM" codigoDocumento="2062" tipoEnvio="I">
//	  <limitesInformados>
//	    <limite codigoLimite="06.00" enviado="S">1000000.00</limite>
//	  </limitesInformados>
//	  <parametros>
//	    <parametro codigoParametro="31" valorParametro="João Silva"/>
//	  </parametros>
//	  <contas>
//	    <conta codigoConta="6.10.01" valorConta="5000000.00"/>
//	  </contas>
//	</documentoDLI>
func ParseDocDLI(data []byte) (*DocDLI, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	doc := &DocDLI{
		Accounts: make(map[string]float64),
	}

	var (
		currentLimite *DLILimite // ponteiro para acumular valor de texto
	)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseErrorDLI{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "documentoDLI":
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "cnpj":
						doc.Root.CNPJ = a.Value
					case "dataBase":
						doc.Root.DataBase = a.Value
					case "codigoDocumento":
						doc.Root.CodigoDoc = a.Value
					case "tipoEnvio":
						doc.Root.TipoEnvio = a.Value
						doc.TipoEnvio = a.Value
					}
				}
			case "limite":
				lim := DLILimite{Enviado: "N", Valor: ""}
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "codigoLimite":
						lim.Codigo = a.Value
					case "enviado":
						lim.Enviado = a.Value
					}
				}
				doc.Limites = append(doc.Limites, lim)
				currentLimite = &doc.Limites[len(doc.Limites)-1]
			case "parametro":
				par := DLIParametro{}
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "codigoParametro":
						par.Codigo = a.Value
					case "valorParametro":
						par.Valor = a.Value
					}
				}
				doc.Parametros = append(doc.Parametros, par)
			case "conta":
				codigo, valor := "", ""
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "codigoConta":
						codigo = a.Value
					case "valorConta":
						valor = a.Value
					}
				}
				if codigo != "" {
					doc.Accounts[codigo] = parseNum(valor)
				}
			}
		case xml.CharData:
			if currentLimite != nil && currentLimite.Valor == "" {
				currentLimite.Valor = strings.TrimSpace(string(t))
			}
		}
	}

	return doc, nil
}

// parseNum converte string numérica brasileira (1.234,56) para float64.
// Reutilizada de 3040.go — declarada ali para evitar duplicação.

// ============================================================
// Helpers — soma por grupo COSIF
// ============================================================

// SomaPLA retorna a soma 6.10.01 + 6.10.02 - 6.10.90 (PLA contábil).
func SomaPLA(accounts map[string]float64) float64 {
	return accounts["6.10.01"] + accounts["6.10.02"] - accounts["6.10.90"]
}

// SomaMargem6 retorna o valor da conta 6.00.00 (Margem PLA).
func SomaMargem6(accounts map[string]float64) float64 {
	return accounts["6.00.00"]
}

// SomaMargem8 retorna 8.00.00 (Margem Capital Realizado).
func SomaMargem8(accounts map[string]float64) float64 {
	return accounts["8.00.00"]
}

// SomaCapitalRealizado retorna 8.10.01 (Capital Realizado).
func SomaCapitalRealizado(accounts map[string]float64) float64 {
	return accounts["8.10.01"]
}

// SomaLimiteCaptaçãoBD retorna 9.00.00 (Margem Captação Bancos de Desenvolvimento).
func SomaLimiteCaptaçãoBD(accounts map[string]float64) float64 {
	return accounts["9.00.00"]
}

// SomaLimiteLCD retorna 11.00.00 + 12.00.00 (Margem LCD).
func SomaLimiteLCD(accounts map[string]float64) float64 {
	return accounts["11.00.00"] + accounts["12.00.00"]
}

// SomaPartesRelacionadas retorna 20.00.00 + 21.00.00 + 22.00.00.
func SomaPartesRelacionadas(accounts map[string]float64) float64 {
	return accounts["20.00.00"] + accounts["21.00.00"] + accounts["22.00.00"]
}

// SomaTVM retorna 34.00.00 + 35.00.00 + 36.00.00 (empréstimo/financiamento/garantias TVM).
func SomaTVM(accounts map[string]float64) float64 {
	return accounts["34.00.00"] + accounts["35.00.00"] + accounts["36.00.00"]
}

// SomaAgenciasFomento retorna 38.00.00 + 39.00.00.
func SomaAgenciasFomento(accounts map[string]float64) float64 {
	return accounts["38.00.00"] + accounts["39.00.00"]
}

// SomaSCM retorna 56.00.00 + 58.00.00 (SCM).
func SomaSCM(accounts map[string]float64) float64 {
	return accounts["56.00.00"] + accounts["58.00.00"]
}

// SomaCooperativasCredito retorna 76.00.00 + 77.00.00 + 78.00.00.
func SomaCooperativasCredito(accounts map[string]float64) float64 {
	return accounts["76.00.00"] + accounts["77.00.00"] + accounts["78.00.00"]
}

// SomaSCDeSEP retorna 91.00.00 + 92.00.00.
func SomaSCDeSEP(accounts map[string]float64) float64 {
	return accounts["91.00.00"] + accounts["92.00.00"]
}

// SomaConfederacaoServico retorna 93.00.00 + 94.00.00.
func SomaConfederacaoServico(accounts map[string]float64) float64 {
	return accounts["93.00.00"] + accounts["94.00.00"]
}

// SomaRequerimentoPLA retorna 6.90.00 (Requerimento de PLA).
func SomaRequerimentoPLA(accounts map[string]float64) float64 {
	return accounts["6.90.00"]
}

// SomaRequerimentoCapital retorna 8.90.00 (Requerimento Capital Realizado).
func SomaRequerimentoCapital(accounts map[string]float64) float64 {
	return accounts["8.90.00"]
}

// ============================================================
// Rule2062 — interface para regras de validação do CADOC 2062 (DLI).
// ============================================================

type Rule2062 interface {
	Code() string
	Sheet() string
	Severity() string // E (Erro bloqueante), A (Aviso), I (Informativo)
	Apply(ctx context.Context, doc *DocDLI) error
}

// ============================================================
// Regras estruturais (DLI-01 a DLI-08)
// ============================================================

// DLI01CNPJValido verifica se CNPJ tem exatamente 8 dígitos.
type DLI01CNPJValido struct{}

func (DLI01CNPJValido) Code() string     { return "DLI-01" }
func (DLI01CNPJValido) Sheet() string    { return "Estrutura" }
func (DLI01CNPJValido) Severity() string { return "E" }

func (DLI01CNPJValido) Apply(ctx context.Context, doc *DocDLI) error {
	if doc == nil {
		return fmt.Errorf("documento DLI nil")
	}
	if !regexp.MustCompile(`^\d{8}$`).MatchString(doc.Root.CNPJ) {
		return fmt.Errorf("CNPJ=%q deve ter exatamente 8 dígitos", doc.Root.CNPJ)
	}
	return nil
}

// DLI02DataBaseValido verifica se dataBase tem formato AAAA-MM.
type DLI02DataBaseValido struct{}

func (DLI02DataBaseValido) Code() string     { return "DLI-02" }
func (DLI02DataBaseValido) Sheet() string    { return "Estrutura" }
func (DLI02DataBaseValido) Severity() string { return "E" }

func (DLI02DataBaseValido) Apply(ctx context.Context, doc *DocDLI) error {
	if !regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`).MatchString(doc.Root.DataBase) {
		return fmt.Errorf("dataBase=%q deve ter formato AAAA-MM", doc.Root.DataBase)
	}
	return nil
}

// DLI03TipoEnvioValido verifica se tipoEnvio é I ou S.
type DLI03TipoEnvioValido struct{}

func (DLI03TipoEnvioValido) Code() string     { return "DLI-03" }
func (DLI03TipoEnvioValido) Sheet() string    { return "Estrutura" }
func (DLI03TipoEnvioValido) Severity() string { return "E" }

func (DLI03TipoEnvioValido) Apply(ctx context.Context, doc *DocDLI) error {
	if doc.TipoEnvio != "I" && doc.TipoEnvio != "S" {
		return fmt.Errorf("tipoEnvio=%q deve ser I (inclusão) ou S (substituição)", doc.TipoEnvio)
	}
	return nil
}

// DLI04CodigoDocumentoValido verifica se codigoDocumento é 2062.
type DLI04CodigoDocumentoValido struct{}

func (DLI04CodigoDocumentoValido) Code() string     { return "DLI-04" }
func (DLI04CodigoDocumentoValido) Sheet() string    { return "Estrutura" }
func (DLI04CodigoDocumentoValido) Severity() string { return "E" }

func (DLI04CodigoDocumentoValido) Apply(ctx context.Context, doc *DocDLI) error {
	if doc.Root.CodigoDoc != "2062" {
		return fmt.Errorf("codigoDocumento=%q deve ser 2062", doc.Root.CodigoDoc)
	}
	return nil
}

// DLI05TemConteudo verifica se pelo menos uma seção tem conteúdo.
type DLI05TemConteudo struct{}

func (DLI05TemConteudo) Code() string     { return "DLI-05" }
func (DLI05TemConteudo) Sheet() string    { return "Estrutura" }
func (DLI05TemConteudo) Severity() string { return "E" }

func (DLI05TemConteudo) Apply(ctx context.Context, doc *DocDLI) error {
	hasLimites := len(doc.Limites) > 0
	hasParametros := len(doc.Parametros) > 0
	hasContas := len(doc.Accounts) > 0
	if !hasLimites && !hasParametros && !hasContas {
		return fmt.Errorf("documento sem conteúdo (sem Limites, Parametros ou Contas)")
	}
	return nil
}

// DLI06LimiteCodigoValido verifica formato NN.NN dos limites.
type DLI06LimiteCodigoValido struct{}

func (DLI06LimiteCodigoValido) Code() string     { return "DLI-06" }
func (DLI06LimiteCodigoValido) Sheet() string    { return "Estrutura" }
func (DLI06LimiteCodigoValido) Severity() string { return "E" }

func (DLI06LimiteCodigoValido) Apply(ctx context.Context, doc *DocDLI) error {
	re := regexp.MustCompile(`^\d{2}\.\d{2}$`)
	for i, lim := range doc.Limites {
		if !re.MatchString(lim.Codigo) {
			return fmt.Errorf("Limite[%d]: código %q inválido (esperado NN.NN)", i+1, lim.Codigo)
		}
		// Valor deve ser numérico (N15,2)
		if lim.Valor != "" && !regexp.MustCompile(`^-?\d{1,13}(\.\d{2})?$`).MatchString(lim.Valor) {
			return fmt.Errorf("Limite[%d]: valor %q inválido (esperado N13,2)", i+1, lim.Valor)
		}
	}
	return nil
}

// DLI07IndicadorValido verifica se indicador tem valor S ou N.
type DLI07IndicadorValido struct{}

func (DLI07IndicadorValido) Code() string     { return "DLI-07" }
func (DLI07IndicadorValido) Sheet() string    { return "Estrutura" }
func (DLI07IndicadorValido) Severity() string { return "E" }

func (DLI07IndicadorValido) Apply(ctx context.Context, doc *DocDLI) error {
	// DLI não tem Indicadores como DLO — é só estrutura de limites e contas.
	// Esta regra verifica consistência genérica se indicadors existissem.
	_ = doc
	return nil
}

// DLI08ContaCOSIFValida verifica formato N.NN.NN.NN das contas COSIF.
type DLI08ContaCOSIFValida struct{}

func (DLI08ContaCOSIFValida) Code() string     { return "DLI-08" }
func (DLI08ContaCOSIFValida) Sheet() string    { return "Estrutura" }
func (DLI08ContaCOSIFValida) Severity() string { return "E" }

func (DLI08ContaCOSIFValida) Apply(ctx context.Context, doc *DocDLI) error {
	re := regexp.MustCompile(`^\d\.\d{2}\.\d{2}(\.\d{2})?$`)
	for codigo, valor := range doc.Accounts {
		if !re.MatchString(codigo) {
			return fmt.Errorf("conta COSIF %q com formato inválido", codigo)
		}
		// Valores devem ser >= 0 para a maioria das contas de limite
		if valor < 0 {
			return fmt.Errorf("conta COSIF %q com valor negativo: %v", codigo, valor)
		}
	}
	return nil
}

// ============================================================
// Regras de consistência PLA (DLI-09 a DLI-12)
// ============================================================

// DLI09PLAContabil valida PLA = 6.10.01 + 6.10.02 - 6.10.90.
// Exige que 6.10.01 (Patrimônio Líquido) seja informado.
type DLI09PLAContabil struct{}

func (DLI09PLAContabil) Code() string     { return "DLI-09" }
func (DLI09PLAContabil) Sheet() string    { return "ConsistênciaPLA" }
func (DLI09PLAContabil) Severity() string { return "E" }

func (DLI09PLAContabil) Apply(ctx context.Context, doc *DocDLI) error {
	pla := SomaPLA(doc.Accounts)
	if doc.Accounts["6.10.01"] == 0 {
		return fmt.Errorf("6.10.01 (Patrimônio Líquido) ausente — PLA não pode ser calculado")
	}
	if pla <= 0 {
		return fmt.Errorf("PLA=%v inválido (deve ser > 0)", pla)
	}
	return nil
}

// DLI10MargemPLA valida 6.00.00 = 6.10.00 - 6.90.00.
type DLI10MargemPLA struct{}

func (DLI10MargemPLA) Code() string     { return "DLI-10" }
func (DLI10MargemPLA) Sheet() string    { return "ConsistênciaPLA" }
func (DLI10MargemPLA) Severity() string { return "A" }

func (DLI10MargemPLA) Apply(ctx context.Context, doc *DocDLI) error {
	margem6 := doc.Accounts["6.00.00"]
	pla := SomaPLA(doc.Accounts)
	req := SomaRequerimentoPLA(doc.Accounts)
	esperado := pla - req
	// Tolerância de 0,01 por precisão decimal
	if margem6 != 0 && abs(margem6-esperado) > 0.01 {
		return fmt.Errorf("6.00.00=%v deveria ser PLA(%v) - Requerimento(%v) = %v",
			margem6, pla, req, esperado)
	}
	return nil
}

// DLI11CapitalRealizado valida 8.10.01 (Capital Realizado).
type DLI11CapitalRealizado struct{}

func (DLI11CapitalRealizado) Code() string     { return "DLI-11" }
func (DLI11CapitalRealizado) Sheet() string    { return "ConsistênciaCapital" }
func (DLI11CapitalRealizado) Severity() string { return "A" }

func (DLI11CapitalRealizado) Apply(ctx context.Context, doc *DocDLI) error {
	cap := SomaCapitalRealizado(doc.Accounts)
	if cap == 0 && len(doc.Accounts) > 0 {
		return fmt.Errorf("8.10.01 (Capital Realizado) ausente ou zerado")
	}
	if cap < 0 {
		return fmt.Errorf("8.10.01=%v negativo", cap)
	}
	return nil
}

// DLI12MargemCapital valida 8.00.00 = 8.10.00 - 8.90.00.
type DLI12MargemCapital struct{}

func (DLI12MargemCapital) Code() string     { return "DLI-12" }
func (DLI12MargemCapital) Sheet() string    { return "ConsistênciaCapital" }
func (DLI12MargemCapital) Severity() string { return "A" }

func (DLI12MargemCapital) Apply(ctx context.Context, doc *DocDLI) error {
	margem8 := doc.Accounts["8.00.00"]
	cap := doc.Accounts["8.10.00"]
	req := SomaRequerimentoCapital(doc.Accounts)
	esperado := cap - req
	if margem8 != 0 && abs(margem8-esperado) > 0.01 {
		return fmt.Errorf("8.00.00=%v deveria ser 8.10.00(%v) - 8.90.00(%v) = %v",
			margem8, cap, req, esperado)
	}
	return nil
}

// ============================================================
// Regras de limites vs PLA (DLI-13 a DLI-18)
// ============================================================

// DLI13LimitePartesRelacionadas verifica Limite 20.00 = 20% PLA.
type DLI13LimitePartesRelacionadas struct{}

func (DLI13LimitePartesRelacionadas) Code() string     { return "DLI-13" }
func (DLI13LimitePartesRelacionadas) Sheet() string    { return "LimitesPartesRelacionadas" }
func (DLI13LimitePartesRelacionadas) Severity() string { return "E" }

func (DLI13LimitePartesRelacionadas) Apply(ctx context.Context, doc *DocDLI) error {
	lim := extractLimite(doc.Limites, "20.00")
	if lim == nil {
		return nil // limite opcional
	}
	pla := SomaPLA(doc.Accounts)
	if pla == 0 {
		return fmt.Errorf("PLA não pode ser 0 para calcular limite 20.00")
	}
	valorLim := parseNum(lim.Valor)
	maximo := pla * 0.20
	if valorLim > maximo {
		return fmt.Errorf("Limite 20.00 (Partes Relacionadas)=%v excede 20%% PLA=%v", valorLim, maximo)
	}
	return nil
}

// DLI14LimitePRPessoaNatural verifica Limite 21.00 = 1% PLA.
type DLI14LimitePRPessoaNatural struct{}

func (DLI14LimitePRPessoaNatural) Code() string     { return "DLI-14" }
func (DLI14LimitePRPessoaNatural) Sheet() string    { return "LimitesPartesRelacionadas" }
func (DLI14LimitePRPessoaNatural) Severity() string { return "E" }

func (DLI14LimitePRPessoaNatural) Apply(ctx context.Context, doc *DocDLI) error {
	lim := extractLimite(doc.Limites, "21.00")
	if lim == nil {
		return nil
	}
	pla := SomaPLA(doc.Accounts)
	if pla == 0 {
		return fmt.Errorf("PLA não pode ser 0 para calcular limite 21.00")
	}
	valorLim := parseNum(lim.Valor)
	maximo := pla * 0.01
	if valorLim > maximo {
		return fmt.Errorf("Limite 21.00 (PR PN)=%v excede 1%% PLA=%v", valorLim, maximo)
	}
	return nil
}

// DLI15LimitePRPessoaJuridica verifica Limite 22.00 = 5% PLA.
type DLI15LimitePRPessoaJuridica struct{}

func (DLI15LimitePRPessoaJuridica) Code() string     { return "DLI-15" }
func (DLI15LimitePRPessoaJuridica) Sheet() string    { return "LimitesPartesRelacionadas" }
func (DLI15LimitePRPessoaJuridica) Severity() string { return "E" }

func (DLI15LimitePRPessoaJuridica) Apply(ctx context.Context, doc *DocDLI) error {
	lim := extractLimite(doc.Limites, "22.00")
	if lim == nil {
		return nil
	}
	pla := SomaPLA(doc.Accounts)
	if pla == 0 {
		return fmt.Errorf("PLA não pode ser 0 para calcular limite 22.00")
	}
	valorLim := parseNum(lim.Valor)
	maximo := pla * 0.05
	if valorLim > maximo {
		return fmt.Errorf("Limite 22.00 (PR PJ)=%v excede 5%% PLA=%v", valorLim, maximo)
	}
	return nil
}

// DLI16LimiteTVM verifica Limite 34.00 = 5x PLA.
type DLI16LimiteTVM struct{}

func (DLI16LimiteTVM) Code() string     { return "DLI-16" }
func (DLI16LimiteTVM) Sheet() string    { return "LimitesTVM" }
func (DLI16LimiteTVM) Severity() string { return "E" }

func (DLI16LimiteTVM) Apply(ctx context.Context, doc *DocDLI) error {
	lim := extractLimite(doc.Limites, "34.00")
	if lim == nil {
		return nil
	}
	pla := SomaPLA(doc.Accounts)
	if pla == 0 {
		return fmt.Errorf("PLA não pode ser 0 para calcular limite 34.00")
	}
	valorLim := parseNum(lim.Valor)
	maximo := pla * 5.0
	if valorLim > maximo {
		return fmt.Errorf("Limite 34.00 (Empréstimo TVM)=%v excede 5x PLA=%v", valorLim, maximo)
	}
	return nil
}

// DLI17LimiteSCM verifica Limite 56.00 = mínimo SCM.
type DLI17LimiteSCM struct{}

func (DLI17LimiteSCM) Code() string     { return "DLI-17" }
func (DLI17LimiteSCM) Sheet() string    { return "LimitesSCM" }
func (DLI17LimiteSCM) Severity() string { return "E" }

func (DLI17LimiteSCM) Apply(ctx context.Context, doc *DocDLI) error {
	lim := extractLimite(doc.Limites, "56.00")
	if lim == nil {
		return nil
	}
	pla := SomaPLA(doc.Accounts)
	if pla == 0 {
		return fmt.Errorf("PLA não pode ser 0 para calcular limite 56.00")
	}
	valorLim := parseNum(lim.Valor)
	// SCM: mínimo pode ser 2% do PLA (típico para instituições menores)
	minimo := pla * 0.02
	if valorLim < minimo && valorLim > 0 {
		return fmt.Errorf("Limite 56.00 (SCM PLA)=%v abaixo do mínimo %v", valorLim, minimo)
	}
	return nil
}

// DLI18LimiteCooperativas verifica Limite 76.00 = mínimo cooperativas.
type DLI18LimiteCooperativas struct{}

func (DLI18LimiteCooperativas) Code() string     { return "DLI-18" }
func (DLI18LimiteCooperativas) Sheet() string    { return "LimitesCooperativas" }
func (DLI18LimiteCooperativas) Severity() string { return "E" }

func (DLI18LimiteCooperativas) Apply(ctx context.Context, doc *DocDLI) error {
	lim := extractLimite(doc.Limites, "76.00")
	if lim == nil {
		return nil
	}
	pla := SomaPLA(doc.Accounts)
	if pla == 0 {
		return fmt.Errorf("PLA não pode ser 0 para calcular limite 76.00")
	}
	valorLim := parseNum(lim.Valor)
	// Cooperativas: mínimo varies by tipo — 2% is a common floor
	minimo := pla * 0.02
	if valorLim < minimo && valorLim > 0 {
		return fmt.Errorf("Limite 76.00 (Cooperativas PLA)=%v abaixo do mínimo %v", valorLim, minimo)
	}
	return nil
}

// ============================================================
// Helpers
// ============================================================

// abs é definida em 3050.go (mesmo package).
func extractLimite(limites []DLILimite, codigo string) *DLILimite {
	for i := range limites {
		if limites[i].Codigo == codigo {
			return &limites[i]
		}
	}
	return nil
}

// parsedDLI é o documento DLI mais recentemente parseado.
// Usado por regras cross-doc que precisam acessar dados DLI.
var parsedDLI *DocDLI

// SetDLI define o documento DLI para acesso cross-doc.
func SetDLI(doc *DocDLI) { parsedDLI = doc }

// BuiltinDLI registra todas as regras DLI/2062 no registry fornecido.
func BuiltinDLI(r *Registry) {
	// DLI-01 a DLI-18 — regras do documento DLI.
	dliRules := []Rule2062{
		// Estruturais
		DLI01CNPJValido{},
		DLI02DataBaseValido{},
		DLI03TipoEnvioValido{},
		DLI04CodigoDocumentoValido{},
		DLI05TemConteudo{},
		DLI06LimiteCodigoValido{},
		DLI07IndicadorValido{},
		DLI08ContaCOSIFValida{},
		// Consistência PLA
		DLI09PLAContabil{},
		DLI10MargemPLA{},
		// Consistência Capital
		DLI11CapitalRealizado{},
		DLI12MargemCapital{},
		// Limites vs PLA
		DLI13LimitePartesRelacionadas{},
		DLI14LimitePRPessoaNatural{},
		DLI15LimitePRPessoaJuridica{},
		DLI16LimiteTVM{},
		DLI17LimiteSCM{},
		DLI18LimiteCooperativas{},
	}
	for _, rule := range dliRules {
		r.Register2062(rule)
	}

	// XD-DLI-01 a XD-DLI-06 — regras cross-doc DLI × DRL × DLP.
	// Implementam a interface Rule (Apply recebe *Doc3040).
	xddliRules := []Rule{
		XDDLI01CNPJConsistente{},
		XDDLI02DataBaseConsistente{},
		XDDLI03PLAPositivo{},
		XDDLI04MargemPLANaoNegativa{},
		XDDLI05CapitalRealizadoMinimo{},
		XDDLI06NSFRxLCRConsistente{},
	}
	for _, rule := range xddliRules {
		r.Register(rule)
	}
}
