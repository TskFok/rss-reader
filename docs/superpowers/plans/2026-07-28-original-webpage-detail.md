# 原始网页详情内嵌功能 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在条目详情中提供「查看原始网页」切换按钮，在当前阅读区内显示文章原文链接，并保留返回 RSS 正文的能力。

**Architecture:** 新增只负责 iframe 与新标签页回退链接的展示组件。首页和收藏页分别保存瞬态的原始网页视图状态，把它作为详情正文区域的条件渲染；文章详情、缓存与服务端 API 均不改变。

**Tech Stack:** React 18、TypeScript、Vitest、React Testing Library、现有 CSS 自定义属性。

## Global Constraints

- 按钮必须放在「手动 AI」左侧，并且仅在文章详情已加载时展示。
- 原始网页地址只能来自 `article.link`；不新增服务端抓取、反向代理、缓存或 API。
- 视图切换不会改写当前正文/译文语言选择，也不会写入 URL、文章缓存或后端。
- 切换文章、关闭详情或详情重新加载时，原始网页视图必须重置为正文视图。
- iframe 必须使用 `loading="lazy"`、可访问标题、`referrerPolicy="strict-origin-when-cross-origin"`，并始终给出新标签页回退链接。

---

### Task 1: 原始网页展示组件

**Files:**
- Create: `web/src/components/ArticleOriginalWebpage.tsx`
- Create: `web/src/components/ArticleOriginalWebpage.test.tsx`
- Modify: `web/src/index.css:3898-3940`

**Interfaces:**
- Consumes: `url: string`，即 `Article.link`。
- Produces: `ArticleOriginalWebpage({ url }: { url: string }): JSX.Element`，在详情内容区渲染 iframe 和新标签页回退链接。

- [ ] **Step 1: 写出 iframe 与回退链接的失败测试**

```tsx
import { render, screen } from '@testing-library/react';
import ArticleOriginalWebpage from './ArticleOriginalWebpage';

test('在详情区域嵌入原始网页并提供新标签页回退链接', () => {
  render(<ArticleOriginalWebpage url="https://example.com/articles/1" />);

  expect(screen.getByTitle('原始网页')).toHaveAttribute('src', 'https://example.com/articles/1');
  expect(screen.getByTitle('原始网页')).toHaveAttribute('loading', 'lazy');
  expect(screen.getByTitle('原始网页')).toHaveAttribute('referrerpolicy', 'strict-origin-when-cross-origin');
  expect(screen.getByRole('link', { name: '在新标签页打开原文' })).toHaveAttribute(
    'href',
    'https://example.com/articles/1'
  );
});
```

- [ ] **Step 2: 验证测试因组件不存在而失败**

Run: `npm test -- --run src/components/ArticleOriginalWebpage.test.tsx`

Expected: FAIL，报错指出无法解析 `./ArticleOriginalWebpage`。

- [ ] **Step 3: 以最小实现创建组件与样式**

```tsx
export default function ArticleOriginalWebpage({ url }: { url: string }) {
  return (
    <section className="article-original-webpage" aria-label="原始网页">
      <iframe
        className="article-original-webpage-frame"
        src={url}
        title="原始网页"
        loading="lazy"
        referrerPolicy="strict-origin-when-cross-origin"
      />
      <p className="article-original-webpage-fallback">
        如果网页无法嵌入，请 <a href={url} target="_blank" rel="noopener noreferrer">在新标签页打开原文</a>。
      </p>
    </section>
  );
}
```

```css
.article-original-webpage-frame {
  display: block;
  width: 100%;
  min-height: 560px;
  border: 0;
  border-radius: 12px;
}
```

- [ ] **Step 4: 验证组件测试通过**

Run: `npm test -- --run src/components/ArticleOriginalWebpage.test.tsx`

Expected: PASS，断言 iframe 的地址、安全属性与回退链接均通过。

- [ ] **Step 5: 提交组件任务**

```bash
git add web/src/components/ArticleOriginalWebpage.tsx web/src/components/ArticleOriginalWebpage.test.tsx web/src/index.css
git commit -m "功能：新增原始网页详情组件"
```

### Task 2: 首页详情区切换原始网页

**Files:**
- Modify: `web/src/pages/Home.tsx:1-20, 513-576`
- Modify: `web/src/pages/Home.test.tsx`

**Interfaces:**
- Consumes: `ArticleOriginalWebpage({ url })`；当前选中的 `selectedDetail.link`。
- Produces: `showOriginalWebpage: boolean`、标题为「查看原始网页」/「返回正文」的切换按钮，以及在切换文章时重置该状态的 effect。

- [ ] **Step 1: 为首页写出失败的详情切换测试**

```tsx
test('点击查看原始网页后替换首页详情正文，并可返回正文', async () => {
  mockLocalStorage();
  const user = userEvent.setup();
  render(<RouterProvider router={renderHomeAt('/?read=all')} />);

  await user.click(await screen.findByText('长文'));
  await waitFor(() => {
    expect(document.querySelector('.article-detail-content')?.innerHTML).toContain('行');
  });
  const trigger = screen.getByRole('button', { name: '查看原始网页' });
  expect(trigger.compareDocumentPosition(screen.getByRole('button', { name: '手动 AI' }))).toBe(
    Node.DOCUMENT_POSITION_FOLLOWING
  );

  await user.click(trigger);
  expect(screen.getByTitle('原始网页')).toHaveAttribute('src', 'http://example.com/p/101');
  expect(document.querySelector('.article-detail-content')).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: '返回正文' })).toBeInTheDocument();

  await user.click(screen.getByRole('button', { name: '返回正文' }));
  expect(document.querySelector('.article-detail-content')?.innerHTML).toContain('行');
});
```

- [ ] **Step 2: 验证首页测试失败**

Run: `npm test -- --run src/pages/Home.test.tsx`

Expected: FAIL，找不到「查看原始网页」按钮。

- [ ] **Step 3: 在首页以最小状态实现切换与重置**

```tsx
const [showOriginalWebpage, setShowOriginalWebpage] = useState(false);

useEffect(() => {
  setShowOriginalWebpage(false);
}, [selectedId, selectedDetail?.id]);
```

```tsx
{selectedDetail && (
  <button
    type="button"
    className="article-detail-original-webpage-trigger"
    onClick={() => setShowOriginalWebpage((value) => !value)}
  >
    {showOriginalWebpage ? '返回正文' : '查看原始网页'}
  </button>
)}
<ArticleManualAI ... />
```

```tsx
{selectedDetail ? (
  showOriginalWebpage ? (
    <ArticleOriginalWebpage url={selectedDetail.link} />
  ) : (
    <ArticleDetailContent article={selectedDetail} displayLang={articleDisplayLang} />
  )
) : null}
```

- [ ] **Step 4: 验证首页测试通过**

Run: `npm test -- --run src/pages/Home.test.tsx`

Expected: PASS，已有首页测试与新增切换测试全部通过。

- [ ] **Step 5: 提交首页任务**

```bash
git add web/src/pages/Home.tsx web/src/pages/Home.test.tsx
git commit -m "功能：首页支持查看原始网页"
```

### Task 3: 收藏页复用详情区切换行为

**Files:**
- Modify: `web/src/pages/Favorites.tsx:1-20, 239-300`
- Create: `web/src/pages/Favorites.test.tsx`

**Interfaces:**
- Consumes: `ArticleOriginalWebpage({ url })` 与 `showOriginalWebpage: boolean`。
- Produces: 与首页一致的按钮位置、切换文案、iframe 内容和选文重置行为。

- [ ] **Step 1: 写出收藏页失败测试**

```tsx
const favoriteArticle: ArticleListItem = {
  id: 101,
  feed_id: 1,
  guid: 'guid-101',
  title: '长文',
  link: 'http://example.com/p/101',
  published_at: null,
  created_at: '2020-01-01T00:00:00Z',
  read: false,
  favorite: true,
  feed_title: '订阅一',
};
const nextFavoriteArticle: ArticleListItem = {
  ...favoriteArticle,
  id: 102,
  guid: 'guid-102',
  title: '另一篇',
  link: 'http://example.com/p/102',
};
const favoriteDetails = [
  { ...favoriteArticle, content: '<p>第一篇正文</p>' },
  { ...nextFavoriteArticle, content: '<p>第二篇正文</p>' },
];

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    articlesApi: {
      ...actual.articlesApi,
      list: vi.fn().mockResolvedValue({ data: { items: [favoriteArticle, nextFavoriteArticle], total: 2 } }),
      get: vi.fn().mockImplementation((id: number) =>
        Promise.resolve({ data: { article: { ...favoriteDetails.find((article) => article.id === id)!, read: true } } })
      ),
      toggleFavorite: vi.fn().mockResolvedValue({ data: { favorite: true } }),
    },
    aiModelsApi: {
      ...actual.aiModelsApi,
      list: vi.fn().mockResolvedValue({ data: [{ id: 1, name: 'm', base_url: 'u', user_id: 1, created_at: '', updated_at: '' }] }),
    },
  };
});

function renderFavoritesWithTwoArticles() {
  return render(
    <MemoryRouter>
      <ThemeProvider><ToastProvider><Favorites /></ToastProvider></ThemeProvider>
    </MemoryRouter>
  );
}

test('收藏页切换文章时不会保留上一条的原始网页视图', async () => {
  const user = userEvent.setup();
  renderFavoritesWithTwoArticles();

  await user.click(await screen.findByText('长文'));
  await user.click(await screen.findByRole('button', { name: '查看原始网页' }));
  expect(screen.getByTitle('原始网页')).toHaveAttribute('src', 'http://example.com/p/101');

  await user.click(screen.getByText('另一篇'));
  await screen.findByText('第二篇正文');
  expect(screen.queryByTitle('原始网页')).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: '查看原始网页' })).toBeInTheDocument();
});
```

- [ ] **Step 2: 验证收藏页测试失败**

Run: `npm test -- --run src/pages/Favorites.test.tsx`

Expected: FAIL，测试辅助函数或「查看原始网页」按钮尚不存在。

- [ ] **Step 3: 在收藏页实现与首页相同的切换和重置**

```tsx
const [showOriginalWebpage, setShowOriginalWebpage] = useState(false);

useEffect(() => {
  setShowOriginalWebpage(false);
}, [selectedId, selectedDetail?.id]);
```

将按钮置于 `ArticleManualAI` JSX 之前，并按 Task 2 的三元渲染替换收藏页原有的 `ArticleDetailContent` 调用。测试文件使用与 `Home.test.tsx` 相同的 API mock、`ThemeProvider` 和 `ToastProvider` 创建 `renderFavoritesWithTwoArticles`，但文章列表请求必须包含 `{ favorite: true }`。

- [ ] **Step 4: 验证收藏页测试通过**

Run: `npm test -- --run src/pages/Favorites.test.tsx`

Expected: PASS，按钮位于手动 AI 左侧，切换、返回与切换文章重置均可验证。

- [ ] **Step 5: 提交收藏页任务**

```bash
git add web/src/pages/Favorites.tsx web/src/pages/Favorites.test.tsx
git commit -m "功能：收藏页支持查看原始网页"
```

### Task 4: 全量前端验证

**Files:**
- Verify only: `web/src/components/ArticleOriginalWebpage.test.tsx`
- Verify only: `web/src/pages/Home.test.tsx`
- Verify only: `web/src/pages/Favorites.test.tsx`

**Interfaces:**
- Consumes: Tasks 1-3 的已提交实现。
- Produces: 通过的前端单元测试和生产构建。

- [ ] **Step 1: 执行相关测试组**

Run: `npm test -- --run src/components/ArticleOriginalWebpage.test.tsx src/pages/Home.test.tsx src/pages/Favorites.test.tsx`

Expected: PASS，无未处理的 React 警告。

- [ ] **Step 2: 执行生产构建**

Run: `npm run build`

Expected: PASS，TypeScript 与 Vite 构建成功。

- [ ] **Step 3: 审查最终改动范围**

Run: `git diff --check && git status --short`

Expected: 无空白错误，工作树无未提交改动。

- [ ] **Step 4: 提交验证后的微调（若存在）**

```bash
git add web/src
git commit -m "测试：验证原始网页详情功能"
```

仅当验证导致实际修复时执行；若无改动，不创建空提交。
