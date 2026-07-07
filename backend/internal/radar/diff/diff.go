// Package diff implementa diff semântico para arquivos regulatórios.
//
// Diferente do radar v1 (que só detecta hash change), o diff v2 parseia
// o conteúdo estruturado (XLSX, XSD) e gera DiffEntry com o que mudou
// especificamente (regra nova, alterada, removida).
package diff

import (
	"fmt"
	"time"
)

// ChangeType representa o tipo de mudança numa regra.
type ChangeType string

const (
	ChangeTypeAdded   ChangeType = "added"
	ChangeTypeRemoved ChangeType = "removed"
	ChangeTypeChanged ChangeType = "changed"
)

// DiffEntry representa uma mudança numa regra ou campo regulatório.
type DiffEntry struct {
	CadocCode  string     // "3040", "3050", "2160"
	RuleCode   string     // "C01", "LCR01", etc. (ou "" se não identificado)
	ChangeType ChangeType // added | removed | changed
	Before     string     // valor antes ("" se added)
	After      string     // valor depois ("" se removed)
	Severity   string     // "E" | "A" | "I" (se aplicável)
	Field      string     // campo que mudou: "descricao", "formula", "obrigatoriedade"
}

// DiffResult é o resultado completo de uma análise de diff.
type DiffResult struct {
	CadocCode  string
	SourceURL  string
	DetectedAt time.Time
	OldHash    string
	NewHash    string
	Entries    []DiffEntry
	Summary    string // texto legível: "2 adicionadas, 1 alterada, 0 removidas"
}

// BuildSummary gera um texto legível a partir de DiffEntries.
func (d *DiffResult) BuildSummary() string {
	var added, removed, changed int
	for _, e := range d.Entries {
		switch e.ChangeType {
		case ChangeTypeAdded:
			added++
		case ChangeTypeRemoved:
			removed++
		case ChangeTypeChanged:
			changed++
		}
	}
	parts := []string{}
	if added > 0 {
		parts = append(parts, fmt.Sprintf("%d adicionada(s)", added))
	}
	if changed > 0 {
		parts = append(parts, fmt.Sprintf("%d alterada(s)", changed))
	}
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("%d removida(s)", removed))
	}
	if len(parts) == 0 {
		return "sem alterações detectadas"
	}
	return parts[0] + remainingJoin(parts[1:])
}

func remainingJoin(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return ", " + parts[0]
	}
	result := ""
	for i := 0; i < len(parts)-1; i++ {
		result += ", " + parts[i]
	}
	result += " e " + parts[len(parts)-1]
	return result
}

// Entry adds a DiffEntry to the result and updates the summary.
func (d *DiffResult) Entry(e DiffEntry) {
	d.Entries = append(d.Entries, e)
	d.Summary = d.BuildSummary()
}

// Differ compara dois conteúdo e gera DiffEntries.
type Differ struct{}

// NewDiffer cria um novo Differ.
func NewDiffer() *Differ { return &Differ{} }

// CompareRowMaps compara dois mapas de linha (key = código da regra) e
// gera DiffEntries para todas as diferenças encontradas.
//
// oldMap: mapa de "código da regra" → map de "campo" → valor
// newMap: idem
//
// changeTypes relevantes para cada regra.
func (d *Differ) CompareRowMaps(oldMap, newMap map[string]map[string]string, cadocCode string) []DiffEntry {
	var entries []DiffEntry

	// Detecta regras novas (em newMap mas não em oldMap).
	for key, newRow := range newMap {
		if _, exists := oldMap[key]; !exists {
			entry := DiffEntry{
				CadocCode:  cadocCode,
				RuleCode:   key,
				ChangeType: ChangeTypeAdded,
				After:      formatRowSummary(newRow),
			}
			entries = append(entries, entry)
		}
	}

	// Detecta regras removidas (em oldMap mas não em newMap).
	for key, oldRow := range oldMap {
		if _, exists := newMap[key]; !exists {
			entry := DiffEntry{
				CadocCode:  cadocCode,
				RuleCode:   key,
				ChangeType: ChangeTypeRemoved,
				Before:     formatRowSummary(oldRow),
			}
			entries = append(entries, entry)
		}
	}

	// Detecta regras alteradas (existem em ambos mas campos diferem).
	for key, oldRow := range oldMap {
		if newRow, exists := newMap[key]; exists {
			diffs := d.compareRowFields(key, oldRow, newRow, cadocCode)
			entries = append(entries, diffs...)
		}
	}

	return entries
}

// compareRowFields compara campos individuais de uma linha e retorna DiffEntries.
func (d *Differ) compareRowFields(ruleCode string, oldRow, newRow map[string]string, cadocCode string) []DiffEntry {
	var entries []DiffEntry
	for field, oldVal := range oldRow {
		if newVal, exists := newRow[field]; exists {
			if oldVal != newVal {
				entries = append(entries, DiffEntry{
					CadocCode:  cadocCode,
					RuleCode:   ruleCode,
					ChangeType: ChangeTypeChanged,
					Field:      field,
					Before:     oldVal,
					After:      newVal,
				})
			}
		}
	}
	// Campos novos em newRow (não existiam em oldRow).
	for field, newVal := range newRow {
		if _, exists := oldRow[field]; !exists {
			entries = append(entries, DiffEntry{
				CadocCode:  cadocCode,
				RuleCode:   ruleCode,
				ChangeType: ChangeTypeChanged,
				Field:      field,
				After:      newVal,
			})
		}
	}
	return entries
}

// formatRowSummary retorna uma string resumida dos valores de uma linha.
func formatRowSummary(row map[string]string) string {
	// Pega o primeiro valor disponível para ter contexto.
	for _, v := range row {
		if len(v) > 50 {
			return v[:47] + "..."
		}
		return v
	}
	return ""
}

// NewResult cria um DiffResult inicializado.
func NewResult(cadocCode, sourceURL, oldHash, newHash string) *DiffResult {
	return &DiffResult{
		CadocCode:  cadocCode,
		SourceURL:  sourceURL,
		DetectedAt: time.Now(),
		OldHash:    oldHash,
		NewHash:    newHash,
		Entries:    []DiffEntry{},
		Summary:    "sem alterações detectadas",
	}
}
