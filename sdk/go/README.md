# Radiant Norma Go SDK

Cliente Go oficial para a [Radiant Norma API](https://radiantnorma.com).

## Instalação

```bash
go get github.com/fortvna/radiant-norma-go
```

## Uso rápido

```go
package main

import (
	"context"
	"fmt"
	"log"

	radiant "github.com/fortvna/radiant-norma-go"
)

func main() {
	c := radiant.New(radiant.Config{
		APIKey:  "sk-...",        // do dashboard radiantnorma.com
		IFID:    "00000",        // seu IF-ID
		BaseURL: "https://api.radiantnorma.com",
	})

	ctx := context.Background()

	// Validar um documento CADOC
	xmlData := []byte(`<Doc3040 ...>...</Doc3040>`)
	result, err := c.Cadocs.Validate(ctx, "3040", xmlData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Valid: %v\n", result.Valid)
	for _, e := range result.Errors {
		fmt.Printf("  [%s] %s\n", e.Code, e.Message)
	}

	// Radar scan — detecta mudanças de layout
	scan, err := c.Radar.Scan(ctx, "3040")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Radar status: %s\n", scan.Status)

	// Listar regras de auditoria
	rules, err := c.Audit.ListRules(ctx, "3040")
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range rules {
		fmt.Printf("  [%s] %s\n", r.Code, r.Message)
	}
}
```

## Serviços

| Serviço | Métodos |
|---|---|
| `Cadocs` | `Validate`, `ValidateCrossDoc` |
| `Audit` | `ListRules` |
| `Radar` | `Scan` |
| `Insights` | `Ask` |
| `Schemas` | `ListVersions`, `GetChangelog` |

## Configuração

```go
c := radiant.New(radiant.Config{
    APIKey:  "sk-...",       // obrigatório
    IFID:    "00000",        // IF-ID do tenant
    BaseURL: "https://api.radiantnorma.com", // opcional, default
})
```

## Validação de documento

```go
result, err := c.Cadocs.Validate(ctx, "3040", xmlData)
if err != nil {
    // erro de rede ou API
}
if !result.Valid {
    for _, e := range result.Errors {
        fmt.Printf("ERRO [%s]: %s\n", e.Code, e.Message)
    }
}
```

## Validação cross-document

```go
docs := map[string][]byte{
    "3040": xml3040,
    "4111": xml4111,
}
result, err := c.Cadocs.ValidateCrossDoc(ctx, docs)
```

## Radar scan

```go
scan, err := c.Radar.Scan(ctx, "3040")
if scan.Status == "changed" {
    for _, ch := range scan.Changes {
        fmt.Printf("  %s %s.%s: %s → %s\n", ch.Kind, ch.Tag, ch.Attr, ch.OldValue, ch.NewValue)
    }
}
```

## Schema registry

```go
versions, _ := c.Schemas.ListVersions(ctx, "3040")
for _, v := range versions {
    fmt.Printf("v%d effective=%s\n", v.ID, v.EffectiveFrom)
}

changelog, _ := c.Schemas.GetChangelog(ctx, "3040")
for _, e := range changelog {
    fmt.Printf("  %s: %s\n", e.EffectiveFrom, e.Changelog)
}
```

## Roadmap

Consulte [ROADMAP.md](../../ROADMAP.md) para sprints futuros.

##License

Proprietary — Radiant Financial Technology
