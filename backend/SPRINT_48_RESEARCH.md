# Sprint 48 — RESEARCH.md

## Pilot2 — Segundo Cliente (IP Médio)

**Sprint:** 48
**Tema:** Pilot2 — Onboarding de segundo cliente IP médio no Radiant Norma
**Período:** 2026-07-07
**Versão alvo:** v3.34.29

---

## 1. Contexto

**Saída da Sprint 48 (conforme ROADMAP):** "Radiant Norma Pro" vendável pra IP média.

O primeiro piloto (Sprint 37) provou que o produto funciona para SCD pequena (S5).
O segundo piloto objetiva demonstrar que o Radiant Norma escala para IPs de segmento médio (S3), que têm:
- Volume maior de operações (SCR mais denso)
- Maior variedade de produtos (crédito, TVM, derivativos)
- Necessidade de dashboard executivo com KPIs agregados
- Integração com sistemas internos (ERP, compliance)

---

## 2. Escopo do Pilot2

### 2.1 Onboarding de Tenant

Criar infraestrutura de lifecycle de tenant (`internal/tenant`):

| Método | Descrição |
|---|---|
| `Create(input)` | Cria tenant com CNPJ, tipo, segmento, plano |
| `Get(id)` | Busca por ID |
| `GetByCNPJ(cnpj)` | Busca por CNPJ |
| `List(segmento)` | Lista tenants (com filtro) |
| `Deactivate(id)` | Soft-delete |
| `UpdatePlano(id, plano)` | Upgrade/downgrade de plano |

### 2.2 Validação de Segmento

Segmentos BACEN (IN_BCB 151/2021):

| Segmento | Definição | Exemplo |
|---|---|---|
| S1 | IF com exposição >= R$ 75 bi | Bancos grandes |
| S2 | IF com exposição >= R$ 25 bi e < R$ 75 bi | Bancos médios |
| S3 | IF com exposição >= R$ 2.5 bi e < R$ 25 bi | Bancos menores, IPs grandes |
| S4 | IF com exposição >= R$ 250 MM e < R$ 2.5 bi | IPs médios |
| S5 | IF com exposição < R$ 250 MM | IPs pequenos, SCD |

**Pilot2 foca em S3-S4** (IPs médios com volume de operações significativo).

### 2.3 Plano Pro para IP Médio

Plano "pro" para S3-S4 inclui:
- Validação completa (3040, 3050, DLO, DDR, DRL, DLP, 2030)
- Dashboard executivo com KPIs
- Radar de mudanças regulatórias
- Cross-Doc L3 (3040 ↔ 4111 ↔ DRSAC)
- 5 usuários
- Suporte via e-mail

---

## 3. O que existe

### 3.1 Estrutura de Tenant

Tabela `ifs` já existe desde Sprint 3 (P0.2):
```sql
CREATE TABLE IF NOT EXISTS ifs (
    id              TEXT PRIMARY KEY,  -- UUID
    cnpj            TEXT NOT NULL UNIQUE,
    nome            TEXT NOT NULL,
    tipo            TEXT NOT NULL,    -- SCD, IP, SEP, BC
    segmento        TEXT,              -- S1-S5
    plano           TEXT DEFAULT 'lite',
    sta_user        TEXT,
    sta_service     TEXT DEFAULT 'PSTA300',
    cert_a1_path    TEXT,
    created_at      DATETIME,
    updated_at      DATETIME,
    deleted_at      DATETIME           -- soft-delete
);
```

### 3.2 Seed de Dados

- `cmd/seed/main.go` — seed genérico de críticas e leiautes
- `cmd/seed-sprint8c/main.go` — seed de envios, audit events e rule_failures

### 3.3 Branding

Sprint 46 (WhiteLabel) adicionou branding por tenant:
- Logo customizável
- Cores primária/secundária
- Domínio customizado
- Tenant slug

### 3.4 Billing

Sprint 45 (StripeBilling) adicionou:
- Criação de Customer + Subscription via Stripe
- Portal de billing self-service
- Webhooks para lifecycle de subscription

---

## 4. Plano de Implementação

### 4.1 Fase 1: Tenant Service (`internal/tenant/tenant.go`)

**Entrega:** Service para lifecycle de tenant com CRUD completo.

**Funcionalidades:**
- Validação de CNPJ (8 dígitos)
- Validação de tipo (SCD, IP, SEP, BC, SCD_S3, IP_S3)
- Validação de segmento (S1-S5)
- Validação de plano (lite, pro, scale, enterprise)
- Soft-delete
- List com filtro por segmento

### 4.2 Fase 2: Seed de Dados para Pilot

**Entrega:** `cmd/seed-pilot2` que popula dados realistas para IP médio.

**Dados:**
- 1 tenant IP S3 com plano "pro"
- 50 envios 3040 ao longo de 12 meses
- 20 envios 3050 (tesouraria)
- 10 envios DLO/DDR/DRL (novos CADOCs)
- Distribuição realista de rule failures
- Audit events de atividade normal

### 4.3 Fase 3: Dashboard KPIs para S3-S4

**Entrega:** Métricas agregadas por segmento no endpoint `/v1/insights/kpis`.

**KPIs para IP médio:**
- Taxa de aprovação (accepted/total)
- Top 5 regras com mais failures
- Volume de operações por CADOC
- Evolução temporal de falhas

---

## 5. Dependências

| Sprint | Dependência |
|---|---|
| Sprint 45 (StripeBilling) | Billing service para subscription |
| Sprint 46 (WhiteLabel) | Branding service |
| Sprint 47 (DRSACResearch) | Parser DRSAC para validação 2030 |

---

## 6. Riscos

| Risco | Mitigação |
|---|---|
| IP médio tem mais CADOCs que SCD | Seed com volume realista, não exagerado |
| Tempo de onboarding do cliente | CLI de seed facilita setup |
| Segmento S3 requer mais validações | Priorizar 3040/3050/DLO/DDR; DRSAC como bônus |
