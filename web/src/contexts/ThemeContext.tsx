import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';

export type Theme = 'light' | 'dark';

export type UiStyle =
  | 'glass'
  | 'eink'
  | 'kinetic'
  | 'motion-driven'
  | 'retro-futurism'
  | 'hud-scifi-fui'
  | 'vibrant-block'
  | 'aurora'
  | 'aurora-evolved'
  | 'memphis'
  | 'y2k'
  | 'cyberpunk'
  | 'pixel-art';

type ThemeContextValue = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
  uiStyle: UiStyle;
  setUiStyle: (style: UiStyle) => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

const STORAGE_KEY = 'ui.theme';
const UI_STYLE_STORAGE_KEY = 'ui.uistyle';

function canUseLocalStorage(): boolean {
  if (typeof window === 'undefined') return false;
  const ls = window.localStorage as unknown as { getItem?: unknown; setItem?: unknown };
  return typeof ls?.getItem === 'function' && typeof ls?.setItem === 'function';
}

function getSystemTheme(): Theme {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function getInitialTheme(): Theme {
  if (typeof window === 'undefined') return 'light';
  if (canUseLocalStorage()) {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored === 'dark' || stored === 'light') return stored;
  }
  return getSystemTheme();
}

const VALID_UI_STYLES: readonly UiStyle[] = [
  'glass',
  'eink',
  'kinetic',
  'motion-driven',
  'retro-futurism',
  'hud-scifi-fui',
  'vibrant-block',
  'aurora',
  'aurora-evolved',
  'memphis',
  'y2k',
  'cyberpunk',
  'pixel-art',
];

function isUiStyle(value: string | null | undefined): value is UiStyle {
  return value != null && (VALID_UI_STYLES as readonly string[]).includes(value);
}

function getInitialUiStyle(): UiStyle {
  if (typeof window === 'undefined') return 'glass';
  if (canUseLocalStorage()) {
    const stored = window.localStorage.getItem(UI_STYLE_STORAGE_KEY);
    if (isUiStyle(stored)) return stored;
  }
  return 'glass';
}

function applyThemeToDocument(theme: Theme) {
  if (typeof document === 'undefined') return;
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
}

function applyUiStyleToDocument(uiStyle: UiStyle) {
  if (typeof document === 'undefined') return;
  document.documentElement.dataset.uiStyle = uiStyle;
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => getInitialTheme());
  const [uiStyle, setUiStyleState] = useState<UiStyle>(() => getInitialUiStyle());

  useEffect(() => {
    applyThemeToDocument(theme);
    if (canUseLocalStorage()) {
      try {
        window.localStorage.setItem(STORAGE_KEY, theme);
      } catch {
        // ignore storage failures (private mode, etc.)
      }
    }
  }, [theme]);

  useEffect(() => {
    applyUiStyleToDocument(uiStyle);
    if (canUseLocalStorage()) {
      try {
        window.localStorage.setItem(UI_STYLE_STORAGE_KEY, uiStyle);
      } catch {
        // ignore storage failures (private mode, etc.)
      }
    }
  }, [uiStyle]);

  const setUiStyle = useCallback((style: UiStyle) => {
    setUiStyleState(style);
  }, []);

  const value = useMemo<ThemeContextValue>(() => {
    return {
      theme,
      setTheme,
      toggleTheme: () => setTheme((t) => (t === 'dark' ? 'light' : 'dark')),
      uiStyle,
      setUiStyle,
    };
  }, [theme, uiStyle, setUiStyle]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider');
  return ctx;
}
