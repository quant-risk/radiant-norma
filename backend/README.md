# Radiant Norma — Backend

Single-binary Go backend for Brazilian regulatory CADOC validation (BACEN).

## Quick Start

```bash
make build      # Build all binaries
make seed       # Populate SQLite with criticas + leiautes
make test       # Run unit tests
make test-race  # Run with -race detector (catches concurrency bugs)
make test-cover # Run with coverage report
make lint       # Run golangci-lint (if installed)
```

## Stack

- **Go 1.24+** (uses `embed.FS`, generics)
- **SQLite** via `modernc.org/sqlite` (pure Go, no CGo)
- **chi** router for HTTP
- **slog** for structured logging
- **No external LLM API** (Light mode — uses MCP sampling instead)

## Architecture

```
backend/
├── cmd/
│   ├── api/      # REST API server
│   ├── worker/   # Background envio processor (STA stub)
│   ├── radar/    # Background Radar Regulatório
│   ├── seed/     # CLI to populate DB from criticas.json
│   └── _verify/  # CLI to verify audit_log chain
└── internal/
    ├── api/         # HTTP handlers (chi router)
    ├── audit/       # Norma Audit (validation engine)
    │   ├── rules/   # 3040-specific rules (25 implemented)
    │   └── service.go
    ├── auditlog/    # Tamper-evident hash chain
    ├── db/          # SQLite + migrations (embed.FS)
    ├── radar/       # Regulatory Radar (change detection)
    ├── schema/      # Schema Registry (versioned by data-base)
    ├── sta/         # STA stub client
    └── testutil/    # Test helpers (in-memory SQLite)
```

## Database

SQLite single-file (`radiant.db`). Uses:
- `journal_mode(WAL)` — concurrent readers + 1 writer
- `foreign_keys(ON)` — referential integrity
- `busy_timeout(5000)` — wait 5s on lock
- `_txlock=immediate` — BEGIN IMMEDIATE on every tx (race-free)

In production, swap to Postgres by changing driver in `internal/db/db.go`.

## Testing

- `internal/testutil/db.go` — `NewTestDB(t)` returns in-memory SQLite with migrations
- All tests are **race-safe** (`go test -race ./...`)
- Coverage targets:
  - `audit/` ≥65% (current: 68.7%)
  - `audit/rules/` ≥60% (current: 63.2%)
  - `auditlog/` ≥85% (current: 90.8%)
  - `radar/` ≥70% (current: 78.1%)

## Critical rules

1. **`_txlock=immediate` is mandatory** — without it, concurrent auditlog.Log loses entries.
   Tests in `auditlog/log_test.go::TestLog_Concurrent` regression-protect this.

2. **`LoadCriticas` uses `sql.NullString` for nullable columns** —
   `gravidade`, `mensagem_erro`, `data_base_inicio` can be NULL.
   Test in `audit/service_test.go::TestLoadCriticas_MensagemErroNULL` regression-protects.

3. **`F02Datas` validates date semantics** (mês 01-12, dia 01-31).
   Test in `audit/rules/3040_test.go::TestF02_Datas_Invalidas` regression-protects.

4. **`recordBaseline` normalizes label** (espaços → underscores).
   Test in `radar/radar_test.go::TestScanSource_FirstScan` regression-protects.

5. **`radar_alerts` no UNIQUE constraint** (migration 003).
   Removed to allow multiple alerts with same title in rapid succession.