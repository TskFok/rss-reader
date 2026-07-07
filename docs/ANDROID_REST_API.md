# Android 客户端 — REST API 说明

本文档与仓库内 `docs/openapi.yaml` 一致，描述与 **rss-reader** 后端对接时所需的接口。

**Base URL**：部署后通常为 `https://<主机>/api`（注意末尾 **无** 斜杠时路径为 `/api/auth/login`）。

**认证**：除「公开」接口外，请求头需携带：

```http
Authorization: Bearer <JWT>
Content-Type: application/json
```

JWT 在登录成功响应的 `token` 字段中取得；失效时接口返回 `401`，客户端应清除本地 token 并跳转登录。

---

## 1. 登录（用户名 / 密码）

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `POST /api/auth/login` |
| 认证 | 不需要 |
| 请求体 | `{ "username": "string", "password": "string" }` |

**成功 `200`**：

```json
{
  "token": "<JWT>",
  "user": {
    "id": 1,
    "username": "alice",
    "status": "active",
    "is_super_admin": false,
    "feishu_id": null,
    "feishu_name": "",
    "created_at": "2026-01-01T12:00:00Z"
  }
}
```

**常见错误**：`401` 用户名或密码错误；`403` 账号被锁定。

---

## 2. 飞书登录

后端采用 **OAuth 授权码** 模式；飞书回调到服务端后，**当前实现返回 HTML 页面**，通过 `postMessage` 把 `token` 交给浏览器父窗口（与 Web 前端一致）。

### 2.1 获取授权入口（公开）

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `GET /api/auth/feishu/login-url` |
| 认证 | 不需要 |

**成功 `200`**：

```json
{
  "url": "/api/auth/feishu/login?state=login",
  "goto": "https://www.feishu.cn/suite/passport/oauth/authorize?..."
}
```

- **`goto`**：在 **WebView / Chrome Custom Tabs** 中打开，引导用户登录飞书。
- 授权成功后，浏览器会跳转到服务端在飞书开放平台配置的 **重定向 URL**，即 `GET /api/auth/feishu/callback?code=...&state=...`。

### 2.2 回调（公开）

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `GET /api/auth/feishu/callback` |
| Query | `code`（必填）、`state` |

响应为 **HTML**，非 JSON。原生 Android 常见做法：

1. 使用 **WebView**，在 `shouldOverrideUrlLoading` / `onPageFinished` 中若检测到重定向 URL 含 `code=`，再自行请求服务端（若后续增加 **JSON 换票接口** 则更稳妥）；或  
2. 由产品侧为 App 配置 **App Link** 与后端共同增加 `POST` 换票接口（当前仓库默认仅 HTML 回调）。

**说明**：若需「纯原生、无 WebView」飞书登录，需要服务端提供 `code` 换 `token` 的 JSON 接口；可基于现有 `Callback` 逻辑扩展。

### 2.3 Code 换票（公开，推荐 App 使用）

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `POST /api/auth/feishu/exchange` |
| 认证 | 不需要 |
| 请求体 | `{ "code": "string", "state": "string(可选)" }` |

**成功 `200`**：

```json
{
  "token": "<JWT>",
  "user": {
    "id": 1,
    "username": "alice",
    "status": "active",
    "is_super_admin": false,
    "created_at": "2026-01-01T12:00:00Z"
  }
}
```

**错误**：
- `400` 参数错误 / code 无效 / 传入了绑定态 `state=bind:*`
- `403` 账号锁定（包含新建后默认锁定）

---

## 3. 注销登录

本服务使用 **无状态 JWT**，**没有** `POST /logout` 接口。

**客户端行为**：删除本地保存的 `token`（及用户信息）即可；可选跳转到登录页。

若将来增加服务端黑名单或刷新令牌，再对接新接口。

---

## 4. 订阅列表

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `GET /api/feeds` |
| 认证 | 需要 Bearer |

**成功 `200`**：JSON 数组，每个元素为一条订阅（字段与 Web 端 `Feed` 一致），例如：

- `id`, `user_id`, `url`, `title`, `update_interval_minutes`, `expire_days`
- `ai_model_id`, `ai_classify_enabled`, `ai_translate_enabled`, `ai_target_language`
- `category`（嵌套）、`last_fetched_at`, `created_at` 等

### 4.1 立即刷新订阅

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `POST /api/feeds/{id}/refresh` |
| 认证 | 需要 Bearer |

**成功 `200`**：返回刷新后的订阅对象。若订阅不存在返回 `404`，抓取失败返回 `502`。

---

## 5. 订阅分类

用于订阅分组展示、创建、编辑、删除以及拖拽排序。

### 5.1 分类列表

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `GET /api/categories` |
| 认证 | 需要 Bearer |

**成功 `200`**：

```json
[
  {
    "id": 1,
    "user_id": 1,
    "name": "科技",
    "sort_order": 0,
    "created_at": "2026-01-01T12:00:00Z",
    "updated_at": "2026-01-01T12:00:00Z"
  }
]
```

### 5.2 创建分类

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `POST /api/categories` |
| 认证 | 需要 Bearer |
| 请求体 | `{ "name": "科技" }` |

**成功 `201`**：返回新建分类对象。

**错误**：
- `400` 参数错误 / 名称为空
- `409` 分类名称已存在

### 5.3 更新分类

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `PUT /api/categories/:id` |
| 认证 | 需要 Bearer |
| 请求体 | `{ "name": "技术" }` |

**成功 `200`**：返回更新后的分类对象。

**错误**：
- `400` 参数错误 / ID 非法
- `404` 分类不存在
- `409` 分类名称已存在

### 5.4 删除分类

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `DELETE /api/categories/:id` |
| 认证 | 需要 Bearer |

**成功 `200`**：

```json
{ "message": "删除成功" }
```

### 5.5 分类排序

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `PUT /api/categories/reorder` |
| 认证 | 需要 Bearer |
| 请求体 | `{ "id_list": [3, 2, 1] }` |

含义：把当前用户分类顺序调整为 `3 -> 2 -> 1`。

**成功 `200`**：

```json
{ "message": "排序已更新" }
```

---

## 6. 订阅内容（文章列表）

对应「点击某条订阅后展示该订阅下的文章」。

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `GET /api/articles` |
| 认证 | 需要 Bearer |
| Query（常用） | 见下表 |

| 参数 | 说明 |
|------|------|
| `feed_id` | 可选，**传入则只返回该订阅下的文章**（用于点击侧边栏某一订阅） |
| `read` | 可选，`true` 仅已读，`false` 仅未读 |
| `page` | 页码，默认 `1` |
| `page_size` | 每页条数，默认 `20`，最大 `100` |

**成功 `200`**：

```json
{
  "items": [ /* ArticleListItem 对象列表 */ ],
  "total": 100
}
```

列表项（`ArticleListItem`）**不包含** `content`、`content_translated`、`guid_raw` 等大字段。常见字段：`id`, `feed_id`, `guid`, `title`, `link`, `published_at`, `read`, `favorite`, `feed_title`，以及 AI 相关 `ai_category`, `title_translated`, `ai_process_status` 等。

> **Breaking change（2026-07）**：列表接口不再返回正文；客户端需在用户打开文章时调用 `GET /api/articles/:id` 获取 `content` / `content_translated` / `guid_raw`。

### 6.1 文章详情

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `GET /api/articles/:id` |
| 认证 | 需要 Bearer |

**成功 `200`**：

```json
{
  "article": { /* ArticleWithRead，含 content、content_translated、guid_raw */ }
}
```

**错误**：未认证 `401`；文章不存在或订阅不属于当前用户 `404`（`{ "error": "文章不存在" }`）。

**其它**（按需）：`PUT /api/articles/:id/read` 标记已读，`PUT /api/articles/:id/favorite` 收藏。

---

## 7. AI 手动翻译（流式）

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `POST /api/articles/:id/ai/translate/stream` |
| 认证 | 需要 Bearer |
| 请求体 | `{ "ai_model_id": 1, "ai_target_language": "zh-CN" }`（两个字段都可选） |
| 响应类型 | `text/event-stream`（SSE） |

请求体规则与同步接口 `POST /api/articles/:id/ai/translate` 一致：
- 不传 `ai_model_id`：使用订阅默认模型
- 不传 `ai_target_language` 或传空串：使用订阅默认目标语言
- 若文章尚无 AI 分类，服务端会在同一次模型调用中同时生成分类与译文；流式 `delta` 仍只返回译文正文片段，完成事件里的 `article` 会包含最新分类字段

### 7.1 事件格式

SSE 每行形如：

```text
data: {...json...}
```

服务端会发送以下三类事件：

1. 增量译文：

```json
{ "delta": "<p>正在输出的译文片段..." }
```

2. 完成事件（最终落库后的完整文章）：

```json
{
  "article": {
    "id": 123,
    "title_translated": "翻译后的标题",
    "content_translated": "<p>完整译文 HTML</p>"
  }
}
```

3. 流内错误：

```json
{ "error": "AI 处理失败", "ai_last_error": "模型超时" }
```

### 7.2 客户端处理建议

- 点击“翻译”后切换到译文视图，不刷新页面
- 收到 `delta` 就拼接到当前文章详情区域
- 收到 `article` 后用最终对象覆盖本地临时流数据
- 收到 `error` 后停止读取并提示错误

---

## 8. 总结历史

| 项目 | 说明 |
|------|------|
| 方法 / 路径 | `GET /api/summary-histories` |
| 认证 | 需要 Bearer |
| Query | `page`, `page_size`（分页规则与文章列表类似，`page_size` 默认 20、最大 100） |

**成功 `200`**：

```json
{
  "items": [
    {
      "id": 1,
      "ai_model_id": 1,
      "ai_model_name": "默认模型",
      "summary_template_id": null,
      "summary_template_name": "",
      "start_time": "2026-01-01 00:00",
      "end_time": "2026-01-02 00:00",
      "page": 1,
      "page_size": 20,
      "order": "desc",
      "article_count": 15,
      "total": 80,
      "content": "……总结正文……",
      "error": "",
      "created_at": "2026-01-02T10:00:00Z"
    }
  ],
  "total": 5
}
```

---

## 9. 机器可读规范与示例代码

| 资源 | 路径 |
|------|------|
| OpenAPI 3.0 | `docs/openapi.yaml` |
| Retrofit 接口与 DTO 示例（Kotlin） | `docs/android/` |

可使用 [Swagger Editor](https://editor.swagger.io/) 打开 `openapi.yaml` 生成其它语言客户端。

---

## 10. 错误格式

多数错误响应体为：

```json
{ "error": "中文或英文说明" }
```

HTTP 状态码：`400` 参数错误，`401` 未登录或 token 无效，`403` 无权限，`404` 资源不存在，`409` 冲突等。
