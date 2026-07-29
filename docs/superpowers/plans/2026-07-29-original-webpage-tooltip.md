# 原始网页 iframe 悬停提示调整 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 消除原始网页 iframe 的浏览器原生悬停提示，同时保留 iframe 的无障碍名称和现有加载行为。

**Architecture:** 只调整共享的 `ArticleOriginalWebpage` 组件，用 `aria-label` 替代会触发浏览器 tooltip 的 `title`。组件、首页和收藏页测试统一通过 iframe 的无障碍名称定位元素，不改变页面状态、URL 或数据流。

**Tech Stack:** React 19、TypeScript、Testing Library、Vitest、Vite

## Global Constraints

- 不再显示由 iframe `title` 触发的「原始网页」悬停提示。
- 保留 iframe 自身清晰的无障碍名称。
- 不改变原始网页的加载、切换、回退链接及按钮行为。
- 不新增依赖，不修改后端。

---

### Task 1: 用无障碍名称替代 iframe title

**Files:**
- Modify: `web/src/components/ArticleOriginalWebpage.tsx:4-10`
- Test: `web/src/components/ArticleOriginalWebpage.test.tsx:4-19`
- Test: `web/src/pages/Home.test.tsx:661-704`
- Test: `web/src/pages/Favorites.test.tsx:98-127`

**Interfaces:**
- Consumes: `ArticleOriginalWebpage({ url }: { url: string }): JSX.Element`
- Produces: 具有 `aria-label="原始网页"` 且没有 `title` 属性的 iframe；组件调用方式不变。

- [ ] **Step 1: 写出 iframe 不再使用 title 的失败测试**

将 `ArticleOriginalWebpage.test.tsx` 中基于 `getByTitle` 的断言改为：

```tsx
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
```

- [ ] **Step 2: 运行组件测试并确认按预期失败**

Run:

```bash
cd web
npm test -- src/components/ArticleOriginalWebpage.test.tsx
```

Expected: FAIL，因为 iframe 尚未设置 `aria-label="原始网页"`。

- [ ] **Step 3: 写入最小实现**

将 `ArticleOriginalWebpage.tsx` 中的 iframe 改为：

```tsx
<iframe
  className="article-original-webpage-frame"
  src={url}
  aria-label="原始网页"
  loading="lazy"
  referrerPolicy="strict-origin-when-cross-origin"
/>
```

- [ ] **Step 4: 运行组件测试并确认通过**

Run:

```bash
cd web
npm test -- src/components/ArticleOriginalWebpage.test.tsx
```

Expected: PASS。

- [ ] **Step 5: 更新首页与收藏页测试的 iframe 定位方式**

将首页和收藏页测试中以下定位：

```tsx
screen.getByTitle('原始网页')
screen.queryByTitle('原始网页')
```

分别改为：

```tsx
screen.getByLabelText('原始网页', { selector: 'iframe' })
screen.queryByLabelText('原始网页', { selector: 'iframe' })
```

保留现有 URL、正文切换、按钮顺序及关闭行为断言。

- [ ] **Step 6: 运行相关测试**

Run:

```bash
cd web
npm test -- src/components/ArticleOriginalWebpage.test.tsx src/pages/Home.test.tsx src/pages/Favorites.test.tsx
```

Expected: 三个测试文件全部 PASS，输出无错误或警告。

- [ ] **Step 7: 运行前端完整验证**

Run:

```bash
cd web
npm test
npm run lint
npm run build
```

Expected: 全部命令退出码为 0。

- [ ] **Step 8: 提交实现**

```bash
git add web/src/components/ArticleOriginalWebpage.tsx \
  web/src/components/ArticleOriginalWebpage.test.tsx \
  web/src/pages/Home.test.tsx \
  web/src/pages/Favorites.test.tsx \
  docs/superpowers/plans/2026-07-29-original-webpage-tooltip.md
git commit -m "修复：移除原始网页悬停提示"
```
