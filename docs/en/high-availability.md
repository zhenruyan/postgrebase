# High Availability (HA)

PostgreBase is designed for clustered environments. Unlike SQLite-based systems, it supports horizontal scaling with shared database backends.

## Architecture

```
                    ┌─────────────┐
                    │ Load Balancer│
                    │  (Nginx/HA)  │
                    └──────┬──────┘
               ┌───────────┼───────────┐
               ▼           ▼           ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │PostgreBase│ │PostgreBase│ │PostgreBase│
        │  Node 1   │ │  Node 2   │ │  Node 3   │
        └─────┬─────┘ └─────┬─────┘ └─────┬─────┘
              │              │              │
        ┌─────┴──────────────┴──────────────┴─────┐
        │           Shared SQL Database            │
        │    (PostgreSQL / MySQL Cluster)          │
        └──────────────────┬───────────────────────┘
                           │
        ┌──────────────────┴───────────────────────┐
        │              Redis Cluster               │
        │     (Cache + Pub/Sub for Realtime)       │
        └──────────────────────────────────────────┘
```

## Deployment Steps

1. **Use a managed database** (AWS RDS, Google Cloud SQL, Azure Database) or a high-availability PostgreSQL/MySQL cluster.

2. **Deploy multiple PostgreBase instances** behind a load balancer (Nginx, HAProxy, AWS ALB).

3. **Connect all instances to the same Redis cluster** for shared caching and realtime synchronization.

4. **Share file storage** using S3-compatible storage or a networked file system (NFS, EFS) for the `pb_data` directory.

## Example Configuration

```bash
# Node 1
./pb serve \
  --dataDsn "postgres://app:secret@pg-cluster:5432/postgrebase?sslmode=disable" \
  --redisDsn "redis://redis-cluster:6379/0" \
  --dir /shared/postgrebase-data

# Node 2 (same config)
./pb serve \
  --dataDsn "postgres://app:secret@pg-cluster:5432/postgrebase?sslmode=disable" \
  --redisDsn "redis://redis-cluster:6379/0" \
  --dir /shared/postgrebase-data
```

## Redis in HA Mode

Redis handles two critical tasks in a cluster:

### 1. Shared Caching

Common queries are cached globally across all nodes. When data changes on one node, the cache is invalidated cluster-wide.

### 2. Realtime Synchronization

Record updates are broadcast via **Redis Pub/Sub** to all connected clients across the entire cluster. This ensures SSE realtime subscriptions work seamlessly regardless of which node a client is connected to.

## Load Balancer Configuration

### Nginx Example

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

        # SSE support
        proxy_set_header Connection '';
        proxy_http_version 1.1;
        chunked_transfer_encoding off;
        proxy_buffering off;
        proxy_cache off;
    }
}
```

## Considerations

- **Session Affinity:** Not required. Any node can handle any request.
- **File Uploads:** Use S3-compatible storage or shared filesystem.
- **Migrations:** Run migrations on one node before starting others.
- **Health Checks:** Use `/api/health` endpoint for load balancer health checks.
