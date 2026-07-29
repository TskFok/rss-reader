# 原始网页 iframe 悬停提示调整设计

## 背景

原始网页组件为 iframe 设置了 `title="原始网页"`。浏览器会将 `title` 渲染为原生悬停提示；由于 iframe 覆盖较大的阅读区域，鼠标停留时提示容易长期显示。

## 目标

- 不再显示由 iframe `title` 触发的「原始网页」悬停提示。
- 保留 iframe 自身清晰的无障碍名称。
- 不改变原始网页的加载、切换、回退链接及按钮行为。

## 方案比较

1. 直接删除 `title`：可以消除提示，但会失去 iframe 自身的无障碍名称，不采用。
2. 用 `aria-label="原始网页"` 替代 `title`：既不会产生原生悬停提示，又保留可访问名称，采用。
3. 使用隐藏文本与 `aria-labelledby`：同样可访问，但需要额外 DOM 和样式，对当前固定名称没有收益，不采用。

## 组件与数据流

仅调整 `ArticleOriginalWebpage`：

- 删除 iframe 的 `title`。
- 添加 `aria-label="原始网页"`。
- 保留外层 section、URL、惰性加载、referrer policy 与回退链接。

首页和收藏页仍通过现有组件渲染 iframe，不改变页面状态或数据流。

## 测试

- 组件测试通过 `aria-label` 精确定位 iframe。
- 明确断言 iframe 不再具有 `title`。
- 首页和收藏页测试改用 iframe 的无障碍名称定位元素。
- 运行相关组件与页面测试，并执行前端构建验证类型和产物。
