# Rerun — Como reproduzir esta auditoria

> **Run:** 20260715-170152-3a51cba
> **Data:** 2026-07-15

Comandos exatos para reproduzir a auditoria em outra máquina. Assume macOS ARM64 com Go 1.25+, Node 20+, Python 3.9+, Playwright instalado.

---

## 1. Preparação

```bash
# Clonar
git clone https://github.com/quant-risk/radiant-norma.git
cd radiant-norma

# Verificar branch/HEAD
git rev-parse --abbrev-ref HEAD
git rev-parse HEAD

# Instalar ferramentas
brew install go@1.25 node postgresql@16 sqlite
pip3 install playwright
playwright install chromium

# Verificar iCloud NÃO está bloqueando I/O:
# - copiar repo para /tmp se necessário
cp -r /Users/henrique/Library/Mobile\ Documents/com~apple~CloudDocs/projects/radiant-norma /tmp/radiant-norma
cd /tmp/radiant-norma
```

## 2. Capability probe + snapshot

```bash
# Reservar run-id
SHORT=$(git rev-parse --short HEAD)
UTC=$(date -u +%Y%m%d-%H%M%S)
RUN_ID="${UTC}-${SHORT}"
echo "$RUN_ID" > .audit-run-id
mkdir -p "e2e-audit/$RUN_ID"/{claims,fixtures,mocks,harness,logs,responses,traces,coverage,benchmarks,defects,reports}

# Snapshot do estado inicial
git status --porcelain=v2 > "e2e-audit/$RUN_ID/git-status-pre.txt"
git diff --name-only HEAD > "e2e-audit/$RUN_ID/git-dirty-names.txt"

# Capability probe
python3 -c "
import json, subprocess, platform
cap = {
  'go': subprocess.run(['go','version'], capture_output=True, text=True).stdout.strip(),
  'node': subprocess.run(['node','--version'], capture_output=True, text=True).stdout.strip(),
  'npm': subprocess.run(['npm','--version'], capture_output=True, text=True).stdout.strip(),
  'python3': subprocess.run(['python3','--version'], capture_output=True, text=True).stdout.strip(),
  'psql': subprocess.run(['psql','--version'], capture_output=True, text=True).stdout.strip(),
  'sqlite3': subprocess.run(['sqlite3','--version'], capture_output=True, text=True).stdout.strip(),
  'playwright': subprocess.run(['playwright','--version'], capture_output=True, text=True).stdout.strip(),
  'docker': 'NOT INSTALLED',
}
with open('e2e-audit/$RUN_ID/capabilities.json','w') as f: json.dump(cap, f, indent=2)
"
```

## 3. Extrair claims (Pass 1, 2, 3)

```bash
cd backend  # importante: scripts assumem backend/
python3 "e2e-audit/$RUN_ID/claims/_build_pass1.py"  # README + REDESIGN + ADRs
python3 "e2e-audit/$RUN_ID/claims/_build_pass2.py"  # OpenAPI + LLM/Postgres/Catálogo
python3 "e2e-audit/$RUN_ID/claims/_build_pass3.py"  # ROADMAP + MASTER_PLAN + CHANGELOG

# Consolidar
python3 -c "
import csv, json, collections
paths = [f'e2e-audit/$RUN_ID/claims/claims-pass{i}.csv' for i in (1,2,3)]
all_claims = []
for p in paths:
    with open(p) as f:
        for r in csv.DictReader(f):
            if r.get('claim_id'): all_claims.append(r)
out = {
    'total_claims': len(all_claims),
    'by_categoria': dict(collections.Counter(c['categoria'] for c in all_claims)),
    'by_criticidade': dict(collections.Counter(c['criticidade'] for c in all_claims)),
    'by_weight': dict(collections.Counter(int(c['peso']) for c in all_claims if c['peso'].isdigit())),
}
with open(f'e2e-audit/$RUN_ID/claims/claim-matrix.json','w') as f:
    json.dump(out, f, indent=2, ensure_ascii=False)
print('Total:', out['total_claims'])
"
```

## 4. Inventário de implementação

```bash
# Contagem de arquivos
find backend -name '*.go' | wc -l
find backend -name '*.sql' | wc -l
ls backend/cmd/ | wc -l
ls backend/internal/ | wc -l

# Generators
ls backend/internal/generator/gen*/

# Adapters
grep -E '^type.*Adapter' backend/internal/ingest/adapter.go

# Cross-doc rules
cat backend/internal/crossdoc/rules/registry.go | head -60

# Rotas reais
grep -hoE 'r\.(Get|Post|Put|Patch|Delete)\("/[^"]*"' backend/internal/api/*.go | grep -v _test.go | sort -u | wc -l
```

## 5. Contagem real de regras

```bash
for cadoc in 3040 3050 4111 2070 3026; do
  count=$(grep -rE "Code\(\) string" backend/internal/audit/rules/${cadoc}*.go 2>/dev/null | grep -v _test.go | wc -l)
  echo "CADOC $cadoc: $count Code() strings"
done
```

## 6. Build baseline (cuidado: requer disco)

```bash
# Liberar cache Go se disco cheio
go clean -cache
df -h /  # verificar > 2Gi disponível

# Build de pacotes individuais (sempre funciona)
cd backend
for pkg in internal/loggerutil internal/canonical internal/generator internal/ingest internal/audit/rules internal/crossdoc/rules internal/sta internal/api; do
  GOMAXPROCS=2 go build ./$pkg/ 2>&1 | tail -3
done

# Build ./... (pode travar por I/O do iCloud)
cd backend && GOMAXPROCS=4 go build ./... 2>&1 | tee /tmp/audit-build.log
```

## 7. Cobertura por pacote

```bash
cd backend
GOMAXPROCS=2 go test -count=1 -cover ./internal/loggerutil/ 2>&1 | tail -1
GOMAXPROCS=2 go test -count=1 -cover ./internal/senhaws/ 2>&1 | tail -1
GOMAXPROCS=2 go test -count=1 -cover ./internal/auditlog/ 2>&1 | tail -1
GOMAXPROCS=2 go test -count=1 -cover ./internal/audit/rules/ 2>&1 | tail -1
GOMAXPROCS=2 go test -count=1 -cover ./internal/crossdoc/rules/ 2>&1 | tail -1

# Race
GOMAXPROCS=2 go test -count=1 -race ./internal/sta/ 2>&1 | tail -3
```

## 8. Subir ambiente (Fase D — quando aplicável)

```bash
# Subir Postgres local
brew services start postgresql@16
createdb radiant_audit
psql radiant_audit -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"

# Subir Redis
brew services start redis

# Subir API
cd backend
DATABASE_URL="postgres://$(whoami)@localhost:5432/radiant_audit?sslmode=disable" \
RADIANT_NORMA_ADMIN_TOKEN="$(openssl rand -hex 32)" \
go run ./cmd/api -addr=:8080 &

# Verificar
curl -s http://localhost:8080/healthz | jq .
curl -s http://localhost:8080/readyz | jq .
```

## 9. Validar artefatos

```bash
# Diff router real vs OpenAPI vs SDKs (Phase D BENCH-07)
python3 << 'PY'
import yaml, re
with open("docs/openapi/v1.yaml") as f:
    spec = yaml.safe_load(f)
openapi_paths = sorted(spec.get("paths", {}).keys())
print("OpenAPI paths:", len(openapi_paths))

# Real routes from grep
import subprocess
real = subprocess.run(
    ["grep","-hoE",'r\\.(Get|Post|Put|Patch|Delete)\\("/[^"]*"',
     "backend/internal/api/*.go"],
    capture_output=True, text=True
).stdout.splitlines()
real_clean = sorted(set(re.sub(r'^r\.\w+\("(/[^"]*)"', r'\1', r) for r in real))
print("Real routes:", len(real_clean))
PY
```

## 10. Limpar recursos criados

```bash
# Matar processos da auditoria
pkill -f "go run ./cmd/api" 2>/dev/null
pkill -f "radiant-api" 2>/dev/null

# Parar serviços
brew services stop postgresql@16 2>/dev/null
brew services stop redis 2>/dev/null

# Limpar bancos efêmeros
dropdb radiant_audit 2>/dev/null

# Limpar caches temporários
rm -rf /tmp/go-cache-audit* /tmp/audit-build*.log /tmp/audit-bin 2>/dev/null

# Verificar que o checkout original não foi modificado
cd /tmp/radiant-norma
git status --porcelain=v2 | head -20
# Espera-se: apenas o diretório e2e-audit/ listado
```

## 11. Cleanup log (prova de não-modificação)

```bash
# Antes e depois da auditoria
git status --porcelain=v2 > e2e-audit/$RUN_ID/git-status-post.txt
diff e2e-audit/$RUN_ID/git-status-pre.txt e2e-audit/$RUN_ID/git-status-post.txt

# Único delta esperado: diretório e2e-audit/ criado
echo "Cleanup verified at $(date -u)" >> e2e-audit/$RUN_ID/cleanup.log
```

---

## Notas importantes

- **NÃO** usar credenciais reais em ambiente cloud (Hetzner, AWS, Vault). Esta auditoria foi projetada para rodar **inteiramente em loopback local**.
- **NÃO** confiar em `go build ./...` no diretório iCloud sem antes copiar para `/tmp` ou usar `GOMAXPROCS=2`.
- **SEMPRE** criar bancos efêmeros; nunca usar `radiant.db` do checkout.
- **SEMPRE** rodar testes contra mocks STA/Radar/Stripe/LLM em loopback, nunca contra produção.
- **PRESERVAR** o worktree e os artefatos em `e2e-audit/<run-id>/` para auditabilidade.