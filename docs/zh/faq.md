# 常见问题

## 通用

### 什么是 PostgreBase？

PostgreBase 是 [PocketBase](https://pocketbase.io) 的分支，将 SQLite 替换为 PostgreSQL、MySQL 和 SQLite 支持。它添加了 Redis 缓存、MCP 服务器集成，并保持 100% 的 PocketBase API 兼容性。

### 为什么不直接用 PocketBase？

PocketBase 非常适合单用户或小规模应用。PostgreBase 适用于以下场景：

- **水平扩展** —— 多服务器实例部署
- **企业级数据库** —— PostgreSQL/MySQL 生产工作负载
- **分布式缓存** —— Redis 高并发场景
- **AI 工具集成** —— 内置 MCP 服务器

### PostgreBase 与 PocketBase API 兼容吗？

是的。REST API、Admin UI 和业务逻辑 100% 兼容。你可以通过将 PostgreBase 指向你的 PostgreSQL/MySQL 数据库从 PocketBase 迁移。

## 数据库

### 生产环境可以用 SQLite 吗？

可以，但不推荐用于多节点部署。SQLite 最适合本地开发和测试。生产环境请使用 PostgreSQL 或 MySQL。

### 如何从 PocketBase SQLite 迁移？

1. 导出你的 PocketBase 数据
2. 设置 PostgreSQL 或 MySQL 数据库
3. 启动 PostgreBase 指向新数据库
4. 通过 API 或 Admin UI 导入数据

### 支持哪些 PostgreSQL 版本？

PostgreBase 使用 `lib/pq` 驱动，支持 PostgreSQL 10+。

### 支持哪些 MySQL 版本？

PostgreBase 使用 `go-sql-driver/mysql`，支持 MySQL 5.7+ 和 MariaDB 10.3+。

## 缓存

### 需要 Redis 吗？

不需要。如果未配置 Redis，PostgreBase 自动回退到内存缓存。多节点部署建议使用 Redis 共享缓存。

### Redis 挂了怎么办？

PostgreBase 会记录错误但继续运行。缓存未命中会直接查询数据库。

## MCP

### 什么是 MCP？

MCP（Model Context Protocol）是标准化的 JSON-RPC 2.0 协议，让 AI 工具与你的数据交互。PostgreBase 内置了 MCP 服务器。

### 哪些 AI 工具支持 MCP？

Claude Desktop、Cursor、Windsurf，以及任何实现了 MCP 规范的工具。

### MCP Token 安全吗？

是的。MCP Token 与 Admin JWT Token 独立，支持过期时间，可单独撤销。完整 Token 值仅在创建时显示一次。

## 部署

### 可以在反向代理后面运行吗？

可以。PostgreBase 支持 Nginx、HAProxy、Caddy 或任何反向代理。SSE 实时订阅需要禁用代理缓冲。

### 如何设置高可用？

参见[高可用部署](high-availability.md)指南。简而言之：共享 PostgreSQL/MySQL + Redis + 多个 PostgreBase 实例在负载均衡器后面。

### 可以用 Docker 吗？

可以。从项目中的 Dockerfile 构建镜像：

```bash
docker build -t postgrebase .
```

按需选择数据库运行：

```bash
# SQLite
docker run -p 8090:8090 -v pb_data:/pb/pb_data postgrebase serve --dataDsn "sqlite:///pb/pb_data/dev.db"

# PostgreSQL
docker run -p 8090:8090 postgrebase serve --dataDsn "postgres://user:pass@host:5432/db?sslmode=disable"
```

镜像基于 Alpine Linux，以非 root 用户 `pocketbase` 运行，暴露 8090 端口，数据卷为 `/pb/pb_data`。

## 开发

### 如何构建 Admin UI？

```bash
cd ui
npm install
npm run build
cd ..
go build -o pb ./build/
```

### 如何运行测试？

```bash
go test ./tools/... ./models/... ./daos/...
```

### vendor 目录在哪里？

所有 Go 依赖都在 `/vendor/` 中。此目录必须保留用于离线构建。
