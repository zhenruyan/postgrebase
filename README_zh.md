# PostgreSQL & MySQL 版 PocketBase (企业级增强版)

本项目是 [PocketBase](https://pocketbase.io) 的深度定制开发版本。针对企业级生产环境进行了核心重构，旨在提供比原版（SQLite）更高性能、更强扩展性的后端服务。

通过将底层存储引擎从 SQLite 替换为 **PostgreSQL** 和 **MySQL**，本项目能够完美胜任高并发、多节点集群以及复杂的数据分片场景。

## 核心特性

- **企业级性能：** 摆脱 SQLite 的写锁限制，利用 PostgreSQL/MySQL 的并发处理能力。
- **集群化架构：** 原生支持多实例部署。由于不再依赖本地文件数据库，您可以轻松地在负载均衡器后运行多个 PocketBase 实例。
- **双数据库引擎：**
    - **PostgreSQL:** 默认引擎，支持标准 DSN 接入。
    - **MySQL:** 完全兼容，通过 `mysql://` 前缀进行连接。
- **灵活缓存：**
    - **Redis 缓存：** 支持集群环境下的分布式缓存，通过 `--redisDsn` 启用。同时，**SSE 实时订阅 (Realtime) 也会自动切换至 Redis Pub/Sub 模式**，确保跨节点的消息同步。
    - **内存缓存：** 在未配置 Redis 时，自动回退到高效的本地内存缓存，确保单机环境下也能获得极致响应。
- **保持 PocketBase 的极致体验：** 100% 兼容原有的 API、Admin UI 和业务逻辑。

## 快速上手

### 1. 环境准备

- Go 1.18+
- PostgreSQL 或 MySQL 实例

### 2. 编译构建

```bash
git clone https://github.com/free/postgresqlbaseapi.git
cd postgresqlbaseapi
# 编译二进制
go build -o pb ./build/
```

### 3. 运行服务

默认情况下，程序会尝试连接 `127.0.0.1:5432` 上的 PostgreSQL。

#### 使用 PostgreSQL (推荐用于生产)
```bash
./pb serve --dataDsn "postgresql://user:password@127.0.0.1:5432/dbname?sslmode=disable"
```

#### 使用 MySQL
```bash
./pb serve --dataDsn "mysql://user:password@tcp(127.0.0.1:3306)/dbname"
```

#### 启用 Redis 缓存 (提升集群性能)
```bash
./pb serve --redisDsn "redis://127.0.0.1:6379/0"
```

## 配置参数说明

- `--dataDsn`: 数据库连接字符串。
    - PostgreSQL: `postgres://user:pass@host:port/db?sslmode=disable`
    - MySQL: `mysql://user:pass@tcp(host:port)/db`
- `--redisDsn`: (可选) Redis 连接字符串。**如果不指定，则默认使用高性能本地内存缓存**。
- `--dir`: PocketBase 数据目录（用于存储文件附件、备份等非数据库数据）。
- `--encryptionEnv`: 用于加密设置的系统环境变量名。

## 开发与扩展

### Admin UI 构建

如果您修改了前端管理界面，请执行以下操作重新构建嵌入资源：

```bash
cd ui
npm install
npm run build
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
