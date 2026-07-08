# Radiant Norma · Design System v3.35

> **Linguagem visual "Institutional Premium Noir"** — fusão de Editorial Luxo Sério (Bloomberg/Stratos), Quiet Authority Tech (Linear/Vercel/Stripe) e Noir Banking Terminal (Mercury/Revolut dark).

---

## 1. Princípios

1. **Restraint é sofisticação.** Violet→magenta gradient aparece em no máximo 1 elemento proeminente por viewport. O resto é monochrome warm-neutral.
2. **Tipografia tripla convive sem brigar.** Inter (corpo), Fraunces (display/editorial), JetBrains Mono (dados críticos). Hierarquia por peso + tracking, não só tamanho.
3. **Densidade calibrada por contexto.** 32px em seções executivas, 24px operacionais, 16px transacionais.
4. **Motion medido.** 180-240ms com `cubic-bezier(0.16, 1, 0.3, 1)`. Sem bounce.
5. **Dark mode não é light invertido.** Noir `#0A0A0B` com accent violet glow dá "trading terminal" feel.

---

## 2. Tokens

### Cores (via CSS variables RGB em `globals.css`)

```css
/* Light mode — warm off-white */
--surface: 247 245 240;          /* warm off-white */
--surface-raised: 252 250 245;   /* warmer paper */
--surface-sunken: 240 237 230;   /* sunk warm */
--ink: 15 14 12;                 /* coal-900 */
--ink-muted: 82 75 65;           /* warm slate-600 */
--ink-subtle: 140 131 117;       /* warm slate-400 */
--accent: 124 58 237;            /* violet-600 */

/* Dark mode — deep noir */
--surface: 10 10 11;             /* noir-950 */
--surface-raised: 18 17 19;      /* noir-900 */
--ink: 242 239 232;              /* warm paper */
--accent: 167 139 250;           /* violet-400 */
```

### Escala tipográfica (`tailwind.config.ts`)

| Token | px/line-height | Uso |
|---|---|---|
| `text-2xs` | 11/16 | Eyebrows, micro-meta, tracking `0.14em` |
| `text-xs` | 12/18 | Badges, captions |
| `text-sm` | 13/20 | UI padrão |
| `text-base` | 14/22 | Corpo |
| `text-md` | 15/24 | Subtítulos |
| `text-lg` | 16/24 | Títulos de card |
| `text-xl` | 18/28 | Títulos de seção |
| `text-2xl` | 22/30 | Títulos de página |
| `text-3xl` | 28/34 | H2 editorial |
| `text-4xl` | 36/40 | H1 editorial |
| `text-5xl` | 48/52 | Hero (login) |
| `display-md` | 56/58 | Hero premium |
| `display-lg` | 72/74 | Manifesto |

### Escala de spacing

4 / 8 / 12 / 16 / 24 / 32 / 48 / 64 px. Padrão: 24px gap entre seções, 16px entre componentes, 8px entre itens irmãos.

### Sombras (`tailwind.config.ts`)

| Token | Uso |
|---|---|
| `shadow-xs` | Cards default (charcoal 5%) |
| `shadow-sm` | Cards raised (charcoal 6%) |
| `shadow-md` | Hover state |
| `shadow-lg/xl` | Modais, dropdowns |
| `glow-accent` | CTAs primários (violet 18%) |
| `glow-critical` | Botões danger (rose 18%) |
| `glow-success` | Confirmações (emerald 18%) |
| `shadow-inner-soft` | Inputs |

### Motion (`tailwind.config.ts`)

| Token | Duração | Easing | Uso |
|---|---|---|---|
| `duration-180` | 180ms | `ease-out-expo` | Hover, focus, pequenas transições |
| `duration-240` | 240ms | `ease-out-expo` | Cards, botões |
| `duration-320` | 320ms | `ease-out-expo` | Mount de página (`fade-up`) |
| `duration-400` | 400ms | `ease-out-expo` | Modais, drawers |
| `animate-fade-in` | 240ms | — | Mount de listas |
| `animate-scale-in` | 180ms | — | Modais |
| `animate-pulse-soft` | 2.4s | — | Dots live |
| `animate-pulse-ring` | 1.8s | — | Alertas críticos |
| `animate-gradient-pan` | 8s | — | Hero glow |

---

## 3. Tipografia em uso

| Função | Família | Quando |
|---|---|---|
| Display | **Fraunces** (serif) | H1/H2 de páginas, títulos de card, wordmark |
| UI / corpo | **Inter** (sans) | Parágrafos, labels, navegação |
| Dados | **JetBrains Mono** | CADOCs, IDs, hashes, valores numéricos, eyebrows |

Convenção: tabs de "uppercase + tracking" usam Inter, mas eyebrows premium em Fraunces italic para ênfase (ex: "pensa junto" no hero do login).

---

## 4. Componentes

### Card

```tsx
<Card variant="default" padding="md" interactive>
  <CardEyebrow>Eyebrow mono</CardEyebrow>
  <CardTitle>Título serif</CardTitle>
  <CardDescription>Descrição muted</CardDescription>
</Card>
```

Variants: `default` (uso geral), `raised` (destaque), `ghost` (transparente), `outlined` (ênfase), `glass` (frosted).

### Button

```tsx
<Button variant="primary" size="md" leftIcon={...} rightIcon={...}>CTA</Button>

// Com Link sem HTML inválido:
<Link href="/x" passHref legacyBehavior={false}>
  <Button asChild variant="primary">Ir para x</Button>
</Link>
```

Polimorfismo `asChild`: clona o único filho e injeta classe/ref/handlers — preserva semântica do elemento (link continua sendo `<a>`).

### Badge

```tsx
<Badge tone="accent" variant="soft" dot size="sm">Live</Badge>
```

Sempre par dot+texto ou icon+texto em badges críticos (WCAG 1.4.1).

### SectionHeader

```tsx
<SectionHeader
  eyebrow="Eyebrow mono"
  title="Título serif"
  description="Descrição muted"
  actions={<Button>CTA</Button>}
/>
```

Padrão editorial: eyebrow → título display → descrição → CTA opcional à direita.

### Divider

```tsx
<Divider />             {/* hairline horizontal */}
<Divider label="OU" />  {/* com label centralizado */}
```

### StatCard (KPI)

```tsx
<StatCard
  label="Envios (30d)"
  value={1247}
  delta={{ value: 12.3, direction: 'up', period: 'vs 30d anteriores' }}
  sparkline={[...]}    // 4-12 pontos
  tone="accent"        // accent | success | warning | critical | neutral
  icon={<Send />}
/>
```

Sparkline com gradient violet→magenta quando `tone="accent"`. SSR-safe via `React.useId()`.

### AlertCard

```tsx
<AlertCard
  id={1}
  cadoc_code="3040"
  severity="critical"
  title="Nova versão do layout BACEN"
  description="..."
  source_url="..."
  detected_at="..."
/>
```

Severity rail (3px) à esquerda em gradient. Sem `border-l-*` hardcoded.

---

## 5. Layouts

### AppShell (root)

```
┌──────────┬──────────────────────────────────────┐
│ Sidebar  │ Topbar (sticky glass)                │
│ 256px    ├──────────────────────────────────────┤
│          │                                       │
│  logo    │       main content                   │
│  nav     │       max-width 1400                 │
│          │       p-10 lg                         │
│  role    │                                       │
└──────────┴──────────────────────────────────────┘
```

### Sidebar items

- Header: wordmark "R" gradient + "Radiant Norma" Fraunces + "Console" Inter 2xs
- Nav groups: "Operação", "Inteligência" (eyebrows mono)
- Active state: rail vertical 2px gradient accent + bg accent-50/30
- Badge live: dot pulsing + "live" (sutil)

### Topbar

- Glass background (frosted `bg-surface-raised/95 backdrop-blur-xl`)
- Esquerda: breadcrumb mono + título serif
- Centro: command palette trigger (search + ⌘K)
- Direita: actions + theme toggle + notifications + avatar

### Command palette (⌘K)

- Glass panel max-w-xl
- Grupos com eyebrow serif (Navegação / Tema / Regras / Alertas / CADOCs)
- Item ativo: rail gradient à esquerda + bg accent-50/30
- Footer: ↑↓ navegar · ↵ abrir · esc fechar + contador
- **Focus trap** ativado quando aberto; foco restaurado ao trigger quando fecha

---

## 6. Padrões de produto

### Severity em alertas

Sempre 2 sinais além da cor (WCAG 1.4.1):
- Glyph específico (AlertTriangle / AlertCircle / Info)
- Badge dot+texto (Crítico / Atenção / Info)
- Hairline rail esquerda (gradient: critical=rose, warning=amber, info=sky)

### KPIs

Hierarquia: eyebrow → número (Fraunces 32px) → delta pill + sparkline → help text footnote. Nunca os 3 visuais juntos no mesmo card.

### Status vazio

Empty state editorial: símbolo Fraunces grande (ex: `∅`) OU ícone + título serif + descrição muted + CTA. Sempre com ação.

### Loading

Skeleton shimmer (animação 2.4s linear infinite) — nunca spinner solto.

---

## 7. Acessibilidade

- Focus rings visíveis em ambos os temas (`*:focus-visible`)
- Badges com dot+texto (cor não é único sinal)
- `prefers-reduced-motion`: animações/transições desativadas
- Focus trap em modais (CommandPalette, RuleDetailModal)
- Landmarks semânticos (`<header>`, `<nav>`, `<aside>`, `<main>`)
- `aria-current="page"` em nav ativo
- `aria-modal="true"` + `role="dialog"` em modais
- Botões acessíveis (avatar do topbar é `<button>`, não `<div>`)

---

## 8. CSP / Segurança

- `themeScript` inline recebe nonce gerado pelo middleware Edge runtime
- Middleware aplica CSP dinâmico `script-src 'self' 'nonce-${nonce}'` em prod
- Cookies httpOnly + SameSite=Lax + Secure em prod
- HSTS 1 ano com includeSubDomains; preload em prod
- `frame-ancestors 'none'` (anti clickjacking)
- `Permissions-Policy` nega mic/camera/geo/interest-cohort

---

## 9. Não-objetivos

- ❌ Gradientes gritantes em massa
- ❌ Animação bouncy/spring
- ❌ Sombras pretas saturadas (cara de 2015)
- ❌ Cores hardcoded fora do token system
- ❌ Componentes interativos aninhados (`<a><button>`, `<button><button>`)
- ❌ Modais sem focus trap
- ❌ Ícones sem par textual em comunicação crítica