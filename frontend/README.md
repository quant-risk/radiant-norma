# Radiant Norma — Console

> Frontend Next.js 14 (App Router) + TailwindCSS 3 + TanStack Query
> dashboard para IFs (Instituições Financeiras) consumirem o backend
> Radiant Norma (Go).

**Sprint:** 7c / v2.0.0
**Status:** MVP (frontend skeleton completo, 6 páginas funcionais)

## Stack

- **Framework:** Next.js 14.2 (App Router)
- **Linguagem:** TypeScript 5.6
- **Styling:** TailwindCSS 3.4 (utility-first) + tailwind-merge
- **State:** TanStack Query 5 (server state) + Zustand 5 (client state)
- **HTTP:** Axios 1.7 (client) + fetch nativo (server components)
- **Auth:** JWT bearer via cookie httpOnly (rsa+jose para decode)
- **OpenAPI:** YAML em `../backend/docs/api/openapi.yaml` (Sprint 7c)

## Setup

```bash
npm install

# Dev (assumindo backend Go rodando em :8080):
export RADIANT_API_URL=http://localhost:8080
export NEXT_PUBLIC_RADIANT_API_JWT_PUBKEY="<RSA PEM>"
export NEXT_PUBLIC_RADIANT_DEV_MODE=1  # X-IF-ID fallback
npm run dev

# Acesso: http://localhost:3000
```

## Estrutura

```
frontend/
├── src/
│   ├── app/                       # Next.js 14 App Router
│   │   ├── layout.tsx             # Root com ReactQueryProvider
│   │   ├── globals.css            # Tailwind + custom utilities
│   │   ├── page.tsx               # / — dashboard
│   │   ├── login/page.tsx         # /login — dev picker
│   │   ├── envios/page.tsx        # /envios — histórico STA
│   │   ├── radar/page.tsx         # /radar — alertas ativas
│   │   ├── regras/page.tsx        # /regras — catálogo de regras
│   │   ├── auditoria/page.tsx     # /auditoria — LGPD/SOC2 view
│   │   ├── api/login/route.ts     # /api/login POST (cookie setter)
│   │   └── v1-api/proxy/[...]/    # JWT-injecting proxy for backend
│   ├── lib/
│   │   ├── api.ts                 # Axios client (client components)
│   │   ├── api-fetch.ts           # Server-side fetch wrapper
│   │   ├── auth.ts                # JWT verify (server) + Zustand (client)
│   │   ├── session.ts             # cookies() → Session
│   │   ├── cookies.ts             # browser cookie helpers
│   │   └── utils.ts                # cn() className helper
│   └── components/
│       └── resolve-alert-button.tsx  # Client component for resolve
├── package.json
├── tsconfig.json
├── next.config.js
├── tailwind.config.ts
├── postcss.config.js
├── next-env.d.ts
└── README.md
```

## Rotas implementadas

| Path | Status | Description |
|------|--------|-------------|
| `/login` | ✅ | Demo picker (3 IFs + role admin) |
| `/` | ✅ | Dashboard com stats agregadas |
| `/radar` | ✅ | Lista alertas ativos + resolve |
| `/regras` | ✅ | Catálogo 60 regras (parse de `../docs/rules-3040-catalog.md`) |
| `/envios` | ⚠️ Sprint 8 | Placeholder — backend endpoint needed |
| `/auditoria` | ⚠️ Sprint 8 | Placeholder — `/v1/audit_log` endpoint needed |
| `/api/login` | ✅ | POST login → cookie httpOnly |
| `/v1-api/proxy/[...]` | ✅ | Server-side proxy com JWT injection |

## Dependências Backend

- `RADIANT_API_URL` — base URL do backend Go (default `http://localhost:8080`)
- `NEXT_PUBLIC_RADIANT_API_JWT_PUBKEY` — chave pública RSA em formato PEM
- `NEXT_PUBLIC_RADIANT_API_JWT_ISSUER` — issuer pin (default `radiant-norma`)
- `NEXT_PUBLIC_RADIANT_DEV_MODE` — se `1`, aceita X-IF-ID via backend dev mode

## Próximo — Sprint 8

1. Endpoint `GET /v1/envios?if_id=X` no backend (filtros, paginação)
2. Endpoint `GET /v1/audit_log` (admin only)
3. PWA / offline support (v2.1)
4. Real auth integration (IdP via OIDC, Sprint 9)
5. Audit log dashboard com timeline visual

## Validation Coverage (Sprint 7c)

- OpenAPI 3.0 spec em `../backend/docs/api/openapi.yaml` (Sprint 7c)
- Frontend usa `apiFetch` server-side e `api` (axios) client-side
- JWT validation server-side via jose
- Cookie httpOnly para JWT (defesa contra XSS)
- Proxy `/v1-api/[...]` injeta JWT antes de forward (CSRF-safe)
- 6 páginas funcionais (dashboard, radar, regras, envios, auditoria, login)

## Test setup

Frontend MVP: testes manuais via `/login`. Testes automatizados (Vitest/Jest) são Sprint 8+.
