export function nextIndex(currentIndex: number | null, delta: -1 | 1, length: number) {
  if (length <= 0) return null;
  if (currentIndex === null) {
    return delta === 1 ? 0 : length - 1;
  }
  const next = currentIndex + delta;
  if (next < 0 || next >= length) return currentIndex;
  return next;
}

