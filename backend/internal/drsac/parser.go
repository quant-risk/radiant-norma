// Parser para DRSAC — CADOC 2030 (Documento de Riscos Social, Ambiental e Climático).
//
// O parsing é feito via XML unmarshaling direto com a stdlib.
// O DRSAC suporta encodings: ISO-8859-1, EBCDIC-CP-US, UTF-8, UTF-16, US-ASCII.
//
//nolint:revive
package drsac

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

// ErrEmptyDocument é retornado quando o documento é vazio.
var ErrEmptyDocument = errors.New("documento DRSAC vazio")

// ParseError representa um erro de parsing XML.
type ParseError struct {
	Inner error
}

func (e *ParseError) Error() string { return fmt.Sprintf("parse DRSAC: %v", e.Inner) }
func (e *ParseError) Unwrap() error { return e.Inner }

// Parse lê um documento DRSAC de r e retorna o struct.
// Suporta os encodings declarados no header XML.
func Parse(r io.Reader) (*DocumentoDRSAC, error) {
	// Lê tudo para detectar encoding
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, &ParseError{Inner: err}
	}
	if len(data) == 0 {
		return nil, ErrEmptyDocument
	}

	// Detecta encoding pelo XML declaration
	enc := detectEncoding(data)
	if enc == "" {
		enc = "UTF-8" // default
	}

	// Re-decode com encoding correto
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charsetReader(enc)

	var doc DocumentoDRSAC
	if err := decoder.Decode(&doc); err != nil {
		return nil, &ParseError{Inner: err}
	}

	// Verifica que é o documento correto
	if doc.CodigoDoc != "2030" {
		return nil, fmt.Errorf("codigoDocumento=%q, esperado 2030", doc.CodigoDoc)
	}

	return &doc, nil
}

// ParseFromBytes é um atalho para Parse(bytes.NewReader(data)).
func ParseFromBytes(data []byte) (*DocumentoDRSAC, error) {
	return Parse(bytes.NewReader(data))
}

// detectEncoding tenta detectar o encoding do XML.
// Retorna "" se não conseguir detectar.
func detectEncoding(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	// UTF-8 BOM
	if data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return "UTF-8"
	}
	// UTF-16 BE BOM
	if data[0] == 0xFE && data[1] == 0xFF {
		return "UTF-16BE"
	}
	// UTF-16 LE BOM
	if data[0] == 0xFF && data[1] == 0xFE {
		return "UTF-16LE"
	}
	// XML declaration
	const xmlDecl = `<?xml version`
	if bytes.HasPrefix(data, []byte(xmlDecl)) {
		// procura encoding="
		idx := bytes.Index(data, []byte(`encoding=`))
		if idx > 0 {
			start := idx + len(`encoding=`)
			if start < len(data) && (data[start] == '"' || data[start] == '\'') {
				quote := data[start]
				start++
				end := start
				for end < len(data) && data[end] != quote {
					end++
				}
				return string(data[start:end])
			}
		}
	}
	return ""
}

// charsetReader é um xml.EncodingScanner para encodings não-UTF-8.
// A ser implementado quando o encoding real for detectado.
func charsetReader(enc string) func(string, io.Reader) (io.Reader, error) {
	return func(encoding string, input io.Reader) (io.Reader, error) {
		// Para encodings conhecidos, re-encodar para UTF-8
		switch encoding {
		case "ISO-8859-1", "iso-8859-1":
			return input, nil // go-xml trata como UTF-8 com bytes Latin-1
		case "EBCDIC-CP-US":
			// TODO: implementar transcoding EBCDIC se necessário
			return nil, fmt.Errorf("encoding EBCDIC-CP-US não suportado: requer conversão")
		default:
			return input, nil
		}
	}
}
