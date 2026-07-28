# 可调整尺寸的文章详情面板 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让桌面端“详情居中”阅读布局的文章详情面板可拖拽调整宽度和高度，并安全持久化用户偏好。

**Architecture:** 新增 `articleDetailSize` 工具模块，集中处理尺寸类型、默认值、边界限制和 localStorage 访问。`Home` 只管理指针拖拽的临时状态，将工具模块提供的尺寸写为 CSS 自定义属性；CSS 在详情居中桌面模式使用这些属性并显示两个手柄，默认与窄屏模式不应用这些规则。

**Tech Stack:** React 19、TypeScript、Vitest、React Testing Library、CSS。

## Global Constraints

- 仅视口宽度大于 860px 且布局为“详情居中”时显示并启用拖拽手柄。
- 宽度必须限制为阅读区域的 40%–75%，高度必须限制为 280px 至当前视口可用高度。
- 尺寸使用独立 localStorage 键保存；无效、缺失或不可访问的存储值必须安全回退。
- 默认布局和窄屏保持既有纵向样式，不显示手柄，也不应用尺寸偏好。
- 不修改后端 API、数据库或 SQL 查询。

---

### Task 1: 尺寸偏好与边界工具

**Files:**
- Create: `web/src/utils/articleDetailSize.ts`
- Create: `web/src/utils/articleDetailSize.test.ts`

**Interfaces:**
- Produces: `ArticleDetailSize`、`DEFAULT_ARTICLE_DETAIL_SIZE`、`clampArticleDetailSize(size, maxHeight)`、`getStoredArticleDetailSize(maxHeight)`、`setStoredArticleDetailSize(size)`。
- Consumes: 浏览器 `localStorage`；存储访问失败时不抛出异常。

- [ ] **Step 1: 写入失败测试**

```ts
import {
  clampArticleDetailSize,
  getStoredArticleDetailSize,
  setStoredArticleDetailSize,
} from './articleDetailSize';

test('将详情尺寸限制在宽度和高度边界内', () => {
  expect(clampArticleDetailSize({ widthPercent: 12, height: 20 }, 720)).toEqual({
    widthPercent: 40,
    height: 280,
  });
  expect(clampArticleDetailSize({ widthPercent: 90, height: 900 }, 720)).toEqual({
    widthPercent: 75,
    height: 720,
  });
});

test('保存后恢复尺寸，缺失或无效存储值使用默认尺寸', () => {
  expect(getStoredArticleDetailSize(720)).toEqual({ widthPercent: 62, height: 520 });
  window.localStorage.setItem('article.detail.size', '{"widthPercent":99,"height":1}');
  expect(getStoredArticleDetailSize(720)).toEqual({ widthPercent: 75, height: 280 });
  setStoredArticleDetailSize({ widthPercent: 55, height: 480 });
  expect(getStoredArticleDetailSize(720)).toEqual({ widthPercent: 55, height: 480 });
});
```

另增加 localStorage 的 `getItem` 与 `setItem` 抛错时仍返回默认值且保存不抛错的测试。测试使用完整的内存 Storage 替身，并在每个测试后恢复全局 `localStorage`。

- [ ] **Step 2: 运行测试，确认因模块不存在而失败**

Run: `npm test -- src/utils/articleDetailSize.test.ts`

Expected: FAIL，错误指向无法解析 `./articleDetailSize`。

- [ ] **Step 3: 编写最小实现**

```ts
export type ArticleDetailSize = { widthPercent: number; height: number };

export const DEFAULT_ARTICLE_DETAIL_SIZE: ArticleDetailSize = {
  widthPercent: 62,
  height: 520,
};

export function clampArticleDetailSize(
  size: ArticleDetailSize,
  maxHeight: number
): ArticleDetailSize {
  return {
    widthPercent: Math.min(75, Math.max(40, Math.round(size.widthPercent))),
    height: Math.min(Math.max(280, Math.round(maxHeight)), Math.max(280, Math.round(size.height))),
  };
}
```

读取存储时解析 JSON，确认 `widthPercent` 和 `height` 均为有限数字后再限制；解析失败、非浏览器环境或任何存储异常均返回限制后的默认值。保存时写入 JSON，并吞掉存储异常。

- [ ] **Step 4: 运行工具测试，确认通过**

Run: `npm test -- src/utils/articleDetailSize.test.ts`

Expected: PASS，所有默认、边界、恢复与受限存储测试通过。

- [ ] **Step 5: 提交工具与测试**

```bash
git add web/src/utils/articleDetailSize.ts web/src/utils/articleDetailSize.test.ts
git commit -m "功能：保存详情面板尺寸偏好"
```

### Task 2: 详情面板拖拽交互

**Files:**
- Modify: `web/src/pages/Home.tsx:1-80,496-620`
- Modify: `web/src/pages/Home.test.tsx`

**Interfaces:**
- Consumes: `ArticleDetailSize`、`clampArticleDetailSize`、`getStoredArticleDetailSize`、`setStoredArticleDetailSize`（Task 1）。
- Produces: `article-detail-width-resize-handle`、`article-detail-height-resize-handle`，以及只在详情居中布局写入的 `--article-detail-width` 和 `--article-detail-height` CSS 变量。

- [ ] **Step 1: 写入失败测试**

在 `Home.test.tsx` 添加以下真实页面行为测试：

```tsx
test('详情居中布局打开文章后显示宽高拖拽手柄并保存拖拽尺寸', async () => {
  const user = userEvent.setup();
  mockLocalStorage();
  const router = renderHomeAt('/');
  const { container } = render(<RouterProvider router={router} />);

  await user.selectOptions(await screen.findByRole('combobox', { name: '阅读布局' }), 'detail-centered');
  await user.click(await screen.findByRole('button', { name: /长文/ }));

  const panels = container.querySelector('.home-reading-panels') as HTMLDivElement;
  vi.spyOn(panels, 'getBoundingClientRect').mockReturnValue({ left: 100, top: 80, width: 1000, height: 700 } as DOMRect);
  const widthHandle = screen.getByRole('separator', { name: '调整详情宽度' });
  fireEvent.pointerDown(widthHandle, { pointerId: 1, clientX: 650 });
  fireEvent.pointerMove(widthHandle, { pointerId: 1, clientX: 800 });
  fireEvent.pointerUp(widthHandle, { pointerId: 1 });

  expect(panels.style.getPropertyValue('--article-detail-width')).toBe('70%');
  expect(JSON.parse(window.localStorage.getItem('article.detail.size')!)).toMatchObject({ widthPercent: 70 });
});
```

再增加测试断言默认布局中打开文章时两个带 `separator` 角色的手柄不存在。

- [ ] **Step 2: 运行首页测试，确认因缺少手柄而失败**

Run: `npm test -- src/pages/Home.test.tsx`

Expected: FAIL，错误指出找不到名称为“调整详情宽度”的 separator。

- [ ] **Step 3: 编写最小拖拽实现**

在 `Home.tsx` 中新增 `readingPanelsRef`、`articleDetailSize` 状态及一个 `resizeAxis` 状态。最大高度使用：

```ts
const getDetailMaxHeight = () => Math.max(280, window.innerHeight - 180);
```

拖拽开始时设置轴；若元素支持 `setPointerCapture`，调用它捕获当前指针；并给 `document.body` 增加 `article-detail-resizing` 类。移动时根据 `.home-reading-panels` 的 rect 计算：

```ts
const widthPercent = ((event.clientX - rect.left) / rect.width) * 100;
const height = event.clientY - rect.top;
setArticleDetailSize((current) => clampArticleDetailSize(
  resizeAxis === 'width' ? { ...current, widthPercent } : { ...current, height },
  getDetailMaxHeight()
));
```

在 `pointerup` 与 `pointercancel` 时移除 body 类、清空拖拽轴并保存最终尺寸。仅当 `homeLayout === 'detail-centered'` 时向阅读容器传入：

```tsx
style={{
  '--article-detail-width': `${articleDetailSize.widthPercent}%`,
  '--article-detail-height': `${articleDetailSize.height}px`,
} as React.CSSProperties}
```

在详情面板内渲染两个 `div role="separator"` 手柄，分别具备 `aria-label="调整详情宽度"` 和 `aria-label="调整详情高度"`、正确的 `aria-orientation`，并绑定对应 `onPointerDown`、`onPointerMove`、`onPointerUp` 和 `onPointerCancel`。手柄不含可见文字，避免干扰正文阅读。

- [ ] **Step 4: 运行首页测试，确认通过**

Run: `npm test -- src/pages/Home.test.tsx`

Expected: PASS，既有首页测试与新增手柄、拖拽持久化、默认布局隐藏手柄测试均通过。

- [ ] **Step 5: 提交交互与测试**

```bash
git add web/src/pages/Home.tsx web/src/pages/Home.test.tsx
git commit -m "功能：支持拖拽调整详情尺寸"
```

### Task 3: 桌面样式、窗口限制与全量验证

**Files:**
- Modify: `web/src/index.css:3264-3515`
- Modify: `web/src/pages/Home.tsx:70-80,496-620`
- Test: `web/src/pages/Home.test.tsx`

**Interfaces:**
- Consumes: Task 2 的 CSS 自定义属性和两个 separator 手柄。
- Produces: 详情居中桌面模式的可调整宽高样式；窄屏隐藏手柄并恢复当前纵向尺寸规则。

- [ ] **Step 1: 写入失败测试**

添加窗口变化测试：先写入 `{ widthPercent: 65, height: 900 }`，将 `window.innerHeight` 设为 `700` 并派发 `resize`，断言阅读容器的 `--article-detail-height` 为 `520px`（即 `700 - 180`）。

- [ ] **Step 2: 运行首页测试，确认当前实现因未监听窗口变化而失败**

Run: `npm test -- src/pages/Home.test.tsx`

Expected: FAIL，期望 `520px`，实际仍为 `900px`。

- [ ] **Step 3: 添加 CSS 与窗口限制监听**

在详情居中样式中使用：

```css
.home-layout--detail-centered .article-detail-dock {
  flex: 0 0 var(--article-detail-width, 62%);
  height: var(--article-detail-height, 520px);
  position: relative;
}

.article-detail-resize-handle {
  position: absolute;
  z-index: 2;
  border: 0;
  background: transparent;
  touch-action: none;
}

.article-detail-width-resize-handle { top: 20%; right: -8px; width: 16px; height: 60%; cursor: col-resize; }
.article-detail-height-resize-handle { bottom: -8px; left: 20%; width: 60%; height: 16px; cursor: row-resize; }
```

增加 `window.resize` 监听，调用 `setArticleDetailSize((size) => clampArticleDetailSize(size, getDetailMaxHeight()))`；在卸载时移除监听。通过 `body.article-detail-resizing { user-select: none; cursor: ...; }` 防止拖拽时选择正文。`max-width: 860px` 媒体查询中把两个手柄设为 `display: none`，并维持已有 `height: 50vh` 与纵向排序。

- [ ] **Step 4: 运行全量验证**

Run: `npm test && npm run build`

Expected: Vitest 以 0 个失败完成，TypeScript 与 Vite 构建以退出码 0 完成。

- [ ] **Step 5: 提交样式与窗口限制**

```bash
git add web/src/index.css web/src/pages/Home.tsx web/src/pages/Home.test.tsx
git commit -m "样式：完善详情面板拖拽尺寸"
```
