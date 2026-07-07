import { expect, test } from '@playwright/test';
import { installAuth, mockHomeApis, type ArticleListRequest } from './helpers';

test.describe('阅读页状态筛选 URL 持久化', () => {
  test('默认进入首页时筛选为未读且 URL 无 read 参数', async ({ page }) => {
    const requests: ArticleListRequest[] = [];
    await installAuth(page);
    await mockHomeApis(page, (request) => requests.push(request));
    await page.goto('/');

    const statusSelect = page.getByRole('combobox', { name: '文章状态筛选' });
    await expect(statusSelect).toHaveValue('unread');
    await expect(page).toHaveURL('/');
    await expect.poll(() => requests.some((r) => r.read === 'false')).toBe(true);
  });

  test('?read=all 时展示全部文章', async ({ page }) => {
    const requests: ArticleListRequest[] = [];
    await installAuth(page);
    await mockHomeApis(page, (request) => requests.push(request));
    await page.goto('/?read=all');

    const statusSelect = page.getByRole('combobox', { name: '文章状态筛选' });
    await expect(statusSelect).toHaveValue('');
    await expect(page).toHaveURL('/?read=all');
    await expect.poll(() => requests.some((r) => r.read === undefined)).toBe(true);
  });

  test('?read=read 时展示已读文章', async ({ page }) => {
    const requests: ArticleListRequest[] = [];
    await installAuth(page);
    await mockHomeApis(page, (request) => requests.push(request));
    await page.goto('/?read=read');

    const statusSelect = page.getByRole('combobox', { name: '文章状态筛选' });
    await expect(statusSelect).toHaveValue('read');
    await expect(page).toHaveURL('/?read=read');
    await expect.poll(() => requests.some((r) => r.read === 'true')).toBe(true);
  });

  test('切换筛选会同步更新 URL', async ({ page }) => {
    await installAuth(page);
    await mockHomeApis(page);
    await page.goto('/');

    const statusSelect = page.getByRole('combobox', { name: '文章状态筛选' });
    await statusSelect.selectOption('');
    await expect(page).toHaveURL('/?read=all');

    await statusSelect.selectOption('read');
    await expect(page).toHaveURL('/?read=read');

    await statusSelect.selectOption('unread');
    await expect(page).toHaveURL('/');
  });

  test('刷新页面后保留 URL 中的筛选状态', async ({ page }) => {
    await installAuth(page);
    await mockHomeApis(page);
    await page.goto('/?read=read');

    const statusSelect = page.getByRole('combobox', { name: '文章状态筛选' });
    await expect(statusSelect).toHaveValue('read');

    await page.reload();
    await expect(statusSelect).toHaveValue('read');
    await expect(page).toHaveURL('/?read=read');
  });

  test('浏览器后退会恢复之前的筛选状态', async ({ page }) => {
    await installAuth(page);
    await mockHomeApis(page);
    await page.goto('/');

    const statusSelect = page.getByRole('combobox', { name: '文章状态筛选' });
    await statusSelect.selectOption('');
    await expect(page).toHaveURL('/?read=all');

    await statusSelect.selectOption('read');
    await expect(page).toHaveURL('/?read=read');

    await page.goBack();
    await expect(page).toHaveURL('/?read=all');
    await expect(statusSelect).toHaveValue('');

    await page.goBack();
    await expect(page).toHaveURL('/');
    await expect(statusSelect).toHaveValue('unread');
  });
});
