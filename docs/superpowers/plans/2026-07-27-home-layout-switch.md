# 首页阅读布局切换 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在首页保持默认三栏布局的前提下，让用户可切换到详情居中、文章列表在右侧的阅读布局，并在浏览器中持久化选择。

**Architecture:** 以 `homeLayout` 工具模块统一布局类型、默认值和受保护的 localStorage 访问。`Home` 保存当前布局状态、渲染选择控件、根布局类及只负责布局的 `.home-reading-panels` 容器；CSS 在详情居中模式将容器改为横向，并使用 flex `order` 与宽度规则改变视觉顺序，不改变文章列表与详情组件自身的挂载关系。

**Tech Stack:** React 19、TypeScript、Vitest、React Testing Library、CSS。

## Global Constraints

- 默认布局必须是现有顺序：订阅栏、文章列表、内容详情。
- 布局偏好只存储于 localStorage，不写入 URL，不改变任何 API 请求。
- localStorage 不可用、抛出异常或值无效时必须回退默认布局。
- 窄屏（`max-width: 860px`）维持现有纵向排列。
- 禁止新增后端、数据库或 SQL 查询。

---

### Task 1: 布局偏好工具

**Files:**
- Create: `web/src/utils/homeLayout.ts`
- Test: `web/src/utils/homeLayout.test.ts`

**Interfaces:**
- Produces: `HomeLayout = 'default' | 'detail-centered'`、`getStoredHomeLayout(): HomeLayout`、`setStoredHomeLayout(layout: HomeLayout): void`。
- Consumes: 浏览器 `window.localStorage`；在非浏览器或存储失败时不抛出异常。

- [ ] **Step 1: 写入失败测试**

```ts
import { afterEach, expect, test, vi } from 'vitest';
import { getStoredHomeLayout, setStoredHomeLayout } from './homeLayout';

afterEach(() => window.localStorage.clear());

test('缺失或无效的存储值时使用默认布局', () => {
  expect(getStoredHomeLayout()).toBe('default');
  window.localStorage.setItem('home.layout', 'unknown');
  expect(getStoredHomeLayout()).toBe('default');
});

test('存储访问抛出异常时回退默认布局且不抛错', () => {
  const getItem = vi.spyOn(window.localStorage, 'getItem').mockImplementation(() => {
    throw new Error('blocked');
  });
  const setItem = vi.spyOn(window.localStorage, 'setItem').mockImplementation(() => {
    throw new Error('blocked');
  });

  expect(getStoredHomeLayout()).toBe('default');
  expect(() => setStoredHomeLayout('detail-centered')).not.toThrow();

  getItem.mockRestore();
  setItem.mockRestore();
});

test('保存布局并在后续读取时恢复', () => {
  setStoredHomeLayout('detail-centered');
  expect(window.localStorage.getItem('home.layout')).toBe('detail-centered');
  expect(getStoredHomeLayout()).toBe('detail-centered');
});
```

- [ ] **Step 2: 运行测试，确认因模块不存在而失败**

Run: `npm test -- src/utils/homeLayout.test.ts`

Expected: FAIL，错误指向无法解析 `./homeLayout`。

- [ ] **Step 3: 编写最小实现**

```ts
export type HomeLayout = 'default' | 'detail-centered';

const STORAGE_KEY = 'home.layout';

export function getStoredHomeLayout(): HomeLayout {
  try {
    return window.localStorage.getItem(STORAGE_KEY) === 'detail-centered'
      ? 'detail-centered'
      : 'default';
  } catch {
    return 'default';
  }
}

export function setStoredHomeLayout(layout: HomeLayout): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, layout);
  } catch {
    // 存储受限时仅保留当前会话状态。
  }
}
```

实现中须先判断 `typeof window === 'undefined'`，避免服务端环境访问 `window`。

- [ ] **Step 4: 运行工具测试，确认通过**

Run: `npm test -- src/utils/homeLayout.test.ts`

Expected: PASS，三个测试均通过。

- [ ] **Step 5: 提交工具与测试**

```bash
git add web/src/utils/homeLayout.ts web/src/utils/homeLayout.test.ts
git commit -m "功能：首页布局偏好持久化"
```

### Task 2: 首页控件与布局状态

**Files:**
- Modify: `web/src/pages/Home.tsx:1-40,315-562`
- Test: `web/src/pages/Home.test.tsx`

**Interfaces:**
- Consumes: `HomeLayout`、`getStoredHomeLayout`、`setStoredHomeLayout`（Task 1）。
- Produces: 具有 `home-layout--detail-centered` 类的详情居中首页根节点，以及可访问名称为“阅读布局”的选择控件。

- [ ] **Step 1: 写入失败测试**

在 `Home.test.tsx` 中添加：

```tsx
test('阅读布局默认保持当前布局，切换后持久化详情居中选择', async () => {
  const user = userEvent.setup();
  mockLocalStorage();
  const router = renderHomeAt('/');
  const { container } = render(<RouterProvider router={router} />);

  const root = document.querySelector('.home-layout');
  expect(root).not.toHaveClass('home-layout--detail-centered');

  await user.selectOptions(await screen.findByRole('combobox', { name: '阅读布局' }), 'detail-centered');
  expect(root).toHaveClass('home-layout--detail-centered');
  expect(window.localStorage.getItem('home.layout')).toBe('detail-centered');
});
```

另增加重挂载测试，先写入 `home.layout=detail-centered`，再断言初始根节点拥有该类。

- [ ] **Step 2: 运行首页测试，确认因缺少控件而失败**

Run: `npm test -- src/pages/Home.test.tsx`

Expected: FAIL，错误指出找不到名称为“阅读布局”的 combobox。

- [ ] **Step 3: 编写最小实现**

在 `Home.tsx`：

```tsx
const [homeLayout, setHomeLayout] = useState<HomeLayout>(() => getStoredHomeLayout());

<div className={`home-layout ${homeLayout === 'detail-centered' ? 'home-layout--detail-centered' : ''}`}>
```

在现有筛选栏的状态选择前加入：

```tsx
<select
  aria-label="阅读布局"
  value={homeLayout}
  onChange={(event) => {
    const next = event.target.value as HomeLayout;
    setHomeLayout(next);
    setStoredHomeLayout(next);
  }}
>
  <option value="default">默认布局</option>
  <option value="detail-centered">详情居中</option>
</select>
```

并从工具模块导入三个接口。不要改动筛选参数、`ArticleList`、详情面板、面板父子结构或任何数据请求逻辑。

- [ ] **Step 4: 运行首页测试，确认通过**

Run: `npm test -- src/pages/Home.test.tsx`

Expected: PASS，既有首页行为与新增布局状态测试全部通过。

- [ ] **Step 5: 提交首页状态与测试**

```bash
git add web/src/pages/Home.tsx web/src/pages/Home.test.tsx
git commit -m "功能：首页支持阅读布局切换"
```

### Task 3: 桌面布局样式与全量验证

**Files:**
- Modify: `web/src/pages/Home.tsx:470-562`
- Modify: `web/src/index.css:3256-3500`
- Test: `web/src/pages/Home.test.tsx`

**Interfaces:**
- Consumes: `.home-layout--detail-centered`（Task 2）。
- Produces: 详情居中时文章详情面板在文章列表左侧显示，详情区域占用更多桌面空间；窄屏继续使用纵向布局。

- [ ] **Step 1: 写入失败测试**

在首页测试中选择详情居中并打开文章后，断言布局容器与两个面板均存在：

```tsx
const root = container.querySelector('.home-layout--detail-centered')!;
const panels = root.querySelector('.home-reading-panels')!;
expect(panels.querySelector('.article-detail-dock')).not.toBeNull();
expect(panels.querySelector('.article-list-scroll')).not.toBeNull();
```

该测试验证切换后的真实 DOM 边界；视觉顺序由 CSS 规则保证，并通过生产构建验证。

- [ ] **Step 2: 运行首页测试，确认因布局容器不存在而失败**

Run: `npm test -- src/pages/Home.test.tsx`

Expected: FAIL，错误指出找不到 `.home-reading-panels`。

- [ ] **Step 3: 完善测试、增加布局容器并添加 CSS 规则**

先在测试中选择“详情居中”并点击文章，使上一步断言可验证真实页面。接着将现有 `.article-list-scroll` 与条件渲染的 `.article-detail-dock` 一起包入：

```tsx
<div className="home-reading-panels">
  <div ref={listScrollRef} className="article-list-scroll">{/* 原有文章列表内容 */}</div>
  {selectedListItem && <div className="article-detail-dock">{/* 原有详情面板内容 */}</div>}
</div>
```

保留容器内两个节点原有的 props、refs、条件与内容。随后在 `index.css` 的 `.home-content` 与 `.article-list-scroll` 规则附近加入：

```css
.home-reading-panels {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  gap: 16px;
}

.home-layout--detail-centered .home-reading-panels {
  flex-direction: row;
}

.home-layout--detail-centered .article-detail-dock {
  order: 1;
  flex: 1.6 1 0;
  height: auto;
}

.home-layout--detail-centered .article-list-scroll {
  order: 2;
  flex: 1 1 0;
}

@media (max-width: 860px) {
  .home-layout--detail-centered .home-reading-panels {
    flex-direction: column;
  }

  .home-layout--detail-centered .article-detail-dock,
  .home-layout--detail-centered .article-list-scroll {
    order: initial;
    flex: initial;
  }
}
```

默认布局中 `.home-reading-panels` 仅复现现有的纵向结构；详情居中样式才改变其方向和详情高度。不要新增固定像素宽度。

- [ ] **Step 4: 运行目标测试、静态检查与生产构建**

Run: `npm test -- src/utils/homeLayout.test.ts src/pages/Home.test.tsx && npm run lint && npm run build`

Expected: 三个命令均以退出码 0 完成；TypeScript、ESLint 与 Vite 均无错误。

- [ ] **Step 5: 提交样式与验证测试**

```bash
git add web/src/index.css web/src/pages/Home.tsx web/src/pages/Home.test.tsx
git commit -m "样式：首页详情居中阅读布局"
```
