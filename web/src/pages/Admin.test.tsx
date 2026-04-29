import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Admin from './Admin';

const getFeishuBindUrl = vi.fn().mockResolvedValue({
  data: { url: 'https://example.com/bind?x=1' },
});

vi.mock('../api/client', () => ({
  adminApi: {
    listUsers: vi.fn().mockResolvedValue({
      data: [
        {
          id: 1,
          username: 'tester',
          status: 'active',
          is_super_admin: false,
          created_at: '',
          feishu_id: null,
        },
      ],
    }),
    unlockUser: vi.fn(),
    getFeishuBindUrl: (...args: unknown[]) => getFeishuBindUrl(...args),
  },
}));

test('绑定飞书弹窗通过 portal 渲染到 body，避免受系统设置卡片 backdrop 影响', async () => {
  const user = userEvent.setup();
  render(<Admin />);

  await screen.findByRole('button', { name: '绑定飞书' });
  await user.click(screen.getByRole('button', { name: '绑定飞书' }));

  await waitFor(() => {
    expect(getFeishuBindUrl).toHaveBeenCalledWith(1);
  });

  const dialog = await screen.findByRole('dialog', { name: '绑定飞书' });
  expect(document.body).toContainElement(dialog);
});
