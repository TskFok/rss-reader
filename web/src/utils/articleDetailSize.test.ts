import { afterEach, beforeEach, expect, test } from 'vitest';
import {
  clampArticleDetailSize,
  getStoredArticleDetailSize,
  setStoredArticleDetailSize,
} from './articleDetailSize';

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
  if (originalLocalStorage) Object.defineProperty(window, 'localStorage', originalLocalStorage);
});

test('将详情尺寸限制在阅读和列表可用的边界内', () => {
  expect(clampArticleDetailSize({ widthPercent: 12, height: 20 }, 720)).toEqual({
    widthPercent: 40,
    height: 280,
  });
  expect(clampArticleDetailSize({ widthPercent: 90, height: 900 }, 720)).toEqual({
    widthPercent: 75,
    height: 720,
  });
});

test('保存后恢复详情尺寸，缺失或无效存储值使用默认尺寸', () => {
  expect(getStoredArticleDetailSize(720)).toEqual({ widthPercent: 62, height: 520 });

  window.localStorage.setItem('article.detail.size', '{"widthPercent":99,"height":1}');
  expect(getStoredArticleDetailSize(720)).toEqual({ widthPercent: 75, height: 280 });

  setStoredArticleDetailSize({ widthPercent: 55, height: 480 });
  expect(getStoredArticleDetailSize(720)).toEqual({ widthPercent: 55, height: 480 });
});

test('受限存储时回退默认尺寸且保存不抛错', () => {
  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    value: {
      getItem: () => { throw new Error('blocked'); },
      setItem: () => { throw new Error('blocked'); },
    } as Storage,
  });

  expect(getStoredArticleDetailSize(720)).toEqual({ widthPercent: 62, height: 520 });
  expect(() => setStoredArticleDetailSize({ widthPercent: 55, height: 480 })).not.toThrow();
});
