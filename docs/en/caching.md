# Caching

PostgreBase supports hybrid caching with Redis (for distributed environments) and automatic in-memory fallback.

## Cache Modes

### Redis Cache (Recommended for Production)

When `--redisDsn` is provided, PostgreBase uses Redis for distributed caching:

```bash
./pb serve --dataDsn "postgres://..." --redisDsn "redis://localhost:6379/0"
```

**Benefits:**
- Cache shared across all cluster nodes
- Automatic invalidation on data updates
- Reduces database load for frequently accessed records

### In-Memory Cache (Default Fallback)

When Redis is not configured, PostgreBase automatically falls back to high-performance local memory caching.

**Benefits:**
- Zero configuration required
- Ultra-low latency
- Perfect for single-node development

## Per-Collection Cache Control

Once Redis is connected, you can toggle caching for individual collections in the **Cache** tab of the collection settings in the Admin UI.

## Cache Invalidation

Cache entries are automatically invalidated when:
- A record is created, updated, or deleted
- A collection's schema is modified
- Application settings are changed

## Redis DSN Format

```
redis://localhost:6379/0
redis://:password@localhost:6379/0
redis://user:password@redis-host:6379/0
rediss://redis-host:6380/0              # TLS connection
```

## Performance Tips

1. **Use Redis for clusters** — In-memory cache is per-node and not shared.
2. **Monitor Redis memory** — Set appropriate `maxmemory` and eviction policies.
3. **Use connection pooling** — Redis connections are managed automatically by PostgreBase.
4. **Colocate Redis** — Deploy Redis in the same datacenter/region as your PostgreBase nodes.
