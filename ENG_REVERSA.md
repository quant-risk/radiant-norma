# Engenharia Reversa Profunda — CADOCs BACEN

> Análise técnica e de negócio de Mitra, Matera, cadoc.ai, Dattos, BIBlue, Regulatório Mais, BTech e outros. Onde eles ganham, onde perdem, como fazer **melhor**.

**Atualização 2026-07-02 22h18:** adicionada cobertura completa do **CADOC 2030 DRSAC** (Risco Social, Ambiental e Climático), peça-chave para ESG que estava faltando.

---

## 1. Quem são os concorrentes (mapa competitivo)

### Tier 1 — Líderes estabelecidos

**Matera** (`matera.com/br/solucoes/reg-tech/`)
- C-LEVEL core banking + pag + crédito + risco + RegTech
- ~35 anos de mercado (fundada 1987)
- **Diferencial:** "Validações prévias e gestão de ciclo de vida — vamos além de apenas empacotar arquivos"
- Atende: SCDs, IPs, SCFIs, Bancos Múltiplos, DTVMs/CTVMs
- **Frase-chave:** *"Nossa solução gerencia o ciclo de vida da informação, monitorando o recebimento, o processamento e a consolidação dos dados"*

**MITRA (LUZ Soluções Financeiras)** (`luz-ef.com/en/mitra/`)
- Front-to-back integrado, com módulo Basileia explícito
- Cliente: gestoras de recursos, bancos, corretoras, empresas não-financeiras
- **Diferencial:** Calculadora Regulatória como módulo isolado — "líder de mercado no módulo de Basileia"
- Cobre: DLO, DDR, DRM, DRL, Basileia, RAROC
- Whitepaper: `LUZ_MITRA_Whitepaper.pdf` (2 MB)

### Tier 2 — SaaS especializados

**cadoc.ai** (`cadoc.ai`)
- Posicionamento: "Inteligência Regulatória"
- **Diferencial:** Norma Skills = RAG jurídico nativo que injeta cérebro regulatório na IA da IF
- Robôs 24/7 monitorando BACEN Oficial — base vetorial atualiza em minutos
- Cada parágrafo cita fonte + metadado + ID — "defensável juridicamente"
- Posts publicados diariamente sobre IN BCB, Resoluções BCB
- **Insight crítico:** eles tratam CADOC + monitoramento regulatório juntos

**Dattos** (`dattos.com.br`)
- Foco em Informes Legais — funcionalidade nativa CADOC 3044 (JSON)
- Plataforma low-code para relatórios
- Clientes: IFs com pouca estrutura técnica

**Regulatório Mais (Celcoin)** (`regulatoriomais.com.br`)
- 4 módulos: Contábil Bacen, Não-Contábeis (IP/SCD), Crédito (SCR), Risco Bacen
- **Diferencial:** Roteiros contábeis configuráveis — converte contabilidade tradicional → COSIF

**BIBlue** (`biblue.com.br`)
- Compliance Regulatório: Bacen, COAF, CVM, Receita, ANPD em uma plataforma
- Templates pré-configurados CADOC/SCR/COS
- Integração via API com core banking

### Tier 3 — Boutique/consultoria

**BTech** (`btechcore.com/cadoc`)
- Implementação por projeto, 4-8 semanas
- Cobra por hora/projeto
- Só implementação, sem SaaS recorrente

**Lerian Studio** (`blog.lerian.studio`)
- Visão arquitetural: "validation-first"
- **Insight:** CADOCs não são problema de ferramenta, são consequência de arquitetura. Se o ledger não captura granularidade na transação, gerar 4010 vira quebra-cabeça manual.

### Tier 4 — Conteúdo/treinamento
- **FBM Educação** — cursos de "Especialista em CADOCs"
- **Compliasset** — alertas regulatórios (não SaaS)
- **B3Bee** — análises técnicas

---

## 2. Comparativo técnico de features

| Feature | Mitra | Matera | cadoc.ai | Dattos | Regulatório+ | BIBlue | **NOSSO** |
|---|---|---|---|---|---|---|---|
| Geração XML | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Validação XSD | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Validação semântica | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Envio STA (Web) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Envio STA (Web Services REST) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (nativo Go) |
| Múltiplos CADOCs (3040, 3042, 3050, 2060/61/62, 2160/70) | ✅ | ✅ | ✅ | parcial | ✅ | ✅ | ✅ |
| 3044 (JSON, eventos) | ✅ | ✅ | ✅ | ✅ | ? | ? | ✅ |
| Integração com core banking via API | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Monitoramento de novas IN BCB | ❌ | ❌ | ✅ (radar) | parcial | parcial | ❌ | ✅ (RAG+IA) |
| Geração de cenários what-if (stress test) | ✅ (Mitra) | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Multi-tenant (várias IFs na mesma instância) | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| UI/UX moderna | parcial | boa | boa | boa | média | boa | **excelente** |
| Self-service (IF cadastra sem precisar de consultoria) | parcial | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Modelo de preço transparente | ❌ (consultoria) | sob consulta | sob consulta | sob consulta | sob consulta | sob consulta | **público** |
| Open API / Webhooks | parcial | ✅ | ✅ | ✅ | parcial | ✅ | ✅ |
| Modo white-label para correspondentes | ? | ? | ? | ? | ? | ? | ✅ |
| Conversor automático de planilhas manuais → XML | parcial | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| IA para detectar inconsistências antes do envio | ❌ | ❌ | ✅ | parcial | ❌ | parcial | ✅ |
| **Preço estimado (entrada)** | R$ 50k+/mês (projeto) | R$ 30k+/mês | R$ 5-15k/mês | R$ 5-10k/mês | R$ 3-8k/mês | R$ 5-10k/mês | **R$ 1.5-3k/mês** |

---

## 3. Onde os concorrentes GANHAM (e onde podemos vencer)

### Onde eles ganham hoje

1. **Bancarização / Network Effect** — Matera está dentro de grandes IFs há décadas; trocar é arriscado.
2. **Suporte "premium" consultivo** — Mitra e Matera entregam "regulatory specialists" como parte do contrato.
3. **Acesso direto ao Desig** — têm canais abertos com o regulador para tirar dúvidas que beneficiam clientes.
4. **Cobertura completa de todos os CADOCs** — Matera cobre 4010, 4060/66, 4076, 3040, 3044, 3050, DRM, DRL, DLO etc.
5. **Validador pré-bacen integrado** — eles reimplementam (ou empacotam) o Validador BCB para dar feedback em tempo real.

### Onde eles PERDEM (e podemos vencer)

1. **Preço opaco e alto** — Matera/Mitra vendem projeto, não SaaS. PME (SCDs, SEPs, pequenas financeiras) paga caro ou fica fora.
2. **Onboarding lento** — 4-8 semanas na BTech, até meses na Matera. SaaS moderno precisa de **15 minutos para self-signup**.
3. **UX datada** — interfaces herdadas de 2010-2015. Muitos IFs ainda imprimem PDF e preenchem manualmente (vide relatórios FBM Educação).
4. **Radar regulatório descoberto** — só cadoc.ai oferece, e mesmo assim via Norma Skills (skill separada, não no fluxo principal).
5. **Sem what-if / simulador** — gerar DRL com cenário hipotético para auditoria é trabalho braçal em Excel.
6. **Falta de feedback inteligente** — "tem 50 críticos no validador, mas onde está o problema mais grave?"
7. **Plataforma não unifica CADOC contábil + SCR + risco** — alguns (Matera) sim, mas a IF tem que comprar três SKUs.

### Nossas vantagens competitivas (a construir)

1. **Onboarding em < 15 min** — upload do leiaute XLS → importador IA → primeiro XML válido em minutos
2. **Pricing público e acessível** — R$ 1.500-3.000/mês (10× mais barato que Mitra para SCD pequena)
3. **Radar regulatório nativo** — cada CADOC tem feed de mudanças; RAG explica impacto em linguagem natural
4. **IA contextual no validador** — "este IPOC já existia no 3040 anterior; mudou de cliente. Tem certeza?" (cross-doc check)
5. **Modo simulação** — "e se eu liquidar 30% da carteira inadimplente hoje, como ficam os indicadores?"
6. **White-label para correspondentes** — Fintech X pode ofertar "BACEN em 1 clique" para 50 SCDs pequenas
7. **Self-host opcional** — IFs grandes podem rodar on-prem (compliance com segurança)
8. **Open API/Webhooks** — eventos `cadoc.processed`, `cadoc.failed`, `cadoc.sent` para o pipeline da IF

---

## 4. Arquitetura técnica do SaaS (proposta inicial)

### Stack sugerido (Go-heavy, alinhado à Fortvna)
```
┌─────────────────────────────────────────────────┐
│   Web Console (Next.js / React + TypeScript)    │
│   - Dashboard regulatório                        │
│   - Wizard de cadastro CADOC por IF              │
│   - Visualizador XML + críticas                  │
│   - Calendário de prazos                         │
└────────────────┬────────────────────────────────┘
                 │ REST/GraphQL
                 ▼
┌─────────────────────────────────────────────────┐
│   API Gateway (Go)                              │
│   - Auth (Keycloak/Auth0)                       │
│   - Rate limiting                                │
│   - Auditoria                                    │
└────────────────┬────────────────────────────────┘
                 │
   ┌─────────────┼─────────────┐
   ▼             ▼             ▼
┌─────────┐ ┌─────────┐ ┌──────────────┐
│Pipeline │ │ STA     │ │ Radar        │
│Generator│ │ Client  │ │ Regulatório  │
│(Go)     │ │(Go)     │ │(Go+RAG)      │
│         │ │         │ │              │
│-Leiaute │ │-HTTPS   │ │-Crawler BACEN│
│-Build   │ │-WS REST │ │-Diff leiaute │
│-Validate│ │-MFT     │ │-Notif push   │
└─────────┘ └─────────┘ └──────────────┘
```

### Componentes críticos

**1. Schema Registry (Postgres + JSON Schema)**
- Cada CADOC tem schema versionado por data-base
- Permite auditar mudanças e fazer replay

**2. Validador Multi-Camada**
- **L1:** XSD (estrutura)
- **L2:** Semântica (regras do Validador BCB — replicar planilha `SCR3040_Criticas.xls`)
- **L3:** Cross-doc (3040 ↔ 3050 ↔ 4076 ↔ 2061)
- **L4:** Histórico (mudanças suspeitas vs. base anterior)

**3. STA Client (Go)**
- Modo 1: **Web Form** (browser automation via Playwright — workarounds quando IF não tem WS)
- Modo 2: **Web Services REST** nativo, com:
  - OAuth-like com certificado A1/A3 (PEM)
  - Hash SHA-256 do payload antes do envio
  - Retry exponencial com backoff
  - Fila de upload (max 10 concurrent por IF)

**4. Radar Regulatório**
- Crawler diário do `bcb.gov.br/estabilidadefinanceira`
- Detecta mudanças em `leiautedocumentoscrd`, IN BCB, Cartas-Circulares
- Compara leiaute novo vs. antigo (XLS diff)
- Gera PR automático com mudança classificada (breaking / minor / new field)
- RAG indexa tudo para "Pergunte à IA"

**5. Engine de Templates**
- Para cada CADOC: template XLSX → mapeamento para XML
- IF faz upload do XLSX preenchido → geramos XML → validamos → enviamos
- Alternativa: integração direta com core banking via API (PL/pgSQL, etc.)

---

## 5. Roadmap sugerido (12-16 semanas MVP → Scale)

### Fase 1 — MVP (4-6 semanas)
- ✅ Schema registry + 3040 e 3050 (leiautes já baixados)
- ✅ Validador XSD + semântico (replicar planilha de críticas)
- ✅ UI básica: upload XLSX → download XML validado
- ✅ STA Client Web (Playwright)
- ✅ 5 IFs piloto (SCDs pequenas que pagam hoje caro)

### Fase 2 — Scale (6-10 semanas)
- ➕ DRM/DLO/DLI/DRL/DLP/DDR (todos os outros)
- ➕ 3044 (eventos, JSON)
- ➕ STA Web Services nativo (sem Playwright)
- ➕ Multi-tenant, billing, Stripe
- ➕ Radar regulatório (crawler + RAG)
- ➕ 30 IFs ativas

### Fase 3 — Diferenciação (10-16 semanas)
- 🚀 IA contextual no validador (cross-doc check)
- 🚀 Simulador what-if
- 🚀 White-label para correspondentes
- 🚀 Open API pública + Webhooks
- 🚀 Self-host opcional (Helm chart)

---

## 6. Riscos regulatórios e operacionais

| Risco | Mitigação |
|---|---|
| BACEN muda leiaute e quebra nosso parser | Radar regulatório + versionamento rigoroso + changelog público |
| IF cliente recebe auto de infração por erro nosso | Disclaimer robusto + auditoria de logs + SLA com teto |
| Concorrente grande copia feature | Velocidade de execução + brand + comunidade |
| ICP-Brasil / cert digital é barreira | Templates prontos + tutorial + parceria com ACs |
| IFs pequenas não têm equipe técnica | UX "Excel-friendly" + suporte humano inicial |

---

## 7. Personas e go-to-market

### Persona 1 — SCD pequena (R$ 50M-500M carteira)
- 1-3 pessoas em compliance
- Hoje paga R$ 5-15k/mês pra Matera/cadoc.ai ou faz manual
- **Dor:** "preciso de alguém que entenda CADOC mas é caro"
- **Pitch:** "15 min de setup, primeira entrega válida hoje, R$ 1.500/mês"

### Persona 2 — IP (Instituição de Pagamento)
- Volume alto, eventos
- Hoje usa Dattos ou similar
- **Dor:** "3044 com JSON é novo, ninguém tem pronto"
- **Pitch:** "suporte nativo a JSON, eventos em streaming, integração API"

### Persona 3 — Fintech "Banking as a Service" (Stone, Cora, etc.)
- Quer oferecer BACEN-complaint pra clientes embedded
- **Dor:** "50 clientes SCD me perguntam se a gente faz CADOC"
- **Pitch:** "API REST por cliente, white-label, billing por uso"

### Persona 4 — Banco médio (S3-S4)
- Já tem Mitra/Matera, quer modernizar
- **Dor:** "estou pagando caro por interface de 2015"
- **Pitch:** "roda on-prem, integra com nosso ESB, RAG para monitoramento"

---

## 8. Insights não-óbvios

1. **O gargalo real NÃO é gerar XML** — é manter atualizado com mudanças regulatórias constantes (IN 733/2026 muda 3040 em 3 ondas em 2026). Quem ganha é quem tem **radar regulatório confiável**.

2. **STA é assíncrono e com hash** — protocolo 18 dígitos numérico, sequencial. Implementar fila robusta > implementar upload.

3. **DDR é DIÁRIO** — DRL é diário. IFs médias têm ~250 envios/ano só desses dois. Errar quebra fiscalização. **Confiabilidade > features.**

4. **O validador do BACEN é Java + planilha XLS** — replicar em Go é viável mas dá trabalho. **Aliada:** podemor rodar o BCValidador original via container em background como fallback.

5. **Crescimento do 3044 = JSON** — quem se preparar para JSON Schema + streaming ganha SCD/IP.

6. **FIDC tem tratamento único no 3040** — segmento alto valor, alta complexidade, pouca concorrência de SaaS especializado. Niche winner.

7. **Comunidade** — Tem "Especialista em CADOCs" como curso (FBM). Educadores podem ser parceiros de conteúdo.

8. **Regulatory Reporting ≠ só CADOC** — inclui DIMP (PIX), CCS, CVM, COAF. Cadoc.ai já percebeu. Roadmap nosso deve expandir.

9. **CROSS-DOC é onde o SaaS pode ser HERO** — ninguém faz check automático "3040 tem 3044 pendente? DLO bate com 4010?". Nosso L4 do validador.

10. **Norma Skills (cadoc.ai) é a melhor feature isolada** — RAG jurídico para IFs. Vale copiar e fazer open-source / freemium.

---

## 9. Pricing sugerido

| Plano | Preço/mês | Inclui |
|---|---|---|
| **Starter** | R$ 1.500 | 1 IF, 3 CADOCs (3040, 3050, 3044), 50 envios/mês, suporte email |
| **Pro** | R$ 4.500 | 1 IF, todos os CADOCs, envios ilimitados, radar regulatório, IA contextual |
| **Scale** | R$ 12.000+ | Multi-tenant, white-label, self-host opcional, SLA 99.9%, suporte dedicado |
| **Enterprise** | sob consulta | On-prem, custom integration, dedicated SRE |

Comparativo:
- Mitra/Matera: R$ 30-80k/mês
- cadoc.ai: R$ 5-15k/mês
- Dattos/Reg Mais: R$ 5-10k/mês
- **NOSSO:** R$ 1.5-12k/mês → **catch whole mid-market que está descoberto**

---

## 11. Cobertura ESG (DRSAC 2030) — gap de mercado real

O **CADOC 2030** (DRSAC) é o único CADOC que **ninguém dos concorrentes SaaS cobre direito**:

- Cobre risco Social, Ambiental e Climático em 3 níveis (setor CNAE / cliente / operação)
- 25-47 campos de avaliação por nível
- Semestral (jun/dez) — picos de demanda conhecidos e calculáveis
- Normativo: IN BCB 222/2021 + IN BCB 328/2022 + IN BCB 423/2023 + IN BCB 694/2025 (vigência dez/2026)
- Cruzamento com 3040 via IPOC + saldo devedor — **cross-doc real**

**Quem cobre atualmente:**
- B3Bee (consultoria) — tem planilha Excel de coleta, não SaaS
- FBM Educação — curso teórico
- Matera — provavelmente cobre mas vira commodity dentro do pacote RegTech

**Oportunidade:**
1. Wizard "respondendo este CADOC você atende simultaneamente DRSAC + PRSAC + GRSAC (Res 139)"
2. Input guiado pelos 3 níveis do leiaute, com taxonomia do que compõe cada "98 Não avaliado" / "99 Fora do escopo"
3. Cross-doc automático: pega o saldo devedor do 3040 e classifica nas contas COSIF que o DRSAC aceita
4. Compliance com a Resolução CMN 4.945/21 (PRSAC) e CMN 4.557 (GIR) **na mesma tela**

### O que precisa pra suportar DRSAC
- ✅ Leiaute DRSAC XLSX
- ✅ Instruções de preenchimento
- ✅ FAQ BACEN (DRSAC@bcb.gov.br é o canal oficial)
- ⏳ Base de CNAEs × dimensões de risco (setor)
- ⏳ Mapeamento das contas COSIF aceitas (3.3.1.10/20/30.13/16/17/18.00)
- ⏳ Tabela de obrigatoriedade por segmento (S1 → dez/2022, S2 → jun/2023, S3 → dez/2023, S4 → jun/2024)

---

## 10. Métricas de sucesso (12 meses)

- 50 IFs pagantes (mix: 35 SCD + 10 IP + 5 bancos pequenos)
- MRR R$ 150k/mês
- NPS > 60
- 99% uptime
- < 24h para atualizar leiaute após publicação BACEN
- 0 autuações de infração reportadas por clientes

---

## Próximo passo imediato

**Construir o protótipo do parser+gerador para o 3040**, validando o fluxo:

1. Ler `SCR3040_Leiaute.xls`
2. Gerar Go struct/json-schema
3. Receber JSON da IF via API
4. Validar contra XSD (gerado do leiaute) + regras semânticas (planilha críticas)
5. Gerar XML no formato BACEN
6. Empacotar ZIP com hash
7. (Futuro) Enviar via STA WS

Em paralelo, validar com 1-2 SCDs pagantes se o fluxo resolve o problema deles.

---

## 12. Mapa BACEN vs. cobertura por concorrente (verificação 2026-07-02)

| CADOC | BACEN | Mitra (LUZ) | Matera | cadoc.ai | Regulatório+ | Dattos | BIBlue | **NOSSO** |
|---|---|---|---|---|---|---|---|---|
| 3040 SCR | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 3042 correção | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 3044 eventos | ✅ (11/2025) | ✅ | ✅ | ✅ | ❓ | ✅ | ❓ | ✅ |
| 3050 estatísticas | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **2030 DRSAC ESG** | ✅ | ❓ | ✅ | ❓ | ❌ | ❌ | ❌ | **diferencial** |
| 2060 DRM | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 2061 DLO | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 2062 DLI | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 2070/2011 DDR | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 2160 DRL | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 2170 DLP | ✅ | ✅ | ✅ | ✅ | ✅ | ❓ | ❓ | ✅ |
| 5050 DRO | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **4111 Saldos Contábeis Diários** | ✅ (2022+) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **4060/4066 Balancete Conglomerado** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **4076 RCP** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **CRD (Controle Remessa) dispensa** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **STA Web Services (envio oficial)** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

> **Gap de mercado confirmado:** DRSAC (2030 ESG) é o único CADOC com **cobertura SaaS fraca**. Quem fizer isso primeiro vira referência obrigatória para ESG/PRSAC compliance.
>
> **Gap secundário:** Consolidação automática entre 3040 + 2030 + 4111 + 4060 — ninguém faz, mas é o que IFs querem.