import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import { ThemeProvider } from '../contexts/ThemeContext';
import Login from './Login';

test('登录页使用玻璃态场景容器并展示标题', () => {
  const store = new Map<string, string>();
  // @ts-expect-error test polyfill
  globalThis.localStorage = {
    getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
    setItem: (k: string, v: string) => {
      store.set(k, String(v));
    },
    removeItem: (k: string) => {
      store.delete(k);
    },
  };

  const { container } = render(
    <MemoryRouter initialEntries={['/login']}>
      <ThemeProvider>
        <AuthProvider>
          <Login />
        </AuthProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  expect(container.querySelector('.auth-scene')).not.toBeNull();
  expect(container.querySelector('.auth-page')).not.toBeNull();
  expect(screen.getByRole('heading', { name: 'RSS 阅读器' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: '账号密码登录' })).toBeInTheDocument();
  expect(screen.getByRole('combobox', { name: /界面风格/ })).toBeInTheDocument();
});
