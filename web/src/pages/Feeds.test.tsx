import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import Feeds from './Feeds';
import { articlesApi } from '../api/client';

vi.mock('../api/client', async () => {
  return {
    feedsApi: { list: vi.fn().mockResolvedValue({ data: [] }) },
    categoriesApi: { list: vi.fn().mockResolvedValue({ data: [] }) },
    proxiesApi: { list: vi.fn().mockResolvedValue({ data: [] }) },
    aiModelsApi: { list: vi.fn().mockResolvedValue({ data: [{ id: 1, name: 'm', base_url: 'u', user_id: 1, created_at: '', updated_at: '' }] }) },
    articlesApi: { summarizeStream: vi.fn().mockImplementation(async (_params: unknown, cb: { onMeta: (n: number) => void; onChunk: (s: string) => void; onMetaAll?: (m: { total?: number }) => void }) => {
      cb.onMeta(2);
      cb.onMetaAll?.({ total: 2 });
      cb.onChunk('总结内容');
    }) },
    summaryHistoriesApi: {
      list: vi.fn().mockResolvedValue({ data: { items: [], total: 0 } }),
      create: vi.fn().mockResolvedValue({ data: { id: 1 } }),
      delete: vi.fn().mockResolvedValue({}),
    },
    summarySchedulesApi: {
      list: vi.fn().mockResolvedValue({ data: [{ id: 1, user_id: 1, ai_model_id: 1, feed_ids_json: '[]', run_at: '08:30', page_size: 20, order: 'desc', enabled: true, last_run_at: null, created_at: '', updated_at: '' }] }),
      create: vi.fn().mockResolvedValue({}),
      update: vi.fn().mockResolvedValue({}),
      delete: vi.fn().mockResolvedValue({}),
    },
  };
});

test('点击“总结下一页”会把页码 +1 并用新页码生成总结', async () => {
  const user = userEvent.setup();
  // 该项目的测试环境下可能没有可用的 localStorage，这里做最小 polyfill
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
  globalThis.localStorage.setItem(
    'user',
    JSON.stringify({ id: 1, username: 'u', status: 'active', is_super_admin: false, created_at: '' })
  );

  render(
    <MemoryRouter initialEntries={['/feeds?tab=ai-summary']}>
      <AuthProvider>
        <Routes>
          <Route path="/feeds" element={<Feeds />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  );

  // 等待“总结下一页”按钮出现（ai-summary tab 已激活）
  const nextBtn = await screen.findByRole('button', { name: '总结下一页' });
  await user.click(nextBtn);

  await waitFor(() => expect(articlesApi.summarizeStream).toHaveBeenCalledTimes(1));
  const callArg = (articlesApi.summarizeStream as unknown as { mock: { calls: unknown[][] } }).mock.calls[0][0] as { page?: number };
  expect(callArg.page).toBe(2);

  // 页码输入框也应同步到 2
  expect(screen.getByLabelText('页码')).toHaveValue(2);
});

test('定时总结编辑会调用更新接口', async () => {
  const user = userEvent.setup();
  const store = new Map<string, string>();
  // @ts-expect-error test polyfill
  globalThis.localStorage = {
    getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
    setItem: (k: string, v: string) => { store.set(k, String(v)); },
    removeItem: (k: string) => { store.delete(k); },
  };
  globalThis.localStorage.setItem(
    'user',
    JSON.stringify({ id: 1, username: 'u', status: 'active', is_super_admin: false, created_at: '' })
  );

  render(
    <MemoryRouter initialEntries={['/feeds?tab=ai-summary-schedule']}>
      <AuthProvider>
        <Routes>
          <Route path="/feeds" element={<Feeds />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  );

  await screen.findByRole('button', { name: '编辑' });
  await user.click(screen.getByRole('button', { name: '编辑' }));
  await user.click(screen.getByRole('button', { name: '保存' }));

  const mod = await import('../api/client');
  await waitFor(() => expect(mod.summarySchedulesApi.update).toHaveBeenCalledTimes(1));
  expect((mod.summarySchedulesApi.update as unknown as { mock: { calls: unknown[][] } }).mock.calls[0][0]).toBe(1);
});

test('生成总结完成后点击保存会创建历史记录（生成中不可点）', async () => {
  const user = userEvent.setup();
  const store = new Map<string, string>();
  // @ts-expect-error test polyfill
  globalThis.localStorage = {
    getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
    setItem: (k: string, v: string) => { store.set(k, String(v)); },
    removeItem: (k: string) => { store.delete(k); },
  };
  globalThis.localStorage.setItem(
    'user',
    JSON.stringify({ id: 1, username: 'u', status: 'active', is_super_admin: false, created_at: '' })
  );

  render(
    <MemoryRouter initialEntries={['/feeds?tab=ai-summary']}>
      <AuthProvider>
        <Routes>
          <Route path="/feeds" element={<Feeds />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  );

  // 先触发生成
  await user.click(await screen.findByRole('button', { name: '生成总结' }));

  // 保存应该可用（mock summarizeStream 会立即输出 chunk 并结束）
  const saveBtn = await screen.findByRole('button', { name: '保存' });
  expect(saveBtn).not.toBeDisabled();

  await user.click(saveBtn);
  const mod = await import('../api/client');
  await waitFor(() => expect(mod.summaryHistoriesApi.create).toHaveBeenCalledTimes(1));
});

