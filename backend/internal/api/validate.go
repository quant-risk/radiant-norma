// Package api — payload validation helpers.
//
// Sprint 13 — v3.5.2 [S15.3]:
// - DisallowUnknownFields nos endpoints JSON (defesa contra typo
//   + mass-assignment attempts).
// - cadoc_code regex [0-9]{4} validator (defesa contra injection via
//   unknown CADOC code → audit log poluído + queries desnecessárias).
// - rule_code regex [A-Z][0-9]{1,3} (mesmo padrão).

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// CadocCodePattern é o regex oficial do BACEN pra códigos CADOC.
// Permite "2030", "3040", "3044", "4111" etc — 4 dígitos numéricos.
// Ancorado em ^ e $ para match exato.
const cadocCodePattern = `^[0-9]{4}$`

// RuleCodePattern é o formato BACEN para códigos de regra.
// "B12", "F23", "S05", "C001" — letra maiúscula seguida de 1-3 dígitos.
const ruleCodePattern = `^[A-Z][0-9]{1,3}$`

// ValidateCadocCode retorna nil se code bate o pattern BACEN [0-9]{4}.
func ValidateCadocCode(code string) error {
	if !matchesPattern(cadocCodePattern, code) {
		return fmt.Errorf("invalid cadoc_code %q (expected 4 digits, e.g. \"3040\")", code)
	}
	return nil
}

// ValidateRuleCode retorna nil se code bate rule_code BACEN format.
func ValidateRuleCode(code string) error {
	if !matchesPattern(ruleCodePattern, code) {
		return fmt.Errorf("invalid rule_code %q (expected [A-Z][0-9]{1,3}, e.g. \"F23\")", code)
	}
	return nil
}

// matchesPattern é uma versão simples de regex match sem expor
// regexp package. Implementação compila on the fly com cache via
// sync.Map-equivalente (ritmo de compilação é OK já que validators
// rodam apenas no path crítico e pattern é fixo).
//
// Aqui usamos uma versão simples que evita regexp.Compiled por call.
// Padrões [0-9]{4} e [A-Z][0-9]{1,3} são simples o suficiente
// pra matching inline (mais rápido que regexp). Se ficar mais
// complexo, migrar para sync.Map+regexp.Compiled.
func matchesPattern(pattern, s string) bool {
	switch pattern {
	case cadocCodePattern:
		if len(s) != 4 {
			return false
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	case ruleCodePattern:
		if len(s) < 2 || len(s) > 4 {
			return false
		}
		if s[0] < 'A' || s[0] > 'Z' {
			return false
		}
		for _, c := range s[1:] {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	return false
}

// decodeJSONStrictly faz Unmarshal mas rejeita campos desconhecidos.
//
// Sprint 13 [S15.3 / MED-S15.3-2]: defesa contra typos + mass-assignment.
// Cliente envia `{"if_id":"X","role":"admin"}` em payload que só deveria
// ter `if_id`. json.Unmarshal aceita silenciosamente. Aqui rejeitamos 400.
//
// Retorna ErrJSONUnknownField se houver campos extras.
func decodeJSONStrictly(body []byte, dst any) error {
	dec := json.NewDecoder(bytesReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("empty body")
		}
		return err
	}
	return nil
}

// bytesReader é wrapper minimal de bytes.NewReader sem importar bytes
// (evita um import e mantém este arquivo focado).
func bytesReader(b []byte) *bytesReaderImpl {
	return &bytesReaderImpl{b: b}
}

type bytesReaderImpl struct {
	b   []byte
	pos int
}

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}
