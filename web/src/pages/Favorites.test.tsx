import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { ThemeProvider } from '../contexts/ThemeContext';
import { ToastProvider } from '../contexts/ToastContext';
import type { Article, ArticleListItem } from '../api/client';
import { aiModelsApi, articlesApi } from '../api/client';
import { clearAiModelsCache } from '../hooks/useAiModels';
import { clearArticleDetailCache } from '../hooks/useArticleDetail';
import Favorites from './Favorites';

const { favoriteArticle, nextFavoriteArticle, favoriteDetails } = vi.hoisted(() => {
  const first: ArticleListItem = {
    id: 101,
    feed_id: 1,
    guid: 'guid-101',
    title: '长文',
    link: 'http://example.com/p/101',
    published_at: null,
    created_at: '2020-01-01T00:00:00Z',
    read: false,
    favorite: true,
    feed_title: '订阅一',
  };
  const second: ArticleListItem = {
    ...first,
    id: 102,
    guid: 'guid-102',
    title: '另一篇',
    link: 'http://example.com/p/102',
  };
  const details: Article[] = [
    { ...first, content: '<p>第一篇正文</p>' },
    { ...second, content: '<p>第二篇正文</p>' },
  ];
  return { favoriteArticle: first, nextFavoriteArticle: second, favoriteDetails: details };
});

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    articlesApi: {
      ...actual.articlesApi,
      list: vi.fn().mockResolvedValue({
        data: { items: [favoriteArticle, nextFavoriteArticle], total: 2 },
      }),
      get: vi.fn().mockImplementation((id: number) => {
        const article = favoriteDetails.find((item) => item.id === id)!;
        return Promise.resolve({ data: { article: { ...article, read: true } } });
      }),
      toggleFavorite: vi.fn().mockResolvedValue({ data: { favorite: true } }),
    },
    aiModelsApi: {
      ...actual.aiModelsApi,
      list: vi.fn().mockResolvedValue({
        data: [{ id: 1, name: 'm', base_url: 'u', user_id: 1, created_at: '', updated_at: '' }],
      }),
    },
  };
});

function mockLocalStorage() {
  const store = new Map<string, string>();
  // @ts-expect-error test polyfill
  globalThis.localStorage = {
    getItem: (key: string) => (store.has(key) ? store.get(key)! : null),
    setItem: (key: string, value: string) => {
      store.set(key, String(value));
    },
    removeItem: (key: string) => {
      store.delete(key);
    },
  };
}

beforeEach(() => {
  mockLocalStorage();
  clearAiModelsCache();
  clearArticleDetailCache();
  vi.mocked(aiModelsApi.list).mockClear();
  vi.mocked(articlesApi.get).mockClear();
  vi.mocked(articlesApi.list).mockClear();
});

function renderFavorites() {
  return render(
    <MemoryRouter>
      <ThemeProvider>
        <ToastProvider>
          <Favorites />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );
}

test('收藏页切换文章时不会保留上一条的原始网页视图', async () => {
  const user = userEvent.setup();
  renderFavorites();

  await user.click(await screen.findByRole('button', { name: /长文/ }));
  await screen.findByText('第一篇正文');
  const originalWebpageTrigger = screen.getByRole('button', { name: '查看原始网页' });
  const manualAiTrigger = screen.getByRole('button', { name: '手动 AI 分类与翻译' });
  expect(originalWebpageTrigger).toHaveClass('article-detail-original-webpage-trigger');
  expect(originalWebpageTrigger.querySelector('svg')).toBeInTheDocument();
  expect(manualAiTrigger).toHaveAttribute('title', '手动 AI 分类与翻译');
  expect(manualAiTrigger.querySelector('svg')).toBeInTheDocument();
  expect(originalWebpageTrigger.compareDocumentPosition(manualAiTrigger)).toBe(
    Node.DOCUMENT_POSITION_FOLLOWING
  );

  await user.click(originalWebpageTrigger);
  expect(screen.getByTitle('原始网页')).toHaveAttribute('src', 'http://example.com/p/101');
  expect(screen.getByRole('button', { name: '返回正文' })).toHaveClass(
    'article-detail-original-webpage-trigger'
  );
  expect(screen.getByRole('button', { name: '返回正文' }).querySelector('svg')).toBeInTheDocument();

  await user.click(screen.getByRole('button', { name: /另一篇/ }));
  await waitFor(() => {
    expect(screen.getByText('第二篇正文')).toBeInTheDocument();
  });
  expect(screen.queryByTitle('原始网页')).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: '查看原始网页' })).toBeInTheDocument();

  const closeButton = screen.getByRole('button', { name: '关闭' });
  expect(closeButton).toHaveAttribute('title', '关闭');
  expect(closeButton.querySelector('svg')).toBeInTheDocument();
  await user.click(closeButton);
  expect(document.querySelector('.article-detail-dock')).not.toBeInTheDocument();
});
