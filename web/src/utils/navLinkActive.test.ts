import { describe, expect, it } from 'vitest';
import { navLinkIsActive } from './navLinkActive';

describe('navLinkIsActive', () => {
  it('精确匹配无查询的 path', () => {
    expect(navLinkIsActive('/', '/', '')).toBe(true);
    expect(navLinkIsActive('/favorites', '/favorites', '')).toBe(true);
  });

  it('子路径前缀匹配（排除仅 /）', () => {
    expect(navLinkIsActive('/feeds', '/feeds/categories', '')).toBe(true);
    expect(navLinkIsActive('/', '/feeds', '')).toBe(false);
  });

  it('带查询的 to 需 path 与 search 均一致', () => {
    expect(navLinkIsActive('/feeds?tab=ai-summary', '/feeds', '?tab=ai-summary')).toBe(true);
    expect(navLinkIsActive('/feeds?tab=ai-summary', '/feeds', '?tab=feeds')).toBe(false);
    expect(navLinkIsActive('/feeds?tab=ai-summary', '/favorites', '?tab=ai-summary')).toBe(false);
  });
});
