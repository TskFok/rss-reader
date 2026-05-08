import { render, screen } from '@testing-library/react';
import SummaryMarkdownContent from './SummaryMarkdownContent';

test('纯文本原样展示', () => {
  render(<SummaryMarkdownContent source="一段说明" />);
  expect(screen.getByText('一段说明')).toBeInTheDocument();
});

test('不解析 Markdown：标题与加粗符号保留为字面量', () => {
  const source = `## 小节

这是**重点**。`;
  render(<SummaryMarkdownContent source={source} />);
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
  expect(screen.getByText(/## 小节/)).toBeInTheDocument();
  expect(screen.getByText(/这是\*\*重点\*\*。/)).toBeInTheDocument();
});

test('链接语法不转换为 <a>', () => {
  render(<SummaryMarkdownContent source="[示例](https://example.com/path)" />);
  expect(screen.queryByRole('link')).not.toBeInTheDocument();
  expect(screen.getByText('[示例](https://example.com/path)')).toBeInTheDocument();
});

test('英文产品名与中文同在最外层容器内展示', () => {
  const { container } = render(
    <SummaryMarkdownContent source="官宣将于今年秋季登陆 **Switch 2** 与 **PS 5**。" />
  );
  const root = container.querySelector('.feeds-summary-plain');
  expect(root?.textContent).toBe('官宣将于今年秋季登陆 **Switch 2** 与 **PS 5**。');
});

test('加粗内为窄不间断空格（U+202F）时原文保留', () => {
  const narrowNbsp = '\u202f';
  const { container } = render(
    <SummaryMarkdownContent
      source={`官宣将于今年秋季登陆 **Switch${narrowNbsp}2** 与 **PS${narrowNbsp}5**。`}
    />
  );
  expect(container.querySelector('.feeds-summary-plain')?.textContent).toContain(`Switch${narrowNbsp}2`);
});
