import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import { ThemeProvider } from '../contexts/ThemeContext';
import { ToastProvider } from '../contexts/ToastContext';
import { userSettingsApi } from '../api/client';
import Me from './Me';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    userSettingsApi: {
      ...actual.userSettingsApi,
      get: vi.fn().mockResolvedValue({
        data: {
          feishu_notify_type: '',
          feishu_bot_webhook: '',
          feishu_id: '',
          password_login_enabled: true,
        },
      }),
      update: vi.fn().mockResolvedValue({ data: { message: '保存成功' } }),
    },
  };
});

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

function renderMe(isSuperAdmin = false) {
  setupLocalStorage();
  localStorage.setItem('token', 'token');
  localStorage.setItem(
    'user',
    JSON.stringify({ id: 1, username: 'alice', status: 'active', is_super_admin: isSuperAdmin, created_at: '' })
  );

  render(
    <MemoryRouter initialEntries={['/me']}>
      <ThemeProvider>
        <AuthProvider>
          <ToastProvider>
            <Routes>
              <Route path="/me" element={<Me />} />
              <Route path="/login" element={<div>登录页</div>} />
            </Routes>
          </ToastProvider>
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

  expect(screen.queryByRole('heading', { name: '账号密码登录' })).not.toBeInTheDocument();

  await user.click(screen.getByRole('button', { name: '退出' }));
  expect(screen.getByText('登录页')).toBeInTheDocument();
  expect(localStorage.getItem('token')).toBeNull();
});

test('超级管理员开关在设置加载完成前显示加载态，不闪错误状态', async () => {
  let resolveGet: (value: {
    data: {
      feishu_notify_type: string;
      feishu_bot_webhook: string;
      feishu_id: string;
      password_login_enabled: boolean;
    };
  }) => void = () => {};
  vi.mocked(userSettingsApi.get).mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        resolveGet = resolve;
      })
  );

  renderMe(true);

  const toggle = screen.getByRole('button', { name: '加载中' });
  expect(toggle).toBeDisabled();
  expect(screen.queryByRole('button', { name: '已开启' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: '已关闭' })).not.toBeInTheDocument();

  await act(async () => {
    resolveGet({
      data: {
        feishu_notify_type: '',
        feishu_bot_webhook: '',
        feishu_id: '',
        password_login_enabled: false,
      },
    });
  });

  expect(await screen.findByRole('button', { name: '已关闭' })).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: '加载中' })).not.toBeInTheDocument();
});

test('超级管理员可确认关闭账号密码登录', async () => {
  const user = userEvent.setup();
  vi.mocked(userSettingsApi.update).mockClear();
  renderMe(true);

  const toggle = await screen.findByRole('button', { name: '已开启' });
  await user.click(toggle);
  expect(screen.getByRole('dialog')).toBeInTheDocument();
  expect(screen.getByText(/同时关闭注册/)).toBeInTheDocument();
  expect(screen.getByText(/不会检查飞书/)).toBeInTheDocument();

  await user.click(screen.getByRole('button', { name: '确认关闭' }));
  await waitFor(() => {
    expect(userSettingsApi.update).toHaveBeenCalledWith({ password_login_enabled: false });
  });
  expect(await screen.findByRole('button', { name: '已关闭' })).toBeInTheDocument();
});

test('超级管理员点击已关闭时直接开启，无需确认', async () => {
  const user = userEvent.setup();
  vi.mocked(userSettingsApi.update).mockClear();
  vi.mocked(userSettingsApi.get).mockResolvedValueOnce({
    data: {
      feishu_notify_type: '',
      feishu_bot_webhook: '',
      feishu_id: '',
      password_login_enabled: false,
    },
  });
  renderMe(true);

  const toggle = await screen.findByRole('button', { name: '已关闭' });
  await user.click(toggle);

  expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  await waitFor(() => {
    expect(userSettingsApi.update).toHaveBeenCalledWith({ password_login_enabled: true });
  });
  expect(await screen.findByRole('button', { name: '已开启' })).toBeInTheDocument();
});

test('保存失败时保持原开关状态并提示错误', async () => {
  const user = userEvent.setup();
  vi.mocked(userSettingsApi.update).mockClear();
  vi.mocked(userSettingsApi.update).mockRejectedValueOnce(new Error('network error'));
  renderMe(true);

  const toggle = await screen.findByRole('button', { name: '已开启' });
  await user.click(toggle);
  await user.click(screen.getByRole('button', { name: '确认关闭' }));

  await waitFor(() => {
    expect(userSettingsApi.update).toHaveBeenCalledWith({ password_login_enabled: false });
  });

  const errorToast = await screen.findByText('保存失败，请重试');
  expect(errorToast).toHaveClass('toast-error');
  expect(screen.getByRole('button', { name: '已开启' })).toBeInTheDocument();
});

test('取消关闭确认框会关闭弹窗且不调用保存接口', async () => {
  const user = userEvent.setup();
  vi.mocked(userSettingsApi.update).mockClear();
  renderMe(true);

  const toggle = await screen.findByRole('button', { name: '已开启' });
  await user.click(toggle);
  expect(screen.getByRole('dialog')).toBeInTheDocument();

  await user.click(screen.getByRole('button', { name: '取消' }));

  expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  expect(userSettingsApi.update).not.toHaveBeenCalled();
  expect(screen.getByRole('button', { name: '已开启' })).toBeInTheDocument();
});
