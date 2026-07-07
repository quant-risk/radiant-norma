# Sprint 44 — RESEARCH — Radar_v2 (Diff Semântico + Auto-PR)

> **Data:** 2026-07-07
> **Sprint:** 44
> **Domínio:** Radar (monitoramento regulatório)
> **Versão atual:** v3.34.24
> **Próxima:** v3.34.25

---

## 1. Contexto

### Radar_v1 — O que existe

O Radar atual (`internal/radar/radar.go`) é um worker que:
1. Faz GET em URLs BACEN (XSDs, críticas, instruções)
2. Calcula SHA-256 do conteúdo
3. Compara com a última hash conhecida (tabela `radar_baselines`)
4. Se mudou → insere alerta em `radar_alerts`

**Limitações de v1:**
- Só detecta que algo mudou — não diz **o quê** mudou
- Não gera diff estruturado (ex: "regra C01 mudou de 'opcional' para 'obrigatório'")
- Não propõe atualização automática de regras

### Radar_v2 — O que vamos construir

| Feature | Descrição |
|---|---|
| **Diff semântico** | Parseia XLS/XLSX anterior e atual, extrai diff estruturado (regra nova, regra alterada, regra removida) |
| **Auto-PR** | Quando detecta mudança, cria GitHub PR com as regras atualizadas no código |

---

## 2. Arquitetura

### Componentes

```
backend/internal/radar/
  diff/                    (NOVO — diff semântico)
    parser.go               (parse XLS/XLSX → strukt)
    differ.go               (compara 2 strukt → diff entries)
    semantic.go             (enriquece diff com contexto de regras)
  autopr/                   (NOVO — GitHub PR automation)
    client.go               (GitHub API client)
    pr.go                   (cria PR com diff rules)
  radar_v2.go              (NOVO — Radar v2 service)
  radar_v2_test.go         (NOVO — testes)
```

### Diff Semântico

```go
// DiffEntry representa uma mudança numa regra.
type DiffEntry struct {
    CadocCode string   // "3040", "3050"
    RuleCode  string   // "C01", "LCR01"
    ChangeType string  // "added" | "removed" | "changed"
    Before    string   // valor antes (ou "")
    After     string   // valor depois (ou "")
    Severity  string  // "E" | "A" | "I" (se aplicável)
}

// DiffResult é o resultado completo de uma análise de diff.
type DiffResult struct {
    CadocCode   string
    SourceURL   string
    DetectedAt  time.Time
    OldHash     string
    NewHash     string
    Entries     []DiffEntry
    Summary     string  // texto legível: "2 adicionadas, 1 alterada, 0 removidas"
}
```

### Auto-PR

```go
// PRCreate cria um GitHub PR com as regras atualizadas.
// Usa GitHub API via Personal Access Token.
// Title: "[Radar] Atualização automática — CADOC {CODE}"
// Body: diff summary + lista de mudanças
// Files: regras Go atualizadas em backend/internal/audit/rules/
func (c *GitHubClient) CreateRuleUpdatePR(ctx context.Context, diff DiffResult) (*PRResult, error)
```

---

## 3. Estratégia de Diff

### Para XLS/XLSX (Críticas BACEN)

1. Baixar arquivo antigo (da baseline) e arquivo novo
2. Parsear XLSX com `excelize` ou planilha → CSV → struct
3. Comparar linha a linha (código da regra como chave)
4. Gerar `DiffEntry` para cada linha que mudou

### Para XSD

1. Baixar XSD antigo e novo
2. Parsear XML com `encoding/xml`
3. Comparar elementos (novos, removidos, com atributos alterados)
4. Gerar `DiffEntry` correspondente

---

## 4. GitHub Auto-PR

### Fluxo

```
Mudança detectada (radar v1)
    ↓
parseDiff(old_content, new_content) → DiffResult
    ↓
enrichWithContext(diff) → DiffResult com rule codes
    ↓
createGitHubPR(diff) → PR criado
    ↓
Notify (Slack/email) com link do PR
```

### Regras de criação de PR

- **Branch:** `radar/update/{cadoc}-{YYYYMMDD}`
- **Title:** `[Radar] {CADOC} atualizado — {N} regra(s) afetada(s)`
- **Reviewers:** configurável (equipe de compliance)
- **Labels:** `radar`, `automated`, `bacen`

---

## 5. Critérios de aceitação

- [ ] `DiffEntry` struct + `DiffResult`
- [ ] Parser XLSX → struct (excelize)
- [ ] `Differ.Compare(old, new) → []DiffEntry`
- [ ] `SemanticDiff.Enrich` (adiciona contexto de severidade)
- [ ] `GitHubClient.CreateRuleUpdatePR` (usa GitHub API)
- [ ] `radar_v2.go` — novo `RadarV2` service integrando tudo
- [ ] `go test ./...` 23/23 PASS
- [ ] `go vet ./...` clean
- [ ] `gofmt -l ./...` clean
- [ ] CHANGELOG entry v3.34.25

---

## 6. Riscos

| Risco | Mitigação |
|---|---|
| XLSX parsing é complexo (fórmulas, formatação) | Usar `excelize` (biblio consolidada); fallback para CSV |
| BACEN muda formato frequentemente | Versão mínima viável: só detecta mudança, não parseia estrutura |
| GitHub token expira | Refresh token automático + alert se auth falha |
| PRs duplicados | Deduplicar por hash + branch idempotente |
