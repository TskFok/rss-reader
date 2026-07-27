export type HomeLayout = 'default' | 'detail-centered';

const STORAGE_KEY = 'home.layout';

export function getStoredHomeLayout(): HomeLayout {
  if (typeof window === 'undefined') return 'default';

  try {
    return window.localStorage.getItem(STORAGE_KEY) === 'detail-centered'
      ? 'detail-centered'
      : 'default';
  } catch {
    return 'default';
  }
}

export function setStoredHomeLayout(layout: HomeLayout): void {
  if (typeof window === 'undefined') return;

  try {
    window.localStorage.setItem(STORAGE_KEY, layout);
  } catch {
    // 浏览器禁止存储时，当前会话仍可使用已选布局。
  }
}
