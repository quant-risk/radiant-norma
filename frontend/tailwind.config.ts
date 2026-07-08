import type { Config } from 'tailwindcss'

/**
 * Radiant Norma — Design System Tokens (Tailwind extension).
 *
 * Filosofia: "Institutional Premium Noir" — fusão de:
 *   - Editorial Luxo Sério (warm off-white, serifa editorial)
 *   - Quiet Authority Tech (Linear/Vercel, density calibrada)
 *   - Noir Banking Terminal (dark mode profundo, mono obsessivo)
 *
 * Princípios:
 *   - Paleta warm-neutral (off-white coal) em ambos modos — foge do cinza
 *     genérico de SaaS e comunica produto high-ticket.
 *   - Accent violet→magenta gradient usado com parcimônia (1 elemento
 *     proeminente por viewport carrega o gradient; resto fica monochrome).
 *   - Tipografia tripla: Inter (corpo), Fraunces (display/editorial),
 *     JetBrains Mono (dados críticos).
 *   - Sombras em 2 famílias: utilitárias (charcoal 5%) + glow (violet/rose
 *     18%, usadas só em CTAs primários e alertas críticos).
 *   - Motion: 180-240ms com cubic-bezier(0.16, 1, 0.3, 1). Sem bounce.
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
        // Brand accent — violet com escala magenta no topo (gradient)
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
        // Magenta — complemento do gradient accent
        magenta: {
          400: '#e879f9',
          500: '#d946ef',
          600: '#c026d3',
        },
        // Semantic colors (HSL fine-tuned pra WCAG AA em ambos modos)
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
        // Surfaces (warm-noir — bind em CSS variables em globals.css)
        surface: {
          DEFAULT: 'rgb(var(--surface) / <alpha-value>)',
          raised: 'rgb(var(--surface-raised) / <alpha-value>)',
          sunken: 'rgb(var(--surface-sunken) / <alpha-value>)',
          overlay: 'rgb(var(--surface-overlay) / <alpha-value>)',
          inverse: 'rgb(var(--surface-inverse) / <alpha-value>)',
        },
        border: {
          DEFAULT: 'rgb(var(--border) / <alpha-value>)',
          subtle: 'rgb(var(--border-subtle) / <alpha-value>)',
          strong: 'rgb(var(--border-strong) / <alpha-value>)',
          accent: 'rgb(var(--border-accent) / <alpha-value>)',
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
        serif: ['var(--font-serif)', 'Fraunces', 'Georgia', 'serif'],
        mono: ['var(--font-mono)', 'JetBrains Mono', 'ui-monospace', 'monospace'],
        display: ['var(--font-serif)', 'Fraunces', 'Georgia', 'serif'],
      },
      fontSize: {
        '2xs': ['0.6875rem', { lineHeight: '1rem', letterSpacing: '0.04em' }],
        xs: ['0.75rem', { lineHeight: '1.125rem' }],
        sm: ['0.8125rem', { lineHeight: '1.25rem' }],
        base: ['0.875rem', { lineHeight: '1.375rem' }],
        md: ['0.9375rem', { lineHeight: '1.5rem' }],
        lg: ['1rem', { lineHeight: '1.5rem' }],
        xl: ['1.125rem', { lineHeight: '1.75rem' }],
        '2xl': ['1.375rem', { lineHeight: '1.875rem', letterSpacing: '-0.015em' }],
        '3xl': ['1.75rem', { lineHeight: '2.125rem', letterSpacing: '-0.02em' }],
        '4xl': ['2.25rem', { lineHeight: '2.5rem', letterSpacing: '-0.025em' }],
        '5xl': ['3rem', { lineHeight: '3.125rem', letterSpacing: '-0.03em' }],
        '6xl': ['3.75rem', { lineHeight: '3.875rem', letterSpacing: '-0.035em' }],
        'display-sm': ['2.5rem', { lineHeight: '2.875rem', letterSpacing: '-0.025em' }],
        'display-md': ['3.5rem', { lineHeight: '3.625rem', letterSpacing: '-0.03em' }],
        'display-lg': ['4.5rem', { lineHeight: '4.625rem', letterSpacing: '-0.035em' }],
      },
      borderRadius: {
        DEFAULT: '0.5rem',
        sm: '0.375rem',
        md: '0.5rem',
        lg: '0.625rem',    // 10px — slightly tighter than before
        xl: '0.875rem',    // 14px
        '2xl': '1.25rem',  // 20px
        '3xl': '1.75rem',  // 28px
      },
      boxShadow: {
        // Utility shadows — charcoal-ink, quase invisíveis (estilo Linear)
        'xs': '0 1px 1px 0 rgb(15 14 12 / 0.04), 0 1px 2px 0 rgb(15 14 12 / 0.03)',
        'sm': '0 1px 2px 0 rgb(15 14 12 / 0.05), 0 2px 4px 0 rgb(15 14 12 / 0.04)',
        DEFAULT: '0 2px 4px 0 rgb(15 14 12 / 0.04), 0 4px 8px 0 rgb(15 14 12 / 0.04)',
        'md': '0 4px 8px -2px rgb(15 14 12 / 0.06), 0 8px 16px -4px rgb(15 14 12 / 0.06)',
        'lg': '0 8px 16px -4px rgb(15 14 12 / 0.08), 0 16px 32px -8px rgb(15 14 12 / 0.08)',
        'xl': '0 16px 32px -8px rgb(15 14 12 / 0.10), 0 24px 48px -12px rgb(15 14 12 / 0.10)',
        '2xl': '0 24px 48px -12px rgb(15 14 12 / 0.18), 0 32px 64px -16px rgb(15 14 12 / 0.14)',
        // Inner soft (input fields)
        'inner-soft': 'inset 0 1px 2px 0 rgb(15 14 12 / 0.04)',
        // Glow effects — reservados pra CTAs primários e alertas críticos
        'glow-accent': '0 0 0 1px rgb(124 58 237 / 0.30), 0 8px 24px -6px rgb(124 58 237 / 0.35)',
        'glow-accent-sm': '0 0 0 1px rgb(124 58 237 / 0.20), 0 2px 8px -2px rgb(124 58 237 / 0.20)',
        'glow-critical': '0 0 0 1px rgb(225 29 72 / 0.25), 0 6px 20px -4px rgb(225 29 72 / 0.30)',
        'glow-success': '0 0 0 1px rgb(16 185 129 / 0.20), 0 6px 20px -4px rgb(16 185 129 / 0.25)',
        'glow-warning': '0 0 0 1px rgb(245 158 11 / 0.25), 0 6px 20px -4px rgb(245 158 11 / 0.30)',
        // Hairline (used as visual divider refinement)
        'hairline': 'inset 0 -1px 0 0 rgb(15 14 12 / 0.06)',
      },
      keyframes: {
        'fade-in': {
          from: { opacity: '0', transform: 'translateY(4px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        'fade-in-fast': {
          from: { opacity: '0' },
          to: { opacity: '1' },
        },
        'fade-up': {
          from: { opacity: '0', transform: 'translateY(8px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        'scale-in': {
          from: { opacity: '0', transform: 'scale(0.97)' },
          to: { opacity: '1', transform: 'scale(1)' },
        },
        'slide-down': {
          from: { opacity: '0', transform: 'translateY(-6px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-up': {
          from: { opacity: '0', transform: 'translateY(6px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-right': {
          from: { opacity: '0', transform: 'translateX(-6px)' },
          to: { opacity: '1', transform: 'translateX(0)' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        'pulse-soft': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.55' },
        },
        'pulse-ring': {
          '0%': { boxShadow: '0 0 0 0 rgb(225 29 72 / 0.45)' },
          '70%': { boxShadow: '0 0 0 10px rgb(225 29 72 / 0)' },
          '100%': { boxShadow: '0 0 0 0 rgb(225 29 72 / 0)' },
        },
        'pulse-accent': {
          '0%': { boxShadow: '0 0 0 0 rgb(124 58 237 / 0.50)' },
          '70%': { boxShadow: '0 0 0 8px rgb(124 58 237 / 0)' },
          '100%': { boxShadow: '0 0 0 0 rgb(124 58 237 / 0)' },
        },
        'spin-slow': {
          to: { transform: 'rotate(360deg)' },
        },
        'gradient-pan': {
          '0%, 100%': { backgroundPosition: '0% 50%' },
          '50%': { backgroundPosition: '100% 50%' },
        },
        'marquee': {
          '0%': { transform: 'translateX(0)' },
          '100%': { transform: 'translateX(-50%)' },
        },
      },
      animation: {
        'fade-in': 'fade-in 240ms cubic-bezier(0.16, 1, 0.3, 1)',
        'fade-in-fast': 'fade-in-fast 160ms cubic-bezier(0.16, 1, 0.3, 1)',
        'fade-up': 'fade-up 320ms cubic-bezier(0.16, 1, 0.3, 1)',
        'scale-in': 'scale-in 180ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-down': 'slide-down 220ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-up': 'slide-up 220ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-right': 'slide-right 220ms cubic-bezier(0.16, 1, 0.3, 1)',
        'shimmer': 'shimmer 2.4s linear infinite',
        'pulse-soft': 'pulse-soft 2.4s ease-in-out infinite',
        'pulse-ring': 'pulse-ring 1.8s cubic-bezier(0.16, 1, 0.3, 1) infinite',
        'pulse-accent': 'pulse-accent 2.2s cubic-bezier(0.16, 1, 0.3, 1) infinite',
        'spin-slow': 'spin-slow 2.4s linear infinite',
        'gradient-pan': 'gradient-pan 8s ease-in-out infinite',
      },
      transitionTimingFunction: {
        'out-quart': 'cubic-bezier(0.25, 1, 0.5, 1)',
        'out-expo': 'cubic-bezier(0.16, 1, 0.3, 1)',
        'in-out-quart': 'cubic-bezier(0.76, 0, 0.24, 1)',
      },
      transitionDuration: {
        '180': '180ms',
        '240': '240ms',
        '320': '320ms',
        '400': '400ms',
      },
      backdropBlur: {
        xs: '2px',
      },
      backgroundImage: {
        'grain': "url(\"data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='3' stitchTiles='stitch'/%3E%3CfeColorMatrix values='0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.5 0'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='0.5'/%3E%3C/svg%3E\")",
        'gradient-accent': 'linear-gradient(135deg, #7C3AED 0%, #D946EF 100%)',
        'gradient-accent-soft': 'linear-gradient(135deg, rgba(124, 58, 237, 0.12) 0%, rgba(217, 70, 239, 0.08) 100%)',
        'gradient-mask-r': 'linear-gradient(to right, transparent, black 8%, black 92%, transparent)',
      },
    },
  },
  plugins: [],
}

export default config