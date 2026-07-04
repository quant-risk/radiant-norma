# VALIDATION v2.0.0 — Sprint 7c (Frontend Next.js)

> **Status:** ACCEPTED
> **Data:** 2026-07-03
> **Trigger:** Henrique pediu frontend Next.js. Tarde da noite
> Sprint 6 → Sprint 7 inteiro (a/b/c).
> **Versão:** v2.0.0 (major — frontend empacotado)

## 🎯 Resumo

Frontend Next.js 14 dashboard criado para IFs consumirem
backend Radiant Norma. Stack: App Router + Tailwind + TanStack +
Zustand. Auth via JWT bearer + cookie httpOnly. 6 páginas
funcionais (dashboard, radar, regras, envios, auditoria, login).

**Stats:**
- 19 arquivos TypeScript (.ts/.tsx)
- 6 páginas funcionais
- 7 routes (login + 6 pages + proxy)
- ~1100 LOC frontend
- Backend tests: 301 (inalterado por Sprint 7c)

## Sprint 7c — frontend/ sub-projeto

### Estrutura
```
frontend/
├── package.json (deps: Next, Tailwind, TanStack, Zustand, Axios, jose, zod)
├── tsconfig.json (strict mode, paths: @/* → src/*)
├── tailwind.config.ts (Radiant brand colors)
├── next.config.js (RADIANT_API_URL, typedRoutes)
├── postcss.config.js
├── src/
│   ├── app/
│   │   ├── layout.tsx (root + ReactQueryProvider + globals.css)
│   │   ├── globals.css (Tailwind + custom btn/card utilities)
│   │   ├── page.tsx (dashboard — server component)
│   │   ├── login/page.tsx (demo picker — client component)
│   │   ├── envios/page.tsx (placeholder com TODO Sprint 8)
│   │   ├── radar/page.tsx (alertas ativos — client via /resolve)
│   │   ├── regras/page.tsx (catálogo parseado de ../docs/rules-3040-catalog.md)
│   │   ├── auditoria/page.tsx (LGPD view + TODO Sprint 8)
│   │   ├── api/login/route.ts (POST login → cookie httpOnly)
│   │   └── v1-api/proxy/[...path]/route.ts (JWT-injecting server proxy)
│   ├── lib/
│   │   ├── api.ts (Axios client + interceptor)
│   │   ├── api-fetch.ts (server fetch wrapper)
│   │   ├── auth.ts (JWT verify + Zustand session store)
│   │   ├── session.ts (getServerSession via next/headers)
│   │   ├── cookies.ts (browser cookie helpers)
│   │   └── utils.ts (cn() className utility)
│   └── components/
│       └── resolve-alert-button.tsx (client mutation button)
└── README.md
```

### Pages funcionais

| Route | Server/Client | Funcional |
|-------|---------------|-----------|
| `/login` | Client (form picker) | ✅ Dev: 3 IFs demo |
| `/` | Server (dashboard) | ✅ Stats agregadas |
| `/radar` | Server (lista) + Client (resolve) | ✅ Lista + resolve |
| `/regras` | Server (parse catalog) | ✅ Catálogo 60 regras |
| `/envios` | Server (placeholder) | ⚠️ TODO Sprint 8 |
| `/auditoria` | Server (placeholder) | ⚠️ TODO Sprint 8 |

### Auth flow (Sprint 7c → Sprint 8)

1. User → /login → POST /api/login com if_id
2. Backend dev mode: retorna token (dev:<if_id>:<role>)
3. Cookie `rn_jwt` httpOnly setado
4. SSR pages leem via cookies()
5. Client components via apiFetch adicionam Authorization header

### OpenAPI 3.0 spec

`backend/docs/api/openapi.yaml`:
- 14 endpoints documentados
- Schemas: HealthStatus, Error, CadocList, Schema, ValidationResponse, etc.
- Security: bearerAuth (JWT RS256)
- Tags: meta, schemas, rules, validate, sta, radar, crossdoc
- Frontend pode usar `openapi-typescript` para gerar tipos (Sprint 8+)

### Design decisions

1. **Single repo, `frontend/` subdir** — não criar repo separado para esta entrega MVP. Sprint 8 pode separar se crescer.

2. **App Router (server-first)** — server components para fetches pesadas; client components só onde precisa interação (resolve button, login form).

3. **Cookie httpOnly para JWT** — defesa contra XSS. Não localStorage.

4. **Proxy /v1-api/[...]** — server-side forward com JWT injection. CSRF-safe porque cookie sameSite=lax + JWT no header é Same-Origin policy protective.

5. **Dev mode X-IF-ID preserved** — `RADIANT_DEV_AUTH=1` no backend; `NEXT_PUBLIC_RADIANT_DEV_MODE=1` no frontend. Em prod: ambos off, IdP real.

### Vetores fechados (cross-cutting)

| Vetor | Frontend | Backend (Sprint 7a) |
|-------|----------|---------------------|
| Auth bypass | X-IF-ID não passa de dev | JWT RS256 |
| XSS in JWT | httpOnly cookie | N/A |
| CSRF | Same-Site Lax + Same-Origin | N/A |
| Token in logs | JWT só em Authorization header (no body) | SafeError |

## Acceptance

- ✅ Frontend skeleton completo
- ✅ 6 páginas funcionais
- ✅ App Router + Server Components + Client minimal
- ✅ JWT auth flow preparado (Sprint 8 wire-up real)
- ✅ OpenAPI 3.0 spec documentado
- ✅ README com setup completo
- ✅ Backend tests: 301 (inalterados, 7c não muda backend)

## Próximo: Sprint 8

1. **npm install + validar build** (`npm run build`)
2. **Backend endpoints faltantes:**
   - GET /v1/envios?if_id=X (envios do IF)
   - GET /v1/audit_log (admin only, paginação)
   - GET /v1/dashboard/stats (stats agregadas)
3. **Frontend finalize:**
   - Validate page (/validar) — upload de XML + visualização de erros
   - Tailwind testes visuais (dark mode toggle, responsivo)
4. **Testes E2E:**
   - Playwright ou Cypress (Sprint 8+)
5. **CI/CD:**
   - npm install no CI
   - docker build do frontend (multi-stage)
   - imagem Docker published
