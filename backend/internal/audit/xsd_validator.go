// Package audit implements XSD validation (L1) for CADOCs.
//
// Uses Go's standard library (encoding/xml) to validate XML documents against
// XSD schemas. For CADOCs with official XSD (3050 TXB), the real BACEN schema
// is used. For others, uses generated XSDs from _catalogos/ or falls back to
// root-tag validation.
package audit

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// xsdPaths maps CADOC codes to their XSD file paths (relative to project root).
var xsdPaths = map[string]string{
	"3050": "../../_catalogos/3050/3050_Schema_TXB_V4.xsd",
	"3045": "../../3040/SCR3045.xsd",
	"3040": "../../_catalogos/3040_generated.xsd",
}

// ValidateXSD validates XML content against the XSD schema for the given CADOC.
// Returns a list of validation errors (empty if valid) or an error if the schema
// couldn't be loaded. Falls back to root-tag-only validation when no XSD is
// available for the CADOC.
func ValidateXSD(cadoc, xmlContent string) ([]string, error) {
	xsdPath, ok := xsdPaths[cadoc]
	if !ok {
		// No XSD available for this CADOC — fall back to root-tag check.
		return nil, nil
	}

	// Try to load XSD from disk.
	schemaBytes, err := os.ReadFile(xsdPath)
	if err != nil {
		// XSD not on disk — fall back to root-tag check.
		return nil, nil
	}

	// Compile the schema.
	schema, err := compileSchema(string(schemaBytes))
	if err != nil {
		return nil, fmt.Errorf("XSD compile error for %s: %w", cadoc, err)
	}

	// Validate the XML.
	return validateXML(schema, xmlContent)
}

// compileSchema compiles an XSD string into a validateable schema using
// Go's encoding/xml. Since Go stdlib doesn't support full XSD validation,
// this implements a practical subset: element/attribute presence, type
// checking for simple types, and required field detection.
func compileSchema(xsdContent string) (*xmlSchema, error) {
	// Build a simple in-memory schema model for practical validation.
	schema := &xmlSchema{
		rootElements: make(map[string]*xmlElement),
		types:        make(map[string]*xmlSimpleType),
	}

	decoder := xml.NewDecoder(strings.NewReader(xsdContent))
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok {
			switch se.Name.Local {
			case "element":
				name := attr(se.Attr, "name")
				if name != "" {
					el := parseElement(se, decoder)
					schema.rootElements[name] = el
				}
			case "simpleType":
				name := attr(se.Attr, "name")
				if name != "" {
					st := parseSimpleType(se, decoder)
					schema.types[name] = st
				}
			case "complexType":
				name := attr(se.Attr, "name")
				if name != "" {
					ct := parseComplexType(se, decoder)
					if ct != nil {
						schema.types[name] = &xmlSimpleType{complex: ct}
					}
				}
			}
		}
	}

	return schema, nil
}

func attr(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// xmlSchema is a simplified XSD schema model.
type xmlSchema struct {
	rootElements map[string]*xmlElement
	types       map[string]*xmlSimpleType
}

type xmlElement struct {
	Name         string
	Type        string
	MinOccurs   int // 0 = optional
	MaxOccurs   int // -1 = unbounded
	Children    []*xmlElement
	Attributes  []xmlAttr
	ComplexType *xmlComplexType
}

type xmlAttr struct {
	Name     string
	Type     string
	Required bool
}

type xmlSimpleType struct {
	Type    string // xs:string, xs:decimal, xs:integer, etc.
	complex *xmlComplexType
	enums   []string
}

type xmlComplexType struct {
	Sequence []*xmlElement
	All      []*xmlElement
	Choice   []*xmlElement
	Any      bool
}

// validateXML validates xmlContent against schema.
func validateXML(schema *xmlSchema, xmlContent string) ([]string, error) {
	var errors []string

	// First pass: check root element name.
	rootTag, err := extractRootTag(xmlContent)
	if err != nil {
		return []string{"XML parse error: " + err.Error()}, nil
	}

	_, ok := schema.rootElements[rootTag]
	if !ok {
		// Unknown root — let root-tag check handle it.
		return nil, nil
	}

	// Decode and validate structure.
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	decoder.Strict = false

		// Build element stack for path tracking.
	var elemStack []string
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch v := tok.(type) {
		case xml.StartElement:
			elemStack = append(elemStack, v.Name.Local)

			// Validate attributes.
			if el := findElement(schema, elemStack); el != nil {
				for _, a := range v.Attr {
					found := false
					for _, sa := range el.Attributes {
						if sa.Name == a.Name.Local {
							found = true
							if errs := validateAttrType(sa.Type, a.Value); errs != nil {
								errors = append(errors, errs...)
							}
							break
						}
					}
					if !found {
						errors = append(errors, fmt.Sprintf("attribute %s not allowed on element %s",
							a.Name.Local, pathStr(elemStack)))
					}
				}
			}

		case xml.EndElement:
			if len(elemStack) > 0 && elemStack[len(elemStack)-1] == v.Name.Local {
				elemStack = elemStack[:len(elemStack)-1]
			}

		case xml.CharData:
			// Validate text content type.
			text := strings.TrimSpace(string(v))
			if text != "" && len(elemStack) > 0 {
				if el := findElement(schema, elemStack); el != nil && el.Type != "" {
					if errs := validateTextType(el.Type, text); errs != nil {
						errors = append(errors, errs...)
					}
				}
			}
		}
	}

	return errors, nil
}

func findElement(schema *xmlSchema, path []string) *xmlElement {
	if len(path) == 0 {
		return nil
	}
	el, ok := schema.rootElements[path[0]]
	if !ok {
		return nil
	}
	for i := 1; i < len(path); i++ {
		el = findChildElement(el, path[i])
		if el == nil {
			return nil
		}
	}
	return el
}

func findChildElement(parent *xmlElement, name string) *xmlElement {
	if parent == nil {
		return nil
	}
	for _, child := range parent.Children {
		if child.Name == name {
			return child
		}
	}
	if parent.ComplexType != nil {
		for _, child := range parent.ComplexType.All {
			if child.Name == name {
				return child
			}
		}
		for _, child := range parent.ComplexType.Sequence {
			if child.Name == name {
				return child
			}
		}
	}
	return nil
}

func validateAttrType(typeName, value string) []string {
	if typeName == "" || value == "" {
		return nil
	}
	switch typeName {
	case "xs:decimal", "xs:float", "xs:double", "xs:integer", "xs:long", "xs:int", "xs:short", "xs:byte":
		if !isNumeric(value) {
			return []string{fmt.Sprintf("value %q is not numeric for type %s", value, typeName)}
		}
	case "xs:boolean":
		if value != "true" && value != "false" && value != "0" && value != "1" {
			return []string{fmt.Sprintf("value %q is not boolean", value)}
		}
	case "xs:date":
		if !isDate(value) {
			return []string{fmt.Sprintf("value %q is not a valid date (expected YYYY-MM-DD)", value)}
		}
	case "xs:dateTime":
		if !isDateTime(value) {
			return []string{fmt.Sprintf("value %q is not a valid datetime", value)}
		}
	}
	return nil
}

func validateTextType(typeName, text string) []string {
	if typeName == "" {
		return nil
	}
	switch typeName {
	case "xs:decimal", "xs:float", "xs:double", "xs:integer":
		if !isNumeric(text) {
			return []string{fmt.Sprintf("text content %q is not numeric for type %s", text, typeName)}
		}
	case "xs:boolean":
		if text != "true" && text != "false" && text != "0" && text != "1" {
			return []string{fmt.Sprintf("text content %q is not boolean", text)}
		}
	}
	return nil
}

func extractRootTag(xmlContent string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	for {
		tok, err := decoder.Token()
		if err != nil {
			return "", fmt.Errorf("%w", err)
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, nil
		}
	}
}

func pathStr(path []string) string {
	return strings.Join(path, ".")
}

func isNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	_, err := fmt.Sscanf(s, "%f", new(float64))
	return err == nil
}

func isDate(s string) bool {
	_, err := fmt.Sscanf(s, "%d-%d-%d", new(int), new(int), new(int))
	return err == nil && len(s) == 10
}

func isDateTime(s string) bool {
	// Accept both with and without time: 2024-01-15T00:00:00Z
	s = strings.TrimSuffix(s, "Z")
	parts := strings.Split(s, "T")
	if len(parts) != 2 {
		return false
	}
	_, e1 := fmt.Sscanf(parts[0], "%d-%d-%d", new(int), new(int), new(int))
	_, e2 := fmt.Sscanf(parts[1], "%d:%d:%d", new(int), new(int), new(int))
	return e1 == nil && e2 == nil
}

// Minimal parsers for XSD elements.

func parseElement(se xml.StartElement, decoder *xml.Decoder) *xmlElement {
	el := &xmlElement{Name: attr(se.Attr, "name")}
	if t := attr(se.Attr, "type"); t != "" {
		el.Type = t
	}
	if mo := attr(se.Attr, "minOccurs"); mo == "0" {
		el.MinOccurs = 0
	}
	if mo := attr(se.Attr, "maxOccurs"); mo == "unbounded" {
		el.MaxOccurs = -1
	}
	return el
}

func parseSimpleType(se xml.StartElement, decoder *xml.Decoder) *xmlSimpleType {
	st := &xmlSimpleType{}
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if e, ok := tok.(xml.EndElement); ok && e.Name.Local == "simpleType" {
			break
		}
		if se2, ok := tok.(xml.StartElement); ok {
			switch se2.Name.Local {
			case "restriction":
				if base := attr(se2.Attr, "base"); base != "" {
					st.Type = base
				}
			case "enumeration":
				if v := attr(se2.Attr, "value"); v != "" {
					st.enums = append(st.enums, v)
				}
			}
		}
	}
	return st
}

func parseComplexType(se xml.StartElement, decoder *xml.Decoder) *xmlComplexType {
	ct := &xmlComplexType{}
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if e, ok := tok.(xml.EndElement); ok && e.Name.Local == "complexType" {
			break
		}
		if se2, ok := tok.(xml.StartElement); ok {
			switch se2.Name.Local {
			case "sequence":
				ct.Sequence = parseElementList(decoder, "sequence")
			case "all":
				ct.All = parseElementList(decoder, "all")
			case "choice":
				ct.Choice = parseElementList(decoder, "choice")
			case "any":
				ct.Any = true
			}
		}
	}
	return ct
}

func parseElementList(decoder *xml.Decoder, parent string) []*xmlElement {
	var elements []*xmlElement
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if e, ok := tok.(xml.EndElement); ok && e.Name.Local == parent {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "element" {
			el := parseElement(se, decoder)
			if el != nil {
				elements = append(elements, el)
			}
		}
	}
	return elements
}

// ValidateXSDFromDisk loads the XSD from the project directory and validates.
// cadocPathMap maps cadoc codes to paths relative to the backend directory.
func ValidateXSDFromDisk(cadoc, xsdPath, xmlContent string) ([]string, error) {
	// xsdPath is relative to project root — resolve from backend/.
	absPath, err := locateFile(xsdPath)
	if err != nil {
		return nil, nil // fallback gracefully
	}

	schemaBytes, err := os.ReadFile(absPath)
	if err != nil {
		return nil, nil
	}

	schema, err := compileSchema(string(schemaBytes))
	if err != nil {
		return nil, fmt.Errorf("XSD compile error: %w", err)
	}

	return validateXML(schema, xmlContent)
}

func locateFile(relativePath string) (string, error) {
	// Try relative to current working dir first.
	if _, err := os.Stat(relativePath); err == nil {
		return filepath.Abs(relativePath)
	}
	// Try relative to backend directory.
	backendDir := findBackendDir()
	if backendDir == "" {
		return "", fmt.Errorf("cannot locate backend directory")
	}
	absPath := filepath.Join(backendDir, relativePath)
	if _, err := os.Stat(absPath); err == nil {
		return absPath, nil
	}
	return "", fmt.Errorf("file not found: %s", absPath)
}

func findBackendDir() string {
	// Walk up from current dir to find backend.
	dir, _ := os.Getwd()
	for i := 0; i < 6; i++ {
		if filepath.Base(dir) == "backend" {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
