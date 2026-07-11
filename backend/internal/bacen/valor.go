// Package bacen provides typed XML structs for all BACEN CADOCs.
// These structs are used by both generators (marshaling) and cross-doc
// rules (unmarshaling). Using encoding/xml eliminates string-scraping with regex.
//
// For package documentation, see doc3040.go.
package bacen

// ValorSimples is a simple wrapper for a value attribute.
type ValorSimples struct {
	Valor string `xml:"valor,attr"`
}
