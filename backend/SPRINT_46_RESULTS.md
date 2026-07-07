# Sprint 46 — RESULTS.md

## WhiteLabel — Branding por Tenant

**Sprint:** 46
**Tema:** WhiteLabel — Branding por Tenant
**Período:** 2026-07-07
**Versão:** v3.34.27
**Status:** ✅ Shipped

---

## 1. Objetivos

Permitir que tenants BaaS que revendem o Radiant Norma com sua própria marca personalizem: logo, cores primárias/secundárias e domínio customizado.

---

## 2. Entregas

### 2.1 Migration (`016_white_label.sql`)

```sql
ALTER TABLE ifs ADD COLUMN logo_url TEXT;
ALTER TABLE ifs ADD COLUMN primary_color TEXT DEFAULT '#3b6ef5';
ALTER TABLE ifs ADD COLUMN secondary_color TEXT DEFAULT '#1a2a5e';
ALTER TABLE ifs ADD COLUMN custom_domain TEXT;
ALTER TABLE ifs ADD COLUMN tenant_slug TEXT;

CREATE UNIQUE INDEX idx_ifs_tenant_slug ON ifs(tenant_slug) WHERE tenant_slug IS NOT NULL;
CREATE INDEX idx_ifs_custom_domain ON ifs(custom_domain) WHERE custom_domain IS NOT NULL;
```

### 2.2 BrandingService (`internal/branding/branding.go`)

| Método | Descrição |
|---|---|
| `GetBranding(tenantID)` | Retorna branding com defaults aplicados |
| `GetBrandingBySlug(slug)` | Lookup público por tenant_slug |
| `UpdateBranding(tenantID, req)` | Update parcial (só campos não-nulos) |
| `slugExistsOtherTenant` | Validação de unicidade de slug entre tenants |

### 2.3 Validadores

| Validador | Regra |
|---|---|
| Hex color | `#RRGGBB` case-insensitive, regex `^#[0-9A-Fa-f]{6}$` |
| Logo URL | `^https?://` — vazio é permitido (opcional) |
| Tenant slug | `^[a-z0-9][a-z0-9-]*[a-z0-9]$` ou `^[a-z0-9]$`, 2-63 chars, único entre tenants |

### 2.4 API Routes (server.go)

| Método | Path | Auth | Descrição |
|---|---|---|---|
| GET | `/v1/tenant/branding` | JWT | Branding do tenant autenticado |
| PUT | `/v1/tenant/branding` | JWT | Atualiza branding (parcial) |
| GET | `/v1/tenant/branding/public/{slug}` | None | Branding público por slug |
| PUT | `/v1/admin/tenant/{id}/branding` | JWT + admin role | Admin atualiza qualquer tenant |

### 2.5 Testes

17 testes unitários cobrindo:
- Get/Update para todos os campos
- Defaults (primary: #3b6ef5, secondary: #1a2a5e)
- Validação hex color (inválido, minúsculo, uppercase)
- Validação logo URL (inválido, vazio OK)
- Validação slug (tamanho, caracteres, underscore, leading/trailing hyphen, espaço)
- Uniqueness de slug entre tenants
- Update parcial (campos não-enviados permanecem)
- Tenant not found / empty

---

## 3. Não escopo (para futuro)

- Upload de logo via API (URL externa por enquanto)
- SSL/wildcard cert provisioning para custom domain
- WhiteLabel no frontend (SSO custom domain redirect)
- Branding preview endpoint

---

## 4. Dependências

Nenhuma nova dependência Go.

---

## 5. Riscos e Mitigações

| Risco | Mitigação |
|---|---|
| Slug uniqueness race condition | Partial unique index (WHERE slug IS NOT NULL) + check app-level |
| Hex color mal formado salvo | Validação via regex antes do UPDATE |
| Custom domain sem validação | Campo livre (CNAME é responsibility do tenant) |

---

## 6. Tempo

Planejado: 1 dia
Realizado: 1 dia
