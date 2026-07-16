# Phase 2 — Wizard funcional ponta a ponta

> **Status**: shipped
> **Branch**: `remediation/gates-1-14`
> **Closes gate**: gate #8 from `RELATORIO_FINAL.md` ("wizard deve gerar XML ao completar")
> **Benchmark coverage**: `WIZARD-*`.

## Problema

Antes desta fase, o wizard (`/generate/wizard/*`) permitia avanzar pelos steps (select_cadoc → select_source → map_fields → preview → generate) mas quando chegava em `StepGenerate`, **não executava a geração de XML**. O XML ficava vazio.

## Solução

Phase 2 adiciona `executeWizardGeneration()` que é chamada quando o wizard avança para `StepGenerate`:

1. Recupera o `CanonicalDocument` do JSON salvo na sessão
2. Resolve o generator correto pelo CADOC
3. Executa `g.Generate(ctx, &doc, dataBase)`
4. Salva o XML gerado na sessão via `SetGeneratedXML()`

## Mudanças de código

### `internal/api/wizard_handlers.go`

```go
// Phase 2: quando avança para StepGenerate, executa a geração.
if updated.Step == wizard.StepGenerate && session.Step != wizard.StepGenerate {
    if err := s.executeWizardGeneration(r.Context(), ifID, updated); err != nil {
        slog.Error("wizard generation failed", "session", id, "err", err)
        s.WizardStore.SetError(r.Context(), id, []string{err.Error()})
    }
    updated, _ = s.WizardStore.Get(r.Context(), id)
}
```

Nova função `executeWizardGeneration`:
```go
func (s *Server) executeWizardGeneration(ctx context.Context, ifID string, session *wizard.Session) error {
    g := s.resolveGenerator(session.CadocCode)
    // ... reconstrói CanonicalDocument do JSON
    // ... executa g.Generate()
    // ... salva XML via s.WizardStore.SetGeneratedXML()
}
```

## Fluxo do wizard

```
1. POST /generate/wizard/start → cria sessão (StepSelectCadoc)
2. PUT /generate/wizard/{id} + {cadoc_code: "3040"} → avança para StepSelectSource
3. PUT /generate/wizard/{id} + {source_type: "manual"} → avança para StepMapFields
4. PUT /generate/wizard/{id} + {canonical_json: {...}} → avança para StepPreview
5. PUT /generate/wizard/{id} → avança para StepGenerate E executa geração
6. GET /generate/wizard/{id}/xml → retorna XML gerado
```

## O que não está em Phase 2

- Phase 3: RBAC readonly middleware
- Phase 4: STA persist + dedupe + retry + DLQ
- Phase 5: Webhook inicializar + assinatura + idempotência
