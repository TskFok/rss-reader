import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ThemeProvider } from '../contexts/ThemeContext';
import Register from './Register';

test('注册页使用玻璃态场景容器并展示表单', () => {
  const { container } = render(
    <MemoryRouter initialEntries={['/register']}>
      <ThemeProvider>
        <Register />
      </ThemeProvider>
    </MemoryRouter>
  );

  expect(container.querySelector('.auth-scene')).not.toBeNull();
  expect(container.querySelector('.auth-page')).not.toBeNull();
  expect(screen.getByRole('heading', { name: '注册' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: '注册' })).toBeInTheDocument();
});
