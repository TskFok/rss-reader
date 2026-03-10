import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ArticleList from './ArticleList';
import type { Article } from '../api/client';

function makeArticle(overrides: Partial<Article> = {}): Article {
  return {
    id: 1,
    feed_id: 1,
    guid: 'g',
    title: '标题1',
    link: 'https://example.com/a',
    content: '<p>内容</p>',
    published_at: null,
    created_at: '2026-01-01T00:00:00Z',
    read: false,
    feed_title: 'Feed',
    ...overrides,
  };
}

test('点击标题会触发 onOpen，并高亮选中项', async () => {
  const user = userEvent.setup();
  const onOpen = vi.fn();
  render(<ArticleList articles={[makeArticle()]} selectedId={null} onOpen={onOpen} />);

  await user.click(screen.getByRole('button', { name: '标题1' }));
  expect(onOpen).toHaveBeenCalledTimes(1);
  expect(onOpen.mock.calls[0][0].link).toBe('https://example.com/a');
});
