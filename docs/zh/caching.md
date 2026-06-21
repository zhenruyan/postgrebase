# 缓存策略

PostgreBase 支持混合缓存：Redis（分布式环境）和自动内存回退。

## 缓存模式

### Redis 缓存（推荐用于生产）

提供 `--redisDsn` 时，PostgreBase 使用 Redis 进行分布式缓存：

```bash
./pb serve --dataDsn "postgres://..." --redisDsn "redis://localhost:6379/0"
```

**优势：**
- 缓存在所有集群节点间共享
- 数据更新时自动失效
- 减少频繁访问记录的数据库负载

### 内存缓存（默认回退）

未配置 Redis 时，PostgreBase 自动回退到高性能本地内存缓存。

**优势：**
- 零配置
- 超低延迟
- 完美适合单节点开发

## 每集合缓存控制

连接 Redis 后，你可以在 Admin UI 的集合设置的 **Cache** 选项卡中为各个集合开关缓存。

## 缓存失效

缓存条目在以下情况自动失效：
- 记录被创建、更新或删除
- 集合的 Schema 被修改
- 应用设置被更改

## Redis DSN 格式

```
redis://localhost:6379/0
redis://:password@localhost:6379/0
redis://user:password@redis-host:6379/0
rediss://redis-host:6380/0              # TLS 连接
```

## 性能建议

1. **集群环境使用 Redis** —— 内存缓存是每节点独立的，不共享。
2. **监控 Redis 内存** —— 设置适当的 `maxmemory` 和淘汰策略。
3. **使用连接池** —— PostgreBase 自动管理 Redis 连接。
4. **就近部署 Redis** —— 将 Redis 部署在与 PostgreBase 节点相同的数据中心/区域。
