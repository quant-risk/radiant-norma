# Radiant Norma Makefile
# Sprint 5 (v1.4.0) — adiciona alvos test, test-race, test-cover, lint.

GO ?= go
BIN_DIR ?= bin
DB ?= radiant.db

.PHONY: all build test test-race test-cover lint seed run-api clean vet fmt

all: build

## Build all binaries (api, worker, radar, verify)
build:
	$(GO) build -o $(BIN_DIR)/radiant-api ./cmd/api
	$(GO) build -o $(BIN_DIR)/radiant-worker ./cmd/worker
	$(GO) build -o $(BIN_DIR)/radiant-radar ./cmd/radar
	$(GO) build -o $(BIN_DIR)/radiant-verify ./cmd/_verify

## Run all unit tests (race-safe)
test:
	$(GO) test -race -v ./...

## Run tests with -race detector (P0 — catches concurrency bugs)
test-race:
	$(GO) test -race ./...

## Run tests with coverage report (fail if below threshold)
test-cover:
	$(GO) test -cover ./...
	@echo ""
	@echo "Coverage thresholds (Sprint 5):"
	@echo "  audit/    ≥65%  (target)"
	@echo "  auditlog/ ≥85%  (target)"

## Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

## Populate SQLite with criticas + leiautes (uses go run, no binary)
seed:
	$(GO) run ./cmd/seed

## Run API server (foreground)
run-api:
	$(GO) run ./cmd/api -db=$(DB)

## Run vet (standard Go static analysis)
vet:
	$(GO) vet ./...

## Format code (idempotent)
fmt:
	gofmt -w .

## Clean build artifacts + DB
clean:
	rm -rf $(BIN_DIR) $(DB) $(DB)-wal $(DB)-shm