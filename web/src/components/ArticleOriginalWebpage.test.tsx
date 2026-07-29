import { render, screen } from '@testing-library/react';
import ArticleOriginalWebpage from './ArticleOriginalWebpage';

test('在详情区域嵌入原始网页并提供新标签页回退链接', () => {
  render(<ArticleOriginalWebpage url="https://example.com/articles/1" />);

  const iframe = screen.getByLabelText('原始网页', { selector: 'iframe' });
  expect(iframe).not.toHaveAttribute('title');
  expect(iframe).toHaveAttribute('src', 'https://example.com/articles/1');
  expect(iframe).toHaveAttribute('loading', 'lazy');
  expect(iframe).toHaveAttribute('referrerpolicy', 'strict-origin-when-cross-origin');
  expect(screen.getByRole('link', { name: '在新标签页打开原文' })).toHaveAttribute(
    'href',
    'https://example.com/articles/1'
  );
});
