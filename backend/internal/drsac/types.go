// Package drsac implementa parsing e validação do CADOC 2030 —
// Documento de Riscos Social, Ambiental e Climático.
//
// Estrutura:
//   - DocumentoDRSAC (root)
//   - Contato
//   - Clientes[] > Cliente > ExpAtivos > ExpOperCred/ExpTVM
//   - Setores[] > ExpSetor
//   - SetoresRestritos[] > SetorRestrito
//
// Fontes: 2030-DRSAC/Leiaute_DRSAC.xlsx + Instrucoes_Preenchimento_DRSAC.pdf
// XSD oficial: PENDENTE — solicitar ao BACEN (Sprint 47)
package drsac

import "time"

// DocumentoDRSAC é o root element do documento.
type DocumentoDRSAC struct {
	CNPJ             string          `xml:"cnpj,attr"`            // A8 — 8 primeiros dígitos do CNPJ
	DataBase         string          `xml:"dataBase,attr"`        // A7 — AAAA-MM
	CodigoDoc        string          `xml:"codigoDocumento,attr"` // "2030"
	TipoEnvio        string          `xml:"tipoEnvio,attr"`       // Anexo 01: I=Inclusão, S=Substituição
	Contato          Contato         `xml:"Contato"`
	Clientes         []Cliente       `xml:"Clientes>Cliente"`
	ClientesOmit     bool            `xml:"-"` // true se Clientes for omitido (opcional desde dez/23)
	Setores          []Setor         `xml:"Setores>ExpSetor"`
	SetoresOmit      bool            `xml:"-"` // true se Setores for omitido
	SetoresRestritos []SetorRestrito `xml:"SetoresRestritos>SetorRestrito"`
}

// Contato representa os dados do responsável pelo envio.
type Contato struct {
	Nome  string `xml:"nome,attr"`  // Nome completo
	Fone  string `xml:"fone,attr"`  // Telefone
	Email string `xml:"email,attr"` // E-mail
}

// Cliente representa um cliente/devedor no documento.
type Cliente struct {
	Ident      string      `xml:"ident,attr"`                // CNPJ(14), CPF(11), ou código interno (até 14 chars)
	Tipo       string      `xml:"tipo,attr"`                 // Anexo 05: 01=PF, 02=PJ, 03-06=especiais
	CNAE       string      `xml:"CNAE,attr,omitempty"`       // N7 — apenas para PJ (tipo=02)
	VersaoCNAE string      `xml:"versaoCNAE,attr,omitempty"` // N2 — versão do CNAE
	ExpAtivos  ExpAtivos   `xml:"ExpAtivos"`
	ExpCliente *ExpCliente `xml:"ExpCliente,omitempty"` // opcional desde dez/23
}

// ExpAtivos agrega as exposições em ativos (operações de crédito e TVM).
type ExpAtivos struct {
	ExpOperCred []ExpOperCred `xml:"ExpOperCred,omitempty"`
	ExpTVM      []ExpTVM      `xml:"ExpTVM,omitempty"`
}

// ExpOperCred representa exposição em operação de crédito.
type ExpOperCred struct {
	IPOC  string `xml:"IPOC,attr"`  // A67 — ID da operação no SCR
	Sicor string `xml:"Sicor,attr"` // Anexo 04: S=Sim, N=Não
	Saldo string `xml:"saldo,attr"` // N13,2 — saldo em reais

	// Riscos
	RiscSoc       Risco `xml:"RiscSoc,omitempty"`
	RiscAmb       Risco `xml:"RiscAmb,omitempty"`
	RiscClimFis   Risco `xml:"RiscClimFis,omitempty"`
	RiscClimTrans Risco `xml:"RiscClimTrans,omitempty"`

	// Contribuição e mitigadores
	ContribPositiva   *ContribPositiva `xml:"ContribPositiva,omitempty"`
	MitRiscClimFis    *MitRiscClimFis  `xml:"MitRiscClimFis,omitempty"`
	HistAbsorEmissGEE *GEEInfo         `xml:"HistAbsorEmissGEE,omitempty"`
	CompEmissGEE      *GEEComp         `xml:"CompEmissGEE,omitempty"`

	// Localização (4 alternativas — só uma por registro)
	LocalizCoord *LocalizCoord `xml:"LocalizCoord,omitempty"`
	LocalizCEP   *LocalizCEP   `xml:"LocalizCEP,omitempty"`
	LocalizMun   *LocalizMun   `xml:"LocalizMun,omitempty"`
	LocalizPais  *LocalizPais  `xml:"LocalizPais,omitempty"`
}

// ExpTVM representa exposição em título e valor mobiliário.
type ExpTVM struct {
	SisReg string `xml:"sisReg,attr"` // Anexo 02: B3, CERC, CSDBR, Outro
	Tipo   string `xml:"tipo,attr"`   // Anexo 03: CPR, CDCA, CRA, DEB, Outro
	Ident  string `xml:"ident,attr"`  // Identificação do TVM
	Valor  string `xml:"valor,attr"`  // N13,2 — valor em reais

	// Mesmos child tags que ExpOperCred
	RiscSoc           Risco            `xml:"RiscSoc,omitempty"`
	RiscAmb           Risco            `xml:"RiscAmb,omitempty"`
	RiscClimFis       Risco            `xml:"RiscClimFis,omitempty"`
	RiscClimTrans     Risco            `xml:"RiscClimTrans,omitempty"`
	ContribPositiva   *ContribPositiva `xml:"ContribPositiva,omitempty"`
	MitRiscClimFis    *MitRiscClimFis  `xml:"MitRiscClimFis,omitempty"`
	HistAbsorEmissGEE *GEEInfo         `xml:"HistAbsorEmissGEE,omitempty"`
	CompEmissGEE      *GEEComp         `xml:"CompEmissGEE,omitempty"`
}

// Risco representa uma avaliação de risco (tipo + avaliação).
type Risco struct {
	Tipo string `xml:"tipo,attr"` // Código do tipo (anexos 06, 07, 08, 18)
	Av   string `xml:"av,attr"`   // Avaliação (Anexo 09): 01-04, 98, 99
}

// ContribPositiva representa a classificação de contribuição positiva.
type ContribPositiva struct {
	Enquad string `xml:"enquad,attr"` // Anexo 10: 01, 02, 03, 98, 99
}

// MitRiscClimFis representa a existência de mitigador de risco climático físico.
type MitRiscClimFis struct {
	Exist string `xml:"exist,attr"` // Anexo 11: 01=Existe, 02=Não existe, 98, 99
}

// GEEInfo representa informações de absorção/emissão de GEE (histórico ou expectativa).
type GEEInfo struct {
	Tipo  string `xml:"tipo,attr"`  // Anexo 12 (histórico) ou Anexo 13 (expectativa)
	Sit   string `xml:"sit,attr"`   // Anexo 15: 01=Absorção, 02=Emissão, 98, 99
	Valor string `xml:"valor,attr"` // N — valor em ton/CO2e/ano
}

// GEEComp representa compensação de emissão de GEE.
type GEEComp struct {
	Tipo  string `xml:"tipo,attr"`  // Anexo 14: tipos de compensação
	Sit   string `xml:"sit,attr"`   // Anexo 15: 01=Absorção, 02=Emissão, 98, 99
	Valor string `xml:"valor,attr"` // N — valor em ton/CO2e/ano
}

// LocalizCoord representa localização por coordenadas geográficas.
type LocalizCoord struct {
	Lat     string `xml:"lat,attr"`               // N2,11 — latitude (-34° a +06°)
	Long    string `xml:"long,attr"`              // N3,11 — longitude (-074° a -030°)
	Alt     string `xml:"alt,attr,omitempty"`     // N1,4,2 — altitude (-100m a 3000m)
	Indice  string `xml:"indice,attr"`            // N2 — índice sequencial (1-60)
	DataObs string `xml:"dataObs,attr,omitempty"` // Data da observação (AAAAMM)
}

// LocalizCEP representa localização por CEP.
type LocalizCEP struct {
	CEP     string `xml:"CEP,attr"`               // N8 — CEP (formato XXXXXXX)
	Indice  string `xml:"indice,attr"`            // N2 — índice sequencial (1-60)
	DataObs string `xml:"dataObs,attr,omitempty"` // Data da observação (AAAAMM)
}

// LocalizMun representa localização por município.
type LocalizMun struct {
	CodMun  string `xml:"codMun,attr"`            // Código IBGE do município
	Indice  string `xml:"indice,attr"`            // N2 — índice sequencial (1-60)
	DataObs string `xml:"dataObs,attr,omitempty"` // Data da observação (AAAAMM)
}

// LocalizPais representa localização por país.
type LocalizPais struct {
	CodPais string `xml:"codPais,attr"` // Anexo 19: código ISO 3166-1 numérico
	Indice  string `xml:"indice,attr"`  // N2 — índice sequencial (1-60)
}

// ExpCliente representa a exposição de nível cliente.
type ExpCliente struct {
	RiscSoc            Risco               `xml:"RiscSoc,omitempty"`
	RiscAmb            Risco               `xml:"RiscAmb,omitempty"`
	RiscClimFis        Risco               `xml:"RiscClimFis,omitempty"`
	RiscClimTrans      Risco               `xml:"RiscClimTrans,omitempty"`
	DetContribPositiva *DetContribPositiva `xml:"DetContribPositiva,omitempty"`
	HistAbsorEmissGEE  *GEEInfo            `xml:"HistAbsorEmissGEE,omitempty"`
	ExpAbsorEmissGEE   *GEEInfo            `xml:"ExpAbsorEmissGEE,omitempty"` // Anexo 13
	CompEmissGEE       *GEEComp            `xml:"CompEmissGEE,omitempty"`
	AgrMit             *AgrMit             `xml:"AgrMit,omitempty"`
}

// DetContribPositiva detalhe da contribuição positiva no nível cliente.
type DetContribPositiva struct {
	Enquad    string `xml:"enquad,attr"`    // Anexo 10
	SaldoCred string `xml:"saldoCred,attr"` // N13,2
	SaldoTVM  string `xml:"saldoTVM,attr"`  // N13,2
}

// AgrMit representa fatores agravantes ou mitigadores.
type AgrMit struct {
	Tipo string `xml:"tipo,attr"` // Anexo 16: 01-10
	Sit  string `xml:"sit,attr"`  // Anexo 17: 01=Existe, 02=Não existe, 98, 99
}

// Setor representa exposição agregada por setor CNAE.
type Setor struct {
	CNAE          string `xml:"CNAE,attr"`       // N7 — código CNAE
	VersaoCNAE    string `xml:"versaoCNAE,attr"` // N2 — versão do CNAE
	RiscSoc       Risco  `xml:"RiscSoc,omitempty"`
	RiscAmb       Risco  `xml:"RiscAmb,omitempty"`
	RiscClimFis   Risco  `xml:"RiscClimFis,omitempty"`
	RiscClimTrans Risco  `xml:"RiscClimTrans,omitempty"`
}

// SetorRestrito representa setor com restrição econômica.
type SetorRestrito struct {
	CNAE       string `xml:"CNAE,attr"`       // N7
	VersaoCNAE string `xml:"versaoCNAE,attr"` // N2
	Restricao  string `xml:"restricao,attr"`  // Anexo 20: 01=Social, 02=Ambiental
}

// Metadata retorna metadados do documento (usado pelo schema registry).
func (d *DocumentoDRSAC) Metadata() (code string, base time.Time, sentAt time.Time) {
	return "2030", time.Time{}, time.Now()
}
