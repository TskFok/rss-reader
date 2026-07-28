import {
  useState,
  useEffect,
  useRef,
  useCallback,
  type CSSProperties,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from 'react';
import { useSearchParams } from 'react-router-dom';
import { articlesApi, feedsApi, categoriesApi } from '../api/client';
import type { ArticleListItem, Feed, FeedCategory } from '../api/client';
import ArticleList from '../components/ArticleList';
import ArticleManualAI from '../components/ArticleManualAI';
import ArticleDetailContent from '../components/ArticleDetailContent';
import ArticleOriginalWebpage from '../components/ArticleOriginalWebpage';
import ArticleOriginalWebpageButton from '../components/ArticleOriginalWebpageButton';
import { useArticleDetail } from '../hooks/useArticleDetail';
import { ensureAiModelsLoaded } from '../hooks/useAiModels';
import { nextIndex } from '../utils/arrowNav';
import {
  articleCategoryForDisplay,
  articleTitleForDisplay,
  getStoredArticleLang,
  setStoredArticleLang,
  type ArticleDisplayLang,
} from '../utils/articleDisplayLang';
import {
  applyReadFilterToSearchParams,
  parseReadFilterParam,
  type ReadFilter,
} from '../utils/homeReadFilter';
import {
  getStoredHomeLayout,
  setStoredHomeLayout,
  type HomeLayout,
} from '../utils/homeLayout';
import {
  clampArticleDetailSize,
  getStoredArticleDetailSize,
  setStoredArticleDetailSize,
  type ArticleDetailSize,
} from '../utils/articleDetailSize';

const PAGE_SIZE = 20;
const DETAIL_MIN_HEIGHT = 280;

function getDetailMaxHeight() {
  if (typeof window === 'undefined') return 520;
  return Math.max(DETAIL_MIN_HEIGHT, window.innerHeight - 180);
}

export default function Home() {
  const [searchParams, setSearchParams] = useSearchParams();
  const feedParam = searchParams.get('feed');
  const initialFeed = feedParam ? (Number.isNaN(Number(feedParam)) ? '' : Number(feedParam)) : '';
  const collapsedParam = searchParams.get('collapsed') ?? '';
  const initialCollapsed = new Set(
    collapsedParam.split(',').map((s) => s.trim()).filter(Boolean).map(Number).filter((n) => !Number.isNaN(n))
  );
  const initialRead = parseReadFilterParam(searchParams.get('read'));

  const [articles, setArticles] = useState<ArticleListItem[]>([]);
  const [feeds, setFeeds] = useState<Feed[]>([]);
  const [categories, setCategories] = useState<FeedCategory[]>([]);
  const [sidebarLoading, setSidebarLoading] = useState(true);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [filterFeed, setFilterFeed] = useState<number | ''>(initialFeed);
  const [filterRead, setFilterRead] = useState<ReadFilter>(initialRead);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [reloadKey, setReloadKey] = useState(0);
  const {
    selectedId,
    selectedListItem,
    selectedDetail,
    detailLoading,
    detailError,
    selectArticle,
    patchArticle,
    patchListItem,
    clearCache,
    retryDetail,
  } = useArticleDetail();
  const [refreshingFeedId, setRefreshingFeedId] = useState<number | null>(null);
  const [refreshError, setRefreshError] = useState('');
  const [collapsedCategories, setCollapsedCategories] = useState<Set<number>>(initialCollapsed);
  const [articleDisplayLang, setArticleDisplayLang] = useState<ArticleDisplayLang>(() =>
    getStoredArticleLang()
  );
  const [showOriginalWebpage, setShowOriginalWebpage] = useState(false);
  const [homeLayout, setHomeLayout] = useState<HomeLayout>(() => getStoredHomeLayout());
  const [articleDetailSize, setArticleDetailSize] = useState<ArticleDetailSize>(() =>
    getStoredArticleDetailSize(getDetailMaxHeight())
  );
  const [isWideViewport, setIsWideViewport] = useState(() =>
    typeof window === 'undefined' || window.innerWidth > 860
  );
  const [resizeAxis, setResizeAxis] = useState<'width' | 'height' | null>(null);
  const sidebarLoadedRef = useRef(false);
  const syncedReadIdRef = useRef<number | null>(null);
  const listScrollRef = useRef<HTMLDivElement>(null);
  const detailDockScrollRef = useRef<HTMLDivElement>(null);
  const readingPanelsRef = useRef<HTMLDivElement>(null);
  const articleDetailSizeRef = useRef(articleDetailSize);
  const resizeAxisRef = useRef<'width' | 'height' | null>(null);

  const updateArticleDetailSize = useCallback((nextSize: ArticleDetailSize) => {
    articleDetailSizeRef.current = nextSize;
    setArticleDetailSize(nextSize);
  }, []);

  useEffect(() => {
    const updateViewport = () => {
      setIsWideViewport(window.innerWidth > 860);
      updateArticleDetailSize(
        clampArticleDetailSize(articleDetailSizeRef.current, getDetailMaxHeight())
      );
    };
    window.addEventListener('resize', updateViewport);
    return () => window.removeEventListener('resize', updateViewport);
  }, [updateArticleDetailSize]);

  useEffect(() => () => {
    document.body.classList.remove('article-detail-resizing');
  }, []);

  useEffect(() => {
    if (reloadKey === 0) return;
    clearCache({ refetch: true });
  }, [reloadKey, clearCache]);

  useEffect(() => {
    let cancelled = false;
    const isFirstPage = page === 1;
    if (isFirstPage) setLoading(true);
    else setLoadingMore(true);

    (async () => {
      if (!sidebarLoadedRef.current) {
        setSidebarLoading(true);
        try {
          const [fr, cr] = await Promise.all([
            feedsApi.list(),
            categoriesApi.list(),
            ensureAiModelsLoaded(),
          ]);
          if (!cancelled) setFeeds(fr.data);
          if (!cancelled) setCategories(cr.data);
        } catch (_) {
          if (!cancelled) {
            setFeeds([]);
            setCategories([]);
          }
        }
        if (!cancelled) setSidebarLoading(false);
        sidebarLoadedRef.current = true;
      }

      const params: { feed_id?: number; read?: boolean; page?: number; page_size?: number } = {
        page,
        page_size: PAGE_SIZE,
      };
      if (filterFeed) params.feed_id = filterFeed;
      if (filterRead === 'read') params.read = true;
      if (filterRead === 'unread') params.read = false;
      try {
        const r = await articlesApi.list(params);
        if (!cancelled) {
          if (isFirstPage) {
            setArticles(r.data.items);
          } else {
            setArticles((prev) => [...prev, ...r.data.items]);
          }
          setTotal(r.data.total);
        }
      } catch (_) {
        if (!cancelled && isFirstPage) setArticles([]);
      } finally {
        if (!cancelled) {
          if (isFirstPage) setLoading(false);
          else setLoadingMore(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [filterFeed, filterRead, page, reloadKey]);

  useEffect(() => {
    const p = searchParams.get('feed');
    const next = p ? (Number.isNaN(Number(p)) ? '' : Number(p)) : '';
    setFilterFeed((prev) => (prev !== next ? next : prev));
    const cp = searchParams.get('collapsed') ?? '';
    const nextCollapsed = new Set(
      cp.split(',').map((s) => s.trim()).filter(Boolean).map(Number).filter((n) => !Number.isNaN(n))
    );
    setCollapsedCategories((prev) => (prev.size !== nextCollapsed.size || [...prev].some((id) => !nextCollapsed.has(id)) ? nextCollapsed : prev));
    const nextRead = parseReadFilterParam(searchParams.get('read'));
    setFilterRead((prev) => (prev !== nextRead ? nextRead : prev));
  }, [searchParams]);

  useEffect(() => {
    const el = detailDockScrollRef.current;
    if (el) el.scrollTop = 0;
  }, [filterFeed]);

  useEffect(() => {
    if (page !== 1 || loading) return;
    const el = listScrollRef.current;
    if (el) el.scrollTop = 0;
  }, [filterFeed, filterRead, loading, page]);

  useEffect(() => {
    const el = detailDockScrollRef.current;
    if (!el || !selectedListItem) return;
    el.scrollTop = 0;
  }, [selectedListItem?.id]);

  useEffect(() => {
    setShowOriginalWebpage(false);
  }, [selectedId, selectedDetail?.id]);

  const toggleCategoryCollapsed = useCallback((categoryId: number) => {
    setCollapsedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(categoryId)) {
        next.delete(categoryId);
      } else {
        next.add(categoryId);
      }
      setSearchParams((sp) => {
        const p = new URLSearchParams(sp);
        if (next.size === 0) {
          p.delete('collapsed');
        } else {
          p.set('collapsed', [...next].sort((a, b) => a - b).join(','));
        }
        return p;
      });
      return next;
    });
  }, [setSearchParams]);

  useEffect(() => {
    if (!selectedListItem) return;
    if (!articles.some((a) => a.id === selectedListItem.id)) {
      selectArticle(null);
    }
  }, [articles, selectedListItem, selectArticle]);

  useEffect(() => {
    if (!selectedDetail?.read) return;
    if (syncedReadIdRef.current === selectedDetail.id) return;
    syncedReadIdRef.current = selectedDetail.id;
    setArticles((prev) =>
      prev.map((a) => (a.id === selectedDetail.id ? { ...a, read: true } : a))
    );
    patchListItem(selectedDetail.id, { read: true });
  }, [selectedDetail, patchListItem]);

  const toggleFavorite = async (id: number) => {
    try {
      const { data } = await articlesApi.toggleFavorite(id);
      setArticles((prev) =>
        prev.map((a) => (a.id === id ? { ...a, favorite: data.favorite } : a))
      );
      patchListItem(id, { favorite: data.favorite });
    } catch {}
  };

  const refreshCurrentFeed = async () => {
    if (!filterFeed || refreshingFeedId !== null) return;
    setRefreshError('');
    setRefreshingFeedId(filterFeed);
    try {
      const { data } = await feedsApi.refresh(filterFeed);
      setFeeds((prev) => prev.map((f) => (f.id === data.id ? data : f)));
      if (page === 1) {
        setReloadKey((v) => v + 1);
      } else {
        setPage(1);
      }
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setRefreshError(msg || '刷新失败');
    } finally {
      setRefreshingFeedId(null);
    }
  };

  const formatDate = (s: string | null) => {
    if (!s) return '';
    const d = new Date(s);
    return d.toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const currentFeed = filterFeed ? feeds.find((f) => f.id === filterFeed) : undefined;
  const showArticleLangToggle =
    feeds.some((f) => f.ai_translate_enabled) ||
    articles.some((a) => !!a.title_translated?.trim());

  const feedsByCategory = categories.map((c) => ({
    category: c,
    feeds: feeds.filter((f) => f.category_id === c.id),
  }));
  const uncategorizedFeeds = feeds.filter((f) => !f.category_id);

  const hasMore = articles.length < total;
  const loadMore = useCallback(() => {
    if (loading || loadingMore || !hasMore) return;
    setPage((p) => p + 1);
  }, [loading, loadingMore, hasMore]);

  useEffect(() => {
    const el = listScrollRef.current;
    if (!el) return;
    const onScroll = () => {
      const { scrollTop, clientHeight, scrollHeight } = el;
      const threshold = 80;
      if (scrollTop + clientHeight >= scrollHeight - threshold) {
        loadMore();
      }
    };
    el.addEventListener('scroll', onScroll);
    return () => el.removeEventListener('scroll', onScroll);
  }, [loadMore]);

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
      if (e.altKey || e.ctrlKey || e.metaKey) return;

      const target = e.target as HTMLElement | null;
      if (target) {
        const tag = target.tagName?.toLowerCase();
        const isTyping =
          tag === 'input' || tag === 'textarea' || tag === 'select' || target.isContentEditable;
        if (isTyping) return;
      }

      if (articles.length === 0) return;

      e.preventDefault();
      const currentIdx = selectedListItem
        ? articles.findIndex((a) => a.id === selectedListItem.id)
        : null;
      const delta = e.key === 'ArrowDown' ? 1 : -1;
      const nextIdx = nextIndex(currentIdx !== null && currentIdx >= 0 ? currentIdx : null, delta as -1 | 1, articles.length);
      if (nextIdx === null) return;
      if (currentIdx !== null && currentIdx === nextIdx) return;

      const nextArticle = articles[nextIdx];
      selectArticle(nextArticle);

      const el = document.querySelector(`[data-article-id="${nextArticle.id}"]`);
      if (el && 'scrollIntoView' in el) {
        (el as HTMLElement).scrollIntoView({ block: 'nearest' });
      }
    };
    window.addEventListener('keydown', onKeyDown, { passive: false });
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [articles, selectedListItem, selectArticle]);

  const openArticle = (a: ArticleListItem) => {
    selectArticle(a);
  };

  const beginDetailResize = (
    axis: 'width' | 'height',
    event: ReactPointerEvent<HTMLDivElement>
  ) => {
    if (homeLayout !== 'detail-centered' || !isWideViewport) return;
    resizeAxisRef.current = axis;
    setResizeAxis(axis);
    event.currentTarget.setPointerCapture?.(event.pointerId);
    document.body.classList.add('article-detail-resizing');
  };

  const resizeDetail = (event: ReactPointerEvent<HTMLDivElement>) => {
    const axis = resizeAxisRef.current;
    const panels = readingPanelsRef.current;
    if (!axis || !panels) return;

    const bounds = panels.getBoundingClientRect();
    if (bounds.width <= 0 || bounds.height <= 0) return;
    const nextSize = clampArticleDetailSize(
      axis === 'width'
        ? {
            ...articleDetailSizeRef.current,
            widthPercent: ((event.clientX - bounds.left) / bounds.width) * 100,
          }
        : {
            ...articleDetailSizeRef.current,
            height: event.clientY - bounds.top,
          },
      getDetailMaxHeight()
    );
    updateArticleDetailSize(nextSize);
  };

  const finishDetailResize = () => {
    if (!resizeAxisRef.current) return;
    resizeAxisRef.current = null;
    setResizeAxis(null);
    document.body.classList.remove('article-detail-resizing');
    setStoredArticleDetailSize(articleDetailSizeRef.current);
  };

  const adjustDetailSizeByKeyboard = (
    axis: 'width' | 'height',
    event: ReactKeyboardEvent<HTMLDivElement>
  ) => {
    const direction =
      axis === 'width'
        ? event.key === 'ArrowRight'
          ? 1
          : event.key === 'ArrowLeft'
            ? -1
            : 0
        : event.key === 'ArrowDown'
          ? 1
          : event.key === 'ArrowUp'
            ? -1
            : 0;
    if (!direction) return;

    event.preventDefault();
    const step = event.shiftKey ? (axis === 'width' ? 5 : 40) : axis === 'width' ? 2 : 20;
    const nextSize = clampArticleDetailSize(
      axis === 'width'
        ? {
            ...articleDetailSizeRef.current,
            widthPercent: articleDetailSizeRef.current.widthPercent + direction * step,
          }
        : {
            ...articleDetailSizeRef.current,
            height: articleDetailSizeRef.current.height + direction * step,
          },
      getDetailMaxHeight()
    );
    updateArticleDetailSize(nextSize);
    setStoredArticleDetailSize(nextSize);
  };

  const canResizeDetail = homeLayout === 'detail-centered' && isWideViewport;
  const readingPanelsStyle = canResizeDetail
    ? ({
        '--article-detail-width': `${articleDetailSize.widthPercent}%`,
        '--article-detail-height': `${articleDetailSize.height}px`,
      } as CSSProperties)
    : undefined;

  return (
    <div className={homeLayout === 'detail-centered' ? 'home-layout home-layout--detail-centered' : 'home-layout'}>
      <aside className="home-sidebar">
        <div className="sidebar-header">订阅</div>
        {sidebarLoading ? (
          <div className="sidebar-empty">加载中...</div>
        ) : (
          <div className="sidebar-tree">
            <button
              type="button"
              className={`sidebar-item ${filterFeed === '' ? 'active' : ''}`}
              onClick={() => {
                setFilterFeed('');
                setPage(1);
                setSearchParams((prev) => {
                  const p = new URLSearchParams(prev);
                  p.delete('feed');
                  return p;
                });
              }}
            >
              全部订阅
            </button>

            {feedsByCategory.map(({ category, feeds: cfeeds }) => {
              const isCollapsed = collapsedCategories.has(category.id);
              return (
              <div key={category.id} className="sidebar-group">
                <button
                  type="button"
                  className={`sidebar-group-title ${isCollapsed ? 'collapsed' : ''}`}
                  onClick={() => toggleCategoryCollapsed(category.id)}
                >
                  <span className="sidebar-group-toggle">{isCollapsed ? '▶' : '▼'}</span>
                  {category.name}
                </button>
                {!isCollapsed && (cfeeds.length === 0 ? (
                  <div className="sidebar-sub-empty">暂无订阅</div>
                ) : (
                  cfeeds.map((f) => (
                    <button
                      key={f.id}
                      type="button"
                      className={`sidebar-sub-item ${filterFeed === f.id ? 'active' : ''}`}
                      onClick={() => {
                        setFilterFeed(f.id);
                        setPage(1);
                        setSearchParams((prev) => {
                          const p = new URLSearchParams(prev);
                          p.set('feed', String(f.id));
                          return p;
                        });
                      }}
                      title={f.title || f.url}
                    >
                      {f.title || f.url}
                    </button>
                  ))
                ))}
              </div>
            );
            })}

            {uncategorizedFeeds.length > 0 && (() => {
              const uncategorizedId = 0;
              const isCollapsed = collapsedCategories.has(uncategorizedId);
              return (
              <div className="sidebar-group">
                <button
                  type="button"
                  className={`sidebar-group-title ${isCollapsed ? 'collapsed' : ''}`}
                  onClick={() => toggleCategoryCollapsed(uncategorizedId)}
                >
                  <span className="sidebar-group-toggle">{isCollapsed ? '▶' : '▼'}</span>
                  未分类
                </button>
                {!isCollapsed && uncategorizedFeeds.map((f) => (
                  <button
                    key={f.id}
                    type="button"
                    className={`sidebar-sub-item ${filterFeed === f.id ? 'active' : ''}`}
                    onClick={() => {
                      setFilterFeed(f.id);
                      setPage(1);
                      setSearchParams((prev) => {
                        const p = new URLSearchParams(prev);
                        p.set('feed', String(f.id));
                        return p;
                      });
                    }}
                    title={f.title || f.url}
                  >
                    {f.title || f.url}
                  </button>
                ))}
              </div>
            );
            })()}

            {categories.length === 0 && feeds.length === 0 && (
              <div className="sidebar-empty">暂无订阅，请先添加订阅</div>
            )}
          </div>
        )}
      </aside>

      <section className="home-content">
        <div className="filters">
          <div className="home-current-feed">
            {currentFeed ? `当前订阅：${currentFeed.title || currentFeed.url}` : '当前订阅：全部订阅'}
          </div>
          {currentFeed && (
            <button
              type="button"
              className="home-refresh-btn"
              onClick={refreshCurrentFeed}
              disabled={refreshingFeedId === currentFeed.id}
              aria-label="立即刷新当前订阅"
              title="立即刷新当前订阅"
            >
              {refreshingFeedId === currentFeed.id ? '刷新中...' : '立即刷新'}
            </button>
          )}
          {refreshError && <span className="home-refresh-error">{refreshError}</span>}
          {showArticleLangToggle && (
            <label className="home-lang-toggle">
              显示
              <select
                value={articleDisplayLang}
                onChange={(e) => {
                  const v = e.target.value as ArticleDisplayLang;
                  setStoredArticleLang(v);
                  setArticleDisplayLang(v);
                }}
              >
                <option value="original">原文</option>
                <option value="translated">译文</option>
              </select>
            </label>
          )}
          <select
            aria-label="阅读布局"
            value={homeLayout}
            onChange={(event) => {
              const next = event.target.value as HomeLayout;
              setHomeLayout(next);
              setStoredHomeLayout(next);
            }}
          >
            <option value="default">默认布局</option>
            <option value="detail-centered">详情居中</option>
          </select>
          <select
            aria-label="文章状态筛选"
            value={filterRead}
            onChange={(e) => {
              const v = e.target.value as ReadFilter;
              setFilterRead(v);
              setPage(1);
              setSearchParams((prev) => applyReadFilterToSearchParams(prev, v));
            }}
          >
            <option value="">全部</option>
            <option value="read">已读</option>
            <option value="unread">未读</option>
          </select>
        </div>

        <div ref={readingPanelsRef} className="home-reading-panels" style={readingPanelsStyle}>
        <div ref={listScrollRef} className="article-list-scroll">
          {loading ? (
            <p className="loading">加载中...</p>
          ) : articles.length === 0 ? (
            <p className="empty">暂无文章，请先添加订阅</p>
          ) : (
            <>
              <ArticleList
                articles={articles}
                selectedId={selectedId}
                displayLang={articleDisplayLang}
                onOpen={openArticle}
              />
              {loadingMore && (
                <p className="loading" style={{ padding: '16px', margin: 0 }}>
                  加载更多...
                </p>
              )}
            </>
          )}
        </div>

        {selectedListItem && (
          <div className={`article-detail-dock${resizeAxis ? ` article-detail-dock--resizing-${resizeAxis}` : ''}`}>
            <div className="article-detail-header">
              <a
                className="article-detail-title"
                href={selectedListItem.link}
                target="_blank"
                rel="noopener noreferrer"
                title="打开原文"
              >
                {articleTitleForDisplay(selectedListItem, articleDisplayLang)}
              </a>
              <div className="article-detail-actions">
                {selectedDetail && (
                  <ArticleOriginalWebpageButton
                    showOriginalWebpage={showOriginalWebpage}
                    onClick={() => setShowOriginalWebpage((value) => !value)}
                  />
                )}
                {selectedDetail && (
                  <ArticleManualAI
                    article={selectedDetail}
                    onTranslateStart={() => {
                      setStoredArticleLang('translated');
                      setArticleDisplayLang('translated');
                    }}
                    onArticlePatched={(next) => {
                      if (next.id !== selectedId) return;
                      setArticles((prev) =>
                        prev.map((x) => (x.id === next.id ? { ...x, ...next } : x))
                      );
                      patchArticle(next);
                    }}
                  />
                )}
                <button
                  type="button"
                  className={`article-detail-favorite ${selectedListItem.favorite ? 'active' : ''}`}
                  onClick={() => toggleFavorite(selectedListItem.id)}
                  title={selectedListItem.favorite ? '取消收藏' : '收藏'}
                  aria-label={selectedListItem.favorite ? '取消收藏' : '收藏'}
                >
                  ★
                </button>
                <button
                  type="button"
                  className="article-detail-close"
                  onClick={() => selectArticle(null)}
                  aria-label="关闭"
                  title="关闭"
                >
                  <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="m6 6 12 12M18 6 6 18" />
                  </svg>
                </button>
              </div>
            </div>
            <div ref={detailDockScrollRef} className="article-detail-scroll">
              <div className="article-detail-meta">
                {articleCategoryForDisplay(selectedListItem, articleDisplayLang) && (
                  <span className="article-detail-ai-cat">
                    {articleCategoryForDisplay(selectedListItem, articleDisplayLang)}
                  </span>
                )}
                {selectedListItem.feed_title && <span className="feed">{selectedListItem.feed_title}</span>}
                <span className="date">{formatDate(selectedListItem.published_at || selectedListItem.created_at)}</span>
              </div>
              {detailError ? (
                <div className="article-detail-error">
                  <p>{detailError}</p>
                  <button type="button" onClick={retryDetail}>
                    重试
                  </button>
                </div>
              ) : detailLoading ? (
                <p className="loading article-detail-skeleton">加载正文中...</p>
              ) : selectedDetail ? (
                showOriginalWebpage ? (
                  <ArticleOriginalWebpage url={selectedDetail.link} />
                ) : (
                  <ArticleDetailContent article={selectedDetail} displayLang={articleDisplayLang} />
                )
              ) : null}
            </div>
            {canResizeDetail && (
              <>
                <div
                  role="separator"
                  aria-label="调整详情宽度"
                  aria-orientation="vertical"
                  aria-valuemin={40}
                  aria-valuemax={75}
                  aria-valuenow={articleDetailSize.widthPercent}
                  tabIndex={0}
                  className="article-detail-resize-handle article-detail-resize-handle--width"
                  onPointerDown={(event) => beginDetailResize('width', event)}
                  onPointerMove={resizeDetail}
                  onPointerUp={finishDetailResize}
                  onPointerCancel={finishDetailResize}
                  onKeyDown={(event) => adjustDetailSizeByKeyboard('width', event)}
                />
                <div
                  role="separator"
                  aria-label="调整详情高度"
                  aria-orientation="horizontal"
                  aria-valuemin={DETAIL_MIN_HEIGHT}
                  aria-valuemax={getDetailMaxHeight()}
                  aria-valuenow={articleDetailSize.height}
                  tabIndex={0}
                  className="article-detail-resize-handle article-detail-resize-handle--height"
                  onPointerDown={(event) => beginDetailResize('height', event)}
                  onPointerMove={resizeDetail}
                  onPointerUp={finishDetailResize}
                  onPointerCancel={finishDetailResize}
                  onKeyDown={(event) => adjustDetailSizeByKeyboard('height', event)}
                />
              </>
            )}
          </div>
        )}
        </div>
      </section>
    </div>
  );
}
