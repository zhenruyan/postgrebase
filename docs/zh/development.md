# 开发指南

## 从源码构建

### 环境准备

- Go 1.26.2+
- Node.js（Admin UI 开发时需要）

### 构建服务器

```bash
git clone https://github.com/zhenruyan/postgrebase.git
cd postgrebase
go build -o pb ./build/
```

静态构建（无 CGO）：

```bash
CGO_ENABLED=0 go build -o pb ./build/
```

### 使用 Makefile 构建

```bash
make build              # 构建当前平台
make build-all          # 构建所有平台（Linux、macOS、Windows）
make dist               # 构建并打包可分发归档
make lint               # 运行 linter
make fmt                # 格式化代码
```

### 构建 Docker 镜像

```bash
docker build -t postgrebase .
```

### 构建 Admin UI

修改 Admin UI 后，需重新构建嵌入资源再编译：

```bash
cd ui
npm install
npm run build
cd ..
go build -o pb ./build/
```

## 运行测试

```bash
go test ./tools/... ./models/... ./daos/...
```

测试使用 `modernc.org/sqlite`（纯 Go）—— 无需外部数据库。

## 静态分析

```bash
go vet ./...
```

Linter 配置在 `golangci.yml` 中。

## 项目结构

```
postgrebase/
├── build/           # 服务器入口 (main.go)
├── core/            # 应用逻辑、数据库连接、缓存
├── daos/            # 数据访问对象（CRUD 操作）
├── models/          # 数据模型和 Schema 定义
├── apis/            # HTTP API 处理器（REST + MCP 路由）
├── mcp/             # MCP 服务器：协议、工具、资源、传输
├── migrations/      # 数据库迁移（Postgres/MySQL/SQLite）
├── cmd/             # CLI 命令（serve、mcp、admin）
├── dbx/             # 数据库查询构建器（ozzo-dbx 分支）
├── tools/           # 共享工具（安全、类型、搜索等）
├── ui/              # Admin UI（Svelte + Vite）
├── vendor/          # Go 依赖（保留用于离线构建）
└── postgrebase.go  # 根包：CLI 设置、配置、引导
```

## 添加新迁移

1. 创建 `migrations/<unix_timestamp>_<description>.go`。
2. 在 `init()` 中使用 `AppMigrations.Register(up, down)`。
3. 对任何 DDL 语句按 `db.DriverName()` 分支。

### 驱动特定 SQL

添加驱动特定 SQL 时，检查 `db.DriverName()`：

- `"mysql"` —— MySQL
- `"sqlite"` / `"sqlite3"` —— SQLite
- 默认 —— PostgreSQL

### SQLite 列检查

SQLite 使用 `PRAGMA table_info` 代替 `information_schema`。

## 添加新 MCP 工具

1. 在 `mcp/server.go` 工具 Schema 列表中注册工具名称。
2. 在 `mcp/tools.go` 中添加处理方法：

```go
func (s *Server) toolMyNewTool(args map[string]interface{}) (*ToolCallResult, error) {
    // 解析参数
    // 使用 s.app.Dao() 访问数据
    return &ToolCallResult{Content: []Content{{Type: "text", Text: result}}}, nil
}
```

3. 在工具分发映射中注册处理器。

## 修改表 Schema

- 更新 `daos/record_table_sync.go:SyncRecordTableSchema()` 以添加新列类型。
- 更新 `models/schema/schema_field.go:ColDefinition()` 以添加新字段类型。
- 始终为 SQLite、MySQL 和 PostgreSQL 添加驱动特定分支。

## 使用 DateTime

使用 `types.DateTime`（封装 `time.Time`）：

```go
now := types.NowDateTime()
t := now.Time()        // 返回 time.Time
zero := types.DateTime{}
zero.IsZero()          // true
```

默认格式：`"2006-01-02 15:04:05.000"`

## 记录操作

```go
// 获取
record, err := dao.FindRecordById(collectionId, recordId)

// 设置值
record.Set("title", "New Title")

// 读取值
title := record.GetString("title")

// 保存（创建或更新）
err := dao.SaveRecord(record)

// 删除
err := dao.DeleteRecord(record)
```

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 发起 Pull Request

## Vendor 目录

`/vendor/` 目录包含所有 Go 依赖，**不能删除**。所有依赖已 vendor 化以支持离线构建。

## CI/CD

- **GitHub Actions** 在 `.github/workflows/`（构建检查和 GoReleaser）
- **版本号** 通过 `.goreleaser.yaml` 中的 `ldflags` 设置

## npm 安装器

PostgreBase 提供平台特定的 npm 包便于安装：

```bash
make npm-version       # 同步版本到 npm 安装器包
make npm-packages      # 构建平台特定 npm 包
make npm-pack          # 打包主包和所有平台包
make npm-publish-all   # 发布主包和所有平台包
make npm-publish-pre   # 发布预发布包
```

## JavaScript SDK

JS SDK（`js-sdk/`）提供了与 PostgreBase 交互的客户端库：

```bash
make sdk-build         # 构建 JavaScript SDK
make sdk-test          # 运行 JS SDK 测试
make sdk-pack          # 打包 JS SDK
make sdk-publish       # 发布 JS SDK 到 npm
make sdk-publish-pre   # 发布 JS SDK 预发布版本
```
