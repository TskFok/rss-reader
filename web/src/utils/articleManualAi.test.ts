import { describe, expect, it } from 'vitest';
import type { Article } from '../api/client';
import { articleNeedsClassifySlot, articleNeedsTranslateSlot } from './articleManualAi';

function base(overrides: Partial<Article> = {}): Article {
  return {
    id: 1,
    feed_id: 1,
    guid: 'g',
    title: 't',
    link: 'https://x',
    content: 'c',
    published_at: null,
    created_at: '2026-01-01',
    read: false,
    ...overrides,
  };
}

describe('articleNeedsClassifySlot', () => {
  it('无分类时 true', () => {
    expect(articleNeedsClassifySlot(base({ ai_category: '' }))).toBe(true);
  });
  it('已有分类则 false', () => {
    expect(articleNeedsClassifySlot(base({ ai_category: '科技' }))).toBe(false);
  });
  it('处理中 false', () => {
    expect(articleNeedsClassifySlot(base({ ai_process_status: 'pending' }))).toBe(false);
  });
});

describe('articleNeedsTranslateSlot', () => {
  it('无译文时 true', () => {
    expect(articleNeedsTranslateSlot(base({}))).toBe(true);
  });
  it('已有译文 false', () => {
    expect(articleNeedsTranslateSlot(base({ title_translated: 'T' }))).toBe(false);
  });
  it('处理中 false', () => {
    expect(articleNeedsTranslateSlot(base({ ai_process_status: 'pending' }))).toBe(false);
  });
});
