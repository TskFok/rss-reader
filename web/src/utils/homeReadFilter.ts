export type ReadFilter = '' | 'read' | 'unread';

export function parseReadFilterParam(param: string | null): ReadFilter {
  if (param === 'read') return 'read';
  if (param === 'all') return '';
  return 'unread';
}

/** 未读为默认态，返回 null 表示从 URL 中省略 read 参数 */
export function readFilterToParam(filter: ReadFilter): string | null {
  if (filter === 'read') return 'read';
  if (filter === '') return 'all';
  return null;
}

export function applyReadFilterToSearchParams(
  params: URLSearchParams,
  filter: ReadFilter
): URLSearchParams {
  const next = new URLSearchParams(params);
  const value = readFilterToParam(filter);
  if (value === null) next.delete('read');
  else next.set('read', value);
  return next;
}
