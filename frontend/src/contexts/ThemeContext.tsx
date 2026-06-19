import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';

export type ThemeMode = 'dark' | 'light';
export type ChatTheme = 'default' | 'ocean' | 'sunset' | 'aurora' | 'midnight';

interface ThemeContextType {
  mode: ThemeMode;
  chatTheme: ChatTheme;
  toggleMode: () => void;
  setChatTheme: (theme: ChatTheme) => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

const STORAGE_KEY_MODE = 'pulsechat_theme_mode';
const STORAGE_KEY_CHAT_THEME = 'pulsechat_chat_theme';

export const CHAT_THEMES: { id: ChatTheme; label: string; color: string }[] = [
  { id: 'default', label: 'Indigo', color: '#6366f1' },
  { id: 'ocean', label: 'Ocean', color: '#06b6d4' },
  { id: 'sunset', label: 'Sunset', color: '#f97316' },
  { id: 'aurora', label: 'Aurora', color: '#10b981' },
  { id: 'midnight', label: 'Midnight', color: '#8b5cf6' },
];

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [mode, setMode] = useState<ThemeMode>(() => {
    const stored = localStorage.getItem(STORAGE_KEY_MODE);
    if (stored === 'light' || stored === 'dark') return stored;
    // Respect system preference
    if (window.matchMedia('(prefers-color-scheme: light)').matches) return 'light';
    return 'dark';
  });

  const [chatTheme, setChatThemeState] = useState<ChatTheme>(() => {
    const stored = localStorage.getItem(STORAGE_KEY_CHAT_THEME);
    if (stored && ['default', 'ocean', 'sunset', 'aurora', 'midnight'].includes(stored)) {
      return stored as ChatTheme;
    }
    return 'default';
  });

  // Sync dark mode class on <html>
  useEffect(() => {
    const root = document.documentElement;
    if (mode === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
    localStorage.setItem(STORAGE_KEY_MODE, mode);
  }, [mode]);

  // Sync chat theme class on <html>
  useEffect(() => {
    const root = document.documentElement;
    // Remove all theme classes
    CHAT_THEMES.forEach(t => root.classList.remove(`theme-${t.id}`));
    // Add current theme class
    root.classList.add(`theme-${chatTheme}`);
    localStorage.setItem(STORAGE_KEY_CHAT_THEME, chatTheme);
  }, [chatTheme]);

  const toggleMode = useCallback(() => {
    setMode(prev => (prev === 'dark' ? 'light' : 'dark'));
  }, []);

  const setChatTheme = useCallback((theme: ChatTheme) => {
    setChatThemeState(theme);
  }, []);

  return (
    <ThemeContext.Provider value={{ mode, chatTheme, toggleMode, setChatTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}
