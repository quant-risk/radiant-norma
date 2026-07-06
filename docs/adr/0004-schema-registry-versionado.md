# ADR-0004: Schema Registry — versionado por data-base, GitHub source-of-truth

> **Status:** Aceito
> **Data:** 2026-07-05
> **Decisor(es):** Henrique Costa · Mavis

## Contexto

BACEN muda leiaute de CADOCs 3-5× por ano. IF precisa se adaptar rápido, sem retrabalho. Hoje:

- **Matera / Mitra / cadoc.ai:** XSD hardcoded em releases do SaaS. IF espera 2-4 semanas por release.
- **BCValidador oficial:** distribuição por download, sem versionamento automático.

Queremos:

1. Detectar mudança em ≤ 24h após BACEN publicar.
2. Publicar nova versão em ≤ 5 dias úteis.
3. IF não mexe em código (apenas config).
4. Histórico auditável e reversível.

## Decisão

Cada release BACEN = PR no repo público **`radiant-norma-schemas`** (GitHub).

**Fluxo:**

```
BACEN publica novo XSD v12.3 (2024-07-15)
         ↓
Radar detecta (24h, ver ADR-0005)
         ↓
Auto-PR aberto em radiant-norma-schemas:
  - schemas/3040/2024-07/3040.xsd
  - schemas/3040/2024-07/CHANGELOG.md (diff vs 2024-06)
  - schemas/3040/2024-07/criticas.json (se houver)
         ↓
CI valida:
  - XSD parse válido
  - XML exemplo do BACEN parseia sem erro
  - Regras públicas atualizadas (não-regressão)
         ↓
Eng lead revisa + merge
         ↓
Webhook publica em plataforma.schemas (Postgres)
         ↓
IFs impacted recebem notificação (SSE + email)
         ↓
Catálogo local do Radiant Norma atualiza em ≤ 5min (cache 5min)
```

**Tabela Postgres:**

```sql
CREATE TABLE plataforma.schemas (
    id              TEXT PRIMARY KEY,  -- sha256(cadoc + version + content)
    cadoc           TEXT NOT NULL,
    version         TEXT NOT NULL,  -- data-base "2024-07"
    xsd             BYTEA NOT NULL,
    changelog       TEXT,
    source_url      TEXT,
    source_sha256   TEXT,
    released_at     TIMESTAMPTZ NOT NULL,
    eol_at          TIMESTAMPTZ,
    UNIQUE(cadoc, version)
);

CREATE INDEX idx_schemas_cadoc_effective ON plataforma.schemas(cadoc, released_at DESC) WHERE eol_at IS NULL;
```

**Schema Registry Go:**

```go
type Schema struct {
    ID         string
    Cadoc      string
    Version    string
    XSD        []byte
    ReleasedAt time.Time
    EOLAt      *time.Time
}

type SchemaRegistry interface {
    Effective(cadoc string, at time.Time) (*Schema, error)
    List(cadoc string) ([]Schema, error)
    Publish(s Schema) error            // platform admin only
    NotifyChange(cadoc string) error   // SSE + webhook
}
```

## Consequências

**Positivas:**
- ✅ Schema-first: zero deploy de código pra BACEN mudar leiaute.
- ✅ Histórico auditável (git log completo).
- ✅ Community contributions (open source model).
- ✅ Marketing orgânico (devs BACEN veem o repo).
- ✅ Trust signal: "auditável publicamente".

**Negativas:**
- ❌ Dependência de GitHub (vendor lock-in). Mitigável: GitLab self-hosted como backup.
- ❌ CI lento se valida XSD contra exemplos BACEN. Mitigável: cache de parse.
- ❌ Race entre múltiplas PRs (ex: 2X atualizações simultâneas). Mitigável: lock file + merge queue.

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| **Confluent Schema Registry** | Feito pra Avro/Protobuf, não XSD/XML. Overhead de infra. |
| **Schema armazenado in-app** | Sem versionamento, sem audit trail, retrabalho a cada release. |
| **Custom registry fechado** | Não permite community contributions, marketing pior. |

## Notas de implementação

- Repo `radiant-norma-schemas`: licença CC-BY 4.0, CONTRIBUTING.md detalhado, CODEOWNERS.
- CI em GitHub Actions: `xmllint --schema`, regression tests, auto-PR review por bot.
- Versioning: SemVer-like `YYYY-MM[-patch]`. Patch = fix dentro da mesma data-base (raro).
- EOL: schema anterior fica `eol_at = now + 90 days` pra dar tempo de IF migrar.
- Cache app-side: in-memory LRU 100MB, refresh a cada 5min.