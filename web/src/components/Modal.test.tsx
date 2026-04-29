import { fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import Modal from './Modal';

test('弹窗通过 portal 渲染到 body，避免被容器层级影响', () => {
  const host = document.createElement('div');
  document.body.appendChild(host);

  const { container, unmount } = render(
    <Modal open onClose={() => {}} title="测试弹窗">
      <div>弹窗内容</div>
    </Modal>,
    { container: host }
  );

  const dialog = screen.getByRole('dialog');
  expect(dialog).toBeInTheDocument();
  expect(container).not.toContainElement(dialog);
  expect(document.body).toContainElement(dialog);

  unmount();
  document.body.removeChild(host);
});

test('支持点击遮罩和按下 Escape 关闭弹窗', () => {
  const onClose = vi.fn();
  render(
    <Modal open onClose={onClose} title="测试弹窗">
      <div>弹窗内容</div>
    </Modal>
  );

  fireEvent.click(screen.getByRole('dialog'));
  fireEvent.keyDown(document, { key: 'Escape' });

  expect(onClose).toHaveBeenCalledTimes(2);
});
