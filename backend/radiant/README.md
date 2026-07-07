# Radiant Norma Go SDK

Cliente Go oficial para a [Radiant Norma API](https://radiantnorma.com).

## Instalação

```bash
go get github.com/fortvna/radiant-norma/backend/radiant
```

## Uso rápido

```go
package main

import (
    "context"
    "log"

    radiant "github.com/fortvna/radiant-norma/backend/radiant"
)

func main() {
    cfg := radiant.Config{
        APIKey:  "rn_live_...",
        BaseURL: "https://api.radiantnorma.com",
    }
    c := radiant.New(cfg)

    ctx := context.Background()

    // Validar documento 3040
    xml3040, _ := os.ReadFile("documento_3040.xml")
    result, err := c.Cadocs.Validate(ctx, "3040", xml3040)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("valid=%v errors=%d warnings=%d", result.Valid, len(result.Errors), len(result.Warnings))

    // Listar regras
    rules, err := c.Audit.ListRules(ctx, "3040")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("%d regras disponíveis", len(rules))

    // Radar scan
    scan, err := c.Radar.Scan(ctx, "3040")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("radar status=%s changes=%d", scan.Status, len(scan.Changes))

    // Insights LLM (requer llm_insights_enabled=true no tenant)
    answer, err := c.Insights.Ask(ctx, "quais são as principais falhas de compliance?")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("insights: %s", answer.Answer)

    // Schema changelog
    versions, _ := c.Schemas.GetChangelog(ctx, "3040")
    for _, v := range versions {
        log.Printf("version %s: %s", v.EffectiveFrom, v.Changelog)
    }
}
```

## Autenticação

Obtenha sua API key em [radiantnorma.com/dashboard](https://radiantnorma.com).

## Pacotes

| Pacote | Descrição |
|--------|-----------|
| `radiant` | Cliente principal + types compartilhados |
| `radiant/cadocs` | Validação e envio de documentos |
| `radiant/audit` | Regras de auditoria |
| `radiant/radar` | Detecção de mudanças de layout |
| `radiant/insights` | Insights LLM |
| `radiant/schemas` | Schema registry |

## Error handling

Todos os métodos retornam `error`. Erros de API são wrapped:

```go
_, err := c.Cadocs.Validate(ctx, "3040", data)
if err != nil {
    // err contém código e mensagem da API
    log.Fatalf("validation failed: %v", err)
}
```

## License

Proprietary — Radiant Norma.
