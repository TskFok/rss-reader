# 原始网页图标按钮 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将详情区的原始网页文字操作改为紧凑、可访问的 SVG 图标按钮。

**Architecture:** 新增一个无状态按钮组件，接收当前视图状态与点击回调，集中管理两个 SVG 图标及对应的可访问文案。首页和收藏页继续持有切换状态，仅以新组件替换现有按钮 JSX。

**Tech Stack:** React 19、TypeScript、Vitest、React Testing Library、CSS。

## Global Constraints

- 按钮必须仍位于「手动 AI」左侧。
- 正文视图使用网页 SVG 图标，`aria-label` 和 `title` 为「查看原始网页」。
- 原始网页视图使用返回 SVG 图标，`aria-label` 和 `title` 为「返回正文」。
- SVG 按钮必须为紧凑正方形；手动 AI 的文字按钮样式不改变。
- 不改变 iframe、原始网页切换状态、文章详情 API 或缓存行为。

---

### Task 1: 可访问的原始网页图标按钮组件

**Files:**
- Create: `web/src/components/ArticleOriginalWebpageButton.tsx`
- Create: `web/src/components/ArticleOriginalWebpageButton.test.tsx`
- Modify: `web/src/index.css:3549-3562`

**Interfaces:**
- Consumes: `showOriginalWebpage: boolean` 与 `onClick: () => void`。
- Produces: `ArticleOriginalWebpageButton({ showOriginalWebpage, onClick })`，渲染带 SVG 的按钮。

- [ ] **Step 1: 写出图标及可访问文案的失败测试**

```tsx
import { render, screen } from '@testing-library/react';
import ArticleOriginalWebpageButton from './ArticleOriginalWebpageButton';

test('按当前视图显示对应的图标按钮语义', () => {
  const { rerender } = render(
    <ArticleOriginalWebpageButton showOriginalWebpage={false} onClick={() => {}} />
  );
  expect(screen.getByRole('button', { name: '查看原始网页' })).toHaveAttribute('title', '查看原始网页');
  expect(screen.getByRole('button', { name: '查看原始网页' }).querySelector('svg')).toBeInTheDocument();

  rerender(<ArticleOriginalWebpageButton showOriginalWebpage onClick={() => {}} />);
  expect(screen.getByRole('button', { name: '返回正文' })).toHaveAttribute('title', '返回正文');
  expect(screen.getByRole('button', { name: '返回正文' }).querySelector('svg')).toBeInTheDocument();
});
```

- [ ] **Step 2: 验证测试因组件缺失而失败**

Run: `npm test -- --run src/components/ArticleOriginalWebpageButton.test.tsx`

Expected: FAIL，无法解析 `ArticleOriginalWebpageButton` 模块。

- [ ] **Step 3: 实现最小 SVG 按钮与紧凑样式**

```tsx
export default function ArticleOriginalWebpageButton({
  showOriginalWebpage,
  onClick,
}: {
  showOriginalWebpage: boolean;
  onClick: () => void;
}) {
  const label = showOriginalWebpage ? '返回正文' : '查看原始网页';
  return (
    <button type="button" className="article-detail-original-webpage-trigger" onClick={onClick} aria-label={label} title={label}>
      {showOriginalWebpage ? <svg aria-hidden="true" viewBox="0 0 24 24">...</svg> : <svg aria-hidden="true" viewBox="0 0 24 24">...</svg>}
    </button>
  );
}
```

```css
.article-detail-original-webpage-trigger {
  width: 32px;
  height: 32px;
  padding: 0;
  display: inline-grid;
  place-items: center;
}

.article-detail-original-webpage-trigger svg { width: 16px; height: 16px; }
```

- [ ] **Step 4: 验证组件测试通过**

Run: `npm test -- --run src/components/ArticleOriginalWebpageButton.test.tsx`

Expected: PASS，两个状态均提供对应的可访问名称、提示和 SVG。

- [ ] **Step 5: 提交组件任务**

```bash
git add web/src/components/ArticleOriginalWebpageButton.tsx web/src/components/ArticleOriginalWebpageButton.test.tsx web/src/index.css
git commit -m "样式：新增原始网页图标按钮"
```

### Task 2: 在首页与收藏页替换为图标按钮

**Files:**
- Modify: `web/src/pages/Home.tsx:1-20, 530-540`
- Modify: `web/src/pages/Favorites.tsx:1-20, 256-266`
- Modify: `web/src/pages/Home.test.tsx:557-588`
- Modify: `web/src/pages/Favorites.test.tsx:98-119`

**Interfaces:**
- Consumes: `ArticleOriginalWebpageButton({ showOriginalWebpage, onClick })`。
- Produces: 两个详情页中相同位置、相同可访问语义的图标按钮。

- [ ] **Step 1: 扩展页面失败测试，断言按钮具有图标样式类**

```tsx
const originalWebpageTrigger = screen.getByRole('button', { name: '查看原始网页' });
expect(originalWebpageTrigger).toHaveClass('article-detail-original-webpage-trigger');
expect(originalWebpageTrigger.querySelector('svg')).toBeInTheDocument();
```

在 `Home.test.tsx` 的返回正文断言后，以及 `Favorites.test.tsx` 点击原始网页后的断言后，各追加：

```tsx
expect(screen.getByRole('button', { name: '返回正文' })).toHaveClass(
  'article-detail-original-webpage-trigger'
);
```

- [ ] **Step 2: 验证页面测试失败**

Run: `npm test -- --run src/pages/Home.test.tsx src/pages/Favorites.test.tsx`

Expected: FAIL，现有文字按钮没有 `article-detail-original-webpage-trigger` 类和 SVG。

- [ ] **Step 3: 在两个页面接入共享组件**

```tsx
<ArticleOriginalWebpageButton
  showOriginalWebpage={showOriginalWebpage}
  onClick={() => setShowOriginalWebpage((value) => !value)}
/>
```

将以上 JSX 放在每个页面的 `ArticleManualAI` 之前，删除原有的文字按钮。保持 `selectedDetail && (...)` 条件不变。

- [ ] **Step 4: 验证首页与收藏页测试通过**

Run: `npm test -- --run src/pages/Home.test.tsx src/pages/Favorites.test.tsx`

Expected: PASS，现有切换、位置、重置行为与新增图标断言全部通过。

- [ ] **Step 5: 执行生产构建**

Run: `npm run build`

Expected: PASS，TypeScript 检查及 Vite 打包成功。

- [ ] **Step 6: 提交页面接入任务**

```bash
git add web/src/pages/Home.tsx web/src/pages/Favorites.tsx web/src/pages/Home.test.tsx web/src/pages/Favorites.test.tsx
git commit -m "样式：使用原始网页图标按钮"
```
