import { afterEach, describe, expect, it } from 'vitest';
import {
  getInitialSidebarCollapsed,
  getStoredSidebarCollapsed,
  setStoredSidebarCollapsed,
} from './sidebarCollapsed';

const STORAGE_KEY = 'ui.sidebarCollapsed';

describe('sidebarCollapsed storage', () => {
  const originalLocalStorage = window.localStorage;
  const originalInnerWidth = window.innerWidth;

  afterEach(() => {
    Object.defineProperty(window, 'localStorage', { value: originalLocalStorage, configurable: true });
    Object.defineProperty(window, 'innerWidth', { value: originalInnerWidth, configurable: true });
  });

  it('无存储值时，桌面端默认展开', () => {
    const store = new Map<string, string>();
    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
        setItem: (k: string, v: string) => {
          store.set(k, String(v));
        },
      },
      configurable: true,
    });
    Object.defineProperty(window, 'innerWidth', { value: 1200, configurable: true });

    expect(getInitialSidebarCollapsed()).toBe(false);
  });

  it('无存储值时，窄屏默认收起', () => {
    const store = new Map<string, string>();
    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
        setItem: (k: string, v: string) => {
          store.set(k, String(v));
        },
      },
      configurable: true,
    });
    Object.defineProperty(window, 'innerWidth', { value: 768, configurable: true });

    expect(getInitialSidebarCollapsed()).toBe(true);
  });

  it('从 localStorage 恢复收起状态', () => {
    const store = new Map<string, string>([[STORAGE_KEY, 'true']]);
    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
        setItem: (k: string, v: string) => {
          store.set(k, String(v));
        },
      },
      configurable: true,
    });
    Object.defineProperty(window, 'innerWidth', { value: 1200, configurable: true });

    expect(getStoredSidebarCollapsed()).toBe(true);
    expect(getInitialSidebarCollapsed()).toBe(true);
  });

  it('从 localStorage 恢复展开状态', () => {
    const store = new Map<string, string>([[STORAGE_KEY, 'false']]);
    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
        setItem: (k: string, v: string) => {
          store.set(k, String(v));
        },
      },
      configurable: true,
    });
    Object.defineProperty(window, 'innerWidth', { value: 375, configurable: true });

    expect(getStoredSidebarCollapsed()).toBe(false);
    expect(getInitialSidebarCollapsed()).toBe(false);
  });

  it('写入 localStorage', () => {
    const store = new Map<string, string>();
    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
        setItem: (k: string, v: string) => {
          store.set(k, String(v));
        },
      },
      configurable: true,
    });

    setStoredSidebarCollapsed(true);
    expect(store.get(STORAGE_KEY)).toBe('true');

    setStoredSidebarCollapsed(false);
    expect(store.get(STORAGE_KEY)).toBe('false');
  });
});
