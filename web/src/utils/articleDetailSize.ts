export type ArticleDetailSize = {
  widthPercent: number;
  height: number;
};

const STORAGE_KEY = 'article.detail.size';
const MIN_WIDTH_PERCENT = 40;
const MAX_WIDTH_PERCENT = 75;
const MIN_HEIGHT = 280;

export const DEFAULT_ARTICLE_DETAIL_SIZE: ArticleDetailSize = {
  widthPercent: 62,
  height: 520,
};

export function clampArticleDetailSize(size: ArticleDetailSize, maxHeight: number): ArticleDetailSize {
  const safeMaxHeight = Math.max(MIN_HEIGHT, Math.round(maxHeight));
  return {
    widthPercent: Math.min(MAX_WIDTH_PERCENT, Math.max(MIN_WIDTH_PERCENT, Math.round(size.widthPercent))),
    height: Math.min(safeMaxHeight, Math.max(MIN_HEIGHT, Math.round(size.height))),
  };
}

export function getStoredArticleDetailSize(maxHeight: number): ArticleDetailSize {
  const fallback = clampArticleDetailSize(DEFAULT_ARTICLE_DETAIL_SIZE, maxHeight);
  if (typeof window === 'undefined') return fallback;

  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (!stored) return fallback;
    const value = JSON.parse(stored) as Partial<ArticleDetailSize>;
    if (!Number.isFinite(value.widthPercent) || !Number.isFinite(value.height)) return fallback;
    return clampArticleDetailSize(value as ArticleDetailSize, maxHeight);
  } catch {
    return fallback;
  }
}

export function setStoredArticleDetailSize(size: ArticleDetailSize): void {
  if (typeof window === 'undefined') return;

  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(size));
  } catch {
    // 浏览器禁止存储时，本次会话仍保留内存中的尺寸。
  }
}
