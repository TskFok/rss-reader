import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ThemeProvider } from '../contexts/ThemeContext';
import { authApi } from '../api/client';
import Register from './Register';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    authApi: {
      ...actual.authApi,
      getLoginOptions: vi.fn().mockResolvedValue({ data: { password_login_enabled: true } }),
    },
  };
});

test('注册页使用玻璃态场景容器并展示表单', async () => {
  const { container } = render(
    <MemoryRouter initialEntries={['/register']}>
      <ThemeProvider>
        <Register />
      </ThemeProvider>
    </MemoryRouter>
  );

  expect(container.querySelector('.auth-scene')).not.toBeNull();
  expect(container.querySelector('.auth-page')).not.toBeNull();
  expect(await screen.findByRole('heading', { name: '注册' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: '注册' })).toBeInTheDocument();
});

test('关闭账号注册后跳转登录页', async () => {
  vi.mocked(authApi.getLoginOptions).mockResolvedValueOnce({
    data: { password_login_enabled: false },
  } as Awaited<ReturnType<typeof authApi.getLoginOptions>>);

  render(
    <MemoryRouter initialEntries={['/register']}>
      <ThemeProvider>
        <Routes>
          <Route path="/register" element={<Register />} />
          <Route path="/login" element={<div>登录页</div>} />
        </Routes>
      </ThemeProvider>
    </MemoryRouter>
  );

  expect(await screen.findByText('登录页')).toBeInTheDocument();
});

test('获取登录选项失败时按失败开放策略继续展示注册表单', async () => {
  vi.mocked(authApi.getLoginOptions).mockRejectedValueOnce(new Error('network error'));

  render(
    <MemoryRouter initialEntries={['/register']}>
      <ThemeProvider>
        <Register />
      </ThemeProvider>
    </MemoryRouter>
  );

  expect(await screen.findByRole('heading', { name: '注册' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: '注册' })).toBeInTheDocument();
});
