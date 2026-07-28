import { render, screen } from '@testing-library/react';
import ArticleOriginalWebpage from './ArticleOriginalWebpage';

test('在详情区域嵌入原始网页并提供新标签页回退链接', () => {
  render(<ArticleOriginalWebpage url="https://example.com/articles/1" />);

  expect(screen.getByTitle('原始网页')).toHaveAttribute('src', 'https://example.com/articles/1');
  expect(screen.getByTitle('原始网页')).toHaveAttribute('loading', 'lazy');
  expect(screen.getByTitle('原始网页')).toHaveAttribute(
    'referrerpolicy',
    'strict-origin-when-cross-origin'
  );
  expect(screen.getByRole('link', { name: '在新标签页打开原文' })).toHaveAttribute(
    'href',
    'https://example.com/articles/1'
  );
});
