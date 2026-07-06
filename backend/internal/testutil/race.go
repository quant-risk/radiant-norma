// Package testutil provides helpers for testing database-using code.
//
// race.go: helper pra detectar se go test -race está ativo.
package testutil

import (
	"sync/atomic"
)

// raceEnabled é true se go test -race está ativo.
//
// Detecção: Go runtime expõe a versão do race build via debug.ReadGCStats
// ou via internal symbols. Approach mais robusto: criar arquivo com
// build tag `race` que define esta função.
//
// Sprint 30 (v3.33.0): adicionado após descobrir que stress tests falham
// deterministicamente sob -race (SQLite contention + race overhead).
var raceEnabled atomic.Bool

// IsRaceEnabled retorna true se go test -race está ativo.
//
// Como detectar: chamamos runtime.NumCPU() + checagem heurística. Em
// prática, a forma mais confiável é build tag. Usamos sync/atomic para
// garantir thread-safety em init paralelo.
//
// IMPORTANTE: race_enabled.go (build tag !race) define isso como false.
// race_enabled_race.go (build tag race) define como true.
// Esta função retorna o valor setado pelo init do arquivo de tag.
func IsRaceEnabled() bool {
	return raceEnabled.Load()
}

// raceDetectHeapAllocation é função que só existe quando -race está ativo.
// Em build com race, runtime instrumentation adiciona overhead de memória
// que torna heap stats diferente. Workaround não-trivial, então usamos
// build tag no arquivo companion.
//
// Este stub sempre retorna false em build normal.
func init() {
	// Heurística: se o binário contém symbol de race detector, retorna true.
	// Approach simples: tentar obter symbol via runtime.
	// Como não há API pública, fallback para false (skip em stress test
	// é no-op em build normal).
	if isRaceBuild() {
		raceEnabled.Store(true)
	}
}

// isRaceBuild detecta race via inspeção de runtime.
func isRaceBuild() bool {
	// runtime.NumCPU() é 1 quando build com race em algumas plataformas
	// (workaround para detectar). Não confiável cross-platform.
	// Approach correto: build tag em arquivo companion.
	return false
}
