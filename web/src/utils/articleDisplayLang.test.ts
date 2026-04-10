import { describe, expect, it } from 'vitest';
import { articleCategoryForDisplay, articleTitleForDisplay } from './articleDisplayLang';
import type { Article } from '../api/client';

function baseArticle(overrides: Partial<Article> = {}): Article {
  return {
    id: 1,
    feed_id: 1,
    guid: 'g',
    title: '原文标题',
    link: 'https://example.com',
    content: '<p>x</p>',
    published_at: null,
    created_at: '',
    read: false,
    ...overrides,
  };
}

describe('articleTitleForDisplay', () => {
  it('译文模式优先使用 title_translated', () => {
    const a = baseArticle({ title_translated: 'Translated' });
    expect(articleTitleForDisplay(a, 'translated')).toBe('Translated');
  });

  it('原文模式忽略译文', () => {
    const a = baseArticle({ title: 'T', title_translated: 'Translated' });
    expect(articleTitleForDisplay(a, 'original')).toBe('T');
  });
});

describe('articleCategoryForDisplay', () => {
  it('未开启分类则返回 null', () => {
    const a = baseArticle({ feed_ai_classify_enabled: false, ai_category: 'X' });
    expect(articleCategoryForDisplay(a, 'original')).toBeNull();
  });

  it('译文模式优先 ai_category_translated', () => {
    const a = baseArticle({
      feed_ai_classify_enabled: true,
      ai_category: '科技',
      ai_category_translated: 'Tech',
    });
    expect(articleCategoryForDisplay(a, 'translated')).toBe('Tech');
  });
});
