import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider, useTheme } from './ThemeContext';

function TestToggle() {
  const { theme, toggleTheme } = useTheme();
  return (
    <button type="button" onClick={toggleTheme}>
      {theme}
    </button>
  );
}

test('ThemeProvider: 初始应用主题并可切换（写入 data-theme 与 localStorage）', async () => {
  const user = userEvent.setup();
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  const mockLocalStorage = {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => {
      store.set(key, value);
    },
    removeItem: (key: string) => {
      store.delete(key);
    },
  };
  Object.defineProperty(window, 'localStorage', { value: mockLocalStorage, configurable: true });

  // 让测试环境稳定：如果有 matchMedia，就先移除它，确保默认走 light
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.theme).toBe('light');
  expect(window.localStorage.getItem('ui.theme')).toBe('light');

  await user.click(screen.getByRole('button', { name: 'light' }));
  expect(document.documentElement.dataset.theme).toBe('dark');
  expect(window.localStorage.getItem('ui.theme')).toBe('dark');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});
