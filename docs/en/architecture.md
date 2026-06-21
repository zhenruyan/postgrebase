# Architecture

PostgreBase is built on the PocketBase architecture with multi-database support and MCP integration.

## Project Structure

```
postgrebase/
├── build/              # Server entry point (main.go)
├── postgrebase.go      # Root package: CLI setup, Config struct, Bootstrap
├── core/               # Application logic
│   ├── base.go         # BaseApp: cache init, app lifecycle
│   ├── db_postgresql.go # connectDB(): DSN parsing and driver detection
│   └── app.go          # App interface definition
├── daos/               # Data access objects (CRUD for all models)
│   ├── base.go         # Dao struct, ModelQuery, RunInTransaction
│   ├── record.go       # Record CRUD
│   ├── record_table_sync.go # Table schema sync (driver-aware DDL)
│   ├── table.go        # HasTable, TableColumns, TableInfo, TableIndexes
│   ├── collection.go   # Collection queries
│   ├── admin.go        # Admin auth and queries
│   └── view.go         # SQL view management
├── models/             # Data models
│   ├── base.go         # BaseModel (Id, Created, Updated)
│   ├── record.go       # Record model
│   ├── collection.go   # Collection model
│   ├── admin.go        # Admin model
│   └── schema/
│       └── schema_field.go # SchemaField.ColDefinition(driverName)
├── apis/               # HTTP API handlers
│   ├── base.go         # InitApi(): route registration
│   ├── serve.go        # HTTP server, migration runner
│   ├── record_crud.go  # Record CRUD endpoints
│   ├── admin.go        # Admin auth endpoints
│   ├── mcp_token.go    # MCP token management API
│   └── middlewares.go  # Auth middleware
├── mcp/                # MCP server (hand-written JSON-RPC 2.0)
│   ├── server.go       # Core router, method routing, tool schemas
│   ├── tools.go        # 8 MCP tool handlers
│   ├── resources.go    # 2 MCP resource handlers
│   ├── auth.go         # Token validation
│   ├── transport_sse.go    # SSE + Streamable HTTP transport
│   └── transport_stdio.go  # Stdin/stdout transport
├── migrations/         # Database migrations (driver-aware SQL)
├── cmd/                # CLI commands (serve, mcp, admin)
├── dbx/                # Database query builder (fork of ozzo-dbx)
├── tools/              # Shared utilities
│   ├── security/       # JWT, bcrypt, random strings
│   ├── types/          # DateTime, custom types
│   ├── search/         # Filter/sort/search provider
│   └── migrate/        # Migration runner
├── ui/                 # Admin UI (Svelte SPA)
│   ├── src/            # Svelte source
│   └── dist/           # Built frontend (embedded in Go binary)
└── vendor/             # Go dependencies (vendored)
```

## Database Driver Detection

All driver-specific code routes through `core/db_postgresql.go:connectDB()`:

| DSN Prefix | Driver Name |
|-----------|-------------|
| `postgres://` | `postgres` |
| `postgresql://` | `postgres` |
| `mysql://` | `mysql` |
| `sqlite://` | `sqlite` |
| `sqlite3://` | `sqlite` |
| `*.db` (suffix) | `sqlite` |
| _(no prefix)_ | `postgres` (default) |

## Key Driver Differences

| Feature | PostgreSQL | MySQL | SQLite |
|---------|-----------|-------|--------|
| Boolean type | `BOOLEAN` | `TINYINT(1)` | `INTEGER` |
| Timestamp default | `now()::TIMESTAMP` | `CURRENT_TIMESTAMP(3)` | `strftime(...)` |
| JSON storage | `text` | `JSON` | `text` |
| ALTER TABLE IF NOT EXISTS | ✅ | ❌ check first | ❌ check first |
| Index IF NOT EXISTS | ✅ | ❌ check first | ✅ |
| Table metadata | `information_schema` | `information_schema` | `sqlite_master` |

## MCP Architecture

The MCP server is hand-written (no external SDK) as a JSON-RPC 2.0 implementation:

- **`mcp/server.go`** — Core JSON-RPC router
- **`mcp/tools.go`** — 8 tool handler functions
- **`mcp/resources.go`** — 2 resource handlers
- **`mcp/auth.go`** — Two auth modes (JWT admin tokens + MCP-specific tokens)
- **`mcp/transport_sse.go`** — SSE + Streamable HTTP
- **`mcp/transport_stdio.go`** — stdin/stdout

## Data Flow

```
Client Request
    ↓
HTTP Handler (apis/)
    ↓
Auth Middleware
    ↓
Dao (daos/)  ←→  Cache Layer (Redis / Memory)
    ↓
Database (PostgreSQL / MySQL / SQLite)
```
