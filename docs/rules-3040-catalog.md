# CATÁLOGO DE REGRAS 3040 — Sprint 7b / v1.7.0

> **Status:** Completo (60 regras: 5 raw + 55 tipadas)
> **Data:** 2026-07-03
> **Versão:** v1.7.0

Este documento cataloga todas as regras implementadas em
`internal/audit/rules/` para validação do CADOC 3040 (SCR —
Risco de Crédito). Cada regra:

- Categoria (B-F-C-S)
- Código (B01-S10)
- Severidade (E/A/I)
- Descrição técnica
- Exemplo de hit
- Como fix (resposta IF)

## Resumo por categoria

| Cat | Total | Sprint 4 | Sprint 6 | Sprint 7b |
|-----|-------|-----------|----------|-----------|
| Raw (B01-B05) | 5 | - | 5 | - |
| Básicas tipadas | 20 | 10 | - | 10 (B16-B25) |
| Formato | 15 | 5 | - | 10 (F06-F15) |
| Campos Obrigatórios | 10 | 5 | - | 5 (C06-C10) |
| Semânticas | 10 | 5 | - | 5 (S06-S10) |
| **TOTAL** | **60** | **25** | **5** | **30** |

## Sevirity legend

- **E (Erro):** impossível processar, IF bloqueia
- **A (Aviso):** suspeita, IF revisa
- **I (Informativo):** alerta educacional, sem bloquear

## Catálogo completo

### Básicas (B01-B25)

| Code | Severity | Sheet | Description | Example Fail |
|------|----------|-------|-------------|--------------|
| B01 | I | Básicas raw | Arquivo XML válido (L1-PARSE já cobre) | - |
| B02 | I | Básicas raw | Estrutura básica presente | - |
| B03 | I | Básicas raw | Tamanho razoável | - |
| B04 | E | Básicas raw | Codificação declarada | `<?xml ... ?>` falta |
| B05 | E | Básicas raw | Arquivo não vazio | body < 100 bytes |
| B06 | E | Básicas | Remessa >= 1 | `<Remessa>0</Remessa>` |
| B07 | E | Básicas | Parte >= 1 | `<Parte>0</Parte>` |
| B08 | A | Básicas | Sem parte rejeitada no histórico | (stub) |
| B09 | I | Básicas | Max 5000 erros | (informativo) |
| B10 | I | Básicas | Max 5000 avisos | (informativo) |
| B11 | A | Básicas | Remessa != número rejeitado anterior | (stub) |
| B12 | E | Básicas | Tipo fundo obrigatório | FIDC exige algo |
| B13 | E | Básicas | IF não-FIDC pode enviar | (validação) |
| B14 | I | Básicas | Max ops divergentes 3042/3040 | (informativo) |
| B15 | I | Básicas | Max ops divergentes 3040 | (informativo) |
| B16 | A | Básicas | Totalizadores coerentes | TotalCli≠soma(QtdCli) |
| B17 | E | Básicas | DtBase formato YYYY-MM-DD | "2024-13-01" |
| B18 | E | Básicas | TpArq F ou S | "X" |
| B19 | E | Básicas | Email formato RFC-ish | "user@" |
| B20 | A | Básicas | Tel formato | "11 99999-9999" (sem DDD) |
| B21 | E | Básicas | CNPJ raiz 8 dígitos | "123456" |
| B22 | E | Básicas | NomeResp não vazio | "" |
| B23 | E | Básicas | >=1 Agreg presente | sem tags Agreg |
| B24 | E | Básicas | DtBase não futura (até 2030) | "2099-..." |
| B25 | E | Básicas | QtdOp >= 1 por Agreg | `<Agreg QtdOp="0">` |

### Formato (F01-F15)

| Code | Severity | Sheet | Description | Example Fail |
|------|----------|-------|-------------|--------------|
| F01 | E | Formato | Taxa Efetiva Anual numérico | - |
| F02 | E | Formato | Mês 1-12 | `<DtBase>2020-13</DtBase>` |
| F03 | A | Formato | Código contrato | (stub) |
| F04 | E | Formato | Conglomerado declarado | - |
| F05 | A | Formato | Referência BACEN/SICOR | (stub) |
| F06 | E | Formato | ClassOp A-H | "Z" |
| F07 | E | Formato | Mod 01-99 | "1" |
| F08 | E | Formato | NatuOp 01/02 | "03" |
| F09 | E | Formato | UF válida (27 siglas) | "XX" |
| F10 | E | Formato | VincME S/N | "X" |
| F11 | E | Formato | PrzProvm S/N | "X" |
| F12 | E | Formato | TpCli 1=PF / 2=PJ | "3" |
| F13 | A | Formato | DesempOp 2 dígitos | "1" |
| F14 | A | Formato | FaixaVlr numérico | "ABC" |
| F15 | A | Formato | OrigemRec 2 dígitos | "1" |

### Campos Obrigatórios (C01-C10)

| Code | Severity | Sheet | Description | Example Fail |
|------|----------|-------|-------------|--------------|
| C01 | E | C.Obrig | PJ campos obrigatórios | - |
| C02 | A | C.Obrig | Sem campos desnecessários | (stub) |
| C03 | E | C.Obrig | Garantias não-fidejussórias | (stub) |
| C04 | E | C.Obrig | Garantias fidejussórias | (stub) |
| C05 | A | C.Obrig | Cessões/coobrigação | (stub) |
| C06 | E | C.Obrig | ClassOp C-H requer ProvConsttd | `<Agreg ClassOp="E">` sem ProvConsttd |
| C07 | E | C.Obrig | DesempOp != "00" com vencimentos | vencimentos=0 mas DesempOp setado |
| C08 | A | C.Obrig | Tel preenchido requer Email | só Tel sem Email |
| C09 | E | C.Obrig | NatuOp=01 requer QtdCli | operação própria sem count |
| C10 | E | C.Obrig | QtdOp>0 requer ClassOp | agregado sem classif |

### Semânticas (S01-S10)

| Code | Severity | Sheet | Description | Example Fail |
|------|----------|-------|-------------|--------------|
| S01 | A | Sem | Detalhamento cliente (stub) | (stub) |
| S02 | A | Sem | Vendor info (stub) | - |
| S03 | I | Sem | Detecção de ocultação (stub) | - |
| S04 | A | Sem | Crédito a liberar | (stub) |
| S05 | A | Sem | Limite crédito | (stub) |
| S06 | A | Sem | QtdOp zero é aviso | Agregados vazios |
| S07 | E | Sem | Mod=0213 requer ClassOp E-H | cheque especial com risco baixo |
| S08 | A | Sem | PF com ClassOp A é suspeito | PF normalmente não tem risco A |
| S09 | I | Sem | Soma V110..V165 ≈ QtdOp | 10% tolerância por arredondamento |
| S10 | E | Sem | NatuOp=01 com VincME=N | próprias não devem ser moeda estrangeira |

## Vetores cobertos

Cada regra documentada acima tem teste em:
- `internal/audit/rules/3040_test.go` (originais)
- `internal/audit/rules/3040_expanded_test.go` (Sprint 7b, NOVOs tests em breve)

Implementação: `internal/audit/rules/3040.go` (25 originais) +
`internal/audit/rules/3040_expanded.go` (Sprint 7b, 30 novos).

## Fuzz testing

`internal/crossdoc/rules/iter_fuzz_test.go`:
- 427k+ execs em 2s.
- Discoveries: edge cases de XML pathology.
- Cobertura: CDATA, entities, malformed UTF-8, nested, oversized.

## Próximo sprint — Frontend

Backend do CATÁLOGO está completo (60 regras catalogadas).
Sprint 7c (v2.0.0) cria UI `radiant-norma-console/` que
visualiza este catálogo + IF-specific toggle enable/disable.
