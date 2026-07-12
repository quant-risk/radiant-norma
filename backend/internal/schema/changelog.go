// Package schema — changelog generation (Sprint 54 v3.34.37).
//
// Diff entre duas versões de schema: computa automaticamente o changelog
// (lista de mudanças) quando um novo schema é inserido.
//
//nolint:revive,stylecheck
package schema

import (
	"fmt"
	"slices"
	"strings"
)

// ComputeChangelog gera texto descritivo diffing prev vs curr.
// Retorna "" se não houver mudança.
//
// Formato de saída (txt, uma linha por mudança):
//
//	+CAMPO ADDED: tag=NOVOCAMPO attr=novo type=A20 desc="Novo campo"
//	-CAMPO REMOVED: tag=VELHOCAMPO type=N13,2
//	~CAMPO MODIFIED: tag=CAMPOEXISTENTE type N13,2 → N15,2
//	~CAMPO MODIFIED: tag=CAMPOEXISTENTE required false → true
//
// nil prev: retorna "versão inicial" se curr não-vazio.
func ComputeChangelog(prev, curr []Field) string {
	if len(prev) == 0 && len(curr) == 0 {
		return ""
	}
	if len(prev) == 0 {
		return fmt.Sprintf("versão inicial (%d campos)", len(curr))
	}

	var lines []string

	currMap := make(map[string]Field)
	for _, f := range curr {
		currMap[fieldKey(f)] = f
	}

	prevMap := make(map[string]Field)
	for _, f := range prev {
		prevMap[fieldKey(f)] = f
	}

	// Removed fields (in prev but not in curr)
	for _, f := range prev {
		k := fieldKey(f)
		if _, inCurr := currMap[k]; !inCurr {
			lines = append(lines, formatRemoved(f))
		}
	}

	// Added fields (in curr but not in prev)
	for _, f := range curr {
		k := fieldKey(f)
		if _, inPrev := prevMap[k]; !inPrev {
			lines = append(lines, formatAdded(f))
		}
	}

	// Modified fields (same key, different properties)
	for _, f := range curr {
		k := fieldKey(f)
		if prevF, inPrev := prevMap[k]; inPrev {
			if diff := formatModified(prevF, f); diff != "" {
				lines = append(lines, diff)
			}
		}
	}

	// Stable order for deterministic output
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}

// fieldKey returns a unique key for a field: tag:attr (attr may be "").
func fieldKey(f Field) string {
	return f.Tag + ":" + f.Attr
}

// formatAdded returns "+CAMPO ADDED" line.
func formatAdded(f Field) string {
	return fmt.Sprintf("+CAMPO ADDED: tag=%s attr=%s type=%s desc=%q",
		f.Tag, f.Attr, f.Type, f.Desc)
}

// formatRemoved returns "-CAMPO REMOVED" line.
func formatRemoved(f Field) string {
	return fmt.Sprintf("-CAMPO REMOVED: tag=%s attr=%s type=%s",
		f.Tag, f.Attr, f.Type)
}

// formatModified returns "~CAMPO MODIFIED" lines for each changed property,
// or "" if nothing changed.
func formatModified(before, after Field) string {
	var changes []string

	if before.Type != after.Type {
		changes = append(changes, typeChange(before.Type, after.Type))
	}
	if before.Required != after.Required {
		changes = append(changes, requiredChange(before.Required, after.Required))
	}
	if before.Desc != after.Desc && after.Desc != "" {
		changes = append(changes, fmt.Sprintf("desc %q → %q", before.Desc, after.Desc))
	}
	if before.Domain != after.Domain && after.Domain != "" {
		changes = append(changes, fmt.Sprintf("domain %q → %q", before.Domain, after.Domain))
	}
	if before.Group != after.Group && after.Group != "" {
		changes = append(changes, fmt.Sprintf("group %q → %q", before.Group, after.Group))
	}

	if len(changes) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("~CAMPO MODIFIED: tag=%s attr=%s", after.Tag, after.Attr))
	for _, c := range changes {
		sb.WriteString(" ")
		sb.WriteString(c)
	}
	return sb.String()
}

func typeChange(before, after string) string {
	return fmt.Sprintf("type %s → %s", before, after)
}

func requiredChange(before, after bool) string {
	return fmt.Sprintf("required %v → %v", before, after)
}

// ============================================================================
// Sprint 54 — Structured changelog (machine-parseable)
// ============================================================================

// ChangeKind classifies the type of field change.
type ChangeKind string

const (
	ChangeAdded    ChangeKind = "added"
	ChangeRemoved ChangeKind = "removed"
	ChangeModified ChangeKind = "modified"
)

// FieldChange represents a single field that was added, removed, or modified.
type FieldChange struct {
	Kind     ChangeKind `json:"kind"`
	Tag      string     `json:"tag"`
	Attr     string     `json:"attr,omitempty"`
	Before   *Field     `json:"before,omitempty"`   // nil for added
	After    *Field     `json:"after,omitempty"`    // nil for removed
	Changes  []string   `json:"changes,omitempty"` // e.g. ["type N13,2 → N15,2"]
}

// StructuredChangelog holds a machine-parseable diff between two schema versions.
type StructuredChangelog struct {
	CadocCode     string         `json:"cadoc_code"`
	PrevVersion   string         `json:"prev_version,omitempty"`   // empty for initial version
	CurrVersion   string         `json:"curr_version"`
	EffectiveFrom string         `json:"effective_from"`
	Added        []FieldChange   `json:"added,omitempty"`
	Removed      []FieldChange   `json:"removed,omitempty"`
	Modified     []FieldChange   `json:"modified,omitempty"`
	TotalChanges int             `json:"total_changes"`
}

// ComputeStructuredChangelog returns a structured diff between prev and curr fields.
func ComputeStructuredChangelog(cadocCode string, prev, curr []Field, prevVersion, currVersion, effectiveFrom string) *StructuredChangelog {
	sc := &StructuredChangelog{
		CadocCode:     cadocCode,
		PrevVersion:   prevVersion,
		CurrVersion:   currVersion,
		EffectiveFrom: effectiveFrom,
	}

	currMap := make(map[string]Field)
	for _, f := range curr {
		currMap[fieldKey(f)] = f
	}
	prevMap := make(map[string]Field)
	for _, f := range prev {
		prevMap[fieldKey(f)] = f
	}

	for _, f := range prev {
		k := fieldKey(f)
		if _, inCurr := currMap[k]; !inCurr {
			sc.Removed = append(sc.Removed, FieldChange{
				Kind:  ChangeRemoved,
				Tag:   f.Tag,
				Attr:  f.Attr,
				Before: &f,
			})
		}
	}

	for _, f := range curr {
		k := fieldKey(f)
		if _, inPrev := prevMap[k]; !inPrev {
			cp := f
			sc.Added = append(sc.Added, FieldChange{
				Kind:  ChangeAdded,
				Tag:   f.Tag,
				Attr:  f.Attr,
				After: &cp,
			})
		} else {
			prev := prevMap[k]
			changes := formatModifiedFields(prev, f)
			if len(changes) > 0 {
				sc.Modified = append(sc.Modified, FieldChange{
					Kind:    ChangeModified,
					Tag:     f.Tag,
					Attr:    f.Attr,
					Before:  &prev,
					After:   &f,
					Changes: changes,
				})
			}
		}
	}

	sc.TotalChanges = len(sc.Added) + len(sc.Removed) + len(sc.Modified)
	return sc
}

func formatModifiedFields(before, after Field) []string {
	var out []string
	if before.Type != after.Type {
		out = append(out, fmt.Sprintf("type %s → %s", before.Type, after.Type))
	}
	if before.Required != after.Required {
		out = append(out, fmt.Sprintf("required %v → %v", before.Required, after.Required))
	}
	if before.Desc != after.Desc {
		out = append(out, fmt.Sprintf("description changed"))
	}
	if before.Domain != after.Domain {
		out = append(out, fmt.Sprintf("domain %s → %s", before.Domain, after.Domain))
	}
	if before.Group != after.Group {
		out = append(out, fmt.Sprintf("group %s → %s", before.Group, after.Group))
	}
	return out
}

// AsText returns the human-readable text representation of the structured changelog.
func (sc *StructuredChangelog) AsText() string {
	var lines []string
	for _, f := range sc.Added {
		lines = append(lines, formatAdded(*f.After))
	}
	for _, f := range sc.Removed {
		lines = append(lines, formatRemoved(*f.Before))
	}
	for _, f := range sc.Modified {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("~CAMPO MODIFIED: tag=%s attr=%s", f.Tag, f.Attr))
		for _, c := range f.Changes {
			sb.WriteString(" ")
			sb.WriteString(c)
		}
		lines = append(lines, sb.String())
	}
	if len(lines) == 0 {
		return "(sem alterações)"
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}
