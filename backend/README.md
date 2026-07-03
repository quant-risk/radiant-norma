# Radiant Sentinel — Backend (Sprint 3+)

> **API REST + Schema Registry + Sentinel Audit microservice + STA client stub.**
> Sprint 3 entregou a primeira iteração funcional end-to-end com SQLite + Go.
> Postgres-ready (abstração `database/sql`, troca de driver em 1 linha).

## Status: ✅ Sprint 3 funcional

| Componente | Status |
|---|---|
| Schema Postgres-ready (5 tabelas) | ✅ |
| Schema Registry (versionado por data-base) | ✅ |
| Sentinel Audit (L1+L2) | ✅ |
| Audit log tamper-evident (hash chain) | ✅ |
| STA client stub | ✅ |
| API REST (7 endpoints) | ✅ |
| Seed CLI (JSON → DB) | ✅ |
| JWT/OAuth | ❌ Sprint 4 |
| Postgres driver real | ❌ Sprint 4 |
| STA Web/WS real (Playwright) | ❌ Sprint 4 |
| Cross-doc engine (L3) | ❌ Sprint 4 |
| Frontend (Sentinel Console) | ❌ Sprint 5 |

## Stack

- **Go** 1.22+
- **chi** router (`github.com/go-chi/chi/v5`)
- **SQLite** com `modernc.org/sqlite` (pure-Go, sem CGo) — produção trocará pra Postgres com `pgx`
- **embed.FS** pra migrations self-contained

## Estrutura

```
backend/
├── cmd/
│   ├── api/main.go              # API REST entrypoint
│   └── seed/main.go             # Popular DB com JSONs extraídos
├── internal/
│   ├── api/server.go            # Handlers HTTP REST
│   ├── audit/service.go         # Sentinel Audit (L1+L2)
│   ├── auditlog/log.go          # Hash chain tamper-evident (LGPD/SOC2)
│   ├── db/
│   │   ├── db.go                # Conexão (SQLite/Postgres)
│   │   ├── migrate.go           # embed.FS migrations
│   │   └── migrations/001_initial.sql
│   ├── schema/registry.go       # Schema Registry versionado
│   └── sta/stub.go              # STA client stub
└── go.mod
```

## Quickstart

### Build

```bash
cd backend
go build -o ./bin/api ./cmd/api
go build -o ./bin/seed ./cmd/seed
```

### Seed (popula banco com JSONs extraídos)

```bash
./bin/seed \
  -json ../_catalogos/criticas.json \
  -leiautes ../_catalogos/leiautes.json \
  -xsd ../_catalogos/3040_generated.xsd \
  -db ./radiant.db
```

Output esperado:
```
importando cadoc=3040 count=361
importando cadoc=3050 count=170
...
✓ criticas importadas total=968
✓ schemas importados total=8
✓ seed completo
```

### Rodar API

```bash
# Default: porta 8080, banco radiant.db no CWD
./bin/api

# Customizado via env vars:
RADIANT_DB=/tmp/radiant.db RADIANT_ADDR=:8888 ./bin/api
```

## Endpoints REST

### `GET /healthz`
Liveness check.
```bash
curl http://localhost:8080/healthz
# {"status":"ok","time":"2026-07-03T17:43:08Z","version":"1.2.0"}
```

### `GET /v1/schemas`
Lista CADOCs suportados.

### `GET /v1/schemas/{cadoc}`
Retorna schema effective de um CADOC.
```bash
curl -H "X-IF-ID: 12345678" http://localhost:8080/v1/schemas/3040
```

### `GET /v1/schemas/{cadoc}/versions`
Histórico de versões do schema.

### `GET /v1/rules/{cadoc}`
Lista críticas habilitadas de um CADOC.
```bash
curl -H "X-IF-ID: 12345678" http://localhost:8080/v1/rules/3040 | jq '.total'
# 320
```

### `POST /v1/validate`
**Sentinel Audit** — valida XML/JSON contra regras do CADOC.

```bash
# Validação com XML
curl -X POST http://localhost:8080/v1/validate \
  -H "X-IF-ID: 12345678" \
  -H "Content-Type: application/json" \
  -d '{
    "cadoc_code": "3040",
    "data_base": "2026-07",
    "content_type": "application/xml",
    "xml": "<?xml version=\"1.0\"?><Doc3040 ...>...</Doc3040>"
  }'

# Validação com JSON (ex: 3044)
curl -X POST http://localhost:8080/v1/validate \
  -H "X-IF-ID: 12345678" \
  -H "Content-Type: application/json" \
  -d '{
    "cadoc_code": "3044",
    "content_type": "application/json",
    "xml": "{\"cnpjIF\":\"12345678\",\"operacoes\":[...]}"
  }'
```

Resposta:
```json
{
  "cadoc_code": "3040",
  "data_base": "2026-07",
  "xml_hash": "7f6ef5d2541e317f...",
  "passed": true,
  "errors": [],
  "warnings": [],
  "executed_at": "2026-07-03T17:43:08Z",
  "duration_ms": 2
}
```

### `POST /v1/sta/submit`
**STA stub** — gera protocolo fake (em Sprint 4 vira cliente Playwright real).
```bash
curl -X POST "http://localhost:8080/v1/sta/submit?cadoc=3040&data_base=2026-07" \
  -H "X-IF-ID: 12345678" \
  -H "Content-Type: application/xml" \
  --data @3040/exemploDesempenhoOperacao.xml
```

Resposta:
```json
{
  "ProtocolSTA": "2026070329287c9b3b181",
  "Accepted": true,
  "Rejection": null
}
```

## Auth

Todos os endpoints `/v1/*` exigem header `X-IF-ID`. Sem ele → 401.

**Sprint 3**: X-IF-ID é apenas identificador (sem JWT/OAuth).
**Sprint 4**: substituir por JWT + OAuth2 com refresh tokens.

## Banco de Dados

### Schema (SQLite no spike, Postgres-ready)

5 tabelas:

| Tabela | Função |
|---|---|
| `ifs` | Multi-tenant (CNPJ, nome, tipo, segmento, plano) |
| `schema_versions` | Versionamento por data-base (effective_from) |
| `criticas` | Regras de validação (regra + descrição + gravidade) |
| `envios` | Histórico de submissões STA (status + protocolo) |
| `audit_log` | Hash chain tamper-evident |

### Trocar pra Postgres (Sprint 4+)

Mudar **2 linhas** no `internal/db/db.go`:

```go
// Antes (SQLite):
import _ "modernc.org/sqlite"
dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)", path)

// Depois (Postgres):
import _ "github.com/jackc/pgx/v5/stdlib"
dsn := "postgres://user:pass@localhost/radiant?sslmode=disable"
```

Abstração `database/sql` cuida do resto. Migrations funcionam em ambos (SQL é portável).

## Variáveis de ambiente

| Variável | Default | Descrição |
|---|---|---|
| `RADIANT_ADDR` | `:8080` | Endereço do servidor HTTP |
| `RADIANT_DB` | `radiant.db` | Caminho do banco SQLite |

## Testes

**Sprint 3**: testes manuais via curl (todos os 7 endpoints).
**Sprint 4**: adicionar `*_test.go` + `go test ./...` + coverage.

## Smoke test rápido

```bash
# Em um terminal
cd backend
RADIANT_DB=/tmp/r.db ./bin/api

# Em outro terminal
curl -s http://localhost:8080/healthz | jq

# Validar XML 3040 exemplo
curl -s -X POST http://localhost:8080/v1/validate \
  -H "X-IF-ID: 12345678" \
  -H "Content-Type: application/json" \
  -d "$(python3 -c "import json; print(json.dumps({'cadoc_code':'3040','content_type':'application/xml','xml':open('../3040/exemploDesempenhoOperacao.xml').read()}))")" | jq

# Ver audit log
sqlite3 /tmp/r.db "SELECT id, action, target, substr(entry_hash,1,12) as hash FROM audit_log ORDER BY id DESC LIMIT 5"
```

## Roadmap

- **Sprint 4**: JWT/OAuth, Postgres driver, STA Playwright, 30+ regras 3040, Radar regulatório
- **Sprint 5**: Frontend (Sentinel Console), cross-doc engine (L3), self-host on-prem
- **Sprint 6**: SOC 2 Type I, LGPD compliance attested, multi-region

---

**Mantido por:** Time do Radiant Sentinel · Radiant Risk Solutions (marca da Fortvna)
**Versão:** 1.2.0 (Sprint 3, 2026-07-03)