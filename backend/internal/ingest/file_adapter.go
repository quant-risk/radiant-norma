// Package ingest — FileAdapter implementation.
//
// Parses XLSX and CSV files into CanonicalDocument.
// Column mapping is configured via SourceConfig.File.Mapping.
// Auto-detection of common column names when mapping is not provided.
package ingest

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

// Fetch parses an XLSX or CSV file and returns a CanonicalDocument.
func (a *FileAdapter) Fetch(ctx context.Context, cfg SourceConfig, cadocCode string, dataBase time.Time) (*canonical.CanonicalDocument, error) {
	if cfg.File == nil {
		return nil, fmt.Errorf("file config is required")
	}

	f, err := os.Open(cfg.File.Path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// 50MB limit.
	data, err := io.ReadAll(io.LimitReader(f, 50<<20))
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	doc := canonical.NewCanonical(cfg.Name, dataBase, canonical.CadocType(cadocCode))
	doc.Metadata.SourceAdapter = string(SourceFile)
	doc.Metadata.SourceRef = cfg.File.Path

	switch strings.ToLower(cfg.File.Format) {
	case "xlsx", "xls":
		if err := parseXLSX(data, cfg, doc); err != nil {
			return nil, fmt.Errorf("parse xlsx: %w", err)
		}
	case "csv":
		if err := parseCSV(data, cfg, doc); err != nil {
			return nil, fmt.Errorf("parse csv: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: xlsx, csv)", cfg.File.Format)
	}

	return doc, nil
}

// --- XLSX parsing ---

func parseXLSX(data []byte, cfg SourceConfig, doc *canonical.CanonicalDocument) error {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("excelize open: %w", err)
	}
	defer f.Close()

	sheet := cfg.File.Sheet
	if sheet == "" {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return fmt.Errorf("no sheets found in XLSX")
		}
		sheet = sheets[0]
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("get rows: %w", err)
	}

	if len(rows) < 2 && cfg.File.HasHeader {
		return fmt.Errorf("XLSX must have at least header + 1 data row")
	}

	header := rows[0]
	hasHeader := cfg.File.HasHeader

	if !hasHeader {
		header = makeHeader(len(rows[0]))
	}

	var operations []canonical.Operacao
	var headerFields map[string]string

	dataRowStart := 1
	if !hasHeader {
		dataRowStart = 0
	}

	for _, row := range rows[dataRowStart:] {
		if len(row) == 0 || (len(row) == 1 && row[0] == "") {
			continue
		}
		row = padRow(row, len(header))
		rowMap := rowByIndex(header, row)

		if !hasHeader && isHeaderRow(rowMap) && len(operations) == 0 {
			headerFields = rowMap
			applyHeaderFields(headerFields, doc)
			continue
		}

		op, ok := tryParseOperacao(rowMap)
		if ok {
			operations = append(operations, op)
		}
	}

	// If no header fields detected, treat all rows as operations.
	if headerFields == nil {
		headerFields = make(map[string]string)
		for _, row := range rows[1:] {
			if len(row) == 0 {
				continue
			}
			row = padRow(row, len(header))
			rowMap := rowByIndex(header, row)
			op, ok := tryParseOperacao(rowMap)
			if ok {
				operations = append(operations, op)
			}
		}
	}

	doc.Operacoes = operations

	if doc.Header.CNPJ == "" {
		doc.Header.CNPJ = headerFields["cnpj"]
	}

	return nil
}

// --- CSV parsing ---

func parseCSV(data []byte, cfg SourceConfig, doc *canonical.CanonicalDocument) error {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // Allow variable length

	records, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("csv read: %w", err)
	}

	if len(records) < 2 && cfg.File.HasHeader {
		return fmt.Errorf("CSV must have at least header + 1 data row")
	}

	var header []string
	var dataRows [][]string

	if cfg.File.HasHeader {
		header = records[0]
		dataRows = records[1:]
	} else {
		header = makeHeader(len(records[0]))
		dataRows = records
	}

	var operations []canonical.Operacao
	var headerFields map[string]string
	foundHeader := false

	for _, row := range dataRows {
		if len(row) == 0 || (len(row) == 1 && row[0] == "") {
			continue
		}
		row = padRow(row, len(header))

		var rowMap map[string]string
		if cfg.File.HasHeader {
			rowMap = rowByIndex(header, row)
			// When hasHeader=true, the first data row carries IF metadata
			// (cnpj, nome_if) alongside operation fields. Extract IF fields
			// before trying to parse as operation, so we don't skip it.
			if !foundHeader {
				applyHeaderFields(rowMap, doc)
				foundHeader = true
			}
		} else {
			// No header: use positional mapping to get meaningful keys.
			rowMap = rowByPos(row)
			if isHeaderRow(rowMap) && len(operations) == 0 {
				applyHeaderFields(rowMap, doc)
				foundHeader = true
				continue
			}
		}

		op, ok := tryParseOperacao(rowMap)
		if ok {
			operations = append(operations, op)
		}
	}

	// Re-process rows only when we never found a dedicated header row
	// (hasHeader=false and the first row had no operation fields).
	// Also guard with len(operations)==0 to avoid re-processing a row
	// that was already successfully parsed as an operation in the main loop.
	if !foundHeader && headerFields == nil && len(operations) == 0 {
		headerFields = make(map[string]string)
		for _, row := range dataRows {
			if len(row) == 0 {
				continue
			}
			row = padRow(row, len(header))
			// Use positional mapping when hasHeader=false.
			rowMap := rowByPos(row)
			applyHeaderFields(rowMap, doc)
			op, ok := tryParseOperacao(rowMap)
			if ok {
				operations = append(operations, op)
			}
		}
	}

	doc.Operacoes = operations

	return nil
}

// --- Column helpers ---

func makeHeader(n int) []string {
	h := make([]string, n)
	for i := 0; i < n; i++ {
		h[i] = colName(i)
	}
	return h
}

func colName(i int) string {
	name := ""
	for i >= 0 {
		name = string(rune('A'+i%26)) + name
		i = i/26 - 1
	}
	return name
}

func padRow(row []string, size int) []string {
	if len(row) >= size {
		return row[:size]
	}
	padded := make([]string, size)
	copy(padded, row)
	for i := len(row); i < size; i++ {
		padded[i] = ""
	}
	return padded
}

// positional keys for no-header CSV (column order maps to Operacao fields).
var posKeys = []string{
	"cnpj", "nomeif", "modalidade", "valor", "uf",
	"tipopessoa", "classificacaoif", "contrato",
	// index 8+ are extras
}

// rowByPos creates a row map using positional keys (for no-header CSV).
func rowByPos(row []string) map[string]string {
	m := make(map[string]string)
	for i, v := range row {
		if i < len(posKeys) {
			m[posKeys[i]] = strings.TrimSpace(v)
		} else {
			m[fmt.Sprintf("col%d", i)] = strings.TrimSpace(v)
		}
	}
	return m
}

// rowByIndex creates a normalized row map by column index.
// key = normalized header name, value = cell value.
func rowByIndex(header, row []string) map[string]string {
	m := make(map[string]string)
	for i, h := range header {
		if i < len(row) {
			m[normalizeHeader(h)] = strings.TrimSpace(row[i])
		}
	}
	return m
}

// normalizeHeader normalizes a column header: strip accents, lowercase, underscore.
func normalizeHeader(col string) string {
	s := stripAccents(col)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			return r
		}
		return '_'
	}, s)
	s = strings.ToLower(s)
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return strings.Trim(s, "_")
}

// stripAccents removes common Portuguese accents.
func stripAccents(s string) string {
	s = strings.ReplaceAll(s, "ã", "a")
	s = strings.ReplaceAll(s, "á", "a")
	s = strings.ReplaceAll(s, "à", "a")
	s = strings.ReplaceAll(s, "â", "a")
	s = strings.ReplaceAll(s, "é", "e")
	s = strings.ReplaceAll(s, "ê", "e")
	s = strings.ReplaceAll(s, "í", "i")
	s = strings.ReplaceAll(s, "ó", "o")
	s = strings.ReplaceAll(s, "ô", "o")
	s = strings.ReplaceAll(s, "õ", "o")
	s = strings.ReplaceAll(s, "ú", "u")
	s = strings.ReplaceAll(s, "ç", "c")
	return s
}

// isHeaderRow returns true if the row looks like IF metadata (not an operation).
// Callers should only invoke this when hasHeader=false (no explicit header row).
func isHeaderRow(row map[string]string) bool {
	// If row has modality or valor → it's an operation row.
	if row["modalidade"] != "" || row["mod"] != "" {
		return false
	}
	if row["valorprincipal"] != "" || row["vlrprincipal"] != "" || row["valor"] != "" {
		return false
	}
	return true
}

// applyHeaderFields extracts IF-level fields into doc.Header.
func applyHeaderFields(fields map[string]string, doc *canonical.CanonicalDocument) {
	if v := fields["cnpj"]; v != "" {
		doc.Header.CNPJ = cleanCNPJ(v)
	}
	if v := fields["nomeif"]; v != "" {
		doc.Header.NomeIF = v
	}
	if v := fields["nome_if"]; v != "" && doc.Header.NomeIF == "" {
		doc.Header.NomeIF = v
	}
}

func cleanCNPJ(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// tryParseOperacao attempts to parse a row as canonical.Operacao.
func tryParseOperacao(row map[string]string) (canonical.Operacao, bool) {
	// Need at least one operation indicator.
	hasData := row["modalidade"] != "" || row["mod"] != "" ||
		row["valorprincipal"] != "" || row["vlrprincipal"] != "" || row["valor"] != "" ||
		row["id"] != "" || row["operacao_id"] != ""
	if !hasData {
		return canonical.Operacao{}, false
	}

	op := canonical.Operacao{
		ID:              coalesce(row["id"], row["operacao_id"]),
		Modalidade:      coalesce(row["modalidade"], row["mod"]),
		NumeroContrato:  row["contrato"],
		TipoPessoa:      coalesce(row["tipopessoa"], row["tpcli"]),
		UF:              coalesce(row["uf"], row["estado"]),
		ClassificacaoIF: coalesce(row["classificacaoif"], row["classif"], row["classop"]),
	}

	// Parse valor principal.
	for _, k := range []string{"valorprincipal", "vlrprincipal", "valor"} {
		if v := row[k]; v != "" {
			op.ValorPrincipal = money(v)
			break
		}
	}

	// Encargos.
	if v := row["encargos"]; v != "" {
		op.EncargosTotais = money(v)
	}

	// IOF.
	if v := row["iof"]; v != "" {
		op.IOF = money(v)
	}

	// Valor atualizado.
	if v := row["valoratualizado"]; v != "" {
		op.ValorAtualizado = money(v)
	}

	// Data vencimento.
	if v := row["datavencimento"]; v != "" {
		op.DataVencimento = parseDate(v)
	}

	// Data constituição.
	if v := row["dataconstituicao"]; v != "" {
		op.DataConstituicao = parseDate(v)
	}

	// Leftover fields → Extra.
	op.Extra = make(map[string]any)
	for k, v := range row {
		if v == "" {
			continue
		}
		switch k {
		case "modalidade", "mod", "valorprincipal", "vlrprincipal", "valor", "contrato",
			"tipopessoa", "tpcli", "uf", "estado",
			"classificacaoif", "classif", "classop", "id", "operacao_id",
			"datavencimento", "dataconstituicao",
			"encargos", "iof", "valoratualizado":
			// already handled
		default:
			op.Extra[k] = v
		}
	}

	return op, true
}

func coalesce(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// money parses a monetary string into canonical.Money.
// money parses a monetary string into canonical.Money.
// Handles both Brazilian ("50.000,00") and US ("50000.00") formats,
// and strips currency symbols (R$, $, €).
func money(s string) canonical.Money {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return canonical.Money{}
	}

	// Detect Brazilian format: comma is decimal separator.
	hasComma := strings.Contains(s, ",")
	hasDot := strings.Contains(s, ".")

	// Remove currency symbols.
	s = strings.NewReplacer("R$", "", "$", "", "€", "", "£", "").Replace(s)
	s = strings.TrimSpace(s)

	if hasComma && hasDot {
		// Brazilian: dots are thousands, comma is decimal.
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if hasComma {
		// Only comma: comma is decimal separator.
		s = strings.ReplaceAll(s, ",", ".")
	} else if hasDot {
		// Only dot: dot is decimal separator (US format).
		// Nothing to do.
	}

	v, _ := strconv.ParseFloat(s, 64)
	return canonical.Money{
		Valor: decimal.NewFromFloat(v),
		Moeda: "BRL",
	}
}

// parseDate tries common date formats.
func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"2006/01/02",
		"02-01-2006",
		"20060102",
		"02.01.2006",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
