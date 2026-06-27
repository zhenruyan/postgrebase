# PostgreBase — AI原生的无代码API开发工具

PostgreBase 是 AI 原生的无代码 API 开发平台，基于 [PocketBase](https://pocketbase.io) 重构。内置 MCP (Model Context Protocol) 服务器，让 AI 工具（Claude、Cursor、Windsurf）直接操作你的数据。支持 PostgreSQL、MySQL、SQLite 三种数据库，提供即时 REST API + 实时订阅 + Admin UI，5分钟上线你的后端服务。

## 核心特性

- **三数据库引擎：**
    - **PostgreSQL**（默认）— 生产级并发、集群和复杂查询。
    - **MySQL** — 完全兼容，通过 `mysql://` DSN 前缀接入。
    - **SQLite**（纯 Go，无 CGO）— 零依赖本地开发和测试，通过 `sqlite://` DSN 或 `.db` 文件路径使用。
- **集群化架构：** 不依赖本地数据库文件，多实例可轻松部署在负载均衡器后。
- **混合缓存：**
    - **Redis 缓存**（`--redisDsn`）— 集群环境分布式缓存；SSE 实时订阅通过 Redis Pub/Sub 跨节点同步。
    - **内存缓存** — 未配置 Redis 时自动回退到高性能本地内存缓存。
- **MCP (Model Context Protocol) 服务器：**
    - JSON-RPC 2.0 协议，让 AI 工具（Claude Desktop、Cursor、Windsurf）直接操作你的数据。
    - 8 个内置工具：`list_collections`、`get_collection`、`list_records`、`get_record`、`create_record`、`update_record`、`delete_record`、`search_records`。
    - 资源：`postgrebase://collections`、`postgrebase://settings`。
    - 三种传输模式：SSE（HTTP）、Streamable HTTP 和 Stdio。
    - MCP 专用 API Token，支持过期时间，可在 Admin UI 中管理。
- **SQLite 零外部依赖：** 使用 `modernc.org/sqlite`（纯 Go 编译的 SQLite），支持 `CGO_ENABLED=0` 构建。
- **保持 PocketBase 的极致体验：** 100% 兼容原有的 API、Admin UI 和业务逻辑。

## 快速上手

### 环境准备

- Go 1.26.2+
- PostgreSQL、MySQL，或者不需要任何数据库（SQLite 开箱即用）

### 编译构建

```bash
git clone https://github.com/zhenruyan/postgrebase.git
cd postgrebase
go build -o pb ./build/
```

静态构建（无 CGO）：

```bash
CGO_ENABLED=0 go build -o pb ./build/
```

### 运行服务

#### SQLite（本地开发）

```bash
# 使用 sqlite:// 前缀
./pb serve --dataDsn "sqlite://./pb_data/dev.db"

# 或者直接传 .db 文件路径
./pb serve --dataDsn "./pb_data/dev.db"
```

#### PostgreSQL（推荐用于生产）

```bash
./pb serve --dataDsn "postgresql://user:password@127.0.0.1:5432/dbname?sslmode=disable"
```

#### MySQL

```bash
./pb serve --dataDsn "mysql://user:password@tcp(127.0.0.1:3306)/dbname"
```

#### 启用 Redis 缓存

```bash
./pb serve --dataDsn "postgres://..." --redisDsn "redis://127.0.0.1:6379/0"
```

#### Docker

```bash
# 构建镜像
docker build -t postgrebase .

# 使用 SQLite 运行
docker run -p 8090:8090 -v pb_data:/pb/pb_data postgrebase serve --dataDsn "sqlite:///pb/pb_data/dev.db"

# 使用 PostgreSQL 运行
docker run -p 8090:8090 postgrebase serve --dataDsn "postgres://user:pass@host:5432/db?sslmode=disable"

# 使用 Redis 运行
docker run -p 8090:8090 postgrebase serve --dataDsn "postgres://..." --redisDsn "redis://host:6379/0"
```

启动后服务器会打印接入地址：

```
PostgreBase — AI原生的无代码API开发工具
├─ REST API: http://127.0.0.1:8090/api/
├─ MCP SSE:  http://127.0.0.1:8090/api/mcp/sse
└─ Admin UI: http://127.0.0.1:8090/_/
```

## 配置参数说明

| 参数 | 说明 | 示例 |
|------|------|------|
| `--dataDsn` | 数据库连接字符串 | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `--redisDsn` | （可选）Redis 连接字符串，未指定时使用内存缓存 | `redis://localhost:6379/0` |
| `--dir` | 数据目录（文件上传、备份等） | `./pb_data` |
| `--encryptionEnv` | 用于加密设置的系统环境变量名 | `PB_ENCRYPTION` |
| `--debug` | 开启调试模式，输出详细日志 | _(flag)_ |

### DSN 格式

| 引擎 | 格式 |
|------|------|
| PostgreSQL | `postgres://user:pass@host:port/db?sslmode=disable` 或 `postgresql://...` |
| MySQL | `mysql://user:pass@tcp(host:port)/db` |
| SQLite | `sqlite:///path/to/file.db`、`sqlite3://...` 或 `./file.db` |

## MCP（Model Context Protocol）

PostgreBase 内置 MCP 服务器，通过标准化 JSON-RPC 2.0 协议让 AI 工具直接操作你的数据。

### 传输模式

#### SSE（Server-Sent Events）— HTTP

运行 `./pb serve` 后可用。连接地址：

```
GET  http://localhost:8090/api/mcp/sse      # SSE 事件流
POST http://localhost:8090/api/mcp/message   # 发送 JSON-RPC 请求
POST http://localhost:8090/api/mcp/stream    # Streamable HTTP（单次请求/响应）
```

#### Stdio — CLI

以独立进程运行 MCP 服务器，通过 stdin/stdout 通信：

```bash
./pb mcp --dataDsn "sqlite://./dev.db" --mcp-token "YOUR_TOKEN"

# 或禁用认证（不推荐用于生产）
./pb mcp --dataDsn "sqlite://./dev.db" --mcp-no-auth
```

| 参数 | 说明 |
|------|------|
| `--mcp-token` | Admin JWT token 或 MCP 专用 token |
| `--mcp-no-auth` | 禁用认证（仅限开发环境） |

### Claude Desktop 配置

```json
{
  "mcpServers": {
    "postgrebase": {
      "command": "/path/to/pb",
      "args": ["mcp", "--dataDsn", "sqlite:///path/to/dev.db", "--mcp-no-auth"]
    }
  }
}
```

### MCP Token

在生产环境中，建议通过 Admin UI 的 **Settings → MCP Tokens** 创建专用 MCP Token（`mcp_` 前缀）。这些 Token：

- 与 Admin JWT Token 独立。
- 可单独撤销。
- 支持可选过期时间。
- 创建时仅显示一次完整值。

### 可用工具

| 工具 | 说明 |
|------|------|
| `list_collections` | 列出所有集合 |
| `get_collection` | 获取集合的 Schema 和设置 |
| `list_records` | 列出记录（支持分页、过滤、排序） |
| `get_record` | 通过 ID 获取单条记录 |
| `create_record` | 创建新记录 |
| `update_record` | 更新已有记录 |
| `delete_record` | 删除记录 |
| `search_records` | 使用 PostgreBase 过滤表达式搜索记录 |

### 可用资源

| URI | 说明 |
|-----|------|
| `postgrebase://collections` | 所有集合及其 Schema |
| `postgrebase://settings` | 应用设置（已脱敏，不含敏感信息） |

## 开发与扩展

### Admin UI 构建

修改前端管理界面后，需重新构建嵌入资源再编译 Go 二进制：

```bash
cd ui
npm install
npm run build
cd ..
go build -o pb ./build/
```

### 项目结构

```
postgrebase/
├── build/           # 服务器入口 (main.go)
├── core/            # 应用逻辑、数据库连接、缓存
├── daos/            # 数据访问对象（CRUD 操作）
├── models/          # 数据模型和 Schema 定义
├── apis/            # HTTP API 处理器（REST + MCP 路由）
├── mcp/             # MCP 服务器：协议、工具、资源、传输
├── migrations/      # 数据库迁移（Postgres/MySQL/SQLite）
├── cmd/             # CLI 命令（serve, mcp, admin）
├── dbx/             # 数据库查询构建器（ozzo-dbx 分支）
├── tools/           # 共享工具（安全、类型、搜索等）
├── ui/              # Admin UI（Svelte + Vite）
├── vendor/          # Go 依赖（保留用于离线构建）
└── postgrebase.go  # 根包：CLI 设置、配置、引导
```

### 贡献指南

1. Fork 本仓库。
2. 创建您的特性分支 (`git checkout -b feature/amazing-feature`)。
3. 提交您的更改 (`git commit -m 'Add some amazing feature'`)。
4. 推送到分支 (`git push origin feature/amazing-feature`)。
5. 发起 Pull Request。

## 致谢

本项目基于 [PocketBase](https://pocketbase.io) 二次开发。感谢 [Gani Georgiev](https://github.com/ganigeorgiev) 创造了如此优秀的框架。

## 开源协议

基于 [MIT 协议](https://opensource.org/licenses/MIT) 开源。
