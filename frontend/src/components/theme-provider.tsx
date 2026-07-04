'use client'

/**
 * ThemeProvider — dark/light mode toggle.
 *
 * Persiste em localStorage; respeita prefers-color-scheme no primeiro
 * load. Aplica classe `dark` em <html>.
 */
import * as React from 'react'

type Theme = 'light' | 'dark'

const ThemeContext = React.createContext<{
  theme: Theme
  setTheme: (t: Theme) => void
  toggle: () => void
}>({
  theme: 'light',
  setTheme: () => {},
  toggle: () => {},
})

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = React.useState<Theme>('light')
  const [mounted, setMounted] = React.useState(false)

  React.useEffect(() => {
    setMounted(true)
    const stored = localStorage.getItem('rn_theme') as Theme | null
    const initial =
      stored ??
      (window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'dark'
        : 'light')
    setThemeState(initial)
    document.documentElement.classList.toggle('dark', initial === 'dark')
  }, [])

  const setTheme = React.useCallback((t: Theme) => {
    setThemeState(t)
    localStorage.setItem('rn_theme', t)
    document.documentElement.classList.toggle('dark', t === 'dark')
  }, [])

  const toggle = React.useCallback(() => {
    setTheme(theme === 'dark' ? 'light' : 'dark')
  }, [theme, setTheme])

  return (
    <ThemeContext.Provider value={{ theme, setTheme, toggle }}>
      {children}
    </ThemeContext.Provider>
  )
}

export const useTheme = () => React.useContext(ThemeContext)

/** Inline script pra evitar FOUC (flash of unstyled content) em dark mode.
 *  Inserido em <head> antes da hidratação.
 */
export const themeScript = `
  (function() {
    try {
      var stored = localStorage.getItem('rn_theme');
      var prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      var theme = stored || (prefersDark ? 'dark' : 'light');
      if (theme === 'dark') document.documentElement.classList.add('dark');
    } catch (e) {}
  })();
`