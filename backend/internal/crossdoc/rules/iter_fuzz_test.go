// Fuzz testing para iterateXMLElements (Sprint 7b / v1.7.0).
//
// Resolve GAP-7.2 (Cross-doc L3 — regras baseadas em agregação de tags
// podem misinterpretar CDATA ou entities).
//
// Targets:
//   - Pathological XML (CDATA, entities, malformed UTF-8)
//   - Edge cases de parsing
//   - Channel leaks (panic recover não bloqueie consumer)
//
// Padrão: fuzz test + assertion CRÍTICA que nenhum panic ou deadlock.
package rules

import (
	"strings"
	"testing"
	"time"
)

// FuzzIterXMLElements_NoPanic: rodado com corpus de vetores maliciosos.
//
// Garante que iterXMLElements não panic, não deadlocka, e sempre
// fecha o canal (mesmo em error path).
func FuzzIterXMLElements_NoPanic(f *testing.F) {
	// Seeds: casos clássicos de XML pathology.
	seeds := []string{
		"",                                                              // vazio
		"<>",                                                            // tag inválida
		"<Agreg><Mod>01</Mod></Agreg>",                                 // happy
		"<Agreg><![CDATA[<Mod>99</Mod>]]></Agreg>",                     // CDATA
		"<Agreg>5 &lt; 10 &amp; ok</Agreg>",                             // entities
		"<Agreg>\x00\x01</Agreg>",                                       // control chars
		strings.Repeat("<Agreg>x</Agreg>", 100000),                     // 1.5MB spam
		"<Agreg><Nested><Deep><Mod>01</Mod></Deep></Nested></Agreg>",     // nested
		"<agreg><Mod>01</Mod></agreg>",                                  // case wrong
		"<Agreg Mod='01' ExtraAttr='evil'>x</Agreg>",                    // mixed attrs
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, content string) {
		// CHANNEL + TIMEOUT evita deadlock se algo panic dentro da goroutine.
		done := make(chan struct{})

		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PANIC no fuzz: %v (input len=%d)", r, len(content))
				}
				close(done)
			}()
			for range iterateXMLElements(content, "Agreg") {
				// Apenas itera. Não conta.
			}
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Errorf("DEADLOCK: iterateXMLElements não finalizou em 2s (input len=%d)", len(content))
		}
	})
}
