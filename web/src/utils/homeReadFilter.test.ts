import { describe, expect, it } from 'vitest';
import {
  applyReadFilterToSearchParams,
  parseReadFilterParam,
  readFilterToParam,
} from './homeReadFilter';

describe('homeReadFilter', () => {
  it('解析 URL read 参数', () => {
    expect(parseReadFilterParam(null)).toBe('unread');
    expect(parseReadFilterParam('unread')).toBe('unread');
    expect(parseReadFilterParam('read')).toBe('read');
    expect(parseReadFilterParam('all')).toBe('');
    expect(parseReadFilterParam('invalid')).toBe('unread');
  });

  it('将筛选状态序列化为 URL 参数', () => {
    expect(readFilterToParam('unread')).toBeNull();
    expect(readFilterToParam('read')).toBe('read');
    expect(readFilterToParam('')).toBe('all');
  });

  it('写入或删除 searchParams 中的 read', () => {
    const base = new URLSearchParams('feed=1&collapsed=2');

    expect(applyReadFilterToSearchParams(base, 'unread').toString()).toBe('feed=1&collapsed=2');
    expect(applyReadFilterToSearchParams(base, 'read').toString()).toBe(
      'feed=1&collapsed=2&read=read'
    );
    expect(applyReadFilterToSearchParams(base, '').toString()).toBe('feed=1&collapsed=2&read=all');

    const withRead = new URLSearchParams('feed=1&read=read');
    expect(applyReadFilterToSearchParams(withRead, 'unread').toString()).toBe('feed=1');
  });
});
