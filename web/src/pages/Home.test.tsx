import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { ThemeProvider } from '../contexts/ThemeContext';
import { ToastProvider } from '../contexts/ToastContext';
import type { Article, Feed } from '../api/client';
import { articlesApi } from '../api/client';
import Home from './Home';

const { feedOne, feedTwo, sharedArticle, articleTwo } = vi.hoisted(() => {
  const one: Feed = {
    id: 1,
    user_id: 1,
    category_id: null,
    proxy_id: null,
    ai_model_id: null,
    ai_classify_enabled: false,
    ai_translate_enabled: false,
    ai_target_language: 'zh-CN',
    url: 'http://example.com/1.xml',
    title: '订阅一',
    update_interval_minutes: 60,
    expire_days: 90,
    last_fetched_at: null,
    created_at: '2020-01-01T00:00:00Z',
  };
  const two: Feed = {
    ...one,
    id: 2,
    url: 'http://example.com/2.xml',
    title: '订阅二',
  };
  const article: Article = {
    id: 101,
    feed_id: 1,
    guid: 'guid-101',
    title: '长文',
    link: 'http://example.com/p/101',
    content: '<p>' + '行<br/>'.repeat(400) + '</p>',
    published_at: null,
    created_at: '2020-01-01T00:00:00Z',
    read: false,
    feed_title: '订阅一',
  };
  const second: Article = {
    ...article,
    id: 102,
    guid: 'guid-102',
    title: '另一篇',
    link: 'http://example.com/p/102',
    read: false,
  };
  return { feedOne: one, feedTwo: two, sharedArticle: article, articleTwo: second };
});

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    feedsApi: {
      ...actual.feedsApi,
      list: vi.fn().mockResolvedValue({ data: [feedOne, feedTwo] }),
    },
    categoriesApi: {
      ...actual.categoriesApi,
      list: vi.fn().mockResolvedValue({ data: [] }),
    },
    articlesApi: {
      ...actual.articlesApi,
      list: vi.fn().mockResolvedValue({ data: { items: [sharedArticle, articleTwo], total: 2 } }),
      markRead: vi.fn().mockResolvedValue({}),
      toggleFavorite: vi.fn().mockResolvedValue({ data: { favorite: true } }),
    },
  };
});

test('切换侧栏订阅筛选时重置文章详情面板滚动位置', async () => {
  const user = userEvent.setup();
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

  const { container } = render(
    <MemoryRouter initialEntries={['/?feed=1']}>
      <ThemeProvider>
        <ToastProvider>
          <Home />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  await user.click(await screen.findByRole('button', { name: /长文/ }));

  const dock = container.querySelector('.article-detail-dock') as HTMLDivElement;
  expect(dock).not.toBeNull();
  dock.scrollTop = 420;
  expect(dock.scrollTop).toBeGreaterThan(0);

  await user.click(screen.getByRole('button', { name: '订阅二' }));

  await waitFor(() => {
    expect(dock.scrollTop).toBe(0);
  });
});

test('在文章列表中切换另一篇文章时重置文章详情面板滚动位置', async () => {
  const user = userEvent.setup();
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

  const { container } = render(
    <MemoryRouter initialEntries={['/']}>
      <ThemeProvider>
        <ToastProvider>
          <Home />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  await user.click(await screen.findByRole('button', { name: /长文/ }));

  const dock = container.querySelector('.article-detail-dock') as HTMLDivElement;
  expect(dock).not.toBeNull();
  dock.scrollTop = 420;
  expect(dock.scrollTop).toBeGreaterThan(0);

  await user.click(screen.getByRole('button', { name: /另一篇/ }));

  await waitFor(() => {
    expect(dock.scrollTop).toBe(0);
  });
});

test('刷新后首屏文章加载完成时重置文章列表滚动位置', async () => {
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

  let resolveArticles: (value: { data: { items: Article[]; total: number } }) => void = () => {};
  vi.mocked(articlesApi.list).mockImplementationOnce(
    () => new Promise((resolve) => {
      resolveArticles = resolve;
    })
  );
  const listCallsBefore = vi.mocked(articlesApi.list).mock.calls.length;

  const { container } = render(
    <MemoryRouter initialEntries={['/?feed=1']}>
      <ThemeProvider>
        <ToastProvider>
          <Home />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  const listScroll = container.querySelector('.article-list-scroll') as HTMLDivElement;
  expect(listScroll).not.toBeNull();
  listScroll.scrollTop = 420;

  await waitFor(() => {
    expect(vi.mocked(articlesApi.list).mock.calls.length).toBeGreaterThan(listCallsBefore);
  });

  await act(async () => {
    resolveArticles({ data: { items: [sharedArticle, articleTwo], total: 2 } });
  });

  await screen.findByRole('button', { name: /长文/ });
  expect(listScroll.scrollTop).toBe(0);
});
