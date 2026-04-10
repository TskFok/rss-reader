import type { Article } from '../api/client';

const KEY = 'rss_article_display_lang';

export type ArticleDisplayLang = 'original' | 'translated';

export function getStoredArticleLang(): ArticleDisplayLang {
  if (typeof localStorage === 'undefined') return 'original';
  return localStorage.getItem(KEY) === 'translated' ? 'translated' : 'original';
}

export function setStoredArticleLang(l: ArticleDisplayLang) {
  localStorage.setItem(KEY, l);
}

export function articleTitleForDisplay(a: Article, lang: ArticleDisplayLang): string {
  if (lang === 'translated' && a.title_translated?.trim()) {
    return a.title_translated.trim();
  }
  return a.title || '(无标题)';
}

export function articleCategoryForDisplay(a: Article, lang: ArticleDisplayLang): string | null {
  if (!a.feed_ai_classify_enabled) return null;
  if (lang === 'translated' && a.ai_category_translated?.trim()) {
    return a.ai_category_translated.trim();
  }
  if (a.ai_category?.trim()) return a.ai_category.trim();
  return null;
}
