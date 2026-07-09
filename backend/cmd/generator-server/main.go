// cmd/generator-server — CLI standalone para geração de CADOCs.
//
// Uso:
//
//	# Gerar um 3040 a partir de JSON via stdin ou arquivo:
//	go run ./cmd/generator-server generate -cadoc=3040 -f data.json
//
//	# Listar generators disponíveis:
//	go run ./cmd/generator-server list
//
//	# Server mode:
//	go run ./cmd/generator-server serve -addr=:8080
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	gen3040 "github.com/fortvna/radiant-norma/backend/internal/generator/gen3040"
	"github.com/fortvna/radiant-norma/backend/internal/ingest"
	"github.com/shopspring/decimal"
)

var genRegistry = generator.NewRegistry()

func init() {
	genRegistry.Register(gen3040.New())
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		cmdGenerate(os.Args[2:])
	case "list":
		cmdList(os.Args[2:])
	case "adapters":
		cmdAdapters(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`generator-server — Norma Generator CLI

Comandos:
  generate  -cadoc=<code> [-i <json> | -f <file>]  Gera um CADOC
  list                           Lista generators disponíveis
  adapters                       Lista conectores disponíveis

Flags (generate):
  -cadoc        Código CADOC (ex: 3040)
  -data-base    Data-base (AAAAMM, default: mês atual)
  -i            JSON inline
  -f            Arquivo JSON de input
  -o            Arquivo de output (default: stdout)
`)
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	cadoc := fs.String("cadoc", "3040", "código CADOC")
	dataBase := fs.String("data-base", "", "data-base (AAAAMM)")
	jsonInput := fs.String("i", "", "JSON inline")
	fileInput := fs.String("f", "", "arquivo JSON de input")
	output := fs.String("o", "", "arquivo de output (default: stdout)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Carrega JSON.
	var data []byte
	switch {
	case *jsonInput != "":
		data = []byte(*jsonInput)
	case *fileInput != "":
		var err error
		data, err = os.ReadFile(*fileInput)
		if err != nil {
			log.Fatalf("ler arquivo: %v", err)
		}
	default:
		var err error
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("ler stdin: %v", err)
		}
		if len(data) == 0 {
			log.Fatal("informe -i <json> ou -f <file> ou pipe stdin")
		}
	}

	// Parse input.
	var doc CanonicalInput
	if err := json.Unmarshal(data, &doc); err != nil {
		log.Fatalf("parsear JSON: %v", err)
	}

	dbTime := parseDataBase(*dataBase)
	canonicalDoc := toCanonical(doc, *cadoc, dbTime)

	g := genRegistry.Get(*cadoc)
	if g == nil {
		log.Fatalf("generator %s não encontrado", *cadoc)
	}

	generated, err := g.Generate(context.Background(), canonicalDoc, dbTime)
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	if *output != "" {
		if err := os.WriteFile(*output, generated.XML, 0600); err != nil {
			log.Fatalf("escrever output: %v", err)
		}
		fmt.Printf("Gerado: %s (%d bytes, SHA256=%s)\n", *output, len(generated.XML), generated.SHA256)
	} else {
		os.Stdout.Write(generated.XML)
	}
}

func cmdList(args []string) {
	_ = args
	gens := genRegistry.List()
	fmt.Printf("Generators disponíveis (%d):\n", len(gens))
	for _, g := range gens {
		complexity := g.EstimateComplexity(&canonical.CanonicalDocument{CadocCode: canonical.CadocType(g.CadocCode())})
		fmt.Printf("  %s  versions=%v  complexity=%.2f\n",
			g.CadocCode(), g.SupportedVersions(), complexity.Score)
	}
}

func cmdAdapters(args []string) {
	_ = args
	adapters := ingest.ListAdapters()
	fmt.Printf("Conectores disponíveis (%d):\n", len(adapters))
	for _, a := range adapters {
		fields := a.DescribeFields("3040")
		fmt.Printf("  %s (%s)  fields=%d\n", a.Name(), a.Type(), len(fields))
	}
}

// parseDataBase parses AAAAMM string to time.Time (first day of month).
// Returns zero time if parsing fails.
func parseDataBase(s string) time.Time {
	if s == "" {
		t := time.Now()
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	y, m := 2026, 1
	if n, _ := fmt.Sscanf(s[:min(len(s), 4)], "%d", &y); n != 1 {
		return time.Time{}
	}
	if len(s) >= 6 {
		fmt.Sscanf(s[4:6], "%d", &m)
	}
	if m < 1 || m > 12 {
		return time.Time{}
	}
	return time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
}

// --- Tipos auxiliares ---

type CanonicalInput struct {
	IFID      string         `json:"if_id"`
	CNPJ      string         `json:"cnpj"`
	NomeIF    string         `json:"nome_if"`
	DataBase  string         `json:"data_base"`
	Source    string         `json:"source,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
	Operacoes []struct {
		ID              string  `json:"id"`
		Modalidade      string  `json:"modalidade"`
		ValorPrincipal  float64 `json:"valor_principal"`
		TipoPessoa      string  `json:"tipo_pessoa,omitempty"`
		UF              string  `json:"uf,omitempty"`
		ClassificacaoIF string  `json:"classificacao_if,omitempty"`
		NumeroContrato  string  `json:"numero_contrato,omitempty"`
	} `json:"operacoes,omitempty"`
}

func toCanonical(in CanonicalInput, cadoc string, dbTime time.Time) *canonical.CanonicalDocument {
	doc := canonical.NewCanonical(in.IFID, dbTime, canonical.CadocType(cadoc))
	doc.Header.CNPJ = in.CNPJ
	doc.Header.NomeIF = in.NomeIF
	doc.Header.DataHoraGeracao = time.Now()
	doc.Metadata.SourceAdapter = in.Source

	for k, v := range in.Fields {
		doc.Extra[k] = v
	}

	for _, op := range in.Operacoes {
		doc.Operacoes = append(doc.Operacoes, canonical.Operacao{
			ID:              op.ID,
			Modalidade:      op.Modalidade,
			TipoPessoa:      op.TipoPessoa,
			UF:              op.UF,
			ClassificacaoIF: op.ClassificacaoIF,
			NumeroContrato:  op.NumeroContrato,
			ValorPrincipal: canonical.Money{
				Valor: decimal.NewFromFloat(op.ValorPrincipal),
				Moeda: "BRL",
			},
		})
	}

	return doc
}
