import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import ErrorLogs from './ErrorLogs';
import { errorLogsApi } from '../api/client';

vi.mock('../api/client', async () => {
  return {
    errorLogsApi: {
      list: vi.fn().mockResolvedValue({
        data: {
          items: [
            {
              id: 1,
              user_id: 1,
              level: 'error',
              message: 'boom',
              location: 'GET /api/x',
              method: 'GET',
              path: '/api/x',
              status: 500,
              stack: 'stack',
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

test('错误日志页删除会调用删除接口', async () => {
  const user = userEvent.setup();

  const store = new Map<string, string>();
  // @ts-expect-error test polyfill
  globalThis.localStorage = {
    getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
    setItem: (k: string, v: string) => { store.set(k, String(v)); },
    removeItem: (k: string) => { store.delete(k); },
  };
  globalThis.localStorage.setItem('user', JSON.stringify({ id: 1, username: 'u', status: 'active', is_super_admin: false, created_at: '' }));

  // @ts-expect-error test mock
  globalThis.confirm = vi.fn().mockReturnValue(true);

  render(
    <MemoryRouter initialEntries={['/error-logs']}>
      <AuthProvider>
        <Routes>
          <Route path="/error-logs" element={<ErrorLogs />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  );

  await screen.findByText('boom');
  await user.click(screen.getByRole('button', { name: '删除' }));

  await waitFor(() => expect(errorLogsApi.delete).toHaveBeenCalledTimes(1));
  expect((errorLogsApi.delete as unknown as { mock: { calls: unknown[][] } }).mock.calls[0][0]).toBe(1);
});

