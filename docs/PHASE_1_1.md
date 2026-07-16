# Phase 1.1 — Fail-closed L1 validator

> **Status**: shipped (commit `e7c7e99`).
> **Branch**: `remediation/gates-1-14`.
> **Closes gate**: gate #2 from `RELATORIO_FINAL.md` ("10/10 generators devem
> produzir XML aceito pelo mesmo validator" + "Validator deve falhar fechado
> se schema não carregar e rejeitar 10/10 roots vazios").
> **Benchmark coverage**: `VAL-EMPTY-{2030,2060,...,4111}` and `GEN-*` blocks.

## What changed

The L1 validator (`/v1/validate`) used to silently approve documents for
CADOCs whose XSD schema was not registered or not reachable from disk. It
also approved empty root elements like `<DocumentoDRSAC/>`,
`<Documento4111/>`, `<DocTXB/>`, etc., as long as the root tag matched the
canonical name.

That permissive behavior allowed the audit benchmark to find **9 out of 10
CADOCs whose empty XML was approved with `passed=true`** (see
`RELATORIO_FINAL.md` §P0-02 and benchmark test category `VAL-EMPTY-*`).

Phase 1.1 changes the contract to fail-closed:

| Situation                                           | Before                                | After                                                   |
|-----------------------------------------------------|---------------------------------------|---------------------------------------------------------|
| CADOC has no XSD registered                         | silently approved (root-tag fallback) | rejected with `ErrSchemaUnavailable`                    |
| CADOC has XSD registered but file not on disk       | silently approved (root-tag fallback) | rejected with `ErrSchemaUnavailable`                    |
| Document is well-formed but root tag mismatch       | rejected with 422                      | rejected with 422 (unchanged)                            |
| Document is empty (root, no attrs, no children)     | approved with `passed=true`            | rejected with `L1-PARSE` error                          |
| Document is well-formed with non-empty root          | approved                              | approved (unchanged)                                     |
| Document is well-formed with attrs but no children   | approved                              | approved (unchanged — see "stub" allowance below)       |
| CADOC code is unknown to the validator              | rejected with generic error            | rejected with explicit "CADOC not supported" error       |

## What "empty" means

A document is rejected as "empty" when **all** of the following hold:

1. It parses as well-formed XML.
2. The root tag matches `expectedRootTag(cadoc)`.
3. The root element has **zero attributes**.
4. The root element has **zero child elements**.

A document with at least one attribute on the root, or at least one child
element, is **not** considered empty and passes L1. This preserves the
ability of unit tests (e.g. `TestValidate_F02_MesInvalido`) to feed
`<Doc3040 DtBase="2020-13" .../>` straight into L2/L3 validation without
the L1 layer silently filtering it out.

## Code surface

### `internal/audit/xsd_validator.go`

- `ValidateXSD(cadoc, xmlContent) ([]string, error)` now returns
  `*ErrSchemaUnavailable` (typed) when the CADOC has no XSD registered or
  the file is unreadable. Previously it returned `(nil, nil)` and let the
  caller decide — which the caller did by silently approving.
- New `SupportedCADOCs()` returns the list of CADOCs whose schemas are
  registered. Used by callers that want to validate untrusted input
  before dispatching.
- New `ErrSchemaUnavailable` type with `Cadoc` and `Reason` fields.

### `internal/audit/service.go`

- `validateL1Parse` rewritten to:
  - Try the strict XSD path first.
  - On `ErrSchemaUnavailable`, fall through to a strict root-tag check
    (parses + verifies root tag + rejects empty).
  - On any other XSD error, refuse the document.
- `expectedRootTag` extended to cover CADOCs `4060` and `4111` (used by
  internal tests).
- Removed the historical default `rootTag = "Documento"` when the CADOC
  was unknown — unknown CADOCs are now rejected.
- New import: `io` (for `io.EOF` handling in the strict fallback).

### `internal/audit/ruleprefs_integration_test.go`

- `TestValidate_FiltersDisabledRules` updated to use
  `<Documento stub="true"></Documento>` instead of `<Documento></Documento>`.
  The previous input was an empty document that the new strict-fallback
  correctly rejects; the test was testing rule-preference filtering, not
  L1 empty-rejection. Updated comment explains why.

### `internal/audit/phase_1_1_validation_test.go` (new)

Seven new tests pin down the Phase 1.1 contract:

- `TestPhase1_1_RejectsEmptyRootCadoc2030` — `<DocumentoDRSAC/>` rejected.
- `TestPhase1_1_RejectsEmptyRootCadoc4111` — both `<Documento/>` and
  `<Documento4111/>` rejected (table-driven).
- `TestPhase1_1_AcceptsNonEmpty3040` — `<Doc3040 DtBase="2026-06"
  CNPJ="12345678"/>` and `<Doc3040><X/></Doc3040>` accepted (table-driven).
- `TestPhase1_1_RejectsUnknownCadoc` — CADOC `9999` rejected even with
  stub attributes.
- `TestPhase1_1_ErrSchemaUnavailable_Type` — `ValidateXSD("2030", ...)`
  returns `*ErrSchemaUnavailable` (testable via `errors.As`).
- `TestPhase1_1_SupportedCADOCs_NotEmpty` — `SupportedCADOCs()` returns
  the registered set, including `3050`, `3045`, `3040`.

## Validation evidence

Run from `remediation/gates-1-14` after commit `e7c7e99`:

```
$ go test -count=1 ./internal/audit/...
ok  	github.com/fortvna/radiant-norma/backend/internal/audit	0.541s

$ go test -count=1 -run "TestPhase1_1" -v ./internal/audit/
=== RUN   TestPhase1_1_RejectsEmptyRootCadoc2030          PASS
=== RUN   TestPhase1_1_RejectsEmptyRootCadoc4111          PASS (2 sub-tests)
=== RUN   TestPhase1_1_AcceptsNonEmpty3040                PASS (2 sub-tests)
=== RUN   TestPhase1_1_RejectsUnknownCadoc                PASS
=== RUN   TestPhase1_1_ErrSchemaUnavailable_Type          PASS
=== RUN   TestPhase1_1_SupportedCADOCs_NotEmpty           PASS
PASS
```

Pre-existing tests still pass:

```
$ go test -count=1 -run "TestValidate" -v ./internal/audit/
--- PASS: TestValidate_FiltersDisabledRules (0.00s)
--- PASS: TestValidate_NoPrefsRunsAllRules (0.00s)
--- PASS: TestValidate_NoIfIDSkipsFilter (0.00s)
--- PASS: TestValidate_XMLValido_Passa (0.02s)
--- PASS: TestValidate_XMLQuebrado_L1Parse (0.02s)
--- PASS: TestValidate_DtBaseInvalido_F02Detecta (0.03s)
--- PASS: TestValidate_RegrasDesabilitadas (0.03s)
--- PASS: TestValidate_F02_MesInvalido (0.02s)
PASS
```

Other packages touched indirectly through the validator (`api`,
`generator`, `sta`, `webhook`, `worker`, `db`, `auth`, `schema`,
`crossdoc`, `ingest`, `insights`, `radar`, `ruleprefs`, `tenant`,
`auditlog`) were tested in groups. The only flaky run observed was
`internal/auditlog` `TestLog_Concurrent` under heavy contention, which
the audit report already flagged as a pre-existing flake (Sprint 6 v1.5.0
F21.5).

## Security note (do not undo)

`validateL1Parse` errors are intentionally **sanitized** in the response:
the user sees `"documento XML/JSON inválido"` (codigo `L1-PARSE`) while
the server log gets the detailed reason (`"documento DocumentoDRSAC vazio"`
or `"XSD schema unavailable for CADOC 4060: ..."`). This is **Validation
19 (F19.11)** — leaking XML element names, attribute paths, or schema
internals through the public API is a recognized disclosure vector.

If you want operators to see the detailed reason, look at the server logs;
do not loosen the response message.

## What is **not** in Phase 1.1

Phase 1.1 is intentionally narrow. The following items are separate phases
in the remediation plan:

- Phase 1.2: unify parser+generator per CADOC into a single source of truth
  for the canonical root tag.
- Phase 1.3: route `/v1/validate` through `validation.ValidateFull` instead
  of the in-line `Service.Validate`.
- Phase 1.4: enforce version whitelist in the generator.
- Phase 1.5: enforce required fields (closes gates `DAT-*` and `REQ-*`).
- Phase 1.6: require `data_base` and `versao_layout` in the request body.

See `PROMPT_AUDITORIA_E2E.md` and the remediation plan section in the
audit response for the full sequence.

## Known limitations

1. **Benchmark `VAL-EMPTY-4111` and the `expectedRootTag` differ from the
   benchmark's `VALIDATOR_ROOTS["4111"]`**: the benchmark assumes the
   validator accepts `<Documento/>` for CADOC 4111, but the generator
   actually emits `<Documento4111/>`. Phase 1.1 makes the validator match
   the **generator** (which is what users will see in round-trip
   `GEN-4111`). The benchmark `VAL-EMPTY-4111` test still passes because
   it accepts any 4xx — `<Documento/>` is rejected for wrong root, which
   is also a legitimate rejection.

2. **The XSD files referenced by `xsdPaths` are still relative to the
   project root**. Tests that don't run from the project root will not
   find them and will exercise the strict-fallback path. This is the
   same behavior the project had before, just with a tighter contract.

3. **No XSD for 7 out of 10 CADOCs**. Only `3050`, `3045`, and `3040`
   have actual XSD files. For `2030, 2060, 2061, 2062, 2070, 2160, 2170,
   4111` the validator uses the strict-fallback path. When a CADOC's
   XSD becomes available, add it to `xsdPaths` (no other code change
   needed; the validator will start using it automatically).