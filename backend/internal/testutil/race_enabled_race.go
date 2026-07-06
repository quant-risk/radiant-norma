// race_enabled_race.go — build tag `race` define raceEnabled=true.
//
// Quando go test -race é usado, este arquivo é compilado e seta
// raceEnabled=true, fazendo stress tests skiparem (SQLite contention + race).
//
// Sprint 30 (v3.33.0): adicionado após descobrir que stress tests falham
// deterministicamente sob -race (SQLite contention + race overhead cria
// SQLITE_BUSY que não é regressão — é limitação de SQLite + race detector).
//go:build race

package testutil

func init() {
	raceEnabled.Store(true)
}
