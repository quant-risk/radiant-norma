# Radiant Sentinel — Produto · Tese · Roadmap

> **Radiant Sentinel é a sentinela regulatória da Radiant Risk Solutions (marca da Fortvna) — plataforma SaaS de CADOCs BACEN que compete contra Mitra (LUZ), Matera, cadoc.ai, Dattos e outros, entendendo que o cliente é uma instituição financeira regulada (banco, SCD, IP, cooperativa, DTVM) e que isso muda tudo: ciclo de venda, due diligence, certificações obrigatórias, regime de compliance.**
>
> **Tagline:** *"Radiant Sentinel — sentinela regulatória pra IF brasileira."*
> **Sub-tagline:** *"Sob a égide da Radiant, sua IF na norma."*

Esta é a **terceira peça** da base documental, complementando:
- `README.md` — mapa do material capturado (BACEN + concorrentes)
- `ENG_REVERSA.md` — análise técnica de concorrentes e gaps

**Marca-mãe:** Radiant Risk Solutions (marca Radiant, da Fortvna Risk Solutions)
**Produto:** Radiant Sentinel
**Sub-produtos da família:** Sentinel ESG · Sentinel Radar · Sentinel Connect · Sentinel Studio · Sentinel Audit
**Planos:** Sentinel Lite · Sentinel Pro · Sentinel Scale · Sentinel Enterprise

Data-base: **2026-07-03**. Autoria: Mavis · Radiant Risk Solutions.

---

## 1. TESE — Radiant Sentinel (em uma frase + desdobramentos)

> **"A corrida regulatória 2024-2027 (IN BCB 530/2024 [3044], IN BCB 733/2026 [3040], IN BCB 754/2025 [4111], IN BCB 694/2025 [DRSAC], Res BCB 139/2022 [GRSAC], Res BCB 205 [Basileia 4.0]) criou uma janela de adaptação obrigatória em todas as ~2.000 IFs brasileiras. IFs que tratam CADOCs como commodity operacional (geração+envio manual) serão reprocessadas ou pagarão multas a cada virada de data-base. Radiant Sentinel é a sentinela que nunca dorme: (1) plataforma que abstrai mudanças de leiaute sem retrabalho, (2) radar regulatório proativo, e (3) cross-doc automática — vendendo tranquilidade, não XML."**

**3 teses paralelas que se reforçam:**

### Tese 1 — Vendedor de "esteira regulatória", não de XML

IFs gastam 1-3 pessoas a 100% do tempo gerando, validando e enviando CADOCs todo mês. Quando o BACEN muda o leiaute (3-5x/ano), refazem tudo. IFs não compram gerador de XML — compram **tranquilidade** de que o sistema se adapta automaticamente.

**Radiant Sentinel entrega:**
- **Schema Registry versionado por data-base**: cada release BACEN entra como nova versão, IF não mexe em código.
- **Validador semântico pré-envio** (Sentinel Audit): reusa a planilha `SCR3040_Criticas.xls` como regras vivas.
- **Cross-doc engine** (Sentinel Radar): detecta "3040 diz X mas 4111 diz Y; quem está errado?" — feature que ninguém tem.

### Tese 2 — ESG/DRSAC é first-mover em vertical sub-explorada

CADOC 2030 (DRSAC — Risco Social, Ambiental e Climático):
- Vigência IN BCB 694/2025 em **dez/2026**
- Obrigatório S1–S4 + IPs (Resolução CMN 4.945/21 PRSAC)
- 25-47 campos de avaliação por nível (setor/cliente/operação)
- Cruzamento com 3040 via IPOC + saldo devedor — **cross-doc real**
- **Nenhum concorrente SaaS cobre direito**: B3Bee é consultoria, Matera mete dentro do RegTech, cadoc.ai cita de passagem

**Módulo Sentinel ESG** da Radiant Sentinel captura essa janela.

### Tese 3 — Mid-market tem fit perfeito pra SaaS acessíveis

| Perfil | Dor | Custo hoje |
|---|---|---|
| SCD pequena (carteira R$ 50-500M) | 1-3 pessoas em compliance full-time | R$ 5-15k/mês Matera/cadoc.ai |
| SEP (empréstimo entre pessoas) | Não tem equipe de compliance | Manual em planilha, multas |
| IP (instituição de pagamento) | Altos volumes de eventos (3044) | Paga caro por Dattos/similar |
| Fintech "BaaS" | 50 clientes SCD perguntam se oferece CADOC | Tem que montar do zero |

**Radiant Sentinel Lite** (R$ 1,5k/mês, onboarding 15 min) + **white-label** para Fintechs BaaS = impossível de Matera igualar (modelo de consultoria).

---

## 2. Personas & Buyers — quem decide, quem usa, quem paga

### Compradores (Budget Holder)

| Persona | Cargo | Tamanho IF | Dor principal | Pressão |
|---|---|---|---|---|
| **Carlos (SCD)** | Head Compliance / CTO | SCD R$ 100M–1B | Equipe 1 pessoa atolada em CADOC, medo de multa | 9º dia útil do mês |
| **Marina (IP)** | Diretora Risco / Sócia | IP médio-grande | 3044 + JSON, integrações BaaS, escala | Eventos em tempo real |
| **Roberto (Banco)** | Head de Tecnologia / CIO | Banco S3–S4 | Sistemas legados, custo de mudar, auditoria | Decisão 6-12 meses, RFP |
| **Ana (Fintech BaaS)** | Head Banking / Cofundadora | Fintech R$ 100M+ valuation | Oferecer BACEN-complaint pros clientes | Compliance dos próprios clientes |
| **Cooperativa** | Diretor Administrativo | Cooperativa grande | POUCOS CADOCs, todos manuais | Pouca equipe técnica |

### Usuários (Day-to-day)

| Persona | Cargo | Dia-a-dia | Quer ver |
|---|---|---|---|
| **Analista Júnior** | Compliance Analyst | Roda o job mensal, sobe XML no STA | "Não ter que preencher 50 planilhas; ver onde está o erro" |
| **Coordenador de Riscos** | Coord. Risco Operacional | Cruza CADOCs entre si, reporta ao regulador | "Validar antes de enviar; audit trail" |
| **Auditor interno** | Auditor Pl/Sr | Revisa evidências, NCs | "Quem enviou, quando, com qual leiaute, ver histórico" |

### Decisor

Pra **SCD/SEP/IP**: 1 decisão, Head Compliance = Head Risco = CTO. 1 venda por ciclo.
Pra **Banco**: comitê (CTO + Risco + Compliance + Compras + Jurídico). 3-9 meses.

---

## 3. Go-to-Market — como entrar vendendo pra IF

### Princípios

1. **Não vender "SaaS", vender "tranquilidade regulatória"** (Risco/Compliance compra isso).
2. **Onboarding em 15 min** (não deployment de 12 semanas como Matera).
3. **Pricing público** (diferencia de Matera/Mitra que escondem).
4. **Pilot gratuito 30-90 dias** antes de cobrar (IF exige PoC).
5. **Referência antes de escalar** (1 SCD-piloto vencido = 10 follow-ons em 60 dias).

### Canais

**Inbound (curto prazo):**
- SEO: "gerador CADOC 3040", "como enviar 3044 JSON", "DRSAC compliance", "RADAR regulatório BACEN"
- Conteúdo: blog Mavis (já existe da Radiant Risk Solutions) com 1 case/mês
- LinkedIn outbound do Henrique + dos fundadores Radiant/Fortvna

**Outbound (médio prazo):**
- Listas ABBC, Febrafar, ABFintechs, Anbima
- Webinars trimestrais com Compliance Officers convidados
- Patrocínio do "Compliance & Regulação" track em eventos (Cibele, Febrafar Tech, Ciab)

**Vendas estruturadas (médio prazo):**
- 1 BDR (Business Development Rep) em mês 6
- 1 Closer/AE em mês 9
- Pipeline target mês 6: 30 IFs em prospecção, 5 demo/semana

### Anticorpos (como vender pra IF)

IF regulada tem **muitos gatekeepers**. Mapeamento de quem fala com quem:

| Função | Pergunta que faz | O que responder |
|---|---|---|
| **Compliance** | "Atende CMN 4.966, Lei 14.155, LGPD?" | Sim + DPIA + DPO + cláusulas específicas |
| **Risco** | "Tem cross-doc 3040↔4111↔DRSAC?" | Roadmap demonstra; roadmap R3 |
| **Tecnologia** | "Stack? On-prem? Onde roda?" | Go + Postgres + AWS São Paulo; Helm on-prem opcional |
| **Jurídico** | "LGPD contrato com operado é DPA?" | Sim + due-dill + SLA + ENCARGO |
| **Compras** | "RFP, homologação interna, ANBIMA?" | Pacote RFP pronto + cases |
| **CIO** | "SOC 2? ISO 27001? Pen test?" | SOC 2 Type II em mês 18, pen test anual |

### Sales Play (script de discovery de 30min)

```
1. "Como vocês tratam [CADOC X] hoje?" → Diagnóstico de dor
2. "Quanto tempo/dinheiro custa hoje esse processo?" → TAM
3. "Se você pudesse tornar automático 80% desse trabalho, R$ Xk/mês seria problema?" → Limite
4. "Em até quando isso vira prioridade?" → Timing
5. "Quem mais precisa estar nessa conversa?" → Multi-thread
```

### Ciclos de venda (realistas)

| Segmento | Ciclo | Por quê |
|---|---|---|
| SCD/SEP | 30-90 dias | Decisor único, dor mensurável |
| IP | 60-120 dias | Compliance + Tecnologia |
| Banco S3-S4 | 6-12 meses | RFP, due-dill múltipla, comitê |
| Cooperativa | 30-60 dias | Decisor único, baixa complexidade |

---

## 4. PRODUTO — fases V0 → V3 (Radiant Sentinel)

### V0 — Beta fechado "Sentinel Pilot" (semanas 1-8)

**Objetivo:** validar com 2-3 SCDs-piloto que o fluxo resolve a dor.

Features:
- ✅ **CLI Go Sentinel**: `sentinel read-xlsx --file base.xlsx --validate > 3040.xml.zip`
- ✅ 3040 + 3050 + 3044 (JSON)
- ✅ Validador XSD + semântico (planilha críticas reimplementada)
- ✅ Landing page `radiant-sentinel.com.br` + pricing público
- ✅ 2-3 SCDs-piloto com 90 dias grátis (Sentinel Pilot Program)

**Métricas de sucesso:**
- 2-3 SCDs com envio real OK no STA
- NPS ≥ 50
- Zero multa BACEN atribuível ao SaaS
- 50% conversam em pagante

### V1 — SaaS comercial "Sentinel Launch" (meses 3-6)

**Objetivo:** R$ 50k MRR com 10-15 IFs ativas.

Features:
- ✅ Backend Go + API REST + frontend Next.js (Sentinel Console)
- ✅ Multi-tenant (cada IF isolada)
- ✅ **Sentinel Console**: dashboard regulatório com calendário de prazos, status de cada documento, críticas em tempo real
- ✅ **Sentinel ESG** — first-mover DRSAC 2030
- ✅ **Sentinel Audit** — log completo (quem enviou, quando, com qual leiaute)
- ✅ **STA Web** automação (Playwright no MVP, WS nativo em V1.5)
- ✅ Auth: SSO gov.br + email+2FA
- ✅ LGPD-compliant: DPA, criptografia at rest/in transit, DPIA, DPO de plantão
- ✅ **Sentinel Radar** — alerta de mudanças de leiaute (push email + in-app)
- ✅ Multitenancy + white-label opcional (BaaS Fintech)

**Planos Radiant Sentinel:**
| Plano | Preço | Inclui |
|---|---|---|
| **Sentinel Lite** | R$ 1.500/mês | 1 IF, 3 CADOCs à escolha, 50 envios/mês, suporte email |
| **Sentinel Pro** | R$ 4.500/mês | 1 IF, todos os CADOCs, envios ilimitados, **Sentinel Radar** |
| **Sentinel Scale** | R$ 12.000/mês | Multi-tenant, white-label, SLA 99.9%, integrações |
| **Sentinel Enterprise** | Sob consulta | Self-host on-prem, dedicated, gerente de conta, pen test |

### V1.5 — STA nativo "Sentinel Direct" (meses 6-8)

- Substituir Playwright por STA Web Services nativo (REST)
- Suporte a A1 (certificado digital ICP-Brasil em arquivo) e A3 (token físico)
- Fila de upload com retry exponencial
- Hash SHA-256 antes do envio
- Logs de protocolo STA (até 18 dígitos numérico)

### V2 — Diferenciação "Sentinel Intelligence" (meses 9-12)

Features:
- ✅ **Sentinel ESG** v2: wizard cruzar PRSAC ↔ DRSAC ↔ GRSAC (Res BCB 139/2022) automaticamente
- ✅ **Sentinel Radar** v2: cross-doc engine detecta inconsistências entre 3040 ↔ 4111 ↔ 4060 ↔ DRSAC
- ✅ **SOC 2 Type I** (preparação 6 meses)
- ✅ **Self-host opcional** (Helm chart Kubernetes)
- ✅ **Sentinel Connect** — Open API pública + Webhooks (`cadoc.validated`, `cadoc.sent`, `cadoc.failed`)
- ✅ Integração nativa com Topázio, Sinacor, ERPs bancários
- ✅ Simulador "what-if" pra DRSAC (impacto de mudança regulatória)

### V3 — Plataforma "Sentinel Network" (meses 13-18+)

- Outros reportes: DIMP (PIX), CCS, COAF, CVM
- **Sentinel Studio** — workspace low-code pra IFs customizarem templates
- Marketplace de IFs/consultorias
- Self-host enterprise
- Expansão LatAm (BACEN equivalente na Argentina, Chile)
- SOC 2 Type II
- API pública pra Fintechs BaaS construirem produtos
- Series A se tração em V2

---

## 5. Arquitetura técnica

### Stack Radiant (Go-heavy, alinhado à Radiant Risk Solutions)

```
┌────────────────────────────────────────────────────────────────┐
│  SENTINEL CONSOLE — Next.js 14 + TypeScript + Tailwind         │
│  Sentinel Dashboard, Sentinel ESG, Sentinel Audit, Calendar     │
│  Vercel + Cloudflare CDN                                         │
└────────────────────────────────┬───────────────────────────────┘
                                 │ REST + Webhooks (Sentinel Connect)
                                 ▼
┌────────────────────────────────────────────────────────────────┐
│  SENTINEL GATEWAY — Go (chi router), OpenAPI, OAuth2 + JWT     │
│  Rate limit, Sentinel Audit (tamper-evident log)               │
│  Multi-tenancy via header X-IF-ID                               │
└────────────────────────────────┬───────────────────────────────┘
                                 │
   ┌─────────────────────────────┼─────────────────────────────┐
   ▼                             ▼                             ▼
┌────────────────┐    ┌────────────────────┐       ┌──────────────────┐
│  SENTINEL      │    │  SENTINEL CORE       │       │  SENTINEL RADAR  │
│  REGISTRY      │    │  (Pipeline)          │       │  Go + Crawler     │
│  Postgres      │    │  Go                   │       │  BACEN            │
│                │    │                       │       │  Diff leiaute     │
│ - IFs          │    │ - XLSX → struct       │       │  LLM summarize    │
│ - Schemas      │    │ - struct → XML        │       │  Push notif       │
│ - Críticas     │    │ - XSD validate        │       │  Cross-doc check  │
│ - Envios       │    │ - Critique validate   │       └──────────────────┘
│ - Audit log    │    │ - Cross-doc check     │
│                │    │ - ZIP pack            │
└────────────────┘    │ - Sentinel ESG wizard │
                       └────────────────────┘
                                  │
                                  ▼
                        ┌────────────────────┐
                        │  SENTINEL DELIVERY  │
                        │  STA Web / WS       │
                        │  Retry queue        │
                        │  Cert A1/A3         │
                        │  Hash SHA-256       │
                        └────────────────────┘
```

### Componentes críticos

**1. Sentinel Registry (Postgres + JSON Schema):**
```sql
CREATE TABLE schema_versions (
  cadoc_code   VARCHAR(4),     -- '3040', '2030'
  effective_from DATE,         -- data-base a partir de quando vale
  doc_uri       TEXT,          -- SHA do arquivo XLSX original
  fields        JSONB,         -- [{tag, attr, type, required, domain}, ...]
  xsd           TEXT,          -- XSD gerado
  changelog     TEXT,
  PRIMARY KEY (cadoc_code, effective_from)
);
```

**2. Sentinel Audit — Validador multi-camada:**
- **L1 (XSD):** sintaxe de XML, tipos de dados, obrigatoriedade
- **L2 (Semântica):** reimplementação das críticas BACEN (`SCR3040_Criticas.xls` → Go rules)
- **L3 (Cross-doc):** 3040 ↔ 4111 ↔ 4060 ↔ DRSAC (regra de consistência)
- **L4 (Histórico):** diff vs. base anterior (mudanças suspeitas)

**3. Sentinel Delivery — STA Client (Go):**
- Modo 1 (Playwright) em V1 → modo 2 (WS nativo) em V1.5
- Auth via certificado A1/A3 (PEM/tokens)
- Hash SHA-256 pré-envio
- Fila com retry exponencial (jitter)
- 10 upload simultâneos/IF (limite BACEN)

**4. Sentinel Radar — Crawler regulatório:**
- Crawler diário do `bcb.gov.br/estabilidadefinanceira`
- Detecta mudanças em `leiautedocumentoscrd` (diff XLSX)
- Notificações push + alerta in-app
- LLM opcional para explicar impacto em linguagem natural

**5. Sentinel ESG — Wizard DRSAC:**
- Cruzar PRSAC ↔ DRSAC ↔ GRSAC (Res BCB 139/2022)
- 25-47 campos de avaliação por nível
- Integração com CNAE setorial + COSIF
- Cross-doc com 3040 via IPOC + saldo devedor

**6. Motor de templates:**
- Cada CADOC tem template XLSX ↔ XML mapping
- IF faz upload do XLSX preenchido → geramos XML → validamos → sugerimos correções
- Alternativa: integração via API com core banking

### Compliance nativo no código (Radiant Sentinel)

- **Criptografia at-rest**: Postgres AES-256, S3 SSE-KMS, volumes EBS criptografados
- **Criptografia in-transit**: TLS 1.3, HSTS, mTLS entre serviços internos
- **Audit log**: append-only, hash chain (cada entrada referencia hash da anterior — tamper-evident)
- **Segregação multi-tenant**: cada IF com schema Postgres separado + row-level security
- **Backup**: RPO < 1h, RTO < 4h, geo-redundant (São Paulo + Sul)
- **WAF + rate limiting + DDoS** (CloudFlare)

---

### 5.1 Como a plataforma vai ser — UX e jornada do usuário

#### Filosofia de produto

Radiant Sentinel é desenhada com **três princípios**:

1. **Compliance Officer não é dev** — UI é **Excel-friendly** (eles vivem em planilha) com wizard visual
2. **Compliance Officer não confia em "mágica"** — toda transformação é auditável (Sentinel Audit mostra origem de cada campo)
3. **Compliance Officer tem pressa** — onboarding 15 min, sem instalação, sem treinamento

#### Jornada do usuário — fluxo end-to-end

```
┌─────────────────────────────────────────────────────────────────────┐
│  PASSO 1 — LOGIN                                                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  [Logo Radiant Sentinel]                                       │  │
│  │                                                                │  │
│  │  Bem-vindo de volta, Carla.                                    │  │
│  │                                                                │  │
│  │  [🔵 Entrar com gov.br]   [📧 Email + 2FA]                     │  │
│  │                                                                │  │
│  │  Status: ✓ Conforme   |  Última crítica: 2 dias atrás          │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
            ↓
┌─────────────────────────────────────────────────────────────────────┐
│  PASSO 2 — SENTINEL CONSOLE (Dashboard principal)                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  ╭─ Calendário Regulatório ────────────────────────────────╮   │  │
│  │  │ ◀ Julho 2026 ▶                                           │   │  │
│  │  │ D  S  T  Q  Q  S  S                                       │   │  │
│  │  │       1  2  3  4  5  6                                     │   │  │
│  │  │ 7  8  ●9  10 11 12 13      ● = prazo BACEN                │   │  │
│  │  │ 14 15 16 17 18 19 20      3040 vence dia 9!               │   │  │
│  │  ╰─────────────────────────────────────────────────────────╯   │  │
│  │                                                                 │  │
│  │  ╭─ Status dos Documentos ──────────────────────────────────╮   │  │
│  │  │ ✅ 3040 (jun) — Enviado em 08/07, prot. 2026070823       │   │  │
│  │  │ ✅ 3050 (jun) — Enviado em 08/07, prot. 2026070824       │   │  │
│  │  │ ⚠️  3044 (jun) — 312 eventos pendentes                  │   │  │
│  │  │ 🔴 2030 DRSAC — Vence 30/07, **NÃO INICIADO**           │   │  │
│  │  │ 🔴 4111 (jun) — Vence 15/07, **NÃO INICIADO**           │   │  │
│  │  ╰─────────────────────────────────────────────────────────╯   │  │
│  │                                                                 │  │
│  │  ╭─ Sentinel Radar — Alertas ──────────────────────────────╮   │  │
│  │  │ 🟡 IN BCB 733/2026 — 3040 muda em 01/08 (15 dias)         │   │  │
│  │  │    → Ação recomendada: revisar schema 3040 v2026.08        │   │  │
│  │  │ 🟢 Sem mudanças em 3050 e 4111                             │   │  │
│  │  ╰─────────────────────────────────────────────────────────╯   │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
            ↓
┌─────────────────────────────────────────────────────────────────────┐
│  PASSO 3 — GERAR CADOC (clica em 3040 → "Gerar agora")              │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  Como você quer gerar?                                         │  │
│  │                                                                │  │
│  │  [📤 Subir planilha XLSX]    [🔌 Integrar via API]            │  │
│  │                                                                │  │
│  │  ┌─ Drop zone ────────────────────────────────────────────┐   │  │
│  │  │                                                         │   │  │
│  │  │   ⬆️  Arraste seu XLSX preenchido aqui                  │   │  │
│  │  │       ou clique para selecionar                          │   │  │
│  │  │                                                         │   │  │
│  │  └─────────────────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
            ↓
┌─────────────────────────────────────────────────────────────────────┐
│  PASSO 4 — VALIDAÇÃO (Sentinel Audit + L1/L2/L3/L4)                 │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  🔍 Validando base_2026-07.xlsx                               │  │
│  │  ────────────────────────────────────────────────              │  │
│  │  ✅ L1 XSD — Sintaxe OK (487 campos)                          │  │
│  │  ✅ L2 Semântica — 312 críticas verificadas                   │  │
│  │  ⚠️  L3 Cross-doc — 2 inconsistências com 4111                │  │
│  │     └─ IPOC 12345678000199: saldo 3040 = R$ 50.000,00        │  │
│  │        saldo 4111 = R$ 49.875,30 (diff R$ 124,70)             │  │
│  │     └─ IPOC 98765432000111: saldo 3040 = R$ 12.300,00        │  │
│  │        4111 não reporta operação                              │  │
│  │  ✅ L4 Histórico — Diff vs base anterior OK                    │  │
│  │                                                                │  │
│  │  [Ver detalhes]  [Gerar mesmo assim]  [Cancelar]               │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
            ↓
┌─────────────────────────────────────────────────────────────────────┐
│  PASSO 5 — PREVIEW + ENVIO (Sentinel Delivery)                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  📦 ZIP gerado: 3040_2026-07_IF1234_202607031530.zip          │  │
│  │  SHA-256: a1b2c3d4e5f6...                                     │  │
│  │  Tamanho: 4.2 MB                                               │  │
│  │                                                                │  │
│  │  Status BACEN: 🟢 STA Web disponível                          │  │
│  │                                                                │  │
│  │  [👁️ Preview XML]  [🚀 Enviar pro BACEN agora]               │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
            ↓
┌─────────────────────────────────────────────────────────────────────┐
│  PASSO 6 — CONFIRMAÇÃO + AUDIT                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  ✅ 3040 enviado com sucesso!                                  │  │
│  │  Protocolo STA: 20260703153045IF1234                           │  │
│  │  Hash: a1b2c3d4...                                            │  │
│  │                                                                │  │
│  │  [Ver no STA]  [Sentinel Audit]  [Voltar ao Console]           │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

#### Stack técnico de UX

- **Frontend:** Next.js 14 (App Router) + TypeScript + Tailwind + shadcn/ui
- **State:** TanStack Query (servidor) + Zustand (UI state)
- **Forms:** React Hook Form + Zod (validação client-side)
- **Drop zone:** react-dropzone (drag & drop XLSX)
- **Tabelas complexas:** TanStack Table (preview do XML antes do envio)
- **Charts:** Recharts (dashboard com histórico de envios)
- **Notificações:** Sonner (toasts) + WebSocket push (Sentinel Radar)
- **Auth:** NextAuth.js com provedor gov.br + credenciais email+2FA
- **i18n:** pt-BR primário, en secundário

#### Princípios de UX aplicados

| Princípio | Como aparece na Radiant Sentinel |
|---|---|
| **Não obriga mudar planilha** | Aceita XLSX no formato que a IF já usa |
| **Mostra antes de enviar** | Preview XML + diff antes do envio |
| **Erro acionável** | Cada crítica vem com link "Como corrigir" |
| **Auditável por construção** | Sentinel Audit registra tudo |
| **Mobile-friendly** | Compliance Officer checa status no celular |
| **Acessibilidade (WCAG AA)** | Contraste, navegação por teclado, ARIA |
| **Onboarding 15 min** | Wizard inicial com 1 CADOC como exemplo |

#### Integrações planejadas

| Integração | Tipo | Quando |
|---|---|---|
| **Sentinel Connect API** | REST + Webhooks (público) | V1 |
| **STA Web / WS** | Cliente BACEN (oficial) | V1 → V1.5 |
| **Topázio** (banco core) | Adapter de extração | V2 |
| **Sinacor** (corretora) | Adapter de extração | V2 |
| **gov.br SSO** | OAuth2 | V1 |
| **Slack / Teams** | Webhook notificação | V1 |
| **Email transacional** | SendGrid/SES | V1 |
| **PagerDuty** | Alertas críticos | V2 |
| **Helm chart (on-prem)** | Kubernetes | V2 |
| **BaaS White-label** | Multi-tenant + custom CSS | V1 (Scale) |

---

## 6. COMPLIANCE — o que muda porque cliente é regulado

IFs não compram SaaS qualquer. Precisamos atender:

### 6.1 LGPD (Lei 13.709/2018)

| Requisito | Como atendemos |
|---|---|
| Encarregado (DPO) | DPO de plantão via Radiant Risk Solutions, contrato SLA |
| Encarregado designado | Oferecemos DPO as a service pra SCD pequena |
| DPIA (Relatório de Impacto) | Feito e atualizado anualmente |
| Cláusulas de operador (art. 39) | DPA anexo ao contrato com IF |
| Registro de operações (art. 37) | Audit log com hash chain, retenção 5 anos |
| Segurança (art. 46) | Criptografia, controle de acesso, MFA |
| Notificação de incidente (art. 48) | SLA 24h, formulário padrão BACEN/LGPD |
| Transferência internacional | **Não faremos** — tudo fica no Brasil (BCloud/AWS São Paulo) |
| Direitos do titular | Endpoints API `/lgpd/dsar` prontos pra IF rotear pedidos |

### 6.2 Resolução CMN 4.658/2018 (Segurança Cibernética)

IFs devem ter controles rígidos. Fornecedor SaaS que processa dados bancários:

| Requisito BCB | Como atendemos |
|---|---|
| Política de Segurança da Informação | Documentação pública, auditada |
| Gestão de acessos | SSO + MFA + RBAC + log |
| Criptografia | at-rest + in-transit (TLS 1.3 + AES-256) |
| Plano de contingência | RPO < 1h, RTO < 4h, geo-redundant |
| Testes de intrusão | Anual por empresa independente |
| Notificação de incidente | SLA 24h, formulário CMN 4.658 |
| Backup | Diário, criptografado, em 2 regiões BR |
| Controle de prestadores | Nosso SaaS é "prestador" — temos controles que IF exige |

### 6.3 Certificações alvo

| Certificação | Quando | Por quê |
|---|---|---|
| **LGPD compliance attested** | Mês 1 | Juridicamente obrigatório |
| **SOC 2 Type I** | Mês 9 | IF médio (S3-S4) exige |
| **SOC 2 Type II** | Mês 18 | IF grande (S1-S2) exige |
| **ISO 27001** | Mês 18-24 | Diferencial competitivo |
| **PCI DSS** | NÃO aplicável | Não processamos cartão diretamente |
| **BCB como "homologado"** | NÃO existe | BACEN não homologa fornecedores SaaS; IFs fazem due-dil |

### 6.4 Onde os dados moram

**Regra:** tudo no Brasil, sempre.

- **Produção:** AWS `sa-east-1` (São Paulo) ou BCloud (Datacenter em SP)
- **DR:** `sa-east-2` ou Filial DF
- **Não-replicamos:** nem metadados para fora do país (LGPD art. 33)
- **Certificado ICP-Brasil A1/A3** instalado nos servidores STA client

### 6.5 Auditoria — o que IF vai pedir

Pra entrar em qualquer banco S3-S4+, **anexa-se um pacote de**:

1. SOC 2 Type II ou relatório anual de pen test + auditoria externa
2. ISO 27001 (se tiver) ou política ISMS documentada
3. DPIA (LGPD)
4. Lista de sub-processadores + localização de dados
5. Histórico de incidentes (12 meses) com SLAs cumpridos
6. Política de backups + teste de restore anual
7. Customer references em outras IFs reguladas
8. Plano de exit (offboarding + portabilidade de dados)

---

## 6.5 ★ Catálogo Estruturado — base do Sentinel Audit

Em **2026-07-03**, extraí todas as planilhas BACEN (críticas + leiautes) para JSON estruturado na pasta `_catalogos/`. Esta é a **base direta do Sentinel Audit**.

### Estatísticas

| Recurso | CADOCs | Linhas |
|---|---|---|
| **`_catalogos/criticas.json`** | 4 (3040, 3050, 2061 DLO, 2070 DDR) | **1.081 regras de validação** |
| **`_catalogos/leiautes.json`** | 8 (3040, 3042, 3050, 2030 DRSAC, 2060 DRM, 2062 DLI, 2070 DDR, 2160 DRL) | **4.244 linhas de campos** |

### Como o Sentinel Audit usa o catálogo

| Camada | O que valida | Fonte de dados |
|---|---|---|
| **L1 — Estrutural (XSD)** | Sintaxe XML/JSON, tipos, obrigatoriedade | `leiautes.json` → XSD gerado em Go |
| **L2 — Semântica** | Regras do BACEN (`SCR3040_Criticas.xls`, etc) | `criticas.json` → regras portadas pra Go |
| **L3 — Cross-doc** | "3040 vs 4111 vs DRSAC" | Multi-CADOC em memória (**exclusivo**) |
| **L4 — Histórico** | Diff vs base anterior | Postgres versioning + LLM explainer (**exclusivo**) |

### Diferencial vs BCValidador (BACEN)

| Capability | BCValidador BACEN | Sentinel Audit Radiant |
|---|---|---|
| L1 XSD | ✅ | ✅ |
| L2 Semântico | ✅ | ✅ |
| Multi-CADOC simultâneo | ❌ | ✅ (L3) |
| Histórico + diff | ❌ | ✅ (L4) |
| API REST + Webhooks | ❌ | ✅ |
| Audit log tamper-evident | ❌ | ✅ (hash chain) |
| LGPD compliance | ❌ | ✅ |
| Multi-tenant SaaS | ❌ | ✅ |
| Self-host on-prem | ❌ | ✅ (Helm chart V2) |
| Atualização automática em mudança de leiaute | ❌ (usuário baixa) | ✅ (Sentinel Radar) |

**Estratégia:** Não copiamos BCValidador (binário Java proprietário). Reimplementamos em Go as regras públicas das planilhas de críticas + adicionamos 3 camadas proprietárias.

### Exemplo de uso

```bash
cd cadocs/
python3 _catalogos/extract.py  # re-gera os JSONs
python3 -c "
import json
d = json.load(open('_catalogos/criticas.json'))
for c in d['criticas']['3040'][:3]:
    print(c['codigo'], '-', c['regra'])
"
# B01 - Erro XML
# B02 - Arquivo .ZIP deve ser gerado pelo aplicativo validador
# B03 - Instituição remetente deve possuir autorização
```

Documentação completa: [`_catalogos/README.md`](_catalogos/README.md).

### Roadmap do catálogo

| # | Ação | Quando |
|---|---|---|
| C.1 | Capturar 3050 críticas V11 (mais recente que V9) | semana 1 |
| C.2 | Extrair 2060 DRM críticas do PDF | semana 1 |
| C.3 | Capturar 2030 DRSAC críticas (URL alternativa) | semana 1 |
| C.4 | Normalizar data-base das críticas | semana 2 |
| C.5 | Gerar XSD Go a partir do `leiautes.json` (3040) | semana 2-3 |
| C.6 | Portar 50% das críticas 3040 pra Go rules | semana 3-4 |
| C.7 | Validar Sentinel Audit L1+L2 contra BCValidador (paralelo) | semana 4 |
| C.8 | Portar críticas 3050 + 2061 DLO | mês 2 |
| C.9 | Schema Registry com versioning por data-base | mês 2 |

---

## 7. ROADMAP — 18 meses

### Fase 0 — Validação (semanas 1-8)

| # | Milestone | Semana | Dono |
|---|---|---|---|
| 0.1 | CLI Go: gera 3040 + 3050 + 3044 (JSON) a partir de XLSX | S2 | Tech Lead |
| 0.2 | Validador XSD (gerado) + semântico (regras das Críticas.xls) | S4 | Dev |
| 0.3 | Landing page + pitch deck 1.0 | S5 | Fundador |
| 0.4 | 5 entrevistas discovery com SCDs (rede Radiant/Fortvna) | S6-7 | Fundador |
| 0.5 | LOI assinado com 2 SCDs-piloto (90 dias grátis) | S8 | Fundador |
| 0.6 | Envio real OK no STA (produção) | S8 | Tech Lead |

**Saída da Fase 0:** 1 SCD-piloto processado na produção, LOIs prontos.

### Fase 1 — MVP Comercial (meses 3-6)

| # | Milestone | Mês | Dono |
|---|---|---|---|
| 1.1 | Backend Go: schema registry + 3040/3050/3044 | M3 | Tech + 1 Dev |
| 1.2 | Frontend Next.js: dashboard + upload + preview XML | M3 | Frontend Dev |
| 1.3 | Multitenancy + auth (SSO gov.br + email/MFA) | M4 | Tech Lead |
| 1.4 | STA Web (Playwright) | M4 | Dev |
| 1.5 | **2030 DRSAC** + wizard ESG | M5 | Dev |
| 1.6 | Audit log + LGPD DPA + DPIA | M5 | Jur + Fundador |
| 1.7 | Alert push pra mudanças de leiaute | M6 | Dev |
| 1.8 | Pricing público + 2 primeiros pagantes | M6 | Fundador |
| 1.9 | SOC 2 Type I (start) | M6 | Fundador + Jur |

**MRR fim da Fase 1:** R$ 50k (10-15 pagantes).

### Fase 1.5 — STA nativo (meses 6-8)

| # | Milestone | Mês | Dono |
|---|---|---|---|
| 1.5.1 | STA WS nativo (REST, sem Playwright) | M7 | Tech Lead |
| 1.5.2 | Cert A1/A3 + fila com retry | M7 | Dev |
| 1.5.3 | Logging de protocolo STA | M8 | Dev |

### Fase 2 — Scale (meses 9-12)

| # | Milestone | Mês | Dono |
|---|---|---|---|
| 2.1 | DRL/DLO/DLI/DDR/DRM/DLP (todos Basileia) | M9 | 2 Devs |
| 2.2 | Radar regulatório (crawler BACEN) | M10 | Dev |
| 2.3 | Cross-doc engine (3040↔4111↔4060↔DRSAC) | M10 | Tech Lead |
| 2.4 | SOC 2 Type I obtido | M11 | Fundador + Jur |
| 2.5 | Self-host Helm chart (preview para S1) | M11 | Dev |
| 2.6 | White-label API (BaaS) | M12 | Dev |
| 2.7 | 1 BDR contratado + outbound | M9-12 | Fundador |

**MRR fim da Fase 2:** R$ 200k (40-50 pagantes).

### Fase 3 — Diferenciação (meses 13-18)

| # | Milestone | Mês | Dono |
|---|---|---|---|
| 3.1 | IA contextual no validador (cross-doc check) | M13 | ML Eng |
| 3.2 | Simulador what-if (DRSAC stress test) | M14 | Dev |
| 3.3 | Open API pública + Webhooks | M15 | Dev |
| 3.4 | Integração Topázio + Sinacor | M16 | Dev |
| 3.5 | SOC 2 Type II obtido | M18 | Fundador + Jur |
| 3.6 | Head of Sales contratado + 2 SDRs | M14-18 | Fundador |
| 3.7 | Cobertura: DIMP (PIX), CCS, COAF | M15-17 | Tech Lead |

**MRR fim da Fase 3:** R$ 500k-1M (80-150 pagantes).

### Marcos de equipe

| Estágio | Equipe |
|---|---|
| Fase 0 | Fundador + Tech Lead (você + eu) |
| Fase 1 | + 1 Dev Go, 1 Dev Front, 1 SRE part-time |
| Fase 1.5 | + 1 Backend |
| Fase 2 | + 1 BDR, 1 SRE full-time, 1 QA |
| Fase 3 | + ML Eng, Head of Sales, 2 SDRs |
| Ano 2 | 15-20 pessoas + escritório/compliance |

### Marcos de capital

| Etapa | Uso | Valor |
|---|---|---|
| Bootstrap (Fase 0-1) | Fundador + Radiant/Fortvna aporta | R$ 300-500k |
| Seed (Fase 1.5) | 4 hires + infra + LGPD/SOC2 | R$ 1.5-2M |
| Series A (Fase 3+) | Head of Sales + 8 hires + expansão | R$ 6-10M |

---

## 8. Métricas de sucesso (12 e 24 meses)

### Phase 0 (semanas 1-8)

- [ ] 2-3 SCDs-piloto com envio real OK
- [ ] 50% conversam em pagante
- [ ] NPS ≥ 50
- [ ] **Zero multa BACEN atribuível ao SaaS**

### Phase 1 (meses 3-6) — MVP Comercial

- [ ] R$ 50k MRR (10-15 pagantes)
- [ ] 95% uptime
- [ ] < 24h para detectar + notificar mudança de leiaute BACEN
- [ ] 5 cross-docs 3040↔4111↔4060↔DRSAC
- [ ] 80% inbound no funil (vs outbound)

### Phase 2 (meses 9-12) — Scale

- [ ] R$ 200k MRR (40-50 pagantes)
- [ ] **5 SCDs de S1-S4** (Tier 1 institutions)
- [ ] **10 IPs** (Instituições de Pagamento)
- [ ] 1 Fintech BaaS usando white-label
- [ ] SOC 2 Type I
- [ ] NRR > 110% (expansão por uso)

### Phase 3 (meses 13-18)

- [ ] R$ 500k-1M MRR (80-150 pagantes)
- [ ] SOC 2 Type II
- [ ] Expansão regional (Argentina? Chile?)
- [ ] Series A fechado

---

## 9. Riscos e mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| BACEN muda leiaute e quebra parser | Alta | Alto | Radar regulatório + versionamento + testes pra cada release |
| IF recebe auto de infração por erro nosso | Média | Alto | Disclaimer + audit log + SLA + seguro E&O (R$ 1M+) |
| Concorrente grande copia feature | Alta | Médio | Velocidade + nicho + brand |
| ICP-Brasil / A1-A3 é barreira | Alta | Médio | Templates + tutorial + parceria com ACs (Serasa, Certisign) |
| IFs pequenas não têm equipe técnica | Alta | Médio | UX Excel-friendly + suporte humano incluso no plano starter |
| LGPD / BACEN barra o SaaS por compliance | Média | Alto | DPIA cedo, contrato DPA, DPO de plantão, certificações roadmap |
| Ciclo de venda longo (banco) | Alta | Médio | Foco em SCD/IP no início (ciclo 30-90d) |
| Engineering founding risk (tech lead sai) | Média | Alto | Documentação forte, 2 Devs no time desde Fase 1 |
| CAPEX alto em SOC 2 / pen test | Média | Médio | Budget Phase 1.5 e Phase 2, ROI claro |
| Concorrente lança DRSAC antes | Baixa | Alto | Velocidade: V1 com 2030 só em M5 |
| Regulador cria DOC novo (ex: criptos ACAM212) | Alta | Médio | Schema Registry extensível + radar |

---

## 10. Próximos passos concretos (esta semana)

### O que fazer HOJE (se for sério)

1. **Validar a tese com stakeholders Radiant/Fortvna:** apresentar isso pro Henrique, decidir se Radiant Sentinel vira spin-off ou nova vertical da Fortvna
2. **Marcar 3 calls discovery:** com SCDs amigas da Fortvna, validar dor + willingness to pay
3. **Tech check (1 dia):** instalar Go 1.22+, clonar repo, ler `SCR3040_Leiaute.xls`, ver se dá pra parsear com `excelize`

### O que fazer esta semana

1. **Tech setup (3 dias):** repo `radicator` (ou nome melhor), pipeline XLSX→XML com 3040
2. **Discovery (2 calls):** diagnosticar dor + ICP (perfil, orçamento, ciclo)
3. **Validação de LP:** rascunho de página "Como funciona" + caso "ANTES/DEPOIS"

### Próximas 2 semanas

1. **MVP CLI rodando** com 3040 + 3050 + 3044
2. **2-3 LOIs** assinados com SCDs-piloto
3. **Decisão:** Go or No-Go na transição F0 → F1 (precisa LOI assinado + capital bootstrap aprovado)

---

## 11. Considerações finais sobre o contexto regulatório

### O que isso significa na prática

1. **Não é SaaS comum:** cada IF precisa homologar o fornecedor (due-dil 60-120 dias). Margens para baixo no início pra construir credibilidade.
2. **Compliance é feature, não custo:** LGPD/SOC2 viram **selling point** ("Aqui nosso DPA, nosso SOC 2 Type II, nosso DPO"). Matera/Mitra não falam disso explicitamente.
3. **Velocidade regulatória importa:** IN BCB 733/2026 muda 3040 em 3 ondas (mai/jul/nov 2026). Quem chegar **adaptado** primeiro ganha contrato novo.
4. **Ciclo de caixa é lento, mas previsível:** SCD paga em dia, maio ARR recorrente. Melhor que startup B2B comum que cancela.
5. **Compliance amadurece a empresa:** fazer SOC 2 + LGPD formal cedo significa que se pivotar pra outro vertical regulatório (saúde, seguros), a base compliance já está feita.

### Próximos movimentos táticos (esta sprint)

- [ ] Definir **CNPJ do produto** (SLU da Radiant Risk Solutions, subsidiária da Fortvna, parceria com consultoria?)
- [ ] **Marcar reunião com Compliance Officer de SCD amiga** pra validar dor
- [ ] Tech spike: 1 dia só pra ver se dá pra parsear SCR3040_Leiaute.xls em Go
- [ ] **Decisão Fundador/Radiant:** "vamos?"

---

**Esta é a base completa do produto Radiant Sentinel. Próximo passo é validação — sem cliente pagante, é só tese bonita.**

---

## 12. IDENTIDADE DA MARCA — Radiant Sentinel

### Estrutura de marca

```
Fortvna Risk Solutions              ← holding/controladora
        │
        └── Radiant Risk Solutions  ← marca umbrella de produtos de risco
                │
                ├── Radiant Harness    ← já existe (CLI de Spec-Driven Development)
                │
                └── Radiant Sentinel   ← ESTE PRODUTO (CADOCs BACEN)
                        │
                        ├── Sentinel Lite    ← plano entry (R$ 1,5k/mês)
                        ├── Sentinel Pro     ← plano completo (R$ 4,5k/mês)
                        ├── Sentinel Scale   ← plano enterprise (R$ 12k/mês)
                        ├── Sentinel Enterprise ← sob consulta
                        │
                        ├── Sentinel ESG       ← módulo DRSAC (first-mover)
                        ├── Sentinel Radar     ← radar regulatório
                        ├── Sentinel Connect   ← API pública + webhooks
                        ├── Sentinel Studio    ← workspace low-code
                        └── Sentinel Audit     ← log + auditoria tamper-evident
```

### Taglines oficiais

| Contexto | Tagline |
|---|---|
| **Principal** | *"Radiant Sentinel — sentinela regulatória pra IF brasileira"* |
| **Curta** | *"Sob a égide da Radiant, sua IF na norma"* |
| **Emocional** | *"Quem nunca dorme no seu compliance"* |
| **Ataque** | *"De R$ 80k/ano Matera pra R$ 18k/ano Radiant Sentinel"* |
| **Vetor ESG** | *"Primeira plataforma brasileira com DRSAC 2030 de fábrica"* |
| **Pitch 30s** | *"A corrida regulatória BACEN 2024-2027 virou IFs refém de geradores de XML. Radiant Sentinel é a esteira regulatória que abstrai leiaute, dispara alertas, cruza CADOCs e entrega SOC 2. Por uma fração do que a consultoria cobra."* |

### Domínios a registrar (estratégia anti-conflito)

| Domínio | Custo estimado | Status |
|---|---|---|
| `radiant-sentinel.com.br` | R$ 50/ano | **REGISTRAR AGORA** |
| `radiantsentinel.com.br` | R$ 50/ano | **REGISTRAR AGORA** |
| `sentinel.radiant.com.br` | depende | reservado |
| `radiant.com.br/sentinel` | subdomínio | reservado |
| `radiant-sentinel.io` | US$ 30/ano | opcional |
| `radiantsentinel.com` | US$ 12/ano | opcional |
| **INPI — marca "Radiant Sentinel"** | R$ 355 + taxa | **REGISTRAR EM MÊS 1** |

### Nome em diferentes contextos

| Contexto | Como aparece |
|---|---|
| Email | *"Time do Radiant Sentinel preparou seu relatório"* |
| Slack interno IF | *"Já subiu pro Sentinel?"* |
| LP headline | *"Radiant Sentinel — sentinela regulatória pra IF"* |
| Deck slide | *"Hoje apresentamos o Radiant Sentinel, da Radiant Risk Solutions"* |
| Contrato | *"Radiant Risk Solutions LTDA — produto Radiant Sentinel"* |
| App icon | S (de Sentinel) estilizado + olho/escudo |
| API | `api.radiant-sentinel.com.br/v1/...` |
| Planos | Sentinel Lite, Pro, Scale, Enterprise |
| Módulos | Sentinel ESG, Radar, Connect, Studio, Audit |

---

**Esta é a base completa do produto Radiant Sentinel. Próximo passo é validação — sem cliente pagante, é só tese bonita.**
