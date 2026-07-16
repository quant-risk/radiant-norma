# Auditoria ponta a ponta — Radiant Norma

Data da execução: 15 de julho de 2026  
Commit auditado: `3a51cba4ce1945c4e554915131617089c9d061bb`  
Run principal: `20260715T174828Z-1a38e519`  
Ambiente: macOS arm64, Go 1.26.4, Python 3.9.6, API local com SQLite e STA stub  
Escopo: produto, documentação, frontend, API, generators, validator, fontes, STA,
auditoria, SDKs, segurança e promessas comerciais.

## Veredito executivo

**NO-GO para produção, homologação regulatória ou envio real ao BACEN.**

O repositório tem uma base técnica ampla e vários componentes isolados funcionam,
mas a promessa central — dados brutos → CADOC válido → submissão STA auditável —
não funciona ponta a ponta de forma confiável.

O benchmark black-box executou 85 oráculos:

| Resultado | Quantidade |
|---|---:|
| PASS | 34 |
| FAIL | 48 |
| PARTIAL | 3 |
| BLOCKED | 0 |
| Total | 85 |
| Score ponderado | 41,76% |

Esse score não é cobertura de código nem certificação regulatória. Ele é apenas a
proporção ponderada dos oráculos deste run (`PASS=1`, `PARTIAL=0,5`, `FAIL=0`).
Qualquer falha nos gates regulatórios mantém o produto em NO-GO.

Na matriz de 28 grupos de promessas públicas:

| Veredito | Grupos |
|---|---:|
| Evidenciado no código/runtime | 1 |
| Parcial ou contraditório | 8 |
| Quebrado, ausente ou sem wiring operacional | 17 |
| Exige prova externa/documental | 2 |

## O que foi realmente comprovado

- `/healthz` e `/readyz` responderam corretamente.
- A rota protegida recusou acesso sem tenant no modo de teste.
- Existem endpoints de geração e metadados para os 10 CADOCs anunciados.
- Os 10 generators produziram XML bem-formado e SHA-256 correto para as fixtures
  sintéticas realistas.
- O header genérico ausente foi rejeitado nos 10 generators.
- O catálogo lista cinco tipos de adapter.
- O CSV simples preservou dois contratos e os valores `50000` e `75000.25`.
- O batch 3040 + 4111 coerente passou e o batch com CNPJ/data-base divergentes
  foi rejeitado com erros cross-document.
- XML malformado e a fixture semântica negativa 3040 foram rejeitados.
- O STA stub local persistiu um envio, e esse envio não apareceu para o segundo
  tenant usado no teste.
- A cadeia SHA-256 de `audit_log` existe no código e tem testes dedicados.
- A suíte backend passou integralmente em uma execução; outra execução reproduziu
  um teste flakey sob contenção, descrito abaixo.
- Os testes unitários do SDK Go passaram; `sdk/py` passou 20 testes mockados.

Esses fatos positivos não provam validade BACEN, XSD oficial, entrega STA real,
RLS/PostgreSQL, SLA, SOC 2, LGPD ou funcionamento da jornada visual.

## Bloqueadores críticos reproduzidos

### P0-01 — O produto rejeita os documentos que ele próprio gera

Dos 10 documentos sintéticos realistas gerados, somente 2062 e 3050 passaram no
próprio `/v1/validate`. Os outros 8 foram rejeitados.

Sete generators usam uma raiz diferente daquela esperada pelo parser do validator:

| CADOC | Generator produz | Validator espera |
|---|---|---|
| 2030 | `DocDRSAC` | `DocumentoDRSAC` |
| 2060 | `DocDRM` | `Doc2060` |
| 2061 | `DocDLO` | `documentoDLO` |
| 2070 | `DocDDR` | `documentoDDR` |
| 2160 | `DocDRL` | `documentoDRL` |
| 2170 | `DocDLP` | `documentoDLP` |
| 4111 | `Documento4111` | `Documento` |

O 3040 usa a mesma raiz, mas a saída realista ainda foi rejeitada por regras do
próprio produto. Portanto, HTTP 200 e XML bem-formado não significam documento
válido.

### P0-02 — A validação aprova documentos vazios

Usando exatamente as raízes aceitas pelo validator, 9 de 10 XMLs vazios foram
aprovados sem qualquer campo obrigatório. Somente `<Doc3040/>` foi rejeitado.

Passaram indevidamente: 2030, 2060, 2061, 2062, 2070, 2160, 2170, 3050 e 4111.
O caso 3050 é especialmente grave porque há 51 críticas desse CADOC no banco e a
documentação declara cobertura completa, mas o dispatcher principal não as executa.

O fluxo `/v1/validate` executa essencialmente L1/L2. L3 e L4 ficam em endpoints ou
pipelines separados; `ValidateFull` não está roteado. O caminho chamado “XSD” usa
caminhos relativos frágeis, cobre poucos CADOCs, implementa apenas um subset e
falha aberto quando não consegue carregar schema.

### P0-03 — Versões e obrigatoriedade não são aplicadas

Nos 10 generators:

- a versão default foi sempre a mais antiga, embora `/fields` anuncie versões mais
  novas;
- o `field_map` não cobriu os campos anunciados como obrigatórios;
- conteúdo de negócio vazio foi aceito;
- a ausência de data-base foi preenchida/inventada em vez de ser rejeitada.

No 3040, `versao_layout: "9.9"`, que não aparece entre `3.0`, `3.1` e `3.2`, foi
aceita com HTTP 200 e devolvida como versão gerada `9.9`.

### P0-04 — O wizard visual não consegue concluir a jornada

A inspeção estática do frontend encontrou falhas determinísticas:

- validação chama `fetch('/v1/validate')`, mas não existe rota/rewrite Next para
  esse caminho;
- o mapping escolhido é salvo no state/localStorage, mas nunca transforma o
  documento enviado;
- refresh restaura só o número do step e perde arquivo, documento e XML;
- a data-base escolhida é descartada em etapas seguintes;
- a UI espera `rules_run` e um shape de erro diferente do backend;
- a UI faz `btoa(xml)` na submissão STA, enquanto o backend espera XML cru;
- a UI espera `protocolo`, mas o backend retorna `protocol_sta`;
- o proxy STA fixa `X-IF-ID: demo`;
- o wizard expõe apenas 3040/3050 apesar dos 10 generators registrados.

O navegador real não pôde ser executado neste run porque os arquivos do frontend,
incluindo `package.json`, fontes e parte de `node_modules`, foram evictados pelo
iCloud e estavam marcados como `dataless`. Cópia limpa, type-check e servidor Next
ficaram bloqueados esperando materialização. Isso foi classificado como limitação
do ambiente, não como PASS. As incompatibilidades acima foram confirmadas no código
antes da evicção.

### P0-05 — Upload pode corromper valores ou aceitar vazio silenciosamente

O XLSX sintético foi realmente aberto e parseado, mas alterou valores:

| Contrato | Esperado | Observado |
|---|---:|---:|
| `E2E-SCR-XLSX-001` | `50000` | `50` |
| `E2E-SCR-XLSX-002` | `75000.25` | `75.00025` |

O CSV UTF-8 BOM + CRLF + `;` + números pt-BR retornou HTTP 200 com
`operacoes: null`, sem erro explícito. Um CSV com quantidade variável de colunas
também retornou HTTP 200 com documento vazio.

O frontend/adapter anuncia limite de 50 MiB, mas um arquivo com 10 MiB−1 byte foi
rejeitado com 413 porque o limite global de 10 MiB inclui o overhead multipart.

### P0-06 — STA local pode produzir falso sucesso

O runtime usa STA stub por default. Ele aceitou `<Doc3040/>`, sem validação
regulatória, e retornou protocolo sintético. Duas submissões idênticas criaram dois
envios e protocolos distintos, comprovando ausência de deduplicação no fluxo.

Uma chamada com `X-Role: readonly` também recebeu HTTP 200. No envio persistido,
`data_base: 2026-06-01` apareceu na listagem como período `00/0000`.

O endpoint chama STA antes de persistir; se a chamada externa falhar, não existe
registro pending para o worker retentar ou mover à DLQ. O worker, por sua vez, usa
sempre `StubClient`, mesmo quando existe configuração WS.

O PASS de persistência no benchmark comprova apenas o stub local. Não prova entrega
ao BACEN.

### P0-07 — Webhooks publicados causam panic/500

`GET /v1/webhooks/` retornou 500. O log capturou `nil pointer dereference` em
`webhook.(*Service).List`: as rotas usam `srv.Webhook`, mas o serviço não é
inicializado em `cmd/api/main.go`.

Além disso, a inspeção encontrou gaps de delivery: dispatch não cria de forma
coerente a linha de entrega, respostas 4xx/5xx podem ser tratadas como sucesso e a
fila pode descartar eventos. Nenhum webhook outbound real foi certificado.

### P0-08 — PostgreSQL/RLS e conectores DB não estão operacionais

O adapter anuncia PostgreSQL, mas o teste black-box retornou:

`sql: unknown driver "postgres" (forgotten import?)`

As migrations e queries também misturam PostgreSQL com sintaxe SQLite:
`AUTOINCREMENT`, `GLOB`, `strftime`, `INSERT OR IGNORE` e placeholders `?` usados
com pgx. Não há evidência de uma suíte real PostgreSQL, e a inicialização tende a
falhar antes de provar RLS.

Oracle não está implementado e MySQL/nomes de driver não têm wiring coerente com o
runtime anunciado.

### P0-09 — Auditoria e Insights leem tabelas que o runtime não alimenta

Depois das ações E2E, a base de teste continha:

- `audit_log`: 10 entradas;
- `audit_events`: 0 entradas;
- `rule_failures`: 0 entradas.

A API/UI de timeline e export consulta `audit_events`, enquanto a cadeia real grava
em `audit_log`. O benchmark recebeu cadeia “válida” com zero eventos — um resultado
formalmente verde sobre um conjunto vazio. Insights depende de `rule_failures`, que
o fluxo real de validação não alimentou.

### P0-10 — A contagem de regras não corresponde ao runtime

O seed anunciou/importou 1.099 linhas, mas colisões de chave descartaram 131. O
banco executado terminou com 968 críticas:

| CADOC armazenado | Regras |
|---|---:|
| 2060-DRM | 22 |
| 2061-DLO | 518 |
| 2070-DDR | 11 |
| 3040 | 349 |
| 3044 | 17 |
| 3050 | 51 |
| Total | 968 |

“Catalogada”, “registrada no Go”, “executada pelo endpoint” e “capaz de detectar
uma violação” são métricas diferentes. A documentação mistura essas categorias e
contabiliza stubs ou regras que o dispatcher pula.

## Matriz de veracidade das 28 promessas

| # | Grupo de promessa | Veredito | Evidência resumida |
|---:|---|---|---|
| 1 | Dados brutos → CADOC validado em 15 min | FAIL | Wizard não conclui e o round-trip geração→validação falha |
| 2 | 10 generators | PARTIAL | 10 registrados; todos violam ao menos um gate e UI contradiz disponibilidade |
| 3 | XML gerado já validado pelo schema registry | FAIL | 8/10 saídas rejeitadas pelo próprio validator |
| 4 | Cinco conectores disponíveis | PARTIAL | Catálogo existe; API/MCP só healthcheck, DB falha, UI não oferece fluxo completo |
| 5 | CSV/XLSX/PDF/DOCX | FAIL | Runtime aceita CSV/XLSX; XLSX corrompe valores; PDF/DOCX não suportados |
| 6 | PostgreSQL/MySQL/Oracle | FAIL | Driver e SQL incompatíveis; Oracle ausente |
| 7 | MCP padrão | PARTIAL | Healthcheck HTTP funciona; lifecycle/tools-call MCP não foi implementado integralmente |
| 8 | 1.099 regras / 90% coverage | FAIL | Runtime tem 968 críticas e muitas não entram no dispatcher |
| 9 | 3040 76% / 3050 100% | FAIL | Claims internos divergem; 3050 vazio passa |
| 10 | L1 XSD real | FAIL | Subset, poucos mapeamentos, caminhos frágeis e fail-open |
| 11 | L1→L4 e explainability | FAIL | Não há um entrypoint único; shapes também divergem da UI |
| 12 | Push STA-h/STA-ws nativo | PARTIAL | Código WS existe; runtime e worker usam stub no caminho testado |
| 13 | Retry, DLQ e dedupe STA | FAIL | Falha síncrona não é enfileirada e duplicatas são aceitas |
| 14 | Radar oficial a cada 6 h | PARTIAL | Scheduler isolado cobre poucos URLs e não está no Compose |
| 15 | Insights/anomalias/recomendações | FAIL | Tabela `rule_failures` não é alimentada pelo fluxo real |
| 16 | Audit log SHA-256 tamper-evident | PASS | Cadeia e verificação existem; teste concorrente ainda é flakey sob carga |
| 17 | Audit imutável, sem PII, retenção 5 anos | FAIL | Sem proteção UPDATE/DELETE ou job de retenção demonstrável |
| 18 | Timeline, filtros e export | PARTIAL | Endpoints existem, mas leem tabela desconectada do audit real |
| 19 | PostgreSQL RLS multi-tenant | FAIL | Backend PostgreSQL não sobe de forma demonstrável; RLS não foi provado |
| 20 | Real-time | PARTIAL | SSE in-process; sem pub/sub multi-réplica e dependente de eventos ausentes |
| 21 | Keycloak/Okta/MFA nativo | FAIL | Verifier JWT existe; fluxo OIDC/MFA do produto não existe no repo |
| 22 | SOC 2, LGPD, SLA 99,95%, onboarding | EXTERNAL | Exige relatórios, contratos e evidência operacional externa |
| 23 | Webhooks outbound | FAIL | Rota publicada panica/500 e serviço não está inicializado |
| 24 | Marketplace de regras | FAIL | CRUD de instalação não altera a validação executada |
| 25 | Stripe, white-label, multi-region | FAIL | Bibliotecas/endpoints isolados, sem produto E2E integrado |
| 26 | SDKs oficiais Go/Python | PARTIAL | Unit tests passam em mocks, há pacotes duplicados e drift de rotas/base path |
| 27 | API reference/status/changelog/termos | FAIL | Links/rotas públicas estão incompletos ou apontam para artefatos inexistentes |
| 28 | “BACEN Ready”, CMN 4.966, IFRS 9 | EXTERNAL | Requer oracle oficial/homologação e revisão regulatória; código atual contradiz readiness |

## Resultados técnicos adicionais

### Testes automatizados existentes

- `backend: go test -count=1 ./...`: PASS em uma execução.
- Em outra execução completa, `TestAuditLog_NoChainBreaks_HighContention` perdeu
  32 de 200 gravações por `context deadline exceeded`; o mesmo teste isolado passou
  10/10. Classificação: **flakey sob contenção global**.
- `go test -race -count=1 ./...`: não concluído neste run porque arquivos Go foram
  evictados pelo iCloud durante a execução; processo ficou sem CPU esperando leitura
  e foi encerrado. Não contabilizado como PASS.
- `sdk/go: go test -count=1 ./...`: PASS.
- `sdk/py: pytest -q`: 20 PASS, com warning de LibreSSL.
- `sdk/python: pytest -q` direto: 2 FAIL + 8 ERROR por
  `ModuleNotFoundError: radiant`; o pacote exige instalação/ambiente que o comando
  direto não prepara.
- Frontend: não há Jest, Vitest, Playwright ou Cypress configurado; type-check/build
  ficaram bloqueados pelo estado `dataless` do workspace.

Há aproximadamente 109 arquivos de teste Go, 1.178 funções de teste, um benchmark
e nenhum fuzz test. Volume de testes unitários não substitui os gates E2E ausentes.

### Contratos e SDKs

- Router, duas versões de OpenAPI e SDKs estão em drift.
- `/v1/generate/adapters` retorna array cru; specs/SDKs esperam shape de objeto.
- O SDK Go mais novo pode perder `/v1` ao resolver paths absolutos.
- Um SDK Python chama `/v1/healthz`, mas o servidor expõe `/healthz`.
- Existem duas árvores Python (`sdk/py` e `sdk/python`) com versões divergentes.

### Funcionalidades sem wiring de produto

Billing/Stripe, multi-region e partes de white-label existem como código isolado,
mas não foram conectados ao runtime/jornada principal. Marketplace instala registros,
mas não injeta regras no validator. O radar automático depende de comando separado
ausente do Compose. LLM/Insights não tem jornada completa no frontend.

## Limitações explícitas do run

Os itens abaixo não foram contados como PASS:

- browser real, build Next e acessibilidade: bloqueados por arquivos iCloud
  `dataless`;
- PostgreSQL, Redis e stack Docker: serviços/daemon indisponíveis no ambiente;
- STA-h/STA-ws BACEN, Radar oficial, AWS, LLM, Sentry/OTel externos: sem credenciais
  ou autorização de homologação;
- SOC 2 Type II, LGPD, SLA, DR e onboarding: exigem evidência documental e
  operacional externa;
- validade regulatória oficial: não foi executado BCValidador nem homologação BACEN.

O STA stub, seeds demo e mocks não podem ser usados como prova desses itens.

## Gates mínimos antes de novo release

1. 10/10 generators devem produzir XML aceito pelo mesmo validator e por oracle
   XSD oficial independente.
2. Validator deve falhar fechado se schema não carregar e rejeitar 10/10 roots
   vazios.
3. Versão default deve ser latest/effective; versão desconhecida deve ser 4xx.
4. Todo campo anunciado como required deve ter par positivo/negativo automatizado.
5. CSV/XLSX devem preservar valores, Unicode, datas e contratos exatamente.
6. Wizard browser deve concluir upload → mapping → validação → geração → download →
   STA stub, inclusive após refresh.
7. Role readonly deve receber 403 em todas as mutações; zero vazamento cross-tenant.
8. PostgreSQL limpo deve migrar, subir e passar a mesma suíte com RLS real.
9. Webhook deve entregar callback assinado, tratar status HTTP e retry sem panic.
10. Falhas STA devem ser persistidas antes da chamada, retentadas e deduplicadas.
11. Auditoria, timeline, export e Insights devem consumir os mesmos eventos reais.
12. OpenAPI, router e SDKs devem ter diff de contrato vazio.
13. Nenhum skip inesperado, flake concorrente ou regressão de dados pode permanecer.
14. Claims externos só podem ser publicados com evidência externa verificável.

## Ordem de correção recomendada

1. Unificar modelo XML/parser/validator por CADOC e aplicar XSD oficial fail-closed.
2. Corrigir required fields, versão default e whitelist de versão.
3. Corrigir ingestão semântica e o wizard completo, adicionando Playwright.
4. Isolar e rotular STA stub; redesenhar persistência/retry/dedupe do fluxo real.
5. Aplicar RBAC e tenancy a todas as rotas mutantes.
6. Corrigir PostgreSQL/migrations/RLS e adicionar CI com banco real.
7. Inicializar ou remover webhooks das superfícies públicas até ficarem operacionais.
8. Unificar `audit_log`/`audit_events`/`rule_failures` e provar exports/Insights.
9. Sincronizar OpenAPI, SDKs, documentação, landing page e roadmap com a realidade.

## Como reproduzir

Com a API de teste rodando em `127.0.0.1:18080`:

```bash
python3 e2e/run_benchmarks.py \
  --base-url http://127.0.0.1:18080 \
  --fixtures e2e/fixtures \
  --xlsx e2e-audit/20260715-170152-3a51cba/fixtures/operacoes-3040.xlsx \
  --output e2e-audit/20260715-170152-3a51cba/runtime/benchmark-results.json
```

O exit code `1` é esperado enquanto houver `FAIL`.

Gerar novamente o corpus sintético:

```bash
python3 e2e/generate_fixtures.py --output /tmp/radiant-fixtures
python3 e2e/generate_fixtures.py --output /tmp/radiant-fixtures-large --include-large
```

O corpus padrão tem 16 fixtures e SHA-256 em `manifest.json`. A opção large adiciona
arquivos de 10 MiB−1, 10 MiB e 10 MiB+1 byte.

## Artefatos

- `PROMPT_AUDITORIA_E2E.md`: prompt mestre para compreensão profunda, inventário
  de claims, criação de fakes, oráculos e benchmarks.
- `e2e/run_benchmarks.py`: harness black-box repetível.
- `e2e/generate_fixtures.py`: fábrica determinística de arquivos sintéticos.
- `e2e/fixtures/generate-all-cadocs.json`: input realista dos 10 generators.
- `e2e-audit/20260715-170152-3a51cba/runtime/benchmark-results.json`: evidência
  completa dos 85 oráculos.
- `e2e-audit/20260715-170152-3a51cba/fixtures/operacoes-3040.xlsx`: XLSX real,
  renderizado e inspecionado visualmente.
- `e2e-audit/20260715-170152-3a51cba/generated-fixtures/manifest.json`: catálogo,
  hashes e resultados esperados do corpus de 16 arquivos.

## Conclusão

O Radiant Norma funciona hoje como uma base/protótipo amplo com vários componentes
úteis, mas não como produto regulatório ponta a ponta confiável. O maior risco é
falso positivo: o sistema pode responder 200, produzir XML, aceitar STA stub ou
aprovar um documento vazio sem que isso represente validade regulatória.

Não foram aplicadas correções em código de produção durante esta auditoria. Os
achados e os benchmarks foram preservados para que cada correção possa ser medida
contra o mesmo conjunto de oráculos.
