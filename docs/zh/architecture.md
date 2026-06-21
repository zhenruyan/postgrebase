# 系统架构

PostgreBase 基于 PocketBase 架构构建，增加了多数据库支持和 MCP 集成。

## 项目结构

```
postgrebase/
├── build/              # 服务器入口 (main.go)
├── postgrebase.go      # 根包：CLI 设置、Config 结构体、Bootstrap
├── core/               # 应用逻辑
│   ├── base.go         # BaseApp：缓存初始化、应用生命周期
│   ├── db_postgresql.go # connectDB()：DSN 解析和驱动检测
│   └── app.go          # App 接口定义
├── daos/               # 数据访问对象（所有模型的 CRUD）
│   ├── base.go         # Dao 结构体、ModelQuery、RunInTransaction
│   ├── record.go       # 记录 CRUD
│   ├── record_table_sync.go # 表 Schema 同步（驱动感知 DDL）
│   ├── table.go        # HasTable、TableColumns、TableInfo、TableIndexes
│   ├── collection.go   # 集合查询
│   ├── admin.go        # 管理员认证和查询
│   └── view.go         # SQL 视图管理
├── models/             # 数据模型
│   ├── base.go         # BaseModel（Id、Created、Updated）
│   ├── record.go       # Record 模型
│   ├── collection.go   # Collection 模型
│   ├── admin.go        # Admin 模型
│   └── schema/
│       └── schema_field.go # SchemaField.ColDefinition(driverName)
├── apis/               # HTTP API 处理器
│   ├── base.go         # InitApi()：路由注册
│   ├── serve.go        # HTTP 服务器、迁移运行器
│   ├── record_crud.go  # 记录 CRUD 端点
│   ├── admin.go        # 管理员认证端点
│   ├── mcp_token.go    # MCP Token 管理 API
│   └── middlewares.go  # 认证中间件
├── mcp/                # MCP 服务器（手写 JSON-RPC 2.0）
│   ├── server.go       # 核心路由器、方法路由、工具 Schema
│   ├── tools.go        # 8 个 MCP 工具处理器
│   ├── resources.go    # 2 个 MCP 资源处理器
│   ├── auth.go         # Token 验证
│   ├── transport_sse.go    # SSE + Streamable HTTP 传输
│   └── transport_stdio.go  # stdin/stdout 传输
├── migrations/         # 数据库迁移（驱动感知 SQL）
├── cmd/                # CLI 命令（serve、mcp、admin）
├── dbx/                # 数据库查询构建器（ozzo-dbx 分支）
├── tools/              # 共享工具
│   ├── security/       # JWT、bcrypt、随机字符串
│   ├── types/          # DateTime、自定义类型
│   ├── search/         # 过滤/排序/搜索提供者
│   └── migrate/        # 迁移运行器
├── ui/                 # Admin UI（Svelte SPA）
│   ├── src/            # Svelte 源码
│   └── dist/           # 构建后的前端（嵌入 Go 二进制）
└── vendor/             # Go 依赖（vendored）
```

## 数据库驱动检测

所有驱动特定代码通过 `core/db_postgresql.go:connectDB()` 路由：

| DSN 前缀 | 驱动名称 |
|---------|---------|
| `postgres://` | `postgres` |
| `postgresql://` | `postgres` |
| `mysql://` | `mysql` |
| `sqlite://` | `sqlite` |
| `sqlite3://` | `sqlite` |
| `*.db`（后缀） | `sqlite` |
| _（无前缀）_ | `postgres`（默认） |

## 关键驱动差异

| 特性 | PostgreSQL | MySQL | SQLite |
|------|-----------|-------|--------|
| 布尔类型 | `BOOLEAN` | `TINYINT(1)` | `INTEGER` |
| 时间戳默认值 | `now()::TIMESTAMP` | `CURRENT_TIMESTAMP(3)` | `strftime(...)` |
| JSON 存储 | `text` | `JSON` | `text` |
| ALTER TABLE IF NOT EXISTS | ✅ | ❌ 先检查 | ❌ 先检查 |
| Index IF NOT EXISTS | ✅ | ❌ 先检查 | ✅ |
| 表元数据 | `information_schema` | `information_schema` | `sqlite_master` |

## MCP 架构

MCP 服务器是手写的（无外部 SDK）JSON-RPC 2.0 实现：

- **`mcp/server.go`** — 核心 JSON-RPC 路由器
- **`mcp/tools.go`** — 8 个工具处理函数
- **`mcp/resources.go`** — 2 个资源处理器
- **`mcp/auth.go`** — 两种认证模式（JWT Admin Token + MCP 专用 Token）
- **`mcp/transport_sse.go`** — SSE + Streamable HTTP
- **`mcp/transport_stdio.go`** — stdin/stdout

## 数据流

```
客户端请求
    ↓
HTTP 处理器 (apis/)
    ↓
认证中间件
    ↓
Dao (daos/)  ←→  缓存层 (Redis / 内存)
    ↓
数据库 (PostgreSQL / MySQL / SQLite)
```
