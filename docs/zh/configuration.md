# 配置详解

PostgreBase 通过 `serve` 命令的 CLI 参数进行配置。

## CLI 参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `--dataDsn` | 数据库连接字符串 | `sqlite://pb_data/data.db` |
| `--redisDsn` | （可选）Redis 连接字符串，未指定时使用内存缓存 | `redis://localhost:6379/0` |
| `--dir` | 数据目录（文件上传、备份等） | `./pb_data` |
| `--encryptionEnv` | 用于加密设置的系统环境变量名 | `PB_ENCRYPTION` |
| `--debug` | 开启调试模式，输出详细日志 | _(flag)_ |

## DSN 格式

### PostgreSQL

```
postgres://user:pass@host:port/dbname?sslmode=disable
postgresql://user:pass@host:port/dbname?sslmode=disable
```

### MySQL

```
mysql://user:pass@tcp(host:port)/dbname
```

### SQLite

```
sqlite:///path/to/file.db
sqlite3:///path/to/file.db
./path/to/file.db          # 简写（通过 .db 后缀检测）
```

## 驱动检测

DSN 前缀决定使用哪个数据库驱动：

| 前缀 | 驱动 |
|------|------|
| `postgres://` | PostgreSQL |
| `postgresql://` | PostgreSQL |
| `mysql://` | MySQL |
| `sqlite://` | SQLite |
| `sqlite3://` | SQLite |
| `*.db`（后缀） | SQLite |
| _（无前缀）_ | PostgreSQL（默认） |

## 环境变量

| 变量 | 说明 |
|------|------|
| `PB_ENCRYPTION` | 设置加密密钥（由 `--encryptionEnv` 使用） |

## 示例

### 本地开发

```bash
./pb serve --dataDsn "sqlite://./pb_data/dev.db" --debug
```

### 生产环境（PostgreSQL）

```bash
./pb serve \
  --dataDsn "postgres://app:secret@db.example.com:5432/postgrebase?sslmode=disable" \
  --dir /var/data/postgrebase
```

### 生产环境（MySQL + Redis）

```bash
./pb serve \
  --dataDsn "mysql://app:secret@tcp(db.example.com:3306)/postgrebase" \
  --redisDsn "redis://redis.example.com:6379/0" \
  --dir /var/data/postgrebase
```

### 高可用集群节点

```bash
./pb serve \
  --dataDsn "postgres://app:secret@pg-cluster:5432/postgrebase?sslmode=disable" \
  --redisDsn "redis://redis-cluster:6379/0"
```
