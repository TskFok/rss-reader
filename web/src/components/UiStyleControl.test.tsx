import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider, useTheme } from '../contexts/ThemeContext';
import UiStyleControl from './UiStyleControl';

function Harness() {
  const { uiStyle } = useTheme();
  return (
    <div>
      <UiStyleControl variant="header" />
      <span data-testid="ui-style-value">{uiStyle}</span>
    </div>
  );
}

test('UiStyleControl: 可选 kinetic / motion-driven / retro-futurism / hud-scifi-fui / vibrant-block / aurora / aurora-evolved / memphis / y2k / cyberpunk / pixel-art 并写入 data 与 localStorage', async () => {
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
  const originalMatchMedia = window.matchMedia;
  Object.defineProperty(window, 'matchMedia', { value: undefined, configurable: true });

  render(
    <ThemeProvider>
      <Harness />
    </ThemeProvider>
  );

  const select = screen.getByRole('combobox', { name: /界面风格/ });
  expect(select).toBeInTheDocument();
  expect(Array.from((select as HTMLSelectElement).options).map((o) => o.value)).toEqual([
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
  ]);

  await user.selectOptions(select, 'kinetic');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('kinetic');
  expect(document.documentElement.dataset.uiStyle).toBe('kinetic');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('kinetic');

  await user.selectOptions(select, 'motion-driven');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('motion-driven');
  expect(document.documentElement.dataset.uiStyle).toBe('motion-driven');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('motion-driven');

  await user.selectOptions(select, 'retro-futurism');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('retro-futurism');
  expect(document.documentElement.dataset.uiStyle).toBe('retro-futurism');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('retro-futurism');

  await user.selectOptions(select, 'hud-scifi-fui');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('hud-scifi-fui');
  expect(document.documentElement.dataset.uiStyle).toBe('hud-scifi-fui');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('hud-scifi-fui');

  await user.selectOptions(select, 'vibrant-block');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('vibrant-block');
  expect(document.documentElement.dataset.uiStyle).toBe('vibrant-block');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('vibrant-block');

  await user.selectOptions(select, 'aurora');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('aurora');
  expect(document.documentElement.dataset.uiStyle).toBe('aurora');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('aurora');

  await user.selectOptions(select, 'aurora-evolved');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('aurora-evolved');
  expect(document.documentElement.dataset.uiStyle).toBe('aurora-evolved');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('aurora-evolved');

  await user.selectOptions(select, 'memphis');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('memphis');
  expect(document.documentElement.dataset.uiStyle).toBe('memphis');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('memphis');

  await user.selectOptions(select, 'y2k');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('y2k');
  expect(document.documentElement.dataset.uiStyle).toBe('y2k');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('y2k');

  await user.selectOptions(select, 'cyberpunk');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('cyberpunk');
  expect(document.documentElement.dataset.uiStyle).toBe('cyberpunk');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('cyberpunk');

  await user.selectOptions(select, 'pixel-art');
  expect(screen.getByTestId('ui-style-value')).toHaveTextContent('pixel-art');
  expect(document.documentElement.dataset.uiStyle).toBe('pixel-art');
  expect(window.localStorage.getItem('ui.uistyle')).toBe('pixel-art');

  Object.defineProperty(window, 'matchMedia', { value: originalMatchMedia, configurable: true });
  Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
});
