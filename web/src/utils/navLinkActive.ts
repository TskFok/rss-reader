/**
 * 判断当前路由是否匹配 React Router 的 `to`（支持 `path` 或 `path?query`）。
 */
export function navLinkIsActive(to: string, pathname: string, search: string): boolean {
  const q = to.indexOf('?');
  if (q >= 0) {
    const path = to.slice(0, q);
    const wantSearch = to.slice(q);
    return pathname === path && search === wantSearch;
  }
  if (pathname === to) return true;
  if (to !== '/' && pathname.startsWith(to)) return true;
  return false;
}
