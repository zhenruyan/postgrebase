# 高可用部署

PostgreBase 专为集群环境设计。不同于基于 SQLite 的系统，它支持水平扩展和共享数据库后端。

## 架构

```
                    ┌─────────────┐
                    │  负载均衡器   │
                    │  (Nginx/HA)  │
                    └──────┬──────┘
               ┌───────────┼───────────┐
               ▼           ▼           ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │PostgreBase│ │PostgreBase│ │PostgreBase│
        │  节点 1   │ │  节点 2   │ │  节点 3   │
        └─────┬─────┘ └─────┬─────┘ └─────┬─────┘
              │              │              │
        ┌─────┴──────────────┴──────────────┴─────┐
        │           共享 SQL 数据库                 │
        │    (PostgreSQL / MySQL 集群)             │
        └──────────────────┬───────────────────────┘
                           │
        ┌──────────────────┴───────────────────────┐
        │              Redis 集群                   │
        │     (缓存 + Pub/Sub 实时同步)             │
        └──────────────────────────────────────────┘
```

## 部署步骤

1. **使用托管数据库**（AWS RDS、Google Cloud SQL、Azure Database）或高可用 PostgreSQL/MySQL 集群。

2. **部署多个 PostgreBase 实例** 在负载均衡器（Nginx、HAProxy、AWS ALB）后面。

3. **将所有实例连接到同一个 Redis 集群** 用于共享缓存和实时同步。

4. **共享文件存储** 使用 S3 兼容存储或网络文件系统（NFS、EFS）共享 `pb_data` 目录。

## 配置示例

```bash
# 节点 1
./pb serve \
  --dataDsn "postgres://app:secret@pg-cluster:5432/postgrebase?sslmode=disable" \
  --redisDsn "redis://redis-cluster:6379/0" \
  --dir /shared/postgrebase-data

# 节点 2（相同配置）
./pb serve \
  --dataDsn "postgres://app:secret@pg-cluster:5432/postgrebase?sslmode=disable" \
  --redisDsn "redis://redis-cluster:6379/0" \
  --dir /shared/postgrebase-data
```

## Redis 在高可用模式中的作用

Redis 在集群中处理两项关键任务：

### 1. 全局共享缓存

常用查询在整个集群中共享缓存。当数据在一个节点上变更时，缓存在整个集群中自动失效。

### 2. 实时状态同步

记录更新通过 **Redis Pub/Sub** 广播至整个集群的所有连接客户端。无论客户端连接到哪个节点，SSE 实时订阅都能无缝工作。

## 负载均衡器配置

### Nginx 示例

```nginx
upstream postgrebase {
    least_conn;
    server 10.0.0.1:8090;
    server 10.0.0.2:8090;
    server 10.0.0.3:8090;
}

server {
    listen 80;
    server_name api.example.com;

    location / {
        proxy_pass http://postgrebase;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE 支持
        proxy_set_header Connection '';
        proxy_http_version 1.1;
        chunked_transfer_encoding off;
        proxy_buffering off;
        proxy_cache off;
    }
}
```

## 注意事项

- **会话亲和性：** 不需要。任何节点都可以处理任何请求。
- **文件上传：** 使用 S3 兼容存储或共享文件系统。
- **迁移：** 在启动其他节点之前，先在一个节点上运行迁移。
- **健康检查：** 使用 `/api/health` 端点进行负载均衡器健康检查。
