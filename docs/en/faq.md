# FAQ

## General

### What is PostgreBase?

PostgreBase is a fork of [PocketBase](https://pocketbase.io) that replaces SQLite with PostgreSQL, MySQL, and SQLite support. It adds Redis caching, MCP server integration, and maintains 100% API compatibility with PocketBase.

### Why not just use PocketBase?

PocketBase is excellent for single-user or small-scale applications. PostgreBase is for when you need:

- **Horizontal scaling** with multiple server instances
- **Enterprise databases** (PostgreSQL/MySQL) for production workloads
- **Distributed caching** with Redis for high-concurrency scenarios
- **AI tool integration** via the built-in MCP server

### Is PostgreBase API-compatible with PocketBase?

Yes. The REST API, Admin UI, and business logic are 100% compatible. You can migrate from PocketBase by simply pointing PostgreBase at your PostgreSQL/MySQL database.

## Database

### Can I use SQLite in production?

You can, but it's not recommended for multi-node deployments. SQLite is best for local development and testing. Use PostgreSQL or MySQL for production.

### How do I migrate from PocketBase SQLite?

1. Export your PocketBase data
2. Set up a PostgreSQL or MySQL database
3. Start PostgreBase pointing at the new database
4. Import your data via the API or Admin UI

### Which PostgreSQL versions are supported?

PostgreBase uses the `lib/pq` driver and supports PostgreSQL 10+.

### Which MySQL versions are supported?

PostgreBase uses `go-sql-driver/mysql` and supports MySQL 5.7+ and MariaDB 10.3+.

## Caching

### Do I need Redis?

No. If Redis is not configured, PostgreBase automatically falls back to in-memory caching. Redis is recommended for multi-node deployments where cache needs to be shared.

### What happens if Redis goes down?

PostgreBase will log errors but continue operating. Cache misses will fall through to the database.

## MCP

### What is MCP?

MCP (Model Context Protocol) is a standardized JSON-RPC 2.0 protocol that allows AI tools to interact with your data. PostgreBase includes a built-in MCP server.

### Which AI tools support MCP?

Claude Desktop, Cursor, Windsurf, and any tool that implements the MCP specification.

### Are MCP tokens secure?

Yes. MCP tokens are separate from admin JWT tokens, support expiration, and can be revoked individually. The full token value is shown only once at creation.

## Deployment

### Can I run PostgreBase behind a reverse proxy?

Yes. PostgreBase works with Nginx, HAProxy, Caddy, or any reverse proxy. For SSE realtime subscriptions, ensure proxy buffering is disabled.

### How do I set up high availability?

See the [High Availability](high-availability.md) guide. In short: shared PostgreSQL/MySQL + Redis + multiple PostgreBase instances behind a load balancer.

### Can I use Docker?

Yes. Build the Docker image from the Dockerfile or use the binary directly in your container.

## Development

### How do I build the Admin UI?

```bash
cd ui
npm install
npm run build
cd ..
go build -o pb ./build/
```

### How do I run tests?

```bash
go test ./tools/... ./models/... ./daos/...
```

### Where is the vendor directory?

All Go dependencies are vendored in `/vendor/`. This directory must be preserved for offline builds.
