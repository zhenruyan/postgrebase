# PostgreBase Documentation

<p align="center">
  <strong>AI-Native No-Code API Development Platform</strong>
</p>

<p align="center">
  Based on <a href="https://pocketbase.io">PocketBase</a>, with native PostgreSQL/MySQL/SQLite support,<br>
  built-in MCP server, hybrid caching (Redis / memory), and 100% PocketBase API compatibility.
</p>

---

Welcome to the PostgreBase Documentation Center!

## Why PostgreBase?

**The Problem:** PocketBase uses SQLite by default — great for single-user apps, but a bottleneck in high-concurrency, clustered, or enterprise environments.

**The Solution:** PostgreBase replaces SQLite with **PostgreSQL** and **MySQL** as primary storage engines, adds **Redis caching** with Pub/Sub for realtime sync, and includes a built-in **MCP server** for AI tool integration.

### Key Highlights

| Feature | What It Means for You |
|---------|----------------------|
| **Three DB Engines** | PostgreSQL, MySQL, and SQLite — pick the right tool for each environment |
| **Cluster-Ready** | Multiple instances behind a load balancer with shared SQL backend |
| **Hybrid Caching** | Redis distributed cache with automatic in-memory fallback |
| **MCP Server** | AI tools (Claude, Cursor, Windsurf) can directly operate your data |
| **100% PocketBase Compatible** | Same API, Admin UI, and business logic — seamless migration |
| **Zero CGO for SQLite** | Pure Go SQLite via `modernc.org/sqlite`, builds with `CGO_ENABLED=0` |

---

## Documentation

### Getting Started
- [Quick Start](getting-started.md) — Installation, build, and first run
- [Configuration](configuration.md) — All CLI flags, DSN formats, and options

### Core Features
- [MCP Protocol](mcp.md) — Model Context Protocol server for AI tool integration
- [High Availability](high-availability.md) — Cluster deployment with Redis Pub/Sub
- [Caching](caching.md) — Redis and in-memory caching strategies

### Reference
- [Architecture](architecture.md) — Project structure and code organization
- [API Reference](api-reference.md) — REST API and MCP tools/resources
- [FAQ](faq.md) — Frequently asked questions

### Development
- [Development Guide](development.md) — Building, testing, and contributing

---

## Quick Example

```bash
# Build
CGO_ENABLED=0 go build -o pb ./build/

# Run with PostgreSQL
./pb serve --dataDsn "postgresql://user:pass@127.0.0.1:5432/db?sslmode=disable"

# Run with Redis cache
./pb serve --dataDsn "postgres://..." --redisDsn "redis://localhost:6379/0"
```

On startup:

```
PostgreBase — AI-Native No-Code API Platform
├─ REST API: http://127.0.0.1:8090/api/
├─ MCP SSE:  http://127.0.0.1:8090/api/mcp/sse
└─ Admin UI: http://127.0.0.1:8090/_/
```
