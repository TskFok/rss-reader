import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { ThemeProvider } from '../contexts/ThemeContext';
import { ToastProvider } from '../contexts/ToastContext';
import type { Article, ArticleListItem, Feed } from '../api/client';
import { articlesApi, feedsApi } from '../api/client';
import Home from './Home';

const { feedOne, feedTwo, sharedArticle, articleTwo, detailArticle, detailArticleTwo } = vi.hoisted(() => {
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
  const listArticle: ArticleListItem = {
    id: 101,
    feed_id: 1,
    guid: 'guid-101',
    title: '长文',
    link: 'http://example.com/p/101',
    published_at: null,
    created_at: '2020-01-01T00:00:00Z',
    read: false,
    feed_title: '订阅一',
  };
  const article: Article = {
    ...listArticle,
    content: '<p>' + '行<br/>'.repeat(400) + '</p>',
  };
  const second: ArticleListItem = {
    ...listArticle,
    id: 102,
    guid: 'guid-102',
    title: '另一篇',
    link: 'http://example.com/p/102',
    read: false,
  };
  const articleTwo: Article = { ...article, ...second };
  return { feedOne: one, feedTwo: two, sharedArticle: listArticle, articleTwo: second, detailArticle: article, detailArticleTwo: articleTwo };
});

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    feedsApi: {
      ...actual.feedsApi,
      list: vi.fn().mockResolvedValue({ data: [feedOne, feedTwo] }),
      refresh: vi.fn().mockResolvedValue({ data: feedOne }),
    },
    categoriesApi: {
      ...actual.categoriesApi,
      list: vi.fn().mockResolvedValue({ data: [] }),
    },
    articlesApi: {
      ...actual.articlesApi,
      list: vi.fn().mockResolvedValue({ data: { items: [sharedArticle, articleTwo], total: 2 } }),
      get: vi.fn().mockImplementation((id: number) => {
        const article = id === sharedArticle.id ? detailArticle : detailArticleTwo;
        return Promise.resolve({ data: { article } });
      }),
      markRead: vi.fn().mockResolvedValue({}),
      toggleFavorite: vi.fn().mockResolvedValue({ data: { favorite: true } }),
    },
  };
});

test('阅读页选中订阅时可以立即刷新当前订阅', async () => {
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
  const listCallsBefore = vi.mocked(articlesApi.list).mock.calls.length;

  render(
    <MemoryRouter initialEntries={['/?feed=1']}>
      <ThemeProvider>
        <ToastProvider>
          <Home />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  const refreshBtn = await screen.findByRole('button', { name: '立即刷新当前订阅' });
  await user.click(refreshBtn);

  await waitFor(() => expect(feedsApi.refresh).toHaveBeenCalledWith(1));
  await waitFor(() => {
    expect(vi.mocked(articlesApi.list).mock.calls.length).toBeGreaterThan(listCallsBefore + 1);
  });
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

  const dock = container.querySelector('.article-detail-scroll') as HTMLDivElement;
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

  const dock = container.querySelector('.article-detail-scroll') as HTMLDivElement;
  expect(dock).not.toBeNull();
  dock.scrollTop = 420;
  expect(dock.scrollTop).toBeGreaterThan(0);

  await user.click(screen.getByRole('button', { name: /另一篇/ }));

  await waitFor(() => {
    expect(dock.scrollTop).toBe(0);
  });
});

test('文章详情标题不放在正文滚动容器内', async () => {
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

  const scroll = container.querySelector('.article-detail-scroll') as HTMLDivElement;
  const title = container.querySelector('.article-detail-title') as HTMLAnchorElement;
  expect(scroll).not.toBeNull();
  expect(title).not.toBeNull();
  expect(scroll.contains(title)).toBe(false);
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

  let resolveArticles: (value: { data: { items: ArticleListItem[]; total: number } }) => void = () => {};
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

test('选中文章时请求详情接口', async () => {
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

  render(
    <MemoryRouter initialEntries={['/']}>
      <ThemeProvider>
        <ToastProvider>
          <Home />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  await user.click(await screen.findByRole('button', { name: /长文/ }));

  await waitFor(() => {
    expect(articlesApi.get).toHaveBeenCalledWith(sharedArticle.id, expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });
});

test('列表无正文时通过详情接口渲染正文', async () => {
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

  render(
    <MemoryRouter initialEntries={['/']}>
      <ThemeProvider>
        <ToastProvider>
          <Home />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  await user.click(await screen.findByRole('button', { name: /长文/ }));

  await waitFor(() => {
    const content = document.querySelector('.article-detail-content');
    expect(content?.innerHTML).toContain('行');
  });
});

test('语言切换选项不依赖列表项 content_translated', async () => {
  vi.mocked(articlesApi.list).mockResolvedValueOnce({
    data: {
      items: [{ ...sharedArticle, title_translated: '译文标题' }],
      total: 1,
    },
  });
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

  render(
    <MemoryRouter initialEntries={['/']}>
      <ThemeProvider>
        <ToastProvider>
          <Home />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  expect(await screen.findByText('显示')).toBeInTheDocument();
  const langSelect = screen.getByText('显示').closest('label')?.querySelector('select');
  expect(langSelect).toBeTruthy();
});

test('刷新订阅后重新请求当前选中文章详情', async () => {
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

  let detailContent = '<p>' + '行<br/>'.repeat(400) + '</p>';
  vi.mocked(articlesApi.get).mockImplementation(async () => ({
    data: { article: { ...detailArticle, content: detailContent } },
  }));

  render(
    <MemoryRouter initialEntries={['/?feed=1']}>
      <ThemeProvider>
        <ToastProvider>
          <Home />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  await user.click(await screen.findByRole('button', { name: /长文/ }));
  await waitFor(() => {
    expect(document.querySelector('.article-detail-content')?.innerHTML).toContain('行');
  });
  const getCallsBeforeRefresh = vi.mocked(articlesApi.get).mock.calls.length;

  detailContent = '<p>刷新后正文</p>';
  await user.click(await screen.findByRole('button', { name: '立即刷新当前订阅' }));

  await waitFor(() => {
    expect(vi.mocked(articlesApi.get).mock.calls.length).toBeGreaterThan(getCallsBeforeRefresh);
  });
  await waitFor(() => {
    expect(document.querySelector('.article-detail-content')?.innerHTML).toContain('刷新后正文');
  });
});
