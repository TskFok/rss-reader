import { useCallback, useRef, useState } from 'react';
import { articlesApi, type Article, type ArticleListItem } from '../api/client';

const cache = new Map<number, Article>();
let activeAbort: AbortController | null = null;

export type ClearArticleDetailCacheOptions = {
  /** reloadKey 等场景：清空后按当前选中项重新拉取详情 */
  refetch?: boolean;
};

/** 登出或 reloadKey 时清空全部详情缓存（仅 Map；React 状态由 hook clearCache 处理） */
export function clearArticleDetailCache() {
  cache.clear();
  if (activeAbort) {
    activeAbort.abort();
    activeAbort = null;
  }
}

export function removeArticleFromDetailCache(id: number) {
  cache.delete(id);
}

/** 测试用：读取模块级缓存 */
export function getArticleDetailCacheForTest() {
  return cache;
}

export function useArticleDetail() {
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [selectedListItem, setSelectedListItem] = useState<ArticleListItem | null>(null);
  const [selectedDetail, setSelectedDetail] = useState<Article | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const selectedIdRef = useRef<number | null>(null);
  const selectedListItemRef = useRef<ArticleListItem | null>(null);
  selectedListItemRef.current = selectedListItem;

  const selectArticle = useCallback((item: ArticleListItem | null) => {
    if (activeAbort) {
      activeAbort.abort();
      activeAbort = null;
    }
    if (!item) {
      selectedIdRef.current = null;
      setSelectedId(null);
      setSelectedListItem(null);
      setSelectedDetail(null);
      setDetailLoading(false);
      setDetailError(null);
      return;
    }

    selectedIdRef.current = item.id;
    setSelectedId(item.id);
    setSelectedListItem(item);
    setDetailError(null);

    const cached = cache.get(item.id);
    if (cached && item.read) {
      setSelectedDetail(cached);
      setDetailLoading(false);
      return;
    }

    setSelectedDetail(null);
    setDetailLoading(true);

    const ac = new AbortController();
    activeAbort = ac;
    articlesApi
      .get(item.id, { signal: ac.signal })
      .then(({ data }) => {
        if (selectedIdRef.current !== item.id) return;
        cache.set(item.id, data.article);
        setSelectedDetail(data.article);
        setDetailLoading(false);
      })
      .catch((err: { code?: string; response?: { status?: number } }) => {
        if (ac.signal.aborted || err.code === 'ERR_CANCELED') return;
        if (selectedIdRef.current !== item.id) return;
        setDetailLoading(false);
        if (err.response?.status === 404) {
          setDetailError('文章不存在');
        } else {
          setDetailError('加载失败，请重试');
        }
      })
      .finally(() => {
        if (activeAbort === ac) {
          activeAbort = null;
        }
      });
  }, []);

  const patchArticle = useCallback((article: Article) => {
    cache.set(article.id, article);
    if (selectedIdRef.current === article.id) {
      setSelectedDetail(article);
      setSelectedListItem((prev) => (prev?.id === article.id ? { ...prev, ...article } : prev));
    }
  }, []);

  const patchListItem = useCallback((id: number, patch: Partial<ArticleListItem>) => {
    setSelectedListItem((prev) => (prev?.id === id ? { ...prev, ...patch } : prev));
    const cached = cache.get(id);
    if (cached) {
      cache.set(id, { ...cached, ...patch });
      if (selectedIdRef.current === id) {
        setSelectedDetail({ ...cached, ...patch });
      }
    }
  }, []);

  const clearCache = useCallback((options?: ClearArticleDetailCacheOptions) => {
    clearArticleDetailCache();
    setSelectedDetail(null);
    setDetailError(null);
    if (options?.refetch && selectedListItemRef.current) {
      selectArticle(selectedListItemRef.current);
    }
  }, [selectArticle]);

  const removeFromCache = useCallback((id: number) => {
    removeArticleFromDetailCache(id);
  }, []);

  const retryDetail = useCallback(() => {
    if (!selectedListItem) return;
    removeArticleFromDetailCache(selectedListItem.id);
    selectArticle(selectedListItem);
  }, [selectedListItem, selectArticle]);

  return {
    selectedId,
    selectedListItem,
    selectedDetail,
    detailLoading,
    detailError,
    selectArticle,
    patchArticle,
    patchListItem,
    clearCache,
    removeFromCache,
    retryDetail,
  };
}
