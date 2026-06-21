# PostgreBase — AI原生的无代码API开发工具

PostgreBase 是 AI 原生的无代码 API 开发平台，基于 [PocketBase](https://pocketbase.io) 重构。内置 MCP (Model Context Protocol) 服务器，让 AI 工具（Claude、Cursor、Windsurf）直接操作你的数据。支持 PostgreSQL、MySQL、SQLite 三种数据库，提供即时 REST API + 实时订阅 + Admin UI，5分钟上线你的后端服务。

## Features

- **Three Database Engines:**
    - **PostgreSQL** (default) — production-grade concurrency, clustering, and complex queries.
    - **MySQL** — fully compatible via the `mysql://` DSN prefix.
    - **SQLite** (pure Go, no CGO) — zero-dependency local development and testing via `sqlite://` DSN or `.db` file path.
- **Cluster-Ready Architecture:** No reliance on local database files means multiple instances can run behind a load balancer with a shared PostgreSQL/MySQL backend.
- **Hybrid Caching:**
    - **Redis Cache** (`--redisDsn`) — distributed caching for cluster environments; SSE Realtime subscriptions synchronize across nodes via Redis Pub/Sub.
    - **In-Memory Cache** — automatic fallback to high-performance local memory caching when Redis is not configured.
- **MCP (Model Context Protocol) Server:**
    - JSON-RPC 2.0 protocol enabling AI tools (Claude Desktop, Cursor, Windsurf) to interact with your data.
    - 8 built-in tools: `list_collections`, `get_collection`, `list_records`, `get_record`, `create_record`, `update_record`, `delete_record`, `search_records`.
    - Resources: `postgrebase://collections`, `postgrebase://settings`.
    - Three transport modes: SSE (HTTP), Streamable HTTP, and Stdio.
    - MCP-specific API tokens with expiration support, manageable from the Admin UI.
- **Zero External Dependencies for SQLite:** Uses `modernc.org/sqlite` (pure Go transpilation of SQLite), builds with `CGO_ENABLED=0`.

## Quick Start

### Prerequisites

- Go 1.18+
- PostgreSQL, MySQL, or nothing (SQLite works out of the box)

### Build

```bash
git clone https://github.com/zhenruyan/postgrebase.git
cd postgrebase
go build -o pb ./build/
```

For static builds (no CGO):

```bash
CGO_ENABLED=0 go build -o pb ./build/
```

### Run

#### SQLite (Local Development)

```bash
# Using sqlite:// prefix
./pb serve --dataDsn "sqlite://./pb_data/dev.db"

# Or just pass a .db file path
./pb serve --dataDsn "./pb_data/dev.db"
```

#### PostgreSQL (Recommended for Production)

```bash
./pb serve --dataDsn "postgresql://user:password@127.0.0.1:5432/dbname?sslmode=disable"
```

#### MySQL

```bash
./pb serve --dataDsn "mysql://user:password@tcp(127.0.0.1:3306)/dbname"
```

#### With Redis Cache

```bash
./pb serve --dataDsn "postgres://..." --redisDsn "redis://127.0.0.1:6379/0"
```

On startup the server prints its endpoints:

```
├─ REST API: http://127.0.0.1:8090/api/
├─ MCP SSE:  http://127.0.0.1:8090/api/mcp/sse
└─ Admin UI: http://127.0.0.1:8090/_/
```

## Configuration Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--dataDsn` | Database connection string | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `--redisDsn` | (Optional) Redis connection string. Falls back to in-memory cache if omitted | `redis://localhost:6379/0` |
| `--dir` | Data directory for file uploads, backups, etc. | `./pb_data` |
| `--encryptionEnv` | Environment variable name for settings encryption key | `PB_ENCRYPTION` |
| `--debug` | Enable debug mode with detailed logs | _(flag)_ |

### DSN Formats

| Engine | Format |
|--------|--------|
| PostgreSQL | `postgres://user:pass@host:port/db?sslmode=disable` or `postgresql://...` |
| MySQL | `mysql://user:pass@tcp(host:port)/db` |
| SQLite | `sqlite:///path/to/file.db`, `sqlite3://...`, or simply `./file.db` |

## MCP (Model Context Protocol)

PostgreBase includes a built-in MCP server that allows AI tools to interact with your data via a standardized JSON-RPC 2.0 protocol.

### Transport Modes

#### SSE (Server-Sent Events) — HTTP

Available when running `./pb serve`. Connect to:

```
GET  http://localhost:8090/api/mcp/sse      # SSE event stream
POST http://localhost:8090/api/mcp/message   # Send JSON-RPC requests
POST http://localhost:8090/api/mcp/stream    # Streamable HTTP (single request/response)
```

#### Stdio — CLI

Run the MCP server as a standalone process communicating over stdin/stdout:

```bash
./pb mcp --dataDsn "sqlite://./dev.db" --mcp-token "YOUR_TOKEN"

# Or disable auth (NOT recommended for production)
./pb mcp --dataDsn "sqlite://./dev.db" --mcp-no-auth
```

| Flag | Description |
|------|-------------|
| `--mcp-token` | Admin JWT token or MCP-specific token for authentication |
| `--mcp-no-auth` | Disable authentication (development only) |

### Claude Desktop Configuration

```json
{
  "mcpServers": {
    "postgrebase": {
      "command": "/path/to/pb",
      "args": ["mcp", "--dataDsn", "sqlite:///path/to/dev.db", "--mcp-no-auth"]
    }
  }
}
```

### MCP Tokens

For production use, create dedicated MCP tokens (prefixed with `mcp_`) from the Admin UI under **Settings → MCP Tokens**. These tokens:

- Are separate from admin JWT tokens.
- Can be revoked individually.
- Support optional expiration dates.
- Are shown in full only once at creation time.

### Available Tools

| Tool | Description |
|------|-------------|
| `list_collections` | List all collections |
| `get_collection` | Get a collection's schema and settings |
| `list_records` | List records with pagination, filtering, and sorting |
| `get_record` | Get a single record by ID |
| `create_record` | Create a new record |
| `update_record` | Update an existing record |
| `delete_record` | Delete a record |
| `search_records` | Search records using PostgreBase filter expressions |

### Available Resources

| URI | Description |
|-----|-------------|
| `postgrebase://collections` | All collections with their schemas |
| `postgrebase://settings` | Application settings (sanitized, no secrets) |

## Development

### Building Admin UI

If you modify the Admin UI, rebuild the embedded assets before compiling the Go binary:

```bash
cd ui
npm install
npm run build
cd ..
go build -o pb ./build/
```

### Project Structure

```
postgrebase/
├── build/           # Main server entry point (main.go)
├── core/            # Application logic, DB connection, caching
├── daos/            # Data access objects (CRUD operations)
├── models/          # Data models and schema definitions
├── apis/            # HTTP API handlers (REST + MCP routes)
├── mcp/             # MCP server: protocol, tools, resources, transports
├── migrations/      # Database migrations (Postgres/MySQL/SQLite)
├── cmd/             # CLI commands (serve, mcp, admin)
├── dbx/             # Database query builder (fork of ozzo-dbx)
├── tools/           # Shared utilities (security, types, search, etc.)
├── ui/              # Admin UI (Svelte + Vite)
├── vendor/          # Go dependencies (preserved for offline builds)
└── postgrebase.go  # Root package: CLI setup, config, bootstrap
```

### Running Tests

```bash
go test ./tools/... ./models/... ./daos/...
```

### Contributing

1. Fork the repository.
2. Create your feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes (`git commit -m 'Add some amazing feature'`).
4. Push to the branch (`git push origin feature/amazing-feature`).
5. Open a Pull Request.

## Credits

This project is a fork of [PocketBase](https://pocketbase.io). Special thanks to [Gani Georgiev](https://github.com/ganigeorgiev) for the original amazing work.

## License

Licensed under the [MIT license](https://opensource.org/licenses/MIT).
