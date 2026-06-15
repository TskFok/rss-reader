import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import { ThemeProvider } from '../contexts/ThemeContext';
import Me from './Me';

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

function renderMe() {
  setupLocalStorage();
  localStorage.setItem('token', 'token');
  localStorage.setItem(
    'user',
    JSON.stringify({ id: 1, username: 'alice', status: 'active', is_super_admin: false, created_at: '' })
  );

  render(
    <MemoryRouter initialEntries={['/me']}>
      <ThemeProvider>
        <AuthProvider>
          <Routes>
            <Route path="/me" element={<Me />} />
            <Route path="/login" element={<div>登录页</div>} />
          </Routes>
        </AuthProvider>
      </ThemeProvider>
    </MemoryRouter>
  );
}

test('我的页面承载原顶部右侧的个人与偏好操作', async () => {
  const user = userEvent.setup();

  renderMe();

  expect(screen.getByRole('heading', { name: '我的' })).toBeInTheDocument();
  expect(screen.getByText('alice')).toBeInTheDocument();
  expect(screen.getByLabelText(/界面风格/)).toBeInTheDocument();

  const themeButton = screen.getByRole('button', { name: '切换到深色模式' });
  await user.click(themeButton);
  expect(document.documentElement.dataset.theme).toBe('dark');

  await user.click(screen.getByRole('button', { name: '退出' }));
  expect(screen.getByText('登录页')).toBeInTheDocument();
  expect(localStorage.getItem('token')).toBeNull();
});
