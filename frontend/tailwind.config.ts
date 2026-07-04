import type { Config } from 'tailwindcss'

/**
 * Radiant Norma — Design System Tokens (Tailwind extension).
 *
 * Filosofia: "institucional moderno" — confiança séria de uma ferramenta
 * regulatória (BACEN / CMN 4.966 / IFRS 9), com a estética minimalista
 * e a densidade calibrada de Linear / Vercel / Stripe.
 *
 * Convenção:
 *   - Surfaces neutros em slate (zinc-700 visualmente, mas slate-900 dark
 *     pra harmonizar com badges azul-violeta).
 *   - Accent primário: violet-600 (decisão consciente: NÃO usar sky/blue,
 *     que é o clichê "fintech trust" — violet comunica tecnologia +
 *     institucional simultaneamente).
 *   - Semantic colors: success=emerald, warning=amber, critical=rose,
 *     info=sky. Sempre em pares -500/-600 com -50/-950 pra backgrounds.
 *   - Radius: 8px padrão, 12px em cards, full em badges/pills.
 *   - Sombras em 3 níveis sutis (nunca pretas saturadas — vai contra
 *     "elevação flat" que produtos modernos usam).
 */

const config: Config = {
  darkMode: 'class',
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    container: {
      center: true,
      padding: '1.5rem',
    },
    extend: {
      colors: {
        // Brand accent (violet — tecnologia + institucional)
        accent: {
          50: '#f5f3ff',
          100: '#ede9fe',
          200: '#ddd6fe',
          300: '#c4b5fd',
          400: '#a78bfa',
          500: '#8b5cf6',
          600: '#7c3aed',  // primary accent
          700: '#6d28d9',
          800: '#5b21b6',
          900: '#4c1d95',
          950: '#2e1065',
        },
        // Semantic colors (HSL fine-tuned pra passar WCAG AA em
        // ambos os modos)
        success: {
          50: '#ecfdf5',
          100: '#d1fae5',
          400: '#34d399',
          500: '#10b981',
          600: '#059669',
          700: '#047857',
          900: '#064e3b',
          950: '#022c22',
        },
        warning: {
          50: '#fffbeb',
          100: '#fef3c7',
          400: '#fbbf24',
          500: '#f59e0b',
          600: '#d97706',
          700: '#b45309',
          900: '#78350f',
          950: '#451a03',
        },
        critical: {
          50: '#fff1f2',
          100: '#ffe4e6',
          400: '#fb7185',
          500: '#f43f5e',
          600: '#e11d48',
          700: '#be123c',
          900: '#881337',
          950: '#4c0519',
        },
        info: {
          50: '#f0f9ff',
          100: '#e0f2fe',
          400: '#38bdf8',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
          900: '#0c4a6e',
          950: '#082f49',
        },
        // Surfaces (light/dark via CSS variables — ver globals.css)
        surface: {
          DEFAULT: 'rgb(var(--surface) / <alpha-value>)',
          raised: 'rgb(var(--surface-raised) / <alpha-value>)',
          sunken: 'rgb(var(--surface-sunken) / <alpha-value>)',
          overlay: 'rgb(var(--surface-overlay) / <alpha-value>)',
        },
        border: {
          DEFAULT: 'rgb(var(--border) / <alpha-value>)',
          subtle: 'rgb(var(--border-subtle) / <alpha-value>)',
          strong: 'rgb(var(--border-strong) / <alpha-value>)',
        },
        ink: {
          DEFAULT: 'rgb(var(--ink) / <alpha-value>)',
          muted: 'rgb(var(--ink-muted) / <alpha-value>)',
          subtle: 'rgb(var(--ink-subtle) / <alpha-value>)',
          inverse: 'rgb(var(--ink-inverse) / <alpha-value>)',
        },
      },
      fontFamily: {
        sans: ['var(--font-sans)', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['var(--font-mono)', 'JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      fontSize: {
        '2xs': ['0.6875rem', { lineHeight: '1rem', letterSpacing: '0.025em' }],
        xs: ['0.75rem', { lineHeight: '1.125rem' }],
        sm: ['0.8125rem', { lineHeight: '1.25rem' }],
        base: ['0.875rem', { lineHeight: '1.375rem' }],
        md: ['0.9375rem', { lineHeight: '1.5rem' }],
        lg: ['1rem', { lineHeight: '1.5rem' }],
        xl: ['1.125rem', { lineHeight: '1.75rem' }],
        '2xl': ['1.375rem', { lineHeight: '1.875rem' }],
        '3xl': ['1.75rem', { lineHeight: '2.125rem', letterSpacing: '-0.01em' }],
        '4xl': ['2.25rem', { lineHeight: '2.5rem', letterSpacing: '-0.015em' }],
        '5xl': ['3rem', { lineHeight: '3.25rem', letterSpacing: '-0.02em' }],
      },
      borderRadius: {
        DEFAULT: '0.5rem',  // 8px
        sm: '0.375rem',     // 6px
        md: '0.5rem',       // 8px
        lg: '0.75rem',      // 12px
        xl: '1rem',         // 16px
        '2xl': '1.25rem',   // 20px
      },
      boxShadow: {
        // 3 níveis sutis — jamais preto saturado (cara de 2015)
        xs: '0 1px 1px 0 rgb(0 0 0 / 0.04), 0 1px 2px 0 rgb(0 0 0 / 0.03)',
        sm: '0 1px 2px 0 rgb(0 0 0 / 0.05), 0 2px 4px 0 rgb(0 0 0 / 0.04)',
        DEFAULT: '0 2px 4px 0 rgb(0 0 0 / 0.04), 0 4px 8px 0 rgb(0 0 0 / 0.04)',
        md: '0 4px 8px -2px rgb(0 0 0 / 0.06), 0 8px 16px -4px rgb(0 0 0 / 0.06)',
        lg: '0 8px 16px -4px rgb(0 0 0 / 0.08), 0 16px 32px -8px rgb(0 0 0 / 0.08)',
        xl: '0 16px 32px -8px rgb(0 0 0 / 0.10), 0 24px 48px -12px rgb(0 0 0 / 0.10)',
        // Glow effects (modos escuros usam mais)
        'glow-accent': '0 0 0 1px rgb(124 58 237 / 0.20), 0 4px 12px -2px rgb(124 58 237 / 0.15)',
        'glow-critical': '0 0 0 1px rgb(244 63 94 / 0.20), 0 4px 12px -2px rgb(244 63 94 / 0.15)',
        'glow-success': '0 0 0 1px rgb(16 185 129 / 0.20), 0 4px 12px -2px rgb(16 185 129 / 0.15)',
        'inner-soft': 'inset 0 1px 2px 0 rgb(0 0 0 / 0.04)',
      },
      keyframes: {
        'fade-in': {
          from: { opacity: '0', transform: 'translateY(2px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        'fade-in-fast': {
          from: { opacity: '0' },
          to: { opacity: '1' },
        },
        'scale-in': {
          from: { opacity: '0', transform: 'scale(0.96)' },
          to: { opacity: '1', transform: 'scale(1)' },
        },
        'slide-down': {
          from: { opacity: '0', transform: 'translateY(-4px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-up': {
          from: { opacity: '0', transform: 'translateY(4px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        'pulse-soft': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.6' },
        },
        'pulse-ring': {
          '0%': { boxShadow: '0 0 0 0 rgb(244 63 94 / 0.45)' },
          '70%': { boxShadow: '0 0 0 8px rgb(244 63 94 / 0)' },
          '100%': { boxShadow: '0 0 0 0 rgb(244 63 94 / 0)' },
        },
        'spin-slow': {
          to: { transform: 'rotate(360deg)' },
        },
      },
      animation: {
        'fade-in': 'fade-in 240ms ease-out',
        'fade-in-fast': 'fade-in-fast 160ms ease-out',
        'scale-in': 'scale-in 160ms ease-out',
        'slide-down': 'slide-down 200ms ease-out',
        'slide-up': 'slide-up 200ms ease-out',
        shimmer: 'shimmer 2s linear infinite',
        'pulse-soft': 'pulse-soft 2s ease-in-out infinite',
        'pulse-ring': 'pulse-ring 1.6s ease-out infinite',
        'spin-slow': 'spin-slow 2s linear infinite',
      },
      transitionTimingFunction: {
        'out-quart': 'cubic-bezier(0.25, 1, 0.5, 1)',
        'out-expo': 'cubic-bezier(0.16, 1, 0.3, 1)',
      },
      backdropBlur: {
        xs: '2px',
      },
    },
  },
  plugins: [],
}

export default config