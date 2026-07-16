# Plano de execução BENCHs pós-cópia para /tmp

## Problema
`go build ./...` no diretório iCloud trava por I/O bloqueante do `Mobile Documents`.

## Solução
Copiar `backend/` para `/tmp/radiant-build/backend/` (filesystem local APFS, sem sync).

## BENCHs a executar (em ordem de prioridade)

### BENCH-00 — Reprodução/build/verdade documental (parcial)
- `go build ./...` em `/tmp/radiant-build/backend` — verificar que **todos** os 287 arquivos compilam
- `go vet ./...` — verificar warnings
- `go test -list ./... | wc -l` — contar testes top-level
- Comparar com README "516 testes"

### BENCH-02 — Matriz de generators (runtime)
- `registry.Get("3040")`, `registry.Get("3050")`, ... cada um
- Verificar quais têm `RequiredFields()` retornando lista não-vazia
- Verificar quais têm `Generate()` real (não stub)

### BENCH-03 — L1-L4 + cobertura de regras (parcial)
- Cobertura por pacote (já feito audit/rules, crossdoc/rules)
- Adicionar: `internal/api`, `internal/audit`, `internal/crossdoc`, `internal/db`, `internal/insights`, `internal/realtime`, `internal/ruleprefs`, `internal/schema`, `internal/sta`, `internal/worker`
- `go test -race ./...` (com paralelismo reduzido)

### BENCH-04 — Ingestão/conectores (parcial)
- Cada adapter: chamar `Fetch` com Canonical vazio e verificar erro
- FileAdapter: parsear CSV/XLSX/JSON simples
- `HealthCheck` retorna nil em mocks válidos?

### BENCH-05 — Cross-doc (parcial)
- Carregar registry e listar 25 regras
- Para cada regra, verificar `RequiredDocs()` e tentar `Apply()` com DocSet vazio

### BENCH-09 — Race detector
- `go test -race ./...` por pacote pequeno

## Limitações aceitas
- Sem Docker → não roda imagem Docker
- Sem Postgres real rodando → testes de RLS não runtime
- Sem STA real → testes STA ficam MOCK_FIEL
- Sem Playwright UI → frontend não exercitado em runtime

## Outputs esperados
- `benchmarks/build-output.txt` — saída do `go build ./...`
- `benchmarks/coverage-all.txt` — tabela de cobertura por pacote
- `benchmarks/test-count.txt` — número real de testes
- `benchmarks/generator-inventory.json` — quais generators têm implementação real
- `benchmarks/crossdoc-inventory.json` — quais cross-doc rules são executáveis
- Atualização de `benchmarks/results.json` com dados reais
- Atualização do scorecard