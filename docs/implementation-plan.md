# GET /api/articles 性能优化 — Implementation Plan

> **Status: Approved for Implementation**  
> Version: v2 (final)  
> Date: 2026-07-07

---

## 1. 背景和目标

### 1.1 背景

当前 `GET /api/articles` 通过 `ArticleService.List` 返回完整 `ArticleWithRead`（嵌入 `models.Article`），包含 `content`、`content_translated`、`guid_raw` 等 `longtext` 字段。前端 `Home` / `Favorites` 将列表项直接用于详情展示，导致：

- 响应体过大（P0 瓶颈）
- DB 执行 `SELECT articles.*`，JSON 裁剪无法避免 I/O
- 列表查询 3 次 DB round-trip（COUNT、Find+Preload、user_articles IN）

项目已有 `GetWithRead` 供 manual AI 使用，但无对外 `GET /api/articles/:id`；`article_ai_backfill.go` 已有 `Select` 跳过大字段的先例。

### 1.2 目标

| 优先级 | 目标 |
|--------|------|
| P0 | 缩小列表响应体，去掉 `content`、`content_translated`、`guid_raw` |
| P1 | DB 列表查询不读取 longtext 列 |
| P1 | 合并 `user_articles` 查询，减少 round-trip |
| P1 | 新增详情接口，前端按需加载正文 |
| P2 | 改善详情切换体验（头部即时 + skeleton + 内存缓存） |

### 1.3 非目标（本轮不做）

- `COUNT(*)` 查询优化
- 复合索引 / `published_at` 索引
- 未读优先排序逻辑改造
- `GetWithRead` 多查询合并优化
- 兼容参数（`?include_content=true`）或 API v2
- Redis、缓存中间件、新 npm/go 依赖
- 首屏或列表批量 prefetch 详情

---

## 2. 已确认需求

| ID | 需求 |
|----|------|
| R1 | P0：缩小列表响应体 |
| R2 | 列表瘦身 + 新增 `GET /api/articles/:id` |
| R3 | 列表不返回 `content`、`content_translated`、`guid_raw` |
| R4 | 语言切换判断改为 `title_translated` 或 `feeds.ai_translate_enabled`，不依赖列表中的 `content_translated` |
| R5 | 详情体验：头部即时展示列表元数据 + 正文 skeleton + 内存缓存 |
| R6 | 列表 DB 查询显式 `Select`，跳过 longtext |
| R7 | 合并 `user_articles` 到主查询，去掉第二次 IN 查询 |
| R8 | 接受 breaking change，同步更新 API 文档 |
| R9 | 缓存失效：手动 AI 成功更新缓存；`reloadKey` 清空全部详情缓存；登出清空缓存 |
| R10 | 本轮仍同步返回准确 `total`，禁止 deferred/partial total |

---

## 3. Phase 0 冻结规格（实施前必须完成）

### 3.1 冻结表 A：`ArticleListItem` 字段

**列表响应包含：**

| 字段 | 用途 |
|------|------|
| `id`, `feed_id`, `guid`, `title`, `link` | 列表/详情头/外链 |
| `ai_process_status`, `ai_last_error` | 列表 pending 态、AI 错误 |
| `ai_category`, `ai_category_translated`, `title_translated` | 列表展示、语言切换 |
| `published_at`, `created_at`, `updated_at` | 日期展示、stale pending 判断 |
| `read`, `favorite` | 列表样式、操作 |
| `feed_title`, `feed_ai_translate_enabled`, `feed_ai_classify_enabled`, `feed_ai_model_id`, `feed_ai_target_language` | 列表 meta、ManualAI 配置 |

**列表响应排除：**

`content`, `content_translated`, `guid_raw`

**详情响应（`GET /api/articles/:id`）包含：**

上述全部字段 + `content` + `content_translated` + `guid_raw`（与现 `GetWithRead` / `models.Article` 一致）

### 3.2 冻结表 B：详情接口权限矩阵

| 场景 | HTTP | 响应 |
|------|------|------|
| 未携带有效 Bearer | **401** | `{ "error": "..." }` |
| 文章不存在 | **404** | `{ "error": "文章不存在" }` |
| 文章存在但 feed 不属于当前用户 | **404** | `{ "error": "文章不存在" }` |
| 正常访问 | **200** | `{ "article": ArticleWithRead }` |

### 3.3 冻结表 C：详情缓存失效矩阵

| 事件 | 行为 | 挂载位置 |
|------|------|----------|
| 选中文章，缓存命中 | 直接展示 `selectedDetail`，不请求 | `useArticleDetail` |
| 选中文章，缓存未命中 | `detailLoading=true`，发起 `GET` | `useArticleDetail` |
| 快速切换文章 | **AbortController 取消进行中的 GET**；仅最新 `selectedId` 可写回 state | `useArticleDetail` |
| 手动 AI 流式/完成 `onArticlePatched` | 写缓存 + 更新 `selectedDetail`；回调内校验 `article.id === selectedId` | `Home` / `Favorites` |
| `markRead` | 更新 list/detail 的 `read`，保留正文缓存 | 页面现有逻辑 |
| `toggleFavorite` | 更新 `favorite`；Favorites 取消收藏时删除该 id 缓存 + `setSelected(null)` | 页面现有逻辑 |
| `reloadKey` 变化（订阅刷新） | `clearCache()` 清空全部 | `Home` |
| 语言切换 | 不清缓存（展示字段本地切换） | `ArticleDetailContent` |
| 登出 | `clearCache()` | `AuthContext.logout` |
| 详情 GET 404 | 展示错误 UI；不写入缓存 | `useArticleDetail` |
| 详情 GET 网络错误 | 展示错误 UI + 重试按钮；不写入缓存 | `useArticleDetail` |

### 3.4 假设与待验证项

| ID | 类型 | 内容 |
|----|------|------|
| H1 | 已闭合 | 登出清缓存挂 `AuthContext.logout` |
| H2 | 假设 | 详情加载完成前不渲染 `ArticleManualAI` |
| H3 | 假设 | 语言切换不清缓存 |
| H4 | 待验证 | `read` + `favorite` 组合筛选与现网结果等价 |
| H5 | 待验证 | GORM `Select`+JOIN 在 SQLite（测试）与 MySQL（生产）字段映射一致 |
| H6 | 假设 | 无外部 Android 产品在产；仅 `docs/android/` 示例需同步 |
| H7 | 必做 | 详情请求必须 AbortController 防竞态 |
| H8 | 假设 | `reloadKey` 后选中文章被清空（现有 Home 行为）保持不变 |

---

## 4. 设计决策

| ID | 决策 | 理由 |
|----|------|------|
| D1 | 继续返回准确 `total` | `hasMore = articles.length < total` 硬依赖 |
| D2 | 统一单 `LEFT JOIN user_articles ua` | `idx_user_article` 保证 1:1，COUNT 不膨胀 |
| D3 | 新建 `ArticleListItem`，不嵌入 `models.Article` | 防 longtext 泄漏；与 backfill `Select` 先例一致 |
| D4 | JOIN feeds 时 SELECT feed 列，去掉 `Preload("Feed")` | 避免重复加载 |
| D5 | `GET /api/articles/:id` → `{ "article": ArticleWithRead }` | 与 manual AI 响应一致 |
| D6 | 越权/不存在 → 404；未认证 → 401 | 项目惯例 |
| D7 | 复用 `GetWithRead`，不优化其 3 次查询 | 范围控制 |
| D8 | TS：`ArticleListItem` + `Article extends ArticleListItem` | 最小类型侵入 |
| D9 | 新建 `useArticleDetail` hook | Home/Favorites 去重 |
| D10 | `Map<id, Article>` 无 LRU | 个人 RSS 规模可接受 |
| D11 | breaking + 文档硬切，禁止拆分部署 | 见 §8.3 |
| D12 | 列表保留 `updated_at`、`ai_last_error` | ManualAI / stale pending 需要 |

---

## 5. 涉及文件和预计修改点

### 5.1 后端

| 文件 | 修改 |
|------|------|
| `internal/services/article.go` | `ArticleListItem`；`List` 重构；JOIN/Select；返回类型变更 |
| `internal/handlers/article.go` | `List` 适配；新增 `Get` |
| `cmd/server/main.go` | `GET /articles/:id` |
| `internal/services/article_test.go` | 列表字段、筛选、JOIN 等价、组合筛选 |
| `internal/handlers/article_test.go` | `Get` 200/404 |

### 5.2 前端

| 文件 | 修改 |
|------|------|
| `web/src/api/client.ts` | `ArticleListItem`、`Article`、`articlesApi.get` |
| `web/src/hooks/useArticleDetail.ts` | 新建；缓存、AbortController、错误态 |
| `web/src/contexts/AuthContext.tsx` | `logout` 时清缓存 |
| `web/src/pages/Home.tsx` | 接入 hook；lang toggle；详情 error UI |
| `web/src/pages/Favorites.tsx` | 同 Home |
| `web/src/pages/Home.test.tsx` | mock `get`、无 content fixture |
| `web/src/components/ArticleList.tsx` | props 类型 |
| `web/src/components/ArticleList.test.tsx` | 瘦 fixture |

**复用（不修改）：** `ArticleDetailContent.tsx`、`articleDisplayLang.ts`、`articleManualAi.ts`

### 5.3 文档

| 文件 | 修改 |
|------|------|
| `README.md` | 新增 `GET /api/articles/:id` |
| `docs/ANDROID_REST_API.md` | 列表字段 + 详情接口 + 迁移说明 |
| `docs/android/src/main/java/com/rssreader/api/RssReaderApi.kt` | 拆分 list/detail DTO + `getArticle` |

---

## 6. 实施步骤

### 6.1 里程碑与发布边界

```
M1  后端列表瘦身（Phase 1）     → 可 PR，不可上线
M2  后端详情接口（Phase 2）     → 可与 M1 同 PR，不可上线
M3  前端 + 文档（Phase 3–6）   → 唯一可上线单元
M4  全量验收（Phase 7）         → 发布 gate
```

**发布禁令：任何环境不得仅部署 M1/M2 而无 M3。**

### Phase 0：冻结确认

1. 确认 §3.1–3.3 三张冻结表无异议
2. 确认 H6：无外部 Android 产品消费者（默认仅文档示例）
3. 确认 H8：接受 `reloadKey` 后详情 dock 被清空

### Phase 1：后端 — 列表瘦身

1. 定义 `ArticleListItem` 及列表 `Select` 列常量（参考 `articleBackfillClassifySelect` 风格）
2. 重构 `List`：
   - 保留 `JOIN feeds` + `feeds.user_id` 过滤
   - 合并为单一 `LEFT JOIN user_articles ua ON ua.article_id = articles.id AND ua.user_id = ?`
   - 筛选改为 WHERE：`read=true` → `ua.read_status = 1`；`read=false` → `ua.id IS NULL OR ua.read_status = 0`；`favorite=true` → `ua.favorite = 1`
   - `read==nil` → `ORDER BY COALESCE(ua.read_status, 0) ASC, articles.published_at DESC, ...`
   - JOIN feeds 时 SELECT feed 所需列；去掉 `Preload("Feed")`
   - `Select` 文章列（跳过 longtext）；去掉第三次 `user_articles IN` 查询
   - 保留 `Count` 并返回准确 `total`
3. 更新 `ArticleHandler.List`
4. 测试：列表无三 longtext 字段；read/favorite/未读优先；read+favorite 组合；JOIN 等价性

### Phase 2：后端 — 详情接口

1. 新增 `ArticleHandler.Get`：`GetWithRead` → `{ "article": ar }`；404 处理
2. `main.go` 注册 `GET /articles/:id`
3. 测试：200 含 content；404 不存在/越权；401 未认证

### Phase 3：前端 — API 与类型

1. `ArticleListItem` 接口；`Article extends ArticleListItem`
2. `articlesApi.list` → `ArticleListItem[]`；`articlesApi.get(id)` → `{ article: Article }`

### Phase 4：前端 — `useArticleDetail` hook

1. 新建 `web/src/hooks/useArticleDetail.ts`：缓存、`selectArticle`、`patchArticle`、`clearCache`、`removeFromCache`、`retryDetail`
2. **必做 AbortController**：切换选中时 abort；写回前校验 `id === selectedId`
3. `AuthContext.logout` 调用 `clearCache`

### Phase 5：前端 — 页面接入

1. `Home.tsx`：接入 hook；lang toggle 去掉 `content_translated`；详情 skeleton/error/retry；`ArticleDetailContent` 和 `ArticleManualAI` 仅在 `selectedDetail` 存在时渲染；`reloadKey` → `clearCache`；流式 AI 校验 `selectedId`
2. `Favorites.tsx`：同 Home；取消收藏时 `removeFromCache(id)`
3. `ArticleList.tsx`：props → `ArticleListItem[]`

### Phase 6：文档与发布检查表

1. 更新 `README.md`、`ANDROID_REST_API.md`、`RssReaderApi.kt`
2. 发布检查表：
   - [ ] 后端 M1+M2 已构建
   - [ ] 前端 `npm run build` 完成且静态资源已嵌入/同发
   - [ ] 文档已更新
   - [ ] 未出现仅后端或仅前端拆分部署
   - [ ] `go test` / `npm test` 通过

### Phase 7：全量验收

按 §9 验收标准逐项勾选（AC1–AC20）。

---

## 7. 测试计划

### 7.1 后端

| 用例 | 文件 |
|------|------|
| 列表不含三 longtext 字段 | `article_test.go` |
| read/favorite/未读优先 | `article_test.go` |
| read+favorite 组合 | `article_test.go` |
| JOIN 后 read/favorite/feed_* 正确 | `article_test.go` |
| total 与 COUNT 一致 | `article_test.go` |
| Handler Get 200/404/401 | `article_test.go` |

### 7.2 前端

| 用例 | 文件 |
|------|------|
| 选中触发 `articlesApi.get` | `Home.test.tsx` |
| 列表无 content 时详情 mock 渲染 | `Home.test.tsx` |
| lang toggle 不依赖 `content_translated` | `Home.test.tsx` |
| 列表点击 | `ArticleList.test.tsx` |
| manual AI slot 逻辑 | `articleManualAi.test.ts` |

### 7.3 验证命令

```bash
go test ./internal/services/ -run 'Article' -v -count=1
go test ./internal/handlers/ -run 'Article' -v -count=1
cd web && npm test -- --run
```

---

## 8. 风险和回滚

### 8.1 风险与缓解

| 风险 | 缓解 |
|------|------|
| JOIN 重构回归 | H4 组合测试 + JOIN 等价性测试 |
| GORM Select 映射错误 | H5 双环境验证 |
| 详情竞态串文 | AbortController |
| 拆分部署导致前端崩溃 | §8.3 发布禁令 |
| ManualAI 误判 | H2：detail 就绪后才渲染 |
| 并发压 DB | 仅选中时单次 GET，不 batch prefetch |

### 8.2 回滚

1. Git revert 整个 PR（后端 + 前端 + 文档）
2. 无 DB schema 变更
3. 禁止先回滚前端保留后端

### 8.3 发布策略

| 规则 | 说明 |
|------|------|
| 唯一发布单元 | M1+M2+M3 同版本同部署 |
| 禁止 | 仅部署后端、仅部署前端、灰度拆分 |
| 构建 | 含 `npm run build` + 静态资源嵌入（若适用） |
| 外部客户端 | H6：仅文档示例 |

---

## 9. 验收标准

### 9.1 功能（必须）

| ID | 标准 |
|----|------|
| AC1 | 列表 `items[]` 不含 `content`、`content_translated`、`guid_raw` |
| AC2 | `GET /api/articles/:id` 返回完整正文 |
| AC3 | 不存在/越权 → 404；未认证 → 401 |
| AC4 | 列表功能不退化：标题、分类、订阅名、已读、AI pending、筛选、load-more |
| AC5 | 点击：头部即时、skeleton、正文正确 |
| AC6 | ↑/↓ 已访问文章缓存秒开 |
| AC7 | 手动 AI / 收藏 / 已读后列表与详情一致 |
| AC8 | `reloadKey` 后无过期正文 |
| AC9 | 登出后缓存清空 |
| AC10 | 文档与实现一致 |
| AC11 | `go test` / `npm test` 通过 |
| AC12 | `read+favorite` 组合筛选与现网一致 |
| AC13 | 快速切换文章详情不串文 |
| AC14 | 详情 404/网络错误有 UI + 重试 |
| AC15 | `ArticleManualAI` 仅在 `selectedDetail` 存在时展示 |
| AC16 | 发布检查表全部勾选 |
| AC17 | 禁止 deferred/partial `total`；`hasMore` 正确 |

### 9.2 性能（建议测量）

| ID | 标准 |
|----|------|
| AC18 | 典型 20 条列表：抓包对比 shrink 前后响应体积 |
| AC19 | SQL log：`List` 不读 longtext 列 |
| AC20 | 单次 `List` DB round-trip ≤ 2（COUNT + Find） |

### 9.3 不在本轮验收范围

COUNT 耗时、排序性能、`GetWithRead` 优化、LRU 缓存

---

## 10. 实现阶段待验证项

| ID | 内容 | 处理时机 |
|----|------|----------|
| U1 | 401 未认证的具体响应文案 | Phase 2 |
| U2 | GORM Select+JOIN 在 MySQL 生产环境字段映射 | Phase 1 后冒烟 |
| U3 | `AuthContext` 与 `useArticleDetail` 缓存清理桥接方式 | Phase 4 |
| U4 | JOIN 等价性：merge 前后排序完全一致 | Phase 1 测试 |
| U5 | 外部 Android 产品是否在产 | 默认 H6 假设 |
| U6 | 性能 Before/After 具体数值 | Phase 7 采样 |
