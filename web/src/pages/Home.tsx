import { useState, useEffect, useRef, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { articlesApi, feedsApi, categoriesApi } from '../api/client';
import type { Article, Feed, FeedCategory } from '../api/client';
import ArticleList from '../components/ArticleList';
import { nextIndex } from '../utils/arrowNav';

const PAGE_SIZE = 20;

export default function Home() {
  const [searchParams, setSearchParams] = useSearchParams();
  const feedParam = searchParams.get('feed');
  const initialFeed = feedParam ? (Number.isNaN(Number(feedParam)) ? '' : Number(feedParam)) : '';
  const clusterParam = searchParams.get('cluster');
  const initialCluster = clusterParam ? (Number.isNaN(Number(clusterParam)) ? '' : Number(clusterParam)) : '';
  const collapsedParam = searchParams.get('collapsed') ?? '';
  const initialCollapsed = new Set(
    collapsedParam.split(',').map((s) => s.trim()).filter(Boolean).map(Number).filter((n) => !Number.isNaN(n))
  );

  const [articles, setArticles] = useState<Article[]>([]);
  const [feeds, setFeeds] = useState<Feed[]>([]);
  const [categories, setCategories] = useState<FeedCategory[]>([]);
  const [sidebarLoading, setSidebarLoading] = useState(true);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [filterFeed, setFilterFeed] = useState<number | ''>(initialFeed);
  const [filterCluster, setFilterCluster] = useState<number | ''>(initialCluster);
  const [filterRead, setFilterRead] = useState<'' | 'read' | 'unread'>('');
  const [query, setQuery] = useState(searchParams.get('q') ?? '');
  const [draftQuery, setDraftQuery] = useState(searchParams.get('q') ?? '');
  const [importance, setImportance] = useState<number | ''>(() => {
    const v = searchParams.get('importance');
    if (!v) return '';
    const parsed = Number(v);
    return Number.isNaN(parsed) ? '' : parsed;
  });
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [selected, setSelected] = useState<Article | null>(null);
  const [collapsedCategories, setCollapsedCategories] = useState<Set<number>>(initialCollapsed);
  const sidebarLoadedRef = useRef(false);
  const listScrollRef = useRef<HTMLDivElement>(null);

  // 串行请求：feeds -> categories -> articles，避免同时请求导致数据库 unexpected EOF
  useEffect(() => {
    let cancelled = false;
    const isFirstPage = page === 1;
    if (isFirstPage) setLoading(true);
    else setLoadingMore(true);

    (async () => {
      if (!sidebarLoadedRef.current) {
        setSidebarLoading(true);
        try {
          const fr = await feedsApi.list();
          if (!cancelled) setFeeds(fr.data);
          const cr = await categoriesApi.list();
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

      const params: {
        feed_id?: number;
        read?: boolean;
        page?: number;
        page_size?: number;
        cluster_id?: number;
        q?: string;
        importance?: number;
        has_ai_metadata?: boolean;
      } = {
        page,
        page_size: PAGE_SIZE,
      };
      if (filterFeed) params.feed_id = filterFeed;
      if (filterCluster) params.cluster_id = filterCluster;
      if (filterRead === 'read') params.read = true;
      if (filterRead === 'unread') params.read = false;
      if (query.trim()) params.q = query.trim();
      if (importance !== '') params.importance = importance;
      params.has_ai_metadata = true;
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
  }, [filterFeed, filterCluster, filterRead, query, importance, page]);

  // URL 变化时同步 filterFeed、collapsedCategories（如浏览器前进/后退）
  useEffect(() => {
    const p = searchParams.get('feed');
    const next = p ? (Number.isNaN(Number(p)) ? '' : Number(p)) : '';
    setFilterFeed((prev) => (prev !== next ? next : prev));
    const cp2 = searchParams.get('cluster');
    const nextCluster = cp2 ? (Number.isNaN(Number(cp2)) ? '' : Number(cp2)) : '';
    setFilterCluster((prev) => (prev !== nextCluster ? nextCluster : prev));
    const nextQuery = searchParams.get('q') ?? '';
    setQuery((prev) => (prev !== nextQuery ? nextQuery : prev));
    setDraftQuery((prev) => (prev !== nextQuery ? nextQuery : prev));
    const iv = searchParams.get('importance');
    const nextImportance = iv ? (Number.isNaN(Number(iv)) ? '' : Number(iv)) : '';
    setImportance((prev) => (prev !== nextImportance ? nextImportance : prev));
    const cp = searchParams.get('collapsed') ?? '';
    const nextCollapsed = new Set(
      cp.split(',').map((s) => s.trim()).filter(Boolean).map(Number).filter((n) => !Number.isNaN(n))
    );
    setCollapsedCategories((prev) => (prev.size !== nextCollapsed.size || [...prev].some((id) => !nextCollapsed.has(id)) ? nextCollapsed : prev));
  }, [searchParams]);

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

  // 当文章列表变化时，如果当前选中的文章不在列表中，清空选择
  useEffect(() => {
    if (!selected) return;
    if (!articles.some((a) => a.id === selected.id)) {
      setSelected(null);
    }
  }, [articles, selected]);

  const markRead = async (id: number) => {
    try {
      await articlesApi.markRead(id);
      setArticles((prev) =>
        prev.map((a) => (a.id === id ? { ...a, read: true } : a))
      );
    } catch {}
  };

  const toggleFavorite = async (id: number) => {
    try {
      const { data } = await articlesApi.toggleFavorite(id);
      setArticles((prev) =>
        prev.map((a) => (a.id === id ? { ...a, favorite: data.favorite } : a))
      );
      if (selected?.id === id) {
        setSelected((prev) => (prev ? { ...prev, favorite: data.favorite } : null));
      }
    } catch {}
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
  const currentCluster = selected?.cluster_title ?? (filterCluster ? `聚类 #${filterCluster}` : '');

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

  // 滚动到底部时加载下一页
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

  // 键盘上下键切换文章详情
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
      const currentIdx = selected ? articles.findIndex((a) => a.id === selected.id) : null;
      const delta = e.key === 'ArrowDown' ? 1 : -1;
      const nextIdx = nextIndex(currentIdx !== null && currentIdx >= 0 ? currentIdx : null, delta as -1 | 1, articles.length);
      if (nextIdx === null) return;
      if (currentIdx !== null && currentIdx === nextIdx) return;

      const nextArticle = articles[nextIdx];
      setSelected(nextArticle);
      markRead(nextArticle.id);

      const el = document.querySelector(`[data-article-id="${nextArticle.id}"]`);
      if (el && 'scrollIntoView' in el) {
        (el as HTMLElement).scrollIntoView({ block: 'nearest' });
      }
    };
    window.addEventListener('keydown', onKeyDown, { passive: false });
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [articles, selected]);

  return (
    <div className="home-layout">
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
            {filterCluster && ` · 当前聚类：${currentCluster}`}
          </div>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              setPage(1);
              setQuery(draftQuery);
              setSearchParams((prev) => {
                const p = new URLSearchParams(prev);
                if (draftQuery.trim()) p.set('q', draftQuery.trim());
                else p.delete('q');
                return p;
              });
            }}
          >
            <input
              type="search"
              placeholder="搜索标题、正文、来源"
              value={draftQuery}
              onChange={(e) => setDraftQuery(e.target.value)}
            />
          </form>
          <select
            aria-label="重要度"
            value={importance}
            onChange={(e) => {
              const value = e.target.value ? Number(e.target.value) : '';
              setImportance(value);
              setPage(1);
              setSearchParams((prev) => {
                const p = new URLSearchParams(prev);
                if (value === '') p.delete('importance');
                else p.set('importance', String(value));
                return p;
              });
            }}
          >
            <option value="">全部重要度</option>
            <option value="2">2+ 星</option>
            <option value="3">3+ 星</option>
            <option value="4">4+ 星</option>
          </select>
          <select
            value={filterRead}
            onChange={(e) => {
              setFilterRead(e.target.value as '' | 'read' | 'unread');
              setPage(1);
            }}
          >
            <option value="">全部</option>
            <option value="read">已读</option>
            <option value="unread">未读</option>
          </select>
        </div>

        <div ref={listScrollRef} className="article-list-scroll">
          {loading ? (
            <p className="loading">加载中...</p>
          ) : articles.length === 0 ? (
            <p className="empty">暂无文章，请先添加订阅</p>
          ) : (
            <>
              <ArticleList
                articles={articles}
                selectedId={selected?.id ?? null}
                onOpen={(a) => {
                  setSelected(a);
                  markRead(a.id);
                }}
              />
              {loadingMore && (
                <p className="loading" style={{ padding: '16px', margin: 0 }}>
                  加载更多...
                </p>
              )}
            </>
          )}
        </div>

        {selected && (
          <div className="article-detail-dock">
            <div className="article-detail-header">
              <a
                className="article-detail-title"
                href={selected.link}
                target="_blank"
                rel="noopener noreferrer"
                title="打开原文"
              >
                {selected.title || '(无标题)'}
              </a>
              <div className="article-detail-actions">
                <button
                  type="button"
                  className={`article-detail-favorite ${selected.favorite ? 'active' : ''}`}
                  onClick={() => toggleFavorite(selected.id)}
                  title={selected.favorite ? '取消收藏' : '收藏'}
                  aria-label={selected.favorite ? '取消收藏' : '收藏'}
                >
                  ★
                </button>
                <button type="button" className="article-detail-close" onClick={() => setSelected(null)}>
                  关闭
                </button>
              </div>
            </div>
            <div className="article-detail-meta">
              {selected.feed_title && <span className="feed">{selected.feed_title}</span>}
              <span className="date">{formatDate(selected.published_at || selected.created_at)}</span>
            </div>
            <div className="article-detail-meta">
              {selected.cluster_title && (
                <button
                  type="button"
                  className="feed"
                  onClick={() => {
                    if (!selected.cluster_id) return;
                    setFilterCluster(selected.cluster_id);
                    setPage(1);
                    setSearchParams((prev) => {
                      const p = new URLSearchParams(prev);
                      p.set('cluster', String(selected.cluster_id));
                      return p;
                    });
                  }}
                >
                  {selected.cluster_title}
                </button>
              )}
              {selected.importance ? <span className="date">重要度 {selected.importance}/5</span> : null}
              {selected.language ? <span className="date">{selected.language.toUpperCase()}</span> : null}
              {selected.sentiment ? <span className="date">{selected.sentiment}</span> : null}
            </div>
            {selected.ai_summary && <div className="feeds-summary-result-content">{selected.ai_summary}</div>}
            {(selected.tags?.length || selected.topics?.length) ? (
              <div className="article-detail-meta">
                {[...(selected.topics ?? []), ...(selected.tags ?? [])].slice(0, 8).map((item) => (
                  <span key={item} className="feed">{item}</span>
                ))}
              </div>
            ) : null}
            <div
              className="article-detail-content"
              dangerouslySetInnerHTML={{ __html: selected.content || '<p>(暂无内容)</p>' }}
            />
          </div>
        )}
      </section>
    </div>
  );
}
