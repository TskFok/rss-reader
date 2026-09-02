import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import { ThemeProvider } from '../contexts/ThemeContext';
import { authApi } from '../api/client';
import Login from './Login';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    authApi: {
      ...actual.authApi,
      getLoginOptions: vi.fn().mockResolvedValue({ data: { password_login_enabled: true } }),
      getFeishuLoginUrl: vi.fn().mockResolvedValue({ data: { url: '/api/auth/feishu/login', goto: 'https://example.com' } }),
    },
  };
});

test('登录页使用玻璃态场景容器并展示标题', async () => {
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
  expect(await screen.findByRole('button', { name: '账号密码登录' })).toBeInTheDocument();
  expect(screen.getByRole('combobox', { name: /界面风格/ })).toBeInTheDocument();
});

test('关闭账号密码登录后只保留飞书登录', async () => {
  vi.mocked(authApi.getLoginOptions).mockResolvedValueOnce({
    data: { password_login_enabled: false },
  } as Awaited<ReturnType<typeof authApi.getLoginOptions>>);

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

  render(
    <MemoryRouter initialEntries={['/login']}>
      <ThemeProvider>
        <AuthProvider>
          <Login />
        </AuthProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  await waitFor(() => {
    expect(screen.queryByRole('button', { name: '账号密码登录' })).not.toBeInTheDocument();
  });
  expect(screen.getByRole('button', { name: '飞书登录' })).toBeInTheDocument();
  expect(screen.queryByText('还没有账号？')).not.toBeInTheDocument();
});
