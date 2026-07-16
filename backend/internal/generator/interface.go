// Package generator implementa o motor de geração de CADOCs.
// Cada documento (3040, 3050, 4111, etc.) tem seu próprio generator
// que implementa a interface CADOCGenerator.
//
// Arquitetura:
//
//	CanonicalDocument (IF-agnóstico, JSON tipado)
//	        ↓
//	 CADOCGenerator (uma implementação por CADOC)
//	        ↓
//	  GeneratedDoc (XML + SHA256 + FieldMap + Errors)
//
// O CanonicalDocument é o contrato central: LLM nunca escreve XML direto,
// sempre escreve o modelo canônico. O generator serializa para XML.
package generator

import (
	"context"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
)

// CADOCGenerator é a interface que todo gerador de CADOC deve implementar.
// Cada documento BACEN (3040, 3050, 4111, etc.) tem seu próprio generator.
//
// O generator é responsável por:
//   - Transformar CanonicalDocument → XML formatado
//   - Mapear campos canônicos → tags COSIF/XML
//   - Validar campos obrigatórios antes de gerar
//   - Retornar GeneratedDoc com XML + auditoria
//
// O generator NÃO valida o XML gerado (isso é feito pela camada L1-L4
// do Norma Audit, separadamente).
type CADOCGenerator interface {
	// CadocCode retorna o código CADOC que este generator produz.
	// Ex: "3040", "3050", "4111", "2061".
	CadocCode() string

	// Generate transforma o CanonicalDocument no documento XML final.
	// dataBase é a data-base de referência (pode ser diferente da data-base
	// do CanonicalDocument em casos de retificação).
	//
	// Se o CanonicalDocument não tem campos suficientes, retorna erro
	// com a lista de campos faltantes.
	Generate(ctx context.Context, doc *canonical.CanonicalDocument, dataBase time.Time) (*GeneratedDoc, error)

	// RequiredFields retorna a lista de campos obrigatórios para este CADOC.
	// Útil para o wizard de geração: mostra ao usuário o que precisa preencher.
	RequiredFields() []schema.Field

	// SupportedVersions retorna as versões de leiaute suportadas.
	SupportedVersions() []string

	// EstimateComplexity avalia a complexidade do documento.
	// Retorna um score 0.0-1.0 (número de operações, dependências, etc.)
	// Útil para estimar tempo de geração e custo de API.
	EstimateComplexity(doc *canonical.CanonicalDocument) ComplexityScore

	// RootTag retorna a tag raiz canônica do XML gerado.
	// Ex: "Doc3040" para 3040, "Documento4111" para 4111.
	// O Norma Audit (L1 validator) usa este método como fonte canônica
	// para evitar divergência entre generator e validator.
	RootTag() string
}

// GeneratedDoc é o produto de um CADOCGenerator.
// Contém o XML pronto, o ZIP (se BACEN exigir), hash para integridade,
// e o mapa de campos para auditoria.
type GeneratedDoc struct {
	// XML é o documento XML gerado (UTF-8, pretty-printed).
	XML []byte `json:"xml"`

	// ZIP é o arquivo ZIP enviado ao BACEN (se aplicável).
	// Para alguns CADOCs, o XML é enviado diretamente; para outros,
	// é compactado. Se nil, usar XML diretamente.
	ZIP []byte `json:"zip,omitempty"`

	// SHA256 é o hash SHA-256 do conteúdo enviado (XML ou ZIP).
	SHA256 string `json:"sha256"`

	// CadocCode é o código do documento (ex: "3040").
	CadocCode string `json:"cadoc_code"`

	// VersaoLayout é a versão do leiaute usada.
	VersaoLayout string `json:"versao_layout"`

	// DataBase é a data-base do documento.
	DataBase time.Time `json:"data_base"`

	// FieldMap é o mapa de campos gerados (auditoria).
	FieldMap []canonical.FieldMapping `json:"field_map"`

	// Errors são erros de geração (campos obrigatórios ausentes, etc).
	// Diferente de ValidationError: aqui é erro de geração, não de validação.
	Errors []GenError `json:"errors,omitempty"`

	// Warnings são avisos não-bloqueantes (ex: campo opcional ausente).
	Warnings []GenWarning `json:"warnings,omitempty"`

	// Metadata é informação de geração (tempo, versão do generator).
	Metadata GenMetadata `json:"metadata"`
}

// GenError é um erro de geração (não de validação).
// Indica que o documento não pôde ser gerado.
type GenError struct {
	// Campo é o campo que causou o erro (pode ser vazio para erros genéricos).
	Campo string `json:"campo,omitempty"`

	// Code é o código de erro (ex: "MISSING_FIELD", "INVALID_FORMAT").
	Code string `json:"code"`

	// Message é a mensagem legível.
	Message string `json:"message"`
}

// GenWarning é um aviso não-bloqueante de geração.
type GenWarning struct {
	Campo   string `json:"campo,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GenMetadata é informação de auditoria da geração.
type GenMetadata struct {
	// GeneratorVersion é a versão do generator que produziu o documento.
	GeneratorVersion string `json:"generator_version"`

	// GeneratedAt é o timestamp de geração.
	GeneratedAt time.Time `json:"generated_at"`

	// DurationMs é o tempo de geração em milissegundos.
	DurationMs int64 `json:"duration_ms"`

	// SourceAdapter é o conector que alimentou o CanonicalDocument.
	SourceAdapter string `json:"source_adapter,omitempty"`
}

// ComplexityScore avalia a complexidade de um documento a ser gerado.
type ComplexityScore struct {
	// Score é 0.0 (trivial) a 1.0 (muito complexo).
	Score float64 `json:"score"`

	// NumOperacoes é o número estimado de operações no documento.
	NumOperacoes int `json:"num_operacoes"`

	// NumParticipantes é o número de participantes.
	NumParticipantes int `json:"num_participantes"`

	// EstimatedAPICalls é o número estimado de chamadas de API necessárias
	// (para conectores que precisam buscar dados).
	EstimatedAPICalls int `json:"estimated_api_calls"`

	// EstimatedTimeMs é o tempo estimado de geração em ms.
	EstimatedTimeMs int64 `json:"estimated_time_ms"`
}

// Registry é o registro global de generators.
// Permite lookup por código CADOC.
type Registry struct {
	generators map[string]CADOCGenerator
}

// NewRegistry cria um novo registry vazio.
//
// Sprint 57 v3.36.3: registro dos generators foi separado para evitar
// import cycle (cada subpacote gen* importa este pacote generator, e
// este pacote não pode importar os subpacotes sem criar ciclo).
//
// O caller deve importar os subpacotes e popular via Register, ou usar
// o helper RegisterDefaults. Em cmd/api/main.go:
//
//	import (
//	    "github.com/fortvna/radiant-norma/backend/internal/generator"
//	    "github.com/fortvna/radiant-norma/backend/internal/generator/gen3040"
//	    ...
//	)
//	registry := generator.NewRegistry()
//	generator.RegisterDefaults(registry, []generator.CADOCGenerator{
//	    gen3040.New(), gen3050.New(), gen4111.New(),
//	    gen2030.New(), gen2060.New(), gen2061.New(),
//	    gen2062.New(), gen2070.New(), gen2160.New(),
//	    gen2170.New(),
//	})
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[string]CADOCGenerator),
	}
}

// RegisterDefaults popula o registry com a lista de generators passada.
// Cada generator implementa CADOCGenerator e seu CadocCode() é usado como chave.
//
// Esta função existe para evitar import cycle: os subpacotes gen* importam
// o pacote generator (para verificar a interface), mas generator não pode
// importar diretamente os subpacotes. O caller (ex: cmd/api/main.go) que
// já importa ambos age como glue.
func RegisterDefaults(r *Registry, generators []CADOCGenerator) {
	if r == nil {
		return
	}
	for _, g := range generators {
		if g == nil {
			continue
		}
		r.Register(g)
	}
}

// Register adiciona um generator ao registry.
func (r *Registry) Register(g CADOCGenerator) {
	r.generators[g.CadocCode()] = g
}

// Get retorna o generator para um código CADOC.
// Retorna nil se não encontrado.
func (r *Registry) Get(cadocCode string) CADOCGenerator {
	return r.generators[cadocCode]
}

// List retorna todos os generators registrados.
func (r *Registry) List() []CADOCGenerator {
	var out []CADOCGenerator
	for _, g := range r.generators {
		out = append(out, g)
	}
	return out
}

// IsRegistered verifica se um CADOC tem generator registrado.
func (r *Registry) IsRegistered(cadocCode string) bool {
	_, ok := r.generators[cadocCode]
	return ok
}
