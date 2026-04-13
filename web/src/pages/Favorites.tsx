import { useState, useEffect, useRef, useCallback } from 'react';
import { articlesApi } from '../api/client';
import type { Article } from '../api/client';
import ArticleList from '../components/ArticleList';
import ArticleManualAI from '../components/ArticleManualAI';
import { nextIndex } from '../utils/arrowNav';
import {
  articleCategoryForDisplay,
  articleTitleForDisplay,
  getStoredArticleLang,
  setStoredArticleLang,
  type ArticleDisplayLang,
} from '../utils/articleDisplayLang';

const PAGE_SIZE = 20;

export default function Favorites() {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [selected, setSelected] = useState<Article | null>(null);
  const [articleDisplayLang, setArticleDisplayLang] = useState<ArticleDisplayLang>(() =>
    getStoredArticleLang()
  );
  const listScrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    const isFirstPage = page === 1;
    if (isFirstPage) setLoading(true);
    else setLoadingMore(true);

    (async () => {
      try {
        const r = await articlesApi.list({
          favorite: true,
          page,
          page_size: PAGE_SIZE,
        });
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
  }, [page]);

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
      if (!data.favorite) {
        setArticles((prev) => prev.filter((a) => a.id !== id));
        if (selected?.id === id) setSelected(null);
      } else {
        setArticles((prev) =>
          prev.map((a) => (a.id === id ? { ...a, favorite: true } : a))
        );
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

  const showArticleLangToggle =
    articles.some((a) => a.feed_ai_translate_enabled) ||
    articles.some((a) => !!(a.title_translated?.trim() || a.content_translated?.trim()));

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
    <div className="home-layout favorites-layout">
      <section className="home-content favorites-content">
        <div className="filters">
          <div className="home-current-feed">收藏列表</div>
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
        </div>

        <div ref={listScrollRef} className="article-list-scroll">
          {loading ? (
            <p className="loading">加载中...</p>
          ) : articles.length === 0 ? (
            <p className="empty">暂无收藏，点击文章标题旁的星标可收藏</p>
          ) : (
            <>
              <ArticleList
                articles={articles}
                selectedId={selected?.id ?? null}
                displayLang={articleDisplayLang}
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
                {articleTitleForDisplay(selected, articleDisplayLang)}
              </a>
              <div className="article-detail-actions">
                <ArticleManualAI
                  article={selected}
                  onArticlePatched={(next) => {
                    setArticles((prev) =>
                      prev.map((x) => (x.id === next.id ? { ...x, ...next } : x))
                    );
                    setSelected((prev) =>
                      prev?.id === next.id ? { ...prev, ...next } : prev
                    );
                  }}
                />
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
              {articleCategoryForDisplay(selected, articleDisplayLang) && (
                <span className="article-detail-ai-cat">
                  {articleCategoryForDisplay(selected, articleDisplayLang)}
                </span>
              )}
              {selected.feed_title && <span className="feed">{selected.feed_title}</span>}
              <span className="date">{formatDate(selected.published_at || selected.created_at)}</span>
            </div>
            {articleDisplayLang === 'translated' && selected.content_translated?.trim() ? (
              <div className="article-detail-content article-detail-plain">{selected.content_translated}</div>
            ) : (
              <div
                className="article-detail-content"
                dangerouslySetInnerHTML={{ __html: selected.content || '<p>(暂无内容)</p>' }}
              />
            )}
          </div>
        )}
      </section>
    </div>
  );
}
