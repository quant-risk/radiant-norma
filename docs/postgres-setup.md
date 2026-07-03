# Postgres Setup (Sprint 6 v1.5.0)

Radiant Norma suporta **dual driver**: SQLite (default, dev) ou Postgres
(produção). A escolha é feita automaticamente pelo prefixo da DSN.

## Quickstart: Postgres local

### 1. Subir Postgres via Docker

```bash
docker compose up -d postgres
```

Postgres:16-alpine rodando em `localhost:5432`, user `radiant`,
db `radiant`. Senha: `radiant` (dev only, mude em produção via env var).

### 2. Configurar a API

```bash
export DATABASE_URL="postgres://radiant:radiant@localhost:5432/radiant?sslmode=disable"
export RADIANT_NORMA_ADMIN_TOKEN="$(openssl rand -hex 32)"  # token admin
./backend/api -db "$DATABASE_URL"
```

A API detecta `postgres://` automaticamente e usa pgx/v5 (Sprint 6 v1.5.0).

### 3. Migrations

A mesma `db.Migrate()` roda em ambos os drivers. As migrations
existentes (001-005) foram escritas com features compatíveis:

- `CREATE TABLE IF NOT EXISTS` — SQLite e Postgres aceitam
- `ALTER TABLE ADD COLUMN` — ambos
- `BEGIN IMMEDIATE` — SQLite specific (no-op em Postgres)
- `INSERT OR IGNORE` — convertido automaticamente pelo pgx? **NÃO**
  - Migrations com `INSERT OR IGNORE` (migration 004 — F3) **NÃO** rodam
    em Postgres. Em produção real, criar migration separada Postgres.

## Arquitetura dual-driver

### Detecção (`db.Open`)

```go
dsn := "postgres://user:pass@host:5432/db?sslmode=disable"
db, err := db.Open(dsn)
//   ↑ IsPostgresDSN detecta prefix → openPostgres com pgx
```

### Pool de conexões

| Driver  | Max Open | Max Idle | Notas |
|---------|----------|----------|-------|
| SQLite   | 8        | 2        | _txlock=immediate serializa writes |
| Postgres | 25       | 5        | pool generoso; pgx suporta muito mais |

### Failover / Test patterns

Em testes, usar SQLite (default). Para integration tests com Postgres,
configurar:

```bash
export DATABASE_URL="postgres://test:test@localhost:5432/radiant_test?sslmode=disable"
go test ./...
```

Sprint 6 v1.5.0 não inclui integration tests com Postgres real (precisa
de Postgres rodando, pularia CI). Adicionar em Sprint 7 com
testcontainers-go.

## Limitações conhecidas

| Limitação | Workaround | Sprint alvo |
|-----------|------------|-------------|
| `_txlock=immediate` é SQLite-only | Sobra `BEGIN IMMEDIATE` ignorado em Postgres | OK (no-op) |
| `INSERT OR IGNORE` migration 004 | Criar migration Postgres-flavor específica | Sprint 7 |
| Sem integration test com Postgres real | Adicionar via testcontainers | Sprint 7 |
| Sem `Listen + Notify` (notification channel) | Polling alternativo | Não urgente |
| SEM read replica support | Conexão única | OK pra v1 |

## Make targets (a serem adicionados em Sprint 7)

```makefile
db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-migrate:
	$(API) -db $$DATABASE_URL

db-test:
	$(GO) test -tags=integration ./...  # tests que precisam de Postgres
```
