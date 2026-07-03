// Command verify roda auditlog.Verify() — checagem rápida de integridade.
// Usage: go run ./cmd/_verify
package main

import (
	"fmt"
	"os"

	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/db"
)

func main() {
	d, err := db.Open("radiant.db")
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer d.Close()

	logger := auditlog.New(d)
	_, n, err := logger.Verify()
	if err != nil {
		fmt.Printf("VERIFY FAIL: %v (verified %d entries before failure)\n", err, n)
		os.Exit(1)
	}
	fmt.Printf("VERIFY OK: %d entries, chain valid\n", n)
}
