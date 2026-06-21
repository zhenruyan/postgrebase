# PostgreBase 文档

<p align="center">
  <strong>AI 原生的无代码 API 开发平台</strong>
</p>

<p align="center">
  基于 <a href="https://pocketbase.io">PocketBase</a> 重构，原生支持 PostgreSQL/MySQL/SQLite，<br>
  内置 MCP 服务器，混合缓存（Redis / 内存），100% 兼容 PocketBase API。
</p>

---

欢迎来到 PostgreBase 文档中心！

## 为什么选择 PostgreBase？

**问题：** PocketBase 默认使用 SQLite —— 对单用户应用很棒，但在高并发、集群或企业环境中成为瓶颈。

**解决方案：** PostgreBase 用 **PostgreSQL** 和 **MySQL** 替换 SQLite 作为主要存储引擎，添加 **Redis 缓存** 和 Pub/Sub 实时同步，并内置 **MCP 服务器** 用于 AI 工具集成。

### 核心亮点

| 特性 | 对你的意义 |
|------|-----------|
| **三数据库引擎** | PostgreSQL、MySQL、SQLite —— 为不同场景选择合适的工具 |
| **集群就绪** | 多实例部署在负载均衡器后，共享 SQL 后端 |
| **混合缓存** | Redis 分布式缓存，自动回退到内存缓存 |
| **MCP 服务器** | AI 工具（Claude、Cursor、Windsurf）直接操作你的数据 |
| **100% PocketBase 兼容** | 相同的 API、Admin UI 和业务逻辑 —— 无缝迁移 |
| **SQLite 零 CGO** | 纯 Go SQLite（`modernc.org/sqlite`），支持 `CGO_ENABLED=0` 构建 |

---

## 文档目录

### 入门指南
- [快速上手](getting-started.md) — 安装、构建和首次运行
- [配置详解](configuration.md) — 所有 CLI 参数、DSN 格式和选项

### 核心功能
- [MCP 协议](mcp.md) — Model Context Protocol 服务器，AI 工具集成
- [高可用部署](high-availability.md) — Redis Pub/Sub 集群部署
- [缓存策略](caching.md) — Redis 和内存缓存

### 参考手册
- [系统架构](architecture.md) — 项目结构和代码组织
- [API 参考](api-reference.md) — REST API 和 MCP 工具/资源
- [常见问题](faq.md) — 常见问题解答

### 开发指南
- [开发指南](development.md) — 构建、测试和贡献

---

## 快速示例

```bash
# 构建
CGO_ENABLED=0 go build -o pb ./build/

# 使用 PostgreSQL 运行
./pb serve --dataDsn "postgresql://user:pass@127.0.0.1:5432/db?sslmode=disable"

# 启用 Redis 缓存
./pb serve --dataDsn "postgres://..." --redisDsn "redis://localhost:6379/0"
```

启动后：

```
PostgreBase — AI原生的无代码API开发平台
├─ REST API: http://127.0.0.1:8090/api/
├─ MCP SSE:  http://127.0.0.1:8090/api/mcp/sse
└─ Admin UI: http://127.0.0.1:8090/_/
```
