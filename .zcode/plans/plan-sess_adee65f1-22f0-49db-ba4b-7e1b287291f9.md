# Plano: Redesign Premium "Radiant Norma Console"

## Direção estética (merge das 3 referências)

Não vou seguir uma direção única. Vou **fundir as três** numa linguagem coesa, chamada internamente de **"Institutional Premium Noir"**:

| Camada | Influência | Decisão concreta |
|---|---|---|
| **Paleta base** | Editorial Luxo Sério (off-white quente) + Noir Terminal (preto profundo) | **Dual-surface system**: light usa `#F7F5F0` (off-white warm) com tinta `#0F0E0C` (ink coal), dark usa `#0A0A0B` (noir) com tinta `#F2EFE8`. Ambos com sutileza de warmth para fugir do cinza genérico de SaaS. |
| **Accent** | Quiet Authority Tech (violet) + Noir (violet glow) | **Violet→Magenta gradient accent**: `#7C3AED` → `#D946EF`. Usado com parcimônia (1 elemento por viewport carrega o gradient; o resto fica em monochrome). |
| **Tipografia** | Editorial (serifa nos títulos) + Terminal (mono nos dados) + Authority (sans neutro no corpo) | **Trio tipográfico**: `Inter` (corpo), `Fraunces` (display/títulos, serifa moderna com bom tracking negativo), `JetBrains Mono` (dados: CADOCs, hashes, IDs, valores numéricos críticos). Carregadas via `next/font/google`. |
| **Hierarquia** | Editorial Luxo + Authority Tech | **Density variável**: páginas analíticas (Dashboard/Insights) em densidade alta com tabelas/grid finas; páginas transacionais (Login/Empty states) com whitespace generoso e tipografia grande. |
| **Motion** | Authority Tech | **Micro-animações medidas**: 180ms ease-out em hover/focus, fade-up 240ms em mount, 400ms em transições de página. Sem bouncy/spring — só cubic-bezier(0.16, 1, 0.3, 1). |
| **Sombras** | Noir Terminal (glow) + Authority Tech (quase invisíveis) | **Sistema dual**: shadow-xs/sm são charcoal-ink 5% (sutil); glow-accent/critical/success são violet/rose/emerald com 18% opacity. Glows só aparecem em elementos primários interativos ou alertas críticos. |
| **Dados** | Terminal (mono obsessivo) + Authority (tabular nums) | Todos os números com `font-variant-numeric: tabular-nums` + JetBrains Mono em valores críticos. KPI cards têm o número com weight 500 (não bold) para elegância editorial. |

---

## Decisões de produto que reforçam "high ticket premium"

1. **Logo + identidade**: troco o ícone SVG de "diamante geométrico" (genérico) por uma **marca tipográfica "RN"** com hairline serif e accent square — algo entre Stripe e Bloomberg. Logo horizontal: "Radiant Norma" em Fraunces + wordmark "Console" em Inter 2xs uppercase.
2. **Command palette (⌘K)** já existe — vou refinar visualmente (glass background, agrupamento com label serif).
3. **Live badges** ficam mais contidos (pill minimalista com dot animado) — não "tapeçaria verde".
4. **Empty states** ganham ilustração tipográfica em vez de ícones sozinhos (símbolo grande Fraunces + caption).

---

## Arquivos a alterar (28 arquivos)

### Foundation (5)
- `tailwind.config.ts` — paleta warm-noir dual, fontes, sombras glow refinadas, novos keyframes
- `src/app/globals.css` — CSS variables warm-noir, gradient utilities, scrollbar premium, focus rings
- `src/app/layout.tsx` — carregar Fraunces + JetBrains Mono via `next/font/google`
- `src/components/theme-provider.tsx` — ajustes mínimos (já funcional)
- `next.config.js` — sem mudanças

### UI Primitives (6) — refresh visual mantendo API
- `src/components/ui/card.tsx` — novo variant `glass` (frosted), shadow system revisado
- `src/components/ui/button.tsx` — gradient accent no primary, micro-press effect
- `src/components/ui/badge.tsx` — uppercase tracking mais elegante, glyph treatment
- `src/components/ui/empty-state.tsx` — versão tipográfica com serifa
- `src/components/ui/tooltip.tsx` — glass variant
- `src/components/ui/kbd.tsx` — proporção ajustada

### Layout (3) — reescrita
- `src/components/layout/app-shell.tsx` — composição nova
- `src/components/layout/sidebar.tsx` — vertical nav com hairline border, accent rail no item ativo, logo "RN" + Fraunces
- `src/components/layout/topbar.tsx` — refinamento glass header, search bar central proeminente
- `src/components/layout/command-palette.tsx` — visual glass com agrupamento editorial

### Domain Components (6) — refresh visual
- `src/components/domain/stat-card.tsx` — KPI card editorial: número mono, delta inline com pill, sparkline mais refinado
- `src/components/domain/alert-card.tsx` — versão "press release": severity como hairline accent à esquerda, typography hierarchy revista
- `src/components/domain/activity-feed.tsx` — timeline editorial com hairline + glyphs serif em datas
- `src/components/domain/heatmap.tsx` — escala de cor mais sofisticada (gradient violet→magenta→rose)
- `src/components/domain/insight-card.tsx` — editorial card com pull-quote treatment
- `src/components/domain/realtime-badge.tsx` — versão minimalista

### Pages (8) — reescrita
- `src/app/login/page.tsx` — capa editorial split-panel com hero copy editorial, wordmark Fraunces, decorative grain texture
- `src/app/page.tsx` — Dashboard com hero editorial + KPI grid + sections
- `src/app/radar/page.tsx` — lista com filtros pills, severity legend editorial
- `src/app/envios/page.tsx` — tabela densa estilo Bloomberg, filtros refinados
- `src/app/regras/regras-client.tsx` — catálogo editorial: cards maiores, typography revista, modal com hairline header
- `src/app/insights/page.tsx` — heatmap proeminente, recommendation cards editorial
- `src/app/auditoria/page.tsx` — timeline style, hash chain visual
- `src/app/auditoria/filter-bar.tsx` — visual polish

### Novos componentes utilitários (2)
- `src/components/ui/divider.tsx` — hairline divider com label opcional (estilo editorial)
- `src/components/ui/section-header.tsx` — header de seção editorial (eyebrow + título + descrição)

---

## Critérios de "premium" que vou validar

1. **Visual restraint**: zero gradientes gritantes; violet accent só onde carrega hierarquia (CTA primário, número-chave, active state)
2. **Tipografia**: 3 famílias convivem sem brigar; hierarquia clara por peso + tracking, não só tamanho
3. **Densidade calibrada**: 32px de padding em seções executivas, 24px em operacionais, 16px em transacionais
4. **Motion**: transições em 180-240ms com `cubic-bezier(0.16, 1, 0.3, 1)`. Nada de bounce.
5. **Dark mode**: não é apenas "light invertido" — charcoal `#0A0A0B` com accent violet glow dá "trading terminal" feel
6. **Acessibilidade preservada**: contraste WCAG AA em ambos os modos, focus rings mantidos, badges com dot+texto

---

## Validação final

1. `npm run build` no diretório `frontend/` — verificar zero erros TS
2. Inspeção visual manual de cada página em ambos os temas
3. Conferir que API calls e lógica de dados **não foram alteradas** — só visual

---

## O que NÃO vou fazer (escopo consciente)

- Não vou trocar a arquitetura de pastas
- Não vou adicionar libs (recharts continua instalado mas não usado; não vou importá-lo agora para evitar escopo)
- Não vou criar páginas novas
- Não vou mexer em backend
- Não vou quebrar a API do `cn()`, `Card`, `Button`, `Badge` — refactor é interno

Tempo estimado: 35-50min de execução focada.