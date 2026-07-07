import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ArticleList from './ArticleList';
import type { ArticleListItem } from '../api/client';

function makeArticle(overrides: Partial<ArticleListItem> = {}): ArticleListItem {
  return {
    id: 1,
    feed_id: 1,
    guid: 'g',
    title: '标题1',
    link: 'https://example.com/a',
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

  await user.click(screen.getByRole('button', { name: /标题1/ }));
  expect(onOpen).toHaveBeenCalledTimes(1);
  expect(onOpen.mock.calls[0][0].link).toBe('https://example.com/a');
});

test('点击行内非标题区域（如订阅名）也会触发 onOpen', async () => {
  const user = userEvent.setup();
  const onOpen = vi.fn();
  render(<ArticleList articles={[makeArticle({ feed_title: '某订阅' })]} selectedId={null} onOpen={onOpen} />);

  await user.click(screen.getByText('某订阅'));
  expect(onOpen).toHaveBeenCalledTimes(1);
  expect(onOpen.mock.calls[0][0].feed_title).toBe('某订阅');
});

test('displayLang=translated 时展示译文标题', () => {
  render(
    <ArticleList
      articles={[makeArticle({ title: '原文', title_translated: '译文标题' })]}
      selectedId={null}
      displayLang="translated"
      onOpen={() => {}}
    />
  );
  expect(screen.getByRole('button', { name: /译文标题/ })).toBeInTheDocument();
});
