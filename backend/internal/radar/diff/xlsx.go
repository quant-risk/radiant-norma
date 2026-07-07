// Package diff implementa diff semântico para arquivos regulatórios.
// Este arquivo contém parsing de XLSX usando excelize.
package diff

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ParseXLSX parseia um arquivo XLSX e retorna mapa de regras.
//
// Estrutura esperada da planilha:
// - Linha 1: cabeçalho (Field1, Field2, ..., FieldN)
// - Linhas 2+: dados (cada linha = uma regra)
// - Coluna "Codigo" ou "Regra": identificador único da regra
//
// Retorna: map["codigo_da_regra"] → map["nome_do_campo"] → valor
func ParseXLSX(data []byte, sheetName string) (map[string]map[string]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("excelize open: %w", err)
	}
	defer f.Close()

	// Seleciona sheet (padrão: primeira se não especificada).
	if sheetName == "" {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("nenhuma sheet encontrada no XLSX")
		}
		sheetName = sheets[0]
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("get rows: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("XLSX com menos de 2 linhas (sem header + dados)")
	}

	// Header = primeira linha.
	header := rows[0]
	if len(header) == 0 {
		return nil, fmt.Errorf("header vazio")
	}

	// Encontra coluna do código (case-insensitive).
	codeCol := -1
	for i, col := range header {
		normalized := normalizeHeader(col)
		if normalized == "codigo" || normalized == "regra" || normalized == "code" || normalized == "rule" {
			codeCol = i
			break
		}
	}
	if codeCol == -1 {
		// Usa primeira coluna como fallback.
		codeCol = 0
	}

	result := make(map[string]map[string]string)

	for rowIdx, row := range rows[1:] {
		if len(row) == 0 || (len(row) == 1 && row[0] == "") {
			continue // pula linha vazia
		}
		// Garante que linha tem pelo menos tantas colunas quanto o header.
		rowPadded := padRow(row, len(header))

		code := rowPadded[codeCol]
		if code == "" {
			continue // pula se código vazio.
		}

		rowMap := make(map[string]string)
		for colIdx, colName := range header {
			if colIdx < len(rowPadded) {
				rowMap[normalizeHeader(colName)] = rowPadded[colIdx]
			}
		}
		result[code] = rowMap
		_ = rowIdx // unused but keeps intent
	}

	return result, nil
}

// normalizeHeader normaliza nome de coluna para chave de mapa.
// Remove espaços, acentos, converte para lowercase.
func normalizeHeader(col string) string {
	// Usa só caracteres alfanuméricos e underscore.
	result := make([]byte, 0, len(col))
	for _, c := range col {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' {
			result = append(result, byte(c))
		}
	}
	// Normaliza para lowercase.
	for i := range result {
		if result[i] >= 'A' && result[i] <= 'Z' {
			result[i] = result[i] + 'a' - 'A'
		}
	}
	return string(result)
}

// padRow preenche uma linha com strings vazios até o tamanho do header.
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
