import { act, renderHook, waitFor } from '@testing-library/react';
import type { Article, ArticleListItem } from '../api/client';
import { articlesApi } from '../api/client';
import {
  clearArticleDetailCache,
  getArticleDetailCacheForTest,
  useArticleDetail,
} from './useArticleDetail';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    articlesApi: {
      ...actual.articlesApi,
      get: vi.fn(),
    },
  };
});

const listItem: ArticleListItem = {
  id: 1,
  feed_id: 10,
  guid: 'g1',
  title: '标题',
  link: 'https://example.com/a',
  published_at: null,
  created_at: '2026-01-01T00:00:00Z',
  read: false,
};

const detail: Article = {
  ...listItem,
  content: '<p>正文 A</p>',
};

const detailB: Article = {
  ...listItem,
  id: 2,
  guid: 'g2',
  title: '标题 B',
  content: '<p>正文 B</p>',
};

const listItemB: ArticleListItem = {
  ...listItem,
  id: 2,
  guid: 'g2',
  title: '标题 B',
};

beforeEach(() => {
  clearArticleDetailCache();
  vi.mocked(articlesApi.get).mockReset();
});

test('clearCache refetch 会清空 selectedDetail 并重新请求详情', async () => {
  vi.mocked(articlesApi.get)
    .mockResolvedValueOnce({ data: { article: detail } })
    .mockResolvedValueOnce({ data: { article: { ...detail, content: '<p>刷新后正文</p>' } } });

  const { result } = renderHook(() => useArticleDetail());

  act(() => {
    result.current.selectArticle(listItem);
  });
  await waitFor(() => expect(result.current.selectedDetail?.content).toBe('<p>正文 A</p>'));
  expect(articlesApi.get).toHaveBeenCalledTimes(1);

  act(() => {
    result.current.clearCache({ refetch: true });
  });

  expect(result.current.selectedDetail).toBeNull();
  await waitFor(() => expect(result.current.selectedDetail?.content).toBe('<p>刷新后正文</p>'));
  expect(articlesApi.get).toHaveBeenCalledTimes(2);
});

test('快速切换文章时仅最新选中项写回详情', async () => {
  let resolveA: (v: { data: { article: Article } }) => void = () => {};
  const pendingA = new Promise<{ data: { article: Article } }>((resolve) => {
    resolveA = resolve;
  });

  vi.mocked(articlesApi.get).mockImplementation((id: number) => {
    if (id === 1) return pendingA;
    return Promise.resolve({ data: { article: detailB } });
  });

  const { result } = renderHook(() => useArticleDetail());

  act(() => {
    result.current.selectArticle(listItem);
    result.current.selectArticle(listItemB);
  });

  await waitFor(() => expect(result.current.selectedDetail?.content).toBe('<p>正文 B</p>'));

  await act(async () => {
    resolveA({ data: { article: detail } });
    await pendingA;
  });

  expect(result.current.selectedDetail?.content).toBe('<p>正文 B</p>');
});

test('详情 404 展示错误且重试会再次请求', async () => {
  vi.mocked(articlesApi.get)
    .mockRejectedValueOnce({ response: { status: 404 } })
    .mockResolvedValueOnce({ data: { article: detail } });

  const { result } = renderHook(() => useArticleDetail());

  act(() => {
    result.current.selectArticle(listItem);
  });

  await waitFor(() => expect(result.current.detailError).toBe('文章不存在'));
  expect(getArticleDetailCacheForTest().has(1)).toBe(false);

  act(() => {
    result.current.retryDetail();
  });

  await waitFor(() => expect(result.current.selectedDetail?.content).toBe('<p>正文 A</p>'));
  expect(articlesApi.get).toHaveBeenCalledTimes(2);
});

test('clearArticleDetailCache 清空模块级缓存', () => {
  getArticleDetailCacheForTest().set(1, detail);
  clearArticleDetailCache();
  expect(getArticleDetailCacheForTest().size).toBe(0);
});
