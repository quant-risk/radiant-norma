# Sprint 46 — RESEARCH — WhiteLabel (Tema Customizável)

> **Data:** 2026-07-07
> **Sprint:** 46
> **Domínio:** WhiteLabel / Theming
> **Versão atual:** v3.34.26
> **Próxima:** v3.34.27

---

## 1. Contexto

WhiteLabel permite que Fintechs BaaS ofereçam o Radiant Norma com a própria marca (logo, cores, domínio) pros seus clientes finais.

**Cenário:** TecBank (Fintech BaaS) quer revender o Radiant Norma como "TecBank Compliance". Eles precisam:
1. Substituir o logo "Radiant Norma" pelo deles
2. Usar cores "tecbank-blue" em vez de "radiant-blue"
3. Usar o domínio `compliance.tecbank.com.br` em vez de `norma.fortvna.com.br`

### O que existe

Tabela `ifs` já tem `plano`, `nome`, `tipo` — mas não tem branding.

### O que vamos construir

Backend Sprint 46 (base para frontend):
1. Colunas de branding na tabela `ifs`
2. API de gerenciamento de branding
3. Endpoint para servir branding do tenant atual

---

## 2. Schema do DB

### Migration 016: branding columns

```sql
ALTER TABLE ifs ADD COLUMN logo_url TEXT;
ALTER TABLE ifs ADD COLUMN primary_color TEXT DEFAULT '#3b6ef5';   -- radiant-blue
ALTER TABLE ifs ADD COLUMN secondary_color TEXT DEFAULT '#1a2a5e'; -- radiant-dark
ALTER TABLE ifs ADD COLUMN custom_domain TEXT;
ALTER TABLE ifs ADD COLUMN tenant_slug TEXT;  -- "tecbank" para /whitelabel/tecbank/
```

---

## 3. Design de API

### Endpoints

| Método | Path | Descrição |
|---|---|---|
| `GET` | `/api/v1/tenant/branding` | Branding do tenant logado |
| `PUT` | `/api/v1/admin/tenant/:id/branding` | Atualiza branding (admin) |
| `GET` | `/api/v1/tenant/branding/public/:slug` | Branding público por slug (sem auth) |

### BrandingResponse

```go
type BrandingResponse struct {
    TenantID      string `json:"tenant_id"`
    TenantName    string `json:"tenant_name"`
    LogoURL       string `json:"logo_url"`
    PrimaryColor  string `json:"primary_color"`
    SecondaryColor string `json:"secondary_color"`
    CustomDomain  string `json:"custom_domain"`
    TenantSlug    string `json:"tenant_slug"`
}
```

### UpdateBrandingRequest

```go
type UpdateBrandingRequest struct {
    LogoURL        *string `json:"logo_url"`
    PrimaryColor  *string `json:"primary_color"`  // hex color
    SecondaryColor *string `json:"secondary_color"`
    CustomDomain  *string `json:"custom_domain"`
    TenantSlug    *string `json:"tenant_slug"`   // URL-safe slug
}
```

### Validações

- `LogoURL`: deve ser URL válida (http/https)
- `PrimaryColor` / `SecondaryColor`: deve ser hex color válido (`^#[0-9A-Fa-f]{6}$`)
- `TenantSlug`: deve ser `^[a-z0-9-]+$` (URL-safe), único

---

## 4. Componentes

```
backend/
  internal/
    branding/              (NOVO — package de branding)
      branding.go           — BrandingService + handlers
    db/
      migrations/
        016_white_label.sql  (NOVO — branding columns)
```

---

## 5. Critérios de aceitação

- [ ] Migration 016 adiciona colunas branding na tabela ifs
- [ ] `GET /api/v1/tenant/branding` retorna branding do tenant logado
- [ ] `PUT /api/v1/admin/tenant/:id/branding` atualiza branding (admin only)
- [ ] Validação de hex color (formato `#[0-9A-Fa-f]{6}`)
- [ ] Validação de URL (logo_url)
- [ ] Validação de slug único
- [ ] `go test ./...` 23/23 PASS
- [ ] `go vet ./...` clean
- [ ] `gofmt -l ./...` clean
- [ ] CHANGELOG entry v3.34.27

---

## 6. Segurança

- **Admin only**: `PUT /admin/tenant/:id/branding` requer `X-Admin-Token`
- **Tenant isolation**: `GET /tenant/branding` retorna só o branding do tenant logado
- **Slug uniqueness**: verificado via DB constraint
- **Color validation**: regex `^#[0-9A-Fa-f]{6}$` antes de persistir

---

## 7. Riscos

| Risco | Mitigação |
|---|---|
| CSS injection via cor | Regex estrito em hex color |
| XSS via logo_url | Validação de URL (somente http/https) |
| Slug collision | UNIQUE constraint no DB + erro 409 Conflict |
| custom_domain hijacking | Ownership verification (futuro: DNS challenge) |
