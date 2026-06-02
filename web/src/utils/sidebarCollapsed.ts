const STORAGE_KEY = 'ui.sidebarCollapsed';

function canUseLocalStorage(): boolean {
  if (typeof window === 'undefined') return false;
  const ls = window.localStorage as unknown as { getItem?: unknown; setItem?: unknown };
  return typeof ls?.getItem === 'function' && typeof ls?.setItem === 'function';
}

function defaultCollapsedByViewport(): boolean {
  if (typeof window === 'undefined') return false;
  return window.innerWidth <= 768;
}

export function getStoredSidebarCollapsed(): boolean | null {
  if (!canUseLocalStorage()) return null;
  const stored = window.localStorage.getItem(STORAGE_KEY);
  if (stored === 'true') return true;
  if (stored === 'false') return false;
  return null;
}

export function getInitialSidebarCollapsed(): boolean {
  const stored = getStoredSidebarCollapsed();
  if (stored !== null) return stored;
  return defaultCollapsedByViewport();
}

export function setStoredSidebarCollapsed(collapsed: boolean): void {
  if (!canUseLocalStorage()) return;
  try {
    window.localStorage.setItem(STORAGE_KEY, String(collapsed));
  } catch {
    // ignore storage failures (private mode, etc.)
  }
}
