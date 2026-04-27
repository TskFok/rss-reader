import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { type UiStyle, ThemeProvider, useTheme } from './ThemeContext';

function TestToggle() {
  const { theme, toggleTheme, uiStyle, setUiStyle } = useTheme();
  return (
    <div>
      <button type="button" onClick={toggleTheme}>
        {theme}
      </button>
      <button
        type="button"
        onClick={() => {
          const order: UiStyle[] = [
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
          setUiStyle(order[(order.indexOf(uiStyle) + 1) % order.length]);
        }}
      >
        toggle-style
      </button>
      <span data-testid="uistyle">{uiStyle}</span>
    </div>
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
  expect(document.documentElement.dataset.uiStyle).toBe('glass');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('glass');

  await user.click(screen.getByRole('button', { name: 'light' }));
  expect(document.documentElement.dataset.theme).toBe('dark');
  expect(window.localStorage.getItem('ui.theme')).toBe('dark');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('eink');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('eink');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('eink');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('kinetic');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('kinetic');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('kinetic');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('motion-driven');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('motion-driven');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('motion-driven');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('retro-futurism');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('retro-futurism');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('retro-futurism');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('hud-scifi-fui');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('hud-scifi-fui');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('hud-scifi-fui');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('vibrant-block');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('vibrant-block');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('vibrant-block');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('aurora');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('aurora');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('aurora');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('aurora-evolved');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('aurora-evolved');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('aurora-evolved');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('memphis');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('memphis');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('memphis');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('y2k');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('y2k');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('y2k');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('cyberpunk');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('cyberpunk');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('cyberpunk');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('pixel-art');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('pixel-art');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('pixel-art');

  await user.click(screen.getByRole('button', { name: 'toggle-style' }));
  expect(document.documentElement.dataset.uiStyle).toBe('glass');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('glass');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 kinetic 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'kinetic');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('kinetic');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('kinetic');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 motion-driven 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'motion-driven');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('motion-driven');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('motion-driven');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 hud-scifi-fui 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'hud-scifi-fui');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('hud-scifi-fui');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('hud-scifi-fui');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 retro-futurism 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'retro-futurism');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('retro-futurism');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('retro-futurism');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 vibrant-block 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'vibrant-block');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('vibrant-block');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('vibrant-block');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 aurora-evolved 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'aurora-evolved');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('aurora-evolved');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('aurora-evolved');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 memphis 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'memphis');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('memphis');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('memphis');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 pixel-art 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'pixel-art');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('pixel-art');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('pixel-art');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 y2k 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'y2k');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('y2k');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('y2k');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 cyberpunk 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'cyberpunk');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('cyberpunk');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('cyberpunk');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});

test('ThemeProvider: 从 localStorage 恢复 aurora 界面风格', () => {
  const originalLocalStorage = window.localStorage;
  const store = new Map<string, string>();
  store.set('ui.uistyle', 'aurora');
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <TestToggle />
    </ThemeProvider>
  );

  expect(document.documentElement.dataset.uiStyle).toBe('aurora');
  expect(screen.getByTestId('uistyle')).toHaveTextContent('aurora');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});
