import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import SummaryHistory from './SummaryHistory';
import { summaryHistoriesApi } from '../api/client';

vi.mock('../api/client', async () => {
  return {
    summaryHistoriesApi: {
      list: vi.fn().mockResolvedValue({
        data: {
          items: [
            {
              id: 1,
              ai_model_id: 1,
              ai_model_name: 'm',
              start_time: '',
              end_time: '',
              page: 1,
              page_size: 20,
              order: 'desc',
              article_count: 2,
              total: 2,
              content: '总结内容',
              error: '',
              created_at: '2026-03-11T00:00:00Z',
            },
          ],
          total: 1,
        },
      }),
      delete: vi.fn().mockResolvedValue({}),
    },
  };
});

test('删除历史记录会调用删除接口', async () => {
  const user = userEvent.setup();

  const store = new Map<string, string>();
  // @ts-expect-error test polyfill
  globalThis.localStorage = {
    getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
    setItem: (k: string, v: string) => { store.set(k, String(v)); },
    removeItem: (k: string) => { store.delete(k); },
  };
  globalThis.localStorage.setItem('user', JSON.stringify({ id: 1, username: 'u', status: 'active', is_super_admin: false, created_at: '' }));

  // confirm() 默认返回 true
  // @ts-expect-error test mock
  globalThis.confirm = vi.fn().mockReturnValue(true);

  render(
    <MemoryRouter initialEntries={['/summary-history']}>
      <AuthProvider>
        <Routes>
          <Route path="/summary-history" element={<SummaryHistory />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  );

  await screen.findByText('总结内容');
  await user.click(screen.getByRole('button', { name: '删除' }));

  await waitFor(() => expect(summaryHistoriesApi.delete).toHaveBeenCalledTimes(1));
  expect((summaryHistoriesApi.delete as unknown as { mock: { calls: unknown[][] } }).mock.calls[0][0]).toBe(1);
});

