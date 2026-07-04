// Command verify roda auditlog.Verify() — checagem rápida de integridade.
//
// Validação 21 (F21.6): convertido para slog (era fmt.Println(err) que
// vazava err.Error() cru).
//
// Usage: go run ./cmd/_verify
package main

import (
	"log/slog"
	"os"

	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/db"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	d, err := db.Open("radiant.db")
	if err != nil {
		logger.Error("open db", "err", err.Error()) // SafeError não aplicável (err ou nada)
		os.Exit(1)
	}
	defer d.Close()

	a := auditlog.New(d)
	ok, n, err := a.Verify()
	if err != nil {
		logger.Error("VERIFY FAIL", "entries", n, "err", err.Error())
		os.Exit(1)
	}
	if !ok {
		logger.Error("VERIFY FAIL (chain break detected)", "entries", n)
		os.Exit(1)
	}
	logger.Info("VERIFY OK", "entries", n, "chain_valid", true)
}
