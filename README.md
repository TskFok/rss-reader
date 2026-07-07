# RSS 阅读器

基于 Go + React 的单可执行文件 RSS 阅读器，支持 MySQL 存储、Docker 部署、用户隔离、订阅管理、AI 处理、AI 总结和飞书集成。

## 功能

### 账号与权限

- 用户注册、用户名密码登录和 JWT 鉴权
- 飞书 OAuth 登录，支持 Web 回调和 JSON code 换票接口
- 新用户默认锁定，超级管理员可解锁用户
- 首个注册用户或 `super_admin.username` 指定用户会成为超级管理员
- 多用户数据隔离：订阅、分类、代理、AI 模型、文章状态和总结记录均按用户归属隔离

### RSS 阅读

- 通过 RSS 地址添加订阅，添加时会抓取并解析订阅标题
- 支持订阅分类、分类拖拽排序、订阅筛选和分类折叠状态记忆
- 支持为订阅设置抓取代理，代理支持 HTTP、HTTPS、SOCKS5 和 SOCKS5H
- 支持设置更新间隔，范围为 5 分钟到 10080 分钟
- 支持手动刷新单个订阅，也支持后台定时刷新
- 支持按订阅设置文章保留天数，`0` 表示永不过期
- 每天 04:00 自动清理过期文章，已收藏文章不会被清理
- 支持 OPML 导入和导出订阅

### 文章管理

- 文章列表支持全部订阅、单订阅、已读、未读、收藏筛选
- 未筛选读/未读时默认未读优先，再按发布时间倒序
- 支持分页加载、键盘上下键切换文章和自动标记已读
- 支持标记已读和收藏/取消收藏
- 文章内容使用 RSS 原文 HTML 展示，并支持切换原文/译文显示

### AI 能力

- 支持配置 OpenAI 兼容的 `chat/completions` 模型接口
- 支持为模型设置 API Key、备用模型和拖拽排序
- 支持测试模型可用性
- 每个订阅可独立开启 AI 分类、AI 翻译和目标语言
- 新文章入库后会异步执行 AI 分类/翻译
- 支持手动触发单篇文章 AI 分类、非流式翻译和流式翻译
- AI 翻译会尽量保留原始 HTML 结构、链接、媒体和嵌入节点
- 支持 AI 补漏任务，定时重试已开启 AI 但分类/翻译未成功的文章

### AI 总结

- 支持按时间范围、订阅范围、分页和排序生成文章总结
- 支持流式返回总结内容
- 支持总结模板，配置 system prompt 和 user prompt 前缀
- 支持保存总结历史、查看历史、删除历史和重试历史总结
- 支持定时总结：按上海时区每天指定时间总结昨天文章
- 定时总结会保存每页历史记录，模型失败时自动重试并继续后续页

### 飞书集成与运维

- 支持飞书机器人通知配置：Webhook 或飞书开放平台 API
- 支持测试当前用户的飞书机器人配置
- 定时总结失败时可发送飞书告警
- 支持错误日志记录、分页查看和删除
- 支持 Gin debug 开关和日志等级配置
- 前后端可打包为单一二进制，便于直接部署

## 快速开始

### 本地运行

1. 复制配置并修改数据库连接：

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml 中的 database.dsn
```

2. 创建 MySQL 数据库：

```sql
CREATE DATABASE rss_reader CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

3. 构建并运行：

```bash
make build
./rss-reader
```

或分别启动前后端开发：

```bash
# 终端 1：启动后端
go run ./cmd/server

# 终端 2：启动前端（需先 cp config.example.yaml config.yaml）
cd web && npm run dev
```

**注意**：若使用单可执行文件（`./rss-reader`），前端来自构建时嵌入的 `cmd/server/static`。修改前端后必须执行 `make build`（会先构建 web 并拷贝到 static 再编译 Go），否则会一直用旧页面；部署后建议浏览器强刷（Ctrl+Shift+R / Cmd+Shift+R）以跳过缓存。

### Docker 部署

```bash
docker-compose up -d
```

访问 http://localhost:8080

### 构建

```bash
# 仅当前平台单二进制，产物：./rss-reader
make build-local

# 各平台单独打包，产物在 dist/
make build-linux-amd64    # dist/rss-reader-linux-amd64
make build-linux-arm64    # dist/rss-reader-linux-arm64
make build-darwin-amd64   # dist/rss-reader-darwin-amd64（Intel Mac）
make build-darwin-arm64   # dist/rss-reader-darwin-arm64（Apple Silicon）
make build-windows-amd64  # dist/rss-reader-windows-amd64.exe

# 打包全部平台
make build
```

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 注册 |
| POST | /api/auth/login | 登录 |
| GET | /api/auth/feishu/login-url | 获取飞书授权入口 |
| GET | /api/auth/feishu/login | 跳转飞书授权页 |
| GET | /api/auth/feishu/callback | 飞书 OAuth 回调 |
| POST | /api/auth/feishu/exchange | 飞书 code 换取站内 JWT |
| GET | /api/feeds/opml | 导出 OPML |
| POST | /api/feeds/opml | 导入 OPML |
| GET | /api/categories | 分类列表 |
| POST | /api/categories | 创建分类 |
| PUT | /api/categories/reorder | 分类排序 |
| PUT | /api/categories/:id | 更新分类 |
| DELETE | /api/categories/:id | 删除分类 |
| GET | /api/proxies | 代理列表 |
| POST | /api/proxies | 创建代理 |
| PUT | /api/proxies/:id | 更新代理 |
| DELETE | /api/proxies/:id | 删除代理 |
| GET | /api/ai-models | AI 模型列表 |
| POST | /api/ai-models | 创建 AI 模型 |
| PUT | /api/ai-models/reorder | AI 模型排序 |
| PUT | /api/ai-models/:id | 更新 AI 模型 |
| DELETE | /api/ai-models/:id | 删除 AI 模型 |
| POST | /api/ai-models/:id/test | 测试 AI 模型 |
| GET | /api/feeds | 订阅列表 |
| POST | /api/feeds | 添加订阅 |
| POST | /api/feeds/:id/refresh | 立即刷新订阅 |
| PUT | /api/feeds/:id | 更新订阅设置 |
| DELETE | /api/feeds/:id | 删除订阅 |
| GET | /api/articles | 文章列表（不含正文大字段） |
| GET | /api/articles/:id | 文章详情（含完整正文，同时标记已读） |
| PUT | /api/articles/:id/favorite | 收藏或取消收藏 |
| POST | /api/articles/:id/ai/classify | 手动 AI 分类 |
| POST | /api/articles/:id/ai/translate | 手动 AI 翻译 |
| POST | /api/articles/:id/ai/translate/stream | 手动 AI 流式翻译 |
| POST | /api/articles/summarize | 流式生成 AI 总结 |
| GET | /api/summary-templates | 总结模板列表 |
| POST | /api/summary-templates | 创建总结模板 |
| PUT | /api/summary-templates/:id | 更新总结模板 |
| DELETE | /api/summary-templates/:id | 删除总结模板 |
| GET | /api/summary-histories | 总结历史列表 |
| POST | /api/summary-histories | 保存总结历史 |
| POST | /api/summary-histories/:id/retry | 重试总结历史 |
| DELETE | /api/summary-histories/:id | 删除总结历史 |
| GET | /api/summary-schedules | 定时总结列表 |
| POST | /api/summary-schedules | 创建定时总结 |
| PUT | /api/summary-schedules/:id | 更新定时总结 |
| DELETE | /api/summary-schedules/:id | 删除定时总结 |
| GET | /api/error-logs | 错误日志列表 |
| DELETE | /api/error-logs/:id | 删除错误日志 |
| GET | /api/users/me/settings | 当前用户设置 |
| PUT | /api/users/me/settings | 更新当前用户设置 |
| POST | /api/users/me/feishu-bot/test | 测试飞书机器人 |
| GET | /api/admin/users | 用户列表（超级管理员） |
| PUT | /api/admin/users/:id/unlock | 解锁用户（超级管理员） |
| GET | /api/admin/users/:id/feishu/bind-url | 获取用户飞书绑定链接（超级管理员） |

除注册、登录和飞书授权相关接口外，其他接口均需要 `Authorization: Bearer <JWT>`。

更详细的移动端对接说明见 `docs/ANDROID_REST_API.md`，OpenAPI 草稿见 `docs/openapi.yaml`。

## 环境变量

- `CONFIG`：配置文件路径
- `DB_DSN`：数据库连接串
- `JWT_SECRET`：JWT 密钥
- `PORT`：服务端口
- `GIN_DEBUG`：是否开启 Gin debug 模式，支持 `1`、`true`、`on`
- `LOG_LEVEL`：日志等级，支持 `debug`、`info`、`warn`、`error`
- `FEISHU_APP_ID`：飞书应用 App ID
- `FEISHU_APP_SECRET`：飞书应用 App Secret
- `FEISHU_REDIRECT`：飞书 OAuth 回调地址

## 配置文件

默认读取 `config.yaml`；若不存在且未指定 `CONFIG`，会回退到 `config.example.yaml`。

主要配置项：

- `server.port`：服务端口
- `server.debug`：是否开启 Gin debug 模式
- `log.level`：日志等级
- `database.dsn`：MySQL 连接串（建议包含 `parseTime=True&loc=Asia%2FShanghai`；未写 `loc` 时应用启动会自动补齐）
- `jwt.secret`：JWT 签名密钥
- `jwt.expire_hours`：JWT 有效期小时数
- `super_admin.username`：可选，指定超级管理员用户名
- `feishu.app_id`、`feishu.app_secret`、`feishu.redirect`：飞书 OAuth 和 API 通知配置
- `ai_backfill.enabled`：是否开启 AI 补漏任务
- `ai_backfill.interval_minutes`：AI 补漏执行周期
- `ai_backfill.classify_batch`：每轮最多补分类条数
- `ai_backfill.translate_batch`：每轮最多补翻译条数
- `ai_backfill.delay_ms_between_calls`：AI 补漏调用间隔毫秒数

### 时区

应用统一使用 **Asia/Shanghai（UTC+8）**：

- 进程启动时设置 `time.Local` 与 `TZ=Asia/Shanghai`
- MySQL DSN 使用 `loc=Asia%2FShanghai`，连接后执行 `SET time_zone = '+08:00'`
- Docker Compose 中 app 与 mysql 均设置 `TZ`，MySQL 默认 `--default-time-zone=+08:00`

非 Docker 部署时，请确保 MySQL 服务器时区亦为 `+08:00`，例如：

```sql
SET GLOBAL time_zone = '+08:00';
```
