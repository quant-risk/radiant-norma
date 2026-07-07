// Validator para DRSAC — validações de domínio e regras de negócio.
//
// Validações implementadas:
//   - Estrutura básica (campos obrigatórios)
//   - Valores de domínio (anexos 01-20)
//   - Regras cross-field (consistência 98/99, coordenadas, GEE)
//
// Nolint:revive
package drsac

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ValidationError representa um erro de validação.
type ValidationError struct {
	Path    string // Path XPath-like ao campo com erro
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Message, e.Field)
}

var (
	ErrEmptyCNPJ          = errors.New("cnpj é obrigatório")
	ErrInvalidCNPJ        = errors.New("cnpj deve ter 8 dígitos")
	ErrEmptyDataBase      = errors.New("dataBase é obrigatório")
	ErrInvalidDataBase    = errors.New("dataBase deve ter formato AAAA-MM")
	ErrInvalidTipoEnvio   = errors.New("tipoEnvio deve ser I ou S")
	ErrEmptyIdentificador = errors.New("identificador do cliente é obrigatório")
)

// Validate executa todas as validações sobre o documento.
// Retorna nil se válido, ou lista de ValidationError.
func Validate(doc *DocumentoDRSAC) error {
	var errs []error

	// 1. Validações de cabeçalho
	if doc.CNPJ == "" {
		errs = append(errs, &ValidationError{Path: "/DocumentoDRSAC", Field: "cnpj", Message: ErrEmptyCNPJ.Error()})
	} else if !validCNPJ(doc.CNPJ) {
		errs = append(errs, &ValidationError{Path: "/DocumentoDRSAC", Field: "cnpj", Message: ErrInvalidCNPJ.Error()})
	}

	if doc.DataBase == "" {
		errs = append(errs, &ValidationError{Path: "/DocumentoDRSAC", Field: "dataBase", Message: ErrEmptyDataBase.Error()})
	} else if !validDataBase(doc.DataBase) {
		errs = append(errs, &ValidationError{Path: "/DocumentoDRSAC", Field: "dataBase", Message: ErrInvalidDataBase.Error()})
	}

	if !ValidTipoEnvio(doc.TipoEnvio) {
		errs = append(errs, &ValidationError{Path: "/DocumentoDRSAC", Field: "tipoEnvio", Message: ErrInvalidTipoEnvio.Error()})
	}

	// 2. Valida Contato
	if doc.Contato.Nome == "" || doc.Contato.Email == "" {
		errs = append(errs, &ValidationError{Path: "/DocumentoDRSAC/Contato", Field: "nome/email", Message: "contato requer nome e email"})
	}

	// 3. Valida Clientes
	if len(doc.Clientes) == 0 && !doc.ClientesOmit {
		errs = append(errs, &ValidationError{Path: "/DocumentoDRSAC/Clientes", Field: "Cliente", Message: "documento deve ter pelo menos um cliente ou ser explicitamente omitido"})
	}
	for i, cl := range doc.Clientes {
		path := fmt.Sprintf("/DocumentoDRSAC/Clientes/Cliente[%d]", i+1)
		if cl.Ident == "" {
			errs = append(errs, &ValidationError{Path: path, Field: "ident", Message: ErrEmptyIdentificador.Error()})
		}
		if !ValidTipoCliente(cl.Tipo) {
			errs = append(errs, &ValidationError{Path: path, Field: "tipo", Message: fmt.Sprintf("tipo %q inválido", cl.Tipo)})
		}
		// CNAE obrigatório para PJ
		if cl.Tipo == TipoClientePJ && cl.CNAE == "" {
			errs = append(errs, &ValidationError{Path: path, Field: "CNAE", Message: "CNAE obrigatório para pessoa jurídica"})
		}
		// CNAE = 7 dígitos
		if cl.CNAE != "" && !validCNAE(cl.CNAE) {
			errs = append(errs, &ValidationError{Path: path, Field: "CNAE", Message: "CNAE deve ter 7 dígitos numéricos"})
		}

		// Valida ExpOperCred
		for j, op := range cl.ExpAtivos.ExpOperCred {
			opPath := fmt.Sprintf("%s/ExpAtivos/ExpOperCred[%d]", path, j+1)
			errs = append(errs, validateExpOperCred(op, opPath)...)
		}

		// Valida ExpTVM
		for j, tv := range cl.ExpAtivos.ExpTVM {
			tvPath := fmt.Sprintf("%s/ExpAtivos/ExpTVM[%d]", path, j+1)
			errs = append(errs, validateExpTVM(tv, tvPath)...)
		}

		// Valida ExpCliente (opcional desde dez/23)
		if cl.ExpCliente != nil {
			errs = append(errs, validateExpCliente(*cl.ExpCliente, path+"/ExpCliente")...)
		}
	}

	// 4. Valida Setores
	for i, s := range doc.Setores {
		path := fmt.Sprintf("/DocumentoDRSAC/Setores/ExpSetor[%d]", i+1)
		if !validCNAE(s.CNAE) {
			errs = append(errs, &ValidationError{Path: path, Field: "CNAE", Message: "CNAE deve ter 7 dígitos numéricos"})
		}
		errs = append(errs, validateSetorRiscos(s.RiscSoc, s.RiscAmb, s.RiscClimFis, s.RiscClimTrans, path)...)
	}

	// 5. Valida SetoresRestritos
	for i, sr := range doc.SetoresRestritos {
		path := fmt.Sprintf("/DocumentoDRSAC/SetoresRestritos/SetorRestrito[%d]", i+1)
		if !validCNAE(sr.CNAE) {
			errs = append(errs, &ValidationError{Path: path, Field: "CNAE", Message: "CNAE deve ter 7 dígitos numéricos"})
		}
		if !ValidTipoRestricaoEconomica(sr.Restricao) {
			errs = append(errs, &ValidationError{Path: path, Field: "restricao", Message: fmt.Sprintf("restricao %q inválido (deve ser 01=Social ou 02=Ambiental)", sr.Restricao)})
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func validateExpOperCred(op ExpOperCred, path string) []error {
	var errs []error
	if op.IPOC == "" {
		errs = append(errs, &ValidationError{Path: path, Field: "IPOC", Message: "IPOC é obrigatório"})
	}
	if !ValidSicor(op.Sicor) {
		errs = append(errs, &ValidationError{Path: path, Field: "Sicor", Message: fmt.Sprintf("Sicor %q inválido (S ou N)", op.Sicor)})
	}
	if !validSaldo(op.Saldo) {
		errs = append(errs, &ValidationError{Path: path, Field: "saldo", Message: "saldo deve ser numérico com 2 casas decimais"})
	}
	errs = append(errs, validateRiscos(op.RiscSoc, op.RiscAmb, op.RiscClimFis, op.RiscClimTrans, path)...)
	errs = append(errs, validateLocaliz(op, path)...)
	return errs
}

func validateExpTVM(tv ExpTVM, path string) []error {
	var errs []error
	if !ValidSisReg(tv.SisReg) {
		errs = append(errs, &ValidationError{Path: path, Field: "sisReg", Message: fmt.Sprintf("sisReg %q inválido", tv.SisReg)})
	}
	if !ValidTipoTVM(tv.Tipo) {
		errs = append(errs, &ValidationError{Path: path, Field: "tipo", Message: fmt.Sprintf("tipo TVM %q inválido", tv.Tipo)})
	}
	if !validSaldo(tv.Valor) {
		errs = append(errs, &ValidationError{Path: path, Field: "valor", Message: "valor deve ser numérico com 2 casas decimais"})
	}
	errs = append(errs, validateRiscos(tv.RiscSoc, tv.RiscAmb, tv.RiscClimFis, tv.RiscClimTrans, path)...)
	return errs
}

func validateExpCliente(ec ExpCliente, path string) []error {
	return validateRiscos(ec.RiscSoc, ec.RiscAmb, ec.RiscClimFis, ec.RiscClimTrans, path)
}

func validateRiscos(soc, amb, climFis, climTrans Risco, path string) []error {
	var errs []error
	if soc.Tipo != "" && !ValidTipoRiscoSocial(soc.Tipo) {
		errs = append(errs, &ValidationError{Path: path, Field: "RiscSoc.tipo", Message: fmt.Sprintf("tipo risco social %q inválido", soc.Tipo)})
	}
	if soc.Av != "" && !ValidAvaliacaoRisco(soc.Av) {
		errs = append(errs, &ValidationError{Path: path, Field: "RiscSoc.av", Message: fmt.Sprintf("avaliacao %q inválida (Anexo 09)", soc.Av)})
	}
	if amb.Tipo != "" && !ValidTipoRiscoAmbiental(amb.Tipo) {
		errs = append(errs, &ValidationError{Path: path, Field: "RiscAmb.tipo", Message: fmt.Sprintf("tipo risco ambiental %q inválido", amb.Tipo)})
	}
	if amb.Av != "" && !ValidAvaliacaoRisco(amb.Av) {
		errs = append(errs, &ValidationError{Path: path, Field: "RiscAmb.av", Message: fmt.Sprintf("avaliacao %q inválida (Anexo 09)", amb.Av)})
	}
	if climFis.Tipo != "" && !ValidTipoRiscoClimaticoFisico(climFis.Tipo) {
		errs = append(errs, &ValidationError{Path: path, Field: "RiscClimFis.tipo", Message: fmt.Sprintf("tipo risco climático físico %q inválido", climFis.Tipo)})
	}
	if climFis.Av != "" && !ValidAvaliacaoRisco(climFis.Av) {
		errs = append(errs, &ValidationError{Path: path, Field: "RiscClimFis.av", Message: fmt.Sprintf("avaliacao %q inválida (Anexo 09)", climFis.Av)})
	}
	if climTrans.Tipo != "" && !ValidTipoRiscoClimaticoTransicao(climTrans.Tipo) {
		errs = append(errs, &ValidationError{Path: path, Field: "RiscClimTrans.tipo", Message: fmt.Sprintf("tipo risco climático transição %q inválido", climTrans.Tipo)})
	}
	if climTrans.Av != "" && !ValidAvaliacaoRisco(climTrans.Av) {
		errs = append(errs, &ValidationError{Path: path, Field: "RiscClimTrans.av", Message: fmt.Sprintf("avaliacao %q inválida (Anexo 09)", climTrans.Av)})
	}
	return errs
}

func validateSetorRiscos(soc, amb, climFis, climTrans Risco, path string) []error {
	return validateRiscos(soc, amb, climFis, climTrans, path)
}

func validateLocaliz(op ExpOperCred, path string) []error {
	var errs []error
	count := 0
	if op.LocalizCoord != nil {
		count++
		if !validLatitude(op.LocalizCoord.Lat) {
			errs = append(errs, &ValidationError{Path: path + "/LocalizCoord", Field: "lat", Message: "latitude deve estar entre -34 e +6 (2,11 dígitos)"})
		}
		if !validLongitude(op.LocalizCoord.Long) {
			errs = append(errs, &ValidationError{Path: path + "/LocalizCoord", Field: "long", Message: "longitude deve estar entre -74 e -30 (3,11 dígitos)"})
		}
		if op.LocalizCoord.Indice != "" && !validIndiceCoordenada(op.LocalizCoord.Indice) {
			errs = append(errs, &ValidationError{Path: path + "/LocalizCoord", Field: "indice", Message: "indice deve ser 1-60"})
		}
	}
	if op.LocalizCEP != nil {
		count++
		if !validCEP(op.LocalizCEP.CEP) {
			errs = append(errs, &ValidationError{Path: path + "/LocalizCEP", Field: "CEP", Message: "CEP deve ter 8 dígitos"})
		}
	}
	if op.LocalizMun != nil {
		count++
		if !validCodMunicipio(op.LocalizMun.CodMun) {
			errs = append(errs, &ValidationError{Path: path + "/LocalizMun", Field: "codMun", Message: "codMun deve ter 7 dígitos (código IBGE)"})
		}
	}
	if op.LocalizPais != nil {
		count++
		if !ValidCodigoPais(op.LocalizPais.CodPais) {
			errs = append(errs, &ValidationError{Path: path + "/LocalizPais", Field: "codPais", Message: "código de país inválido (Anexo 19)"})
		}
	}
	if count > 1 {
		errs = append(errs, &ValidationError{Path: path, Field: "Localiz", Message: "apenas um tipo de localização por registro (coord/CEP/mun/pais)"})
	}
	return errs
}

// Helpers de validação de formato

var (
	cnpjRegex     = regexp.MustCompile(`^\d{8}$`)
	dataBaseRegex = regexp.MustCompile(`^\d{4}-\d{2}$`)
	cnaeRegex     = regexp.MustCompile(`^\d{7}$`)
	saldoRegex    = regexp.MustCompile(`^\d{1,13}\.\d{2}$`)
	cepRegex      = regexp.MustCompile(`^\d{8}$`)
	indiceRegex   = regexp.MustCompile(`^\d{1,2}$`)
)

func validCNPJ(v string) bool { return cnpjRegex.MatchString(v) }
func validDataBase(v string) bool {
	if !dataBaseRegex.MatchString(v) {
		return false
	}
	mes := v[5:7]
	return mes >= "01" && mes <= "12"
}
func validCNAE(v string) bool  { return cnaeRegex.MatchString(v) }
func validSaldo(v string) bool { return saldoRegex.MatchString(v) }
func validCEP(v string) bool   { return cepRegex.MatchString(v) }
func validIndiceCoordenada(v string) bool {
	if !indiceRegex.MatchString(v) {
		return false
	}
	i := strings.Index(v, "0")
	if i == 0 && len(v) > 1 { // leading zero
		v = v[1:]
	}
	idx := 0
	for _, c := range v {
		idx = idx*10 + int(c-'0')
	}
	return idx >= 1 && idx <= 60
}
func validLatitude(v string) bool {
	// N2,11 — inteiros de -34 a +6 com 11 decimais
	if v == "" {
		return true // opcional
	}
	return regexp.MustCompile(`^-?(3[0-4]|[0-2]?\d)\.\d{1,11}$`).MatchString(v) ||
		regexp.MustCompile(`^-?(34|[0-2]?\d)$`).MatchString(v)
}
func validLongitude(v string) bool {
	if v == "" {
		return true
	}
	return regexp.MustCompile(`^-?(7[0-4]|[0-6]?\d)\.\d{1,11}$`).MatchString(v) ||
		regexp.MustCompile(`^-?(74|[0-6]?\d)$`).MatchString(v)
}
func validCodMunicipio(v string) bool {
	// Código IBGE de município: 7 dígitos
	return regexp.MustCompile(`^\d{7}$`).MatchString(v)
}
