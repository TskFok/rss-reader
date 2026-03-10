# RSS 阅读器

基于 Go + React 的单可执行文件 RSS 阅读器，支持 MySQL 存储、Docker 部署。

## 功能

- RSS 订阅：通过链接添加订阅，可设置更新间隔（30 分钟 ~ 24 小时）
- 用户系统：注册、登录，新用户默认锁定，需超级管理员解锁
- 订阅隔离：每个用户仅能访问自己的订阅和文章
- 定时更新：后台每分钟检查并抓取待更新的订阅
- 单可执行文件：前后端打包为单一二进制，便于部署

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
| GET | /api/feeds | 订阅列表 |
| POST | /api/feeds | 添加订阅 |
| PUT | /api/feeds/:id | 更新订阅设置 |
| DELETE | /api/feeds/:id | 删除订阅 |
| GET | /api/articles | 文章列表 |
| PUT | /api/articles/:id/read | 标记已读 |
| GET | /api/admin/users | 用户列表（超级管理员） |
| PUT | /api/admin/users/:id/unlock | 解锁用户（超级管理员） |

## 环境变量

- `DB_DSN`：数据库连接串
- `JWT_SECRET`：JWT 密钥
- `PORT`：服务端口
- `CONFIG`：配置文件路径
