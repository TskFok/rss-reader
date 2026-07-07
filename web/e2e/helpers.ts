import type { Page, Route } from '@playwright/test';

const testUser = {
  id: 1,
  username: 'e2e-user',
  status: 'active',
  is_super_admin: false,
  created_at: '2020-01-01T00:00:00Z',
};

const testArticle = {
  id: 101,
  feed_id: 1,
  guid: 'e2e-guid-101',
  title: 'E2E 测试文章',
  link: 'http://example.com/e2e/101',
  published_at: '2020-01-02T00:00:00Z',
  created_at: '2020-01-01T00:00:00Z',
  read: false,
  feed_title: 'E2E 订阅',
};

export type ArticleListRequest = {
  read?: string;
  feed_id?: string;
  page?: string;
  page_size?: string;
};

export async function installAuth(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('token', 'e2e-test-token');
    localStorage.setItem(
      'user',
      JSON.stringify({
        id: 1,
        username: 'e2e-user',
        status: 'active',
        is_super_admin: false,
        created_at: '2020-01-01T00:00:00Z',
      })
    );
  });
}

export async function mockHomeApis(
  page: Page,
  onArticlesList?: (request: ArticleListRequest) => void
) {
  await page.route('**/api/feeds', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          id: 1,
          user_id: 1,
          category_id: null,
          proxy_id: null,
          ai_model_id: null,
          ai_classify_enabled: false,
          ai_translate_enabled: false,
          ai_target_language: 'zh-CN',
          url: 'http://example.com/e2e.xml',
          title: 'E2E 订阅',
          update_interval_minutes: 60,
          expire_days: 90,
          last_fetched_at: null,
          created_at: '2020-01-01T00:00:00Z',
        },
      ]),
    });
  });

  await page.route('**/api/categories', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  await page.route('**/api/ai-models', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    });
  });

  await page.route('**/api/articles**', async (route: Route) => {
    const url = new URL(route.request().url());
    const request: ArticleListRequest = {
      read: url.searchParams.get('read') ?? undefined,
      feed_id: url.searchParams.get('feed_id') ?? undefined,
      page: url.searchParams.get('page') ?? undefined,
      page_size: url.searchParams.get('page_size') ?? undefined,
    };
    onArticlesList?.(request);

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [testArticle],
        total: 1,
      }),
    });
  });
}

export async function openHome(page: Page, path = '/') {
  await installAuth(page);
  await mockHomeApis(page);
  await page.goto(path);
  await page.getByRole('combobox', { name: '文章状态筛选' }).waitFor();
}

export { testArticle, testUser };
