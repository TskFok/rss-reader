# 固定订阅内容详情标题设计

## 目标

阅读页打开订阅内容详情后，详情标题和操作区需要固定在详情面板顶部；用户滚动长正文时，标题不随正文一起滑动。

## 当前问题

[web/src/pages/Home.tsx](/Users/tskfok/workspace/myself/rss-reader/web/src/pages/Home.tsx) 中 `.article-detail-dock` 同时承载标题、元信息和正文，并且该容器自身负责滚动。正文滚动时，标题区域也会被卷出视口。

## 方案

将详情面板拆成两层：

- `.article-detail-dock`：外层面板，只负责布局，不再作为正文滚动容器。
- `.article-detail-scroll`：内层正文滚动容器，包含元信息和 `ArticleDetailContent`。

标题、AI 操作、收藏和关闭按钮保留在 `.article-detail-header` 中，作为面板的固定顶部区域。现有切换订阅、切换文章时重置滚动位置的逻辑改为作用于 `.article-detail-scroll`。

## 测试

在 `web/src/pages/Home.test.tsx` 增加行为测试：打开文章详情后，确认 `.article-detail-scroll` 存在，并且 `.article-detail-title` 不在该滚动容器内。这样可以防止后续把标题重新放回正文滚动区。

## 范围

只改 Web 阅读页结构、相关样式和前端测试。不改 API、不改数据库、不新增 SQL 查询。
