import { nextIndex } from './arrowNav';

test('nextIndex: 空列表返回 null', () => {
  expect(nextIndex(null, 1, 0)).toBeNull();
});

test('nextIndex: 未选中时 ArrowDown 选第一条，ArrowUp 选最后一条', () => {
  expect(nextIndex(null, 1, 3)).toBe(0);
  expect(nextIndex(null, -1, 3)).toBe(2);
});

test('nextIndex: 在边界不越界', () => {
  expect(nextIndex(0, -1, 3)).toBe(0);
  expect(nextIndex(2, 1, 3)).toBe(2);
});

test('nextIndex: 正常上下移动', () => {
  expect(nextIndex(0, 1, 3)).toBe(1);
  expect(nextIndex(1, -1, 3)).toBe(0);
});

