import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import { ThemeProvider } from '../contexts/ThemeContext';
import Layout from './Layout';

function setupLocalStorage() {
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
}

function renderLayoutAt(path: string) {
  setupLocalStorage();
  localStorage.setItem('token', 'token');
  localStorage.setItem(
    'user',
    JSON.stringify({ id: 1, username: 'alice', status: 'active', is_super_admin: false, created_at: '' })
  );

  render(
    <MemoryRouter initialEntries={[path]}>
      <ThemeProvider>
        <AuthProvider>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<div>首页内容</div>} />
              <Route path="me" element={<div>我的页面</div>} />
            </Route>
            <Route path="/login" element={<div>登录页</div>} />
          </Routes>
        </AuthProvider>
      </ThemeProvider>
    </MemoryRouter>
  );
}

test('侧边栏提供“我的”入口并移除顶部栏', () => {
  renderLayoutAt('/me');

  expect(screen.getByRole('link', { name: /我的/ })).toHaveClass('active');
  expect(screen.queryByRole('banner')).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: '打开或关闭侧边导航' })).toBeInTheDocument();
  expect(screen.getByText('我的页面')).toBeInTheDocument();
});
