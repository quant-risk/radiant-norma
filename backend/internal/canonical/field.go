package canonical

// FieldMapping registra como um campo COSIF foi preenchido.
// Útil para auditoria: mostra a origem de cada valor no XML gerado.
type FieldMapping struct {
	// CampoCOSIF é o nome do campo no XML COSIF (ex: "NRO_CONTRATO_BCB").
	CampoCOSIF string `json:"campo_cosif"`

	// CampoXML é o nome exato da tag no XML gerado (pode diferir do COSIF
	// por convenções de nomenclatura do gerador).
	CampoXML string `json:"campo_xml"`

	// ValorOriginal é o valor lido da fonte (sem formatação).
	ValorOriginal any `json:"valor_original,omitempty"`

	// ValorFormatado é o valor após transformação (ex: CNPJ com máscara).
	ValorFormatado string `json:"valor_formatado,omitempty"`

	// Fonte identifica de onde veio o valor.
	// Valores: "MANUAL", "FILE", "API", "DB", "MCP", "CALCULATED".
	Fonte FieldSource `json:"fonte"`

	// RowIndex é o índice da linha (para dados tabulares como planilhas).
	RowIndex int `json:"row_index,omitempty"`

	// LinhaOrigem é uma referência legível à origem (ex: "Planilha!A5").
	LinhaOrigem string `json:"linha_origem,omitempty"`

	// Transformacoes lista as transformações aplicadas (ex: "uppercase",
	// "CNPJmask", "dateFormat").
	Transformacoes []string `json:"transformacoes,omitempty"`
}

// FieldSource indica a origem de um campo no CanonicalDocument.
type FieldSource string

const (
	// FonteManual indica que o campo foi inserido manualmente pelo usuário.
	FontSourceManual FieldSource = "MANUAL"

	// FonteFile indica que o campo veio de um arquivo (XLSX, CSV, etc.).
	FonteFile FieldSource = "FILE"

	// FonteAPI indica que o campo veio de uma API.
	FonteAPI FieldSource = "API"

	// FonteDB indica que o campo veio de um banco de dados.
	FonteDB FieldSource = "DB"

	// FonteMCP indica que o campo veio de um agente IA via MCP.
	FonteMCP FieldSource = "MCP"

	// FonteCalculated indica que o campo foi calculado pelo motor.
	FonteCalculated FieldSource = "CALCULATED"
)

// MappingRecord é uma coleção de FieldMappings para um documento.
// Útil para exportar como CSV/JSON para auditoria.
type MappingRecord struct {
	DocID        string         `json:"doc_id"`
	CadocCode    string         `json:"cadoc_code"`
	DataBase     string         `json:"data_base"`
	Source       string         `json:"source"`
	Fields       []FieldMapping `json:"fields"`
	TotalFields  int            `json:"total_fields"`
	MappedCount  int            `json:"mapped_count"`
	MissingCount int            `json:"missing_count"`
}

// NewMappingRecord cria um MappingRecord para um documento.
func NewMappingRecord(docID, cadocCode, dataBase, source string, fields []FieldMapping) MappingRecord {
	missing := 0
	for _, f := range fields {
		if f.ValorOriginal == nil && f.ValorFormatado == "" {
			missing++
		}
	}
	return MappingRecord{
		DocID:        docID,
		CadocCode:    cadocCode,
		DataBase:     dataBase,
		Source:       source,
		Fields:       fields,
		TotalFields:  len(fields),
		MappedCount:  len(fields) - missing,
		MissingCount: missing,
	}
}
