# 快速上手

本指南帮助你在 5 分钟内启动 PostgreBase。

## 环境准备

- **Go 1.26.2+**（从源码构建时需要）
- **PostgreSQL** 或 **MySQL**（可选 —— SQLite 开箱即用）

## 安装

### 方式一：从源码构建

```bash
git clone https://github.com/zhenruyan/postgrebase.git
cd postgrebase
go build -o pb ./build/
```

静态构建（无 CGO）：

```bash
CGO_ENABLED=0 go build -o pb ./build/
```

### 方式二：下载二进制文件

从 [GitHub Releases](https://github.com/zhenruyan/postgrebase/releases) 下载最新版本。

## 启动服务

### SQLite（本地开发）

```bash
# 使用 sqlite:// 前缀
./pb serve --dataDsn "sqlite://./pb_data/dev.db"

# 或者直接传 .db 文件路径
./pb serve --dataDsn "./pb_data/dev.db"
```

### PostgreSQL（推荐用于生产）

```bash
./pb serve --dataDsn "postgresql://user:password@127.0.0.1:5432/dbname?sslmode=disable"
```

### MySQL

```bash
./pb serve --dataDsn "mysql://user:password@tcp(127.0.0.1:3306)/dbname"
```

### 启用 Redis 缓存

```bash
./pb serve --dataDsn "postgres://..." --redisDsn "redis://127.0.0.1:6379/0"
```

### Docker

从项目中的 Dockerfile 构建镜像：

```bash
docker build -t postgrebase .
```

使用 SQLite 运行：

```bash
docker run -p 8090:8090 -v pb_data:/pb/pb_data postgrebase serve --dataDsn "sqlite:///pb/pb_data/dev.db"
```

使用 PostgreSQL 运行：

```bash
docker run -p 8090:8090 postgrebase serve --dataDsn "postgres://user:pass@host:5432/dbname?sslmode=disable"
```

使用 Redis 运行：

```bash
docker run -p 8090:8090 postgrebase serve --dataDsn "postgres://..." --redisDsn "redis://host:6379/0"
```

Docker 镜像使用 Alpine Linux，暴露 8090 端口，数据存储在 `/pb/pb_data`。

## 首次使用

1. 打开 Admin UI：`http://127.0.0.1:8090/_/`
2. 创建管理员账户
3. 创建集合（表）并定义字段
4. 通过 REST API 或 MCP 与数据交互

## 下一步

- [配置详解](configuration.md) — 所有 CLI 参数和 DSN 格式
- [MCP 协议](mcp.md) — 连接 AI 工具到你的数据
- [高可用部署](high-availability.md) — 集群部署
