import { afterEach, beforeEach, expect, test } from 'vitest';
import { getStoredHomeLayout, setStoredHomeLayout } from './homeLayout';

const originalLocalStorage = Object.getOwnPropertyDescriptor(window, 'localStorage');

function installLocalStorage() {
  const values = new Map<string, string>();
  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, String(value)),
      removeItem: (key: string) => values.delete(key),
      clear: () => values.clear(),
    } as Storage,
  });
}

beforeEach(() => {
  installLocalStorage();
});

afterEach(() => {
  if (originalLocalStorage) {
    Object.defineProperty(window, 'localStorage', originalLocalStorage);
  }
});

test('缺失或无效的存储值时使用默认布局', () => {
  expect(getStoredHomeLayout()).toBe('default');

  window.localStorage.setItem('home.layout', 'unknown');
  expect(getStoredHomeLayout()).toBe('default');
});

test('保存布局并在后续读取时恢复', () => {
  setStoredHomeLayout('detail-centered');

  expect(window.localStorage.getItem('home.layout')).toBe('detail-centered');
  expect(getStoredHomeLayout()).toBe('detail-centered');
});

test('存储访问抛出异常时回退默认布局且不抛错', () => {
  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    value: {
      getItem: () => {
        throw new Error('blocked');
      },
      setItem: () => {
        throw new Error('blocked');
      },
    } as Storage,
  });

  expect(getStoredHomeLayout()).toBe('default');
  expect(() => setStoredHomeLayout('detail-centered')).not.toThrow();
});
