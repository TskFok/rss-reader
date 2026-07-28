import { render, screen } from '@testing-library/react';
import ArticleOriginalWebpageButton from './ArticleOriginalWebpageButton';

test('按当前视图显示对应的图标按钮语义', () => {
  const { rerender } = render(
    <ArticleOriginalWebpageButton showOriginalWebpage={false} onClick={() => {}} />
  );

  expect(screen.getByRole('button', { name: '查看原始网页' })).toHaveAttribute(
    'title',
    '查看原始网页'
  );
  expect(screen.getByRole('button', { name: '查看原始网页' }).querySelector('svg')).toBeInTheDocument();

  rerender(<ArticleOriginalWebpageButton showOriginalWebpage onClick={() => {}} />);

  expect(screen.getByRole('button', { name: '返回正文' })).toHaveAttribute('title', '返回正文');
  expect(screen.getByRole('button', { name: '返回正文' }).querySelector('svg')).toBeInTheDocument();
});
