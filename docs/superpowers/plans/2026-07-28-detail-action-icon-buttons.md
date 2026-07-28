# 条目详情操作图标按钮 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将详情区的手动 AI 与关闭操作改为紧凑、可访问的 SVG 图标按钮。

**Architecture:** 手动 AI 触发器在 `ArticleManualAI` 内直接改用 SVG；首页和收藏页的关闭按钮各自替换为相同的 × SVG。CSS 为两种操作设置独立的紧凑样式，不触碰已有原始网页、收藏与详情状态逻辑。

**Tech Stack:** React 19、TypeScript、Vitest、React Testing Library、CSS。

## Global Constraints

- 手动 AI 按钮的 `aria-label` 与 `title` 为「手动 AI 分类与翻译」，点击仍打开右侧抽屉。
- 关闭按钮的 `aria-label` 与 `title` 为「关闭」，点击仍清空当前详情。
- 两类按钮均为 32px 正方形内联 SVG；SVG 必须 `aria-hidden="true"`。
- 原始网页按钮仍在手动 AI 左侧，操作顺序和原有详情切换行为不改变。

---

### Task 1: 手动 AI 图标触发器

**Files:**
- Modify: `web/src/components/ArticleManualAI.tsx:332-342`
- Modify: `web/src/index.css:3549-3562`
- Modify: `web/src/pages/Home.test.tsx:576-590`
- Modify: `web/src/pages/Favorites.test.tsx:104-122`

**Interfaces:**
- Consumes: 既有 `setDrawerOpen(true)` 处理函数。
- Produces: 带魔杖星光 SVG、可访问名称和提示的 `article-detail-ai-trigger` 按钮。

- [ ] **Step 1: 为现有页面回归测试添加手动 AI 图标断言**

```tsx
const manualAiTrigger = screen.getByRole('button', { name: '手动 AI 分类与翻译' });
expect(manualAiTrigger).toHaveAttribute('title', '手动 AI 分类与翻译');
expect(manualAiTrigger.querySelector('svg')).toBeInTheDocument();
```

在首页与收藏页已有的原始网页测试中，用以上查询替代对「手动 AI」文字名称的查询；保留 `compareDocumentPosition` 断言，以验证原始网页按钮仍在其左侧。

- [ ] **Step 2: 验证页面测试失败**

Run: `npm test -- --run src/pages/Home.test.tsx src/pages/Favorites.test.tsx`

Expected: FAIL，现有手动 AI 按钮没有「手动 AI 分类与翻译」可访问名称，也不包含 SVG。

- [ ] **Step 3: 使用魔杖星光 SVG 替换手动 AI 文字**

```tsx
<button
  type="button"
  className="article-detail-ai-trigger"
  onClick={() => setDrawerOpen(true)}
  aria-label="手动 AI 分类与翻译"
  title="手动 AI 分类与翻译"
>
  <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="m15 4 1.3 3.7L20 9l-3.7 1.3L15 14l-1.3-3.7L10 9l3.7-1.3L15 4Z" />
    <path d="m4 20 9-9" />
    <path d="m6 15 1 1M9 18l1 1" />
  </svg>
</button>
```

将 `.article-detail-ai-trigger` 改为 32px 正方形、`inline-grid` 居中、零内边距；添加 `.article-detail-ai-trigger svg { width: 16px; height: 16px; }`，保留现有边框、颜色和 hover 背景。

- [ ] **Step 4: 验证页面测试通过**

Run: `npm test -- --run src/pages/Home.test.tsx src/pages/Favorites.test.tsx`

Expected: PASS，原始网页、手动 AI 图标和既有详情切换测试全部通过。

- [ ] **Step 5: 提交手动 AI 图标任务**

```bash
git add web/src/components/ArticleManualAI.tsx web/src/index.css web/src/pages/Home.test.tsx web/src/pages/Favorites.test.tsx
git commit -m "样式：将手动AI改为图标按钮"
```

### Task 2: 首页与收藏页关闭图标

**Files:**
- Modify: `web/src/pages/Home.tsx:563-565`
- Modify: `web/src/pages/Favorites.tsx:289-291`
- Modify: `web/src/index.css:3709-3721`
- Modify: `web/src/pages/Home.test.tsx`
- Modify: `web/src/pages/Favorites.test.tsx`

**Interfaces:**
- Consumes: 既有 `selectArticle(null)` 处理函数。
- Produces: 带 × SVG、可访问名称和提示的 `article-detail-close` 按钮。

- [ ] **Step 1: 写出关闭图标与关闭详情的失败测试**

```tsx
const closeButton = screen.getByRole('button', { name: '关闭' });
expect(closeButton).toHaveAttribute('title', '关闭');
expect(closeButton.querySelector('svg')).toBeInTheDocument();
await user.click(closeButton);
expect(document.querySelector('.article-detail-dock')).not.toBeInTheDocument();
```

在首页原始网页测试完成返回正文后追加该断言；在收藏页测试完成第二篇正文断言后追加相同断言。

- [ ] **Step 2: 验证关闭测试失败**

Run: `npm test -- --run src/pages/Home.test.tsx src/pages/Favorites.test.tsx`

Expected: FAIL，现有关闭按钮不含 `title` 属性和 SVG。

- [ ] **Step 3: 用 × SVG 替换两个页面的关闭文字并收紧样式**

```tsx
<button
  type="button"
  className="article-detail-close"
  onClick={() => selectArticle(null)}
  aria-label="关闭"
  title="关闭"
>
  <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="m6 6 12 12M18 6 6 18" />
  </svg>
</button>
```

将 `.article-detail-close` 改为 32px 正方形、`inline-grid` 居中、零内边距；添加 `.article-detail-close svg { width: 16px; height: 16px; }`，保留现有弱化颜色与 hover 规则。

- [ ] **Step 4: 验证页面测试和生产构建通过**

Run: `npm test -- --run src/pages/Home.test.tsx src/pages/Favorites.test.tsx && npm run build`

Expected: PASS，关闭详情、图标语义、详情操作顺序与 TypeScript/Vite 构建均通过。

- [ ] **Step 5: 提交关闭图标任务**

```bash
git add web/src/pages/Home.tsx web/src/pages/Favorites.tsx web/src/index.css web/src/pages/Home.test.tsx web/src/pages/Favorites.test.tsx
git commit -m "样式：将详情关闭改为图标按钮"
```
