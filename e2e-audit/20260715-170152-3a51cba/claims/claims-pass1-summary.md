# Claims Pass 1 - Resumo

**Total**: 120 claims extraidos de 10 documentos.


## Distribuicao por peso

- Peso 1: 11 claims
- Peso 3: 55 claims
- Peso 5: 54 claims

## Distribuicao por criticidade

- core: 54
- supporting: 61
- marketing: 5

## Distribuicao por categoria

- validacao: 31
- deployment: 24
- seguranca: 21
- geracao: 15
- ui: 10
- ingestao: 8
- integracao: 7
- contratos: 2
- negocio: 2

## Peso 5 por categoria

- seguranca: 17
- validacao: 15
- geracao: 10
- deployment: 5
- integracao: 4
- ingestao: 3

## Conflitos e contradicoes (14)

- CLM-A0-006: README linha 57 declara 0 regras 2030 DRSAC; gaps conhecidos linha 325 dizem 35 regras D01-D35
- CLM-A0-014: tabela linha 66 diz 10 generators entregues; gaps conhecidos linha 322 dizem apenas 3040 implementado
- CLM-A0-020: README linha 237 diz 13 migrations; linha 268 da Estrutura diz 2 SQL files
- CLM-A0-024: README linha 149 diz 968 criticas no seed; linha 233 diz 1099 regras extraidas
- CLM-A0-025: README linha 150 espera versao 1.2.0 em /healthz; badge diz v3.36.2
- CLM-A0-027: curl /v1/rules/3040 retorna 320 (linha 158); tabela linha 54 diz 275; linha 235 diz 126; linha 328 diz 275/361
- CLM-A0-035: README linha 232 lista 9 binarios CLI; linha 258 lista 4; cmd/* na verdade tem 13
- CLM-A0-038/047: README linha 235 diz 126 regras portadas; linha 54/328 dizem 275; linha 249 confirma 126
- CLM-A0-039/088: README linha 236 diz 1 regra cross-doc entregue; ADR-0006 lista 12
- CLM-A0-041/056: README linha 238 diz Next.js 14; gaps conhecidos linha 335 dizem Next.js 15
- CLM-A0-062: Diagrama arquitetura (linha 93) diz 25 regras 3040 portadas; tabela linha 54 diz 275; linha 235 diz 126
- CLM-A0-100: ADR-0008 lista 5 conectores; README linha 323 diz 4 stubs e 1 funcional (Manual)
- CLM-A0-107: ADR-0007/0008 status 'Proposto (Sprint 57 - nao iniciado)' em 2026-07-08
- CLM-A0-017/069: README linha 128 diz Go 1.22+; ADR-0001 linha 20 diz 1.25+; go.mod declara 1.25.0; instalado 1.26.4

## Top claims peso 5 a verificar primeiro

- CLM-A0-001 [geracao] Gera 10 CADOCs (SCR/DRSAC/DRM/DLO/DLI/DDR/DRL/DLP/eventos) com leiautes versionados
- CLM-A0-002 [geracao] BCValidador apenas valida; Radiant Norma tambem gera (motor proprietario)
- CLM-A0-003 [validacao] Cobertura 3040: 275 regras Go (B/F/C/S/I/H) ja portadas
- CLM-A0-004 [validacao] 3044 Eventos de Credito (JSON) com 17 regras T01-T19
- CLM-A0-005 [validacao] 3050 Estatisticas Agregadas com 170 regras TXB
- CLM-A0-006 [validacao] 2030 DRSAC: 0 regras declaradas na tabela; 35 regras D01-D35 nos gaps conhecidos
- CLM-A0-013 [validacao] Catalogo declara 1099 regras de validacao; 275+ portadas em Go
- CLM-A0-014 [geracao] Generator 3040, 3050, 4111, 2060, 2061, 2062, 2070, 2160, 2170, 2030 entregues (Sprint 57-73)
- CLM-A0-019 [deployment] DB: SQLite (modernc.org/sqlite) em dev, Postgres em prod
- CLM-A0-021 [seguranca] Audit log SHA-256 hash chain tamper-evident
- CLM-A0-023 [seguranca] Concorrencia: pgx + Postgres RLS multi-tenant
- CLM-A0-027 [validacao] Endpoint /v1/rules/3040 retorna 320 regras
- CLM-A0-028 [validacao] Endpoint POST /v1/validate aceita {cadoc, xml}
- CLM-A0-033 [deployment] Testes Go: 516 top-level, 21/21 packages PASS, race clean
- CLM-A0-036 [validacao] Catalogo critico: 1099 regras extraidas; 6 CADOCs (3040, 3044, 3050, 2060, 2061, 2070)
- CLM-A0-038 [validacao] Regras portadas Go: 126 de 3040 (34.9%)
- CLM-A0-039 [validacao] Cross-doc: 1 regra (3040 <-> 4111) entregue; meta 12 regras (Sprint 43)
- CLM-A0-042 [seguranca] Audit log entries: hash chain SHA-256 validado, tamper-evident, trigger Postgres imutavel
- CLM-A0-043 [seguranca] Endpoints REST: 20+ funcionais, JWT + CSRF + RateLimit + CORS
- CLM-A0-046 [seguranca] Postgres RLS: FORCE ROW LEVEL SECURITY em 6 tabelas tenant-scoped + helper WithTenantTx
- CLM-A0-047 [validacao] Cobertura real 3040: 126/361 regras (34.9%); Sprint 32 Fases 1+2+3+4 entregas 66 regras
- CLM-A0-048 [geracao] Motor de Geracao 3040 implementado em v3.36.0; outros 9 CADOCs em desenvolvimento/planejamento (Sprint 58+)
- CLM-A0-053 [integracao] STA real implementado em Sprints 18-22 (WS client, retry, DLQ)
- CLM-A0-054 [seguranca] Auth: JWT + cookie implementado (Sprints 8-10, simplificado em v3.35.5)
- CLM-A0-055 [seguranca] Postgres RLS ativado em Sprint 30