# PostgreBase — Agent Development Guide

This file provides context for AI coding agents (Cursor, Claude, Windsurf, etc.) working on the PostgreBase codebase. Read this before exploring or modifying code.

## Overview

PostgreBase 是 AI 原生的无代码 API 开发平台，基于 [PocketBase](https://pocketbase.io) 重构。内置 MCP (Model Context Protocol) 服务器，让 AI 工具直接操作数据。支持 **PostgreSQL**、**MySQL**、**SQLite** 三种数据库，混合缓存（**Redis** / 内存），100% 兼容 PocketBase API、Admin UI 和业务逻辑。

## Tech Stack

- **Language:** Go 1.26.2+ (builds with `CGO_ENABLED=0`)
- **HTTP Framework:** `github.com/labstack/echo/v5`
- **CLI:** `github.com/spf13/cobra`
- **Database Drivers:**
  - `github.com/lib/pq` — PostgreSQL
  - `github.com/go-sql-driver/mysql` — MySQL
  - `modernc.org/sqlite` — SQLite (pure Go, no CGO)
- **Caching:** `github.com/redis/go-redis/v9` (Redis) / built-in memory store
- **Frontend:** Svelte 3, svelte-spa-router, Vite 4, PocketBase JS SDK
- **Vendor:** `/vendor/` directory **must be preserved** — all dependencies are vendored for offline builds.

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
│   ├── table.go        # HasTable, TableColumns, TableInfo, TableIndexes (driver-aware)
│   ├── collection.go   # Collection queries
│   ├── admin.go        # Admin auth and queries
│   └── view.go         # SQL view management
├── models/             # Data models
│   ├── base.go         # BaseModel (Id, Created, Updated, RefreshId)
│   ├── record.go       # Record model
│   ├── collection.go   # Collection model
│   ├── admin.go        # Admin model (bcrypt password, JWT token key)
│   └── schema/
│       └── schema_field.go # SchemaField.ColDefinition(driverName) — driver-aware DDL
├── apis/               # HTTP API handlers
│   ├── base.go         # InitApi(): route registration, MCP route binding
│   ├── serve.go        # HTTP server, migration runner, startup banner
│   ├── record_crud.go  # Record CRUD endpoints
│   ├── admin.go        # Admin auth endpoints
│   ├── mcp_token.go    # MCP token management API (CRUD)
│   └── middlewares.go   # Auth middleware (RequireAdminAuth, etc.)
├── mcp/                # MCP (Model Context Protocol) server
│   ├── server.go       # JSON-RPC 2.0 core, method routing, tool schemas
│   ├── tools.go        # 8 MCP tools (CRUD + search for records/collections)
│   ├── resources.go    # 2 MCP resources (collections, settings)
│   ├── auth.go         # Token validation (JWT admin tokens + mcp_ prefixed tokens)
│   ├── transport_sse.go    # SSE + Streamable HTTP transport
│   └── transport_stdio.go  # Stdin/stdout transport
├── migrations/         # Database migrations (driver-aware SQL)
│   ├── 1640988000_init.go              # Core tables (admins, collections, params, externalAuths)
│   ├── 1691747914_add_cache_columns.go # Cache columns migration
│   └── 1704067200_mcp_tokens.go        # MCP tokens collection
├── cmd/                # CLI commands
│   ├── serve.go        # `serve` command
│   ├── admin.go        # `admin create` command
│   └── mcp.go          # `mcp` command (stdio transport)
├── dbx/                # Database query builder (fork of ozzo-dbx)
│   ├── builder.go      # BaseBuilder: CreateTable, AddColumn, DropColumn, etc.
│   ├── builder_sqlite.go # SqliteBuilder: SQLite-specific overrides
│   ├── builder_pgsql.go  # PgsqlBuilder: PostgreSQL-specific overrides
│   └── builder_mysql.go  # MysqlBuilder: MySQL-specific overrides
├── tools/              # Shared utilities
│   ├── security/       # JWT, bcrypt, random string generation
│   ├── types/          # DateTime (custom time type with Scan/MarshalJSON)
│   ├── search/         # Filter/sort/search provider
│   ├── migrate/        # Migration runner
│   └── list/           # Slice/string helpers
├── ui/                 # Admin UI (Svelte SPA)
│   ├── src/
│   │   ├── components/settings/PageMCPTokens.svelte # MCP token management page
│   │   └── routes.js   # SPA routes (includes /settings/mcp-tokens)
│   └── dist/           # Built frontend (embedded in Go binary)
└── vendor/             # **DO NOT DELETE.** All Go dependencies.
```

## Build & Run

```bash
# Build (static, no CGO)
CGO_ENABLED=0 go build -o pb ./build/

# Run with SQLite (local development)
./pb serve --dataDsn "sqlite://./pb_data/dev.db"
./pb serve --dataDsn "./pb_data/dev.db"          # shorthand

# Run with PostgreSQL (production)
./pb serve --dataDsn "postgres://user:pass@127.0.0.1:5432/db?sslmode=disable"

# Run with MySQL
./pb serve --dataDsn "mysql://user:pass@tcp(127.0.0.1:3306)/db"

# Run with Redis cache
./pb serve --dataDsn "postgres://..." --redisDsn "redis://localhost:6379/0"

# Run MCP stdio server
./pb mcp --dataDsn "sqlite://./dev.db" --mcp-no-auth
./pb mcp --dataDsn "postgres://..." --mcp-token "TOKEN"

# Rebuild Admin UI
cd ui && npm install && npm run build && cd ..
go build -o pb ./build/    # must rebuild to embed updated dist/
```

## Database Driver Detection

All driver-specific code routes through `core/db_postgresql.go:connectDB()`:

```
DSN prefix          → driver name
─────────────────────────────────
postgres://         → "postgres"
postgresql://       → "postgres"
mysql://            → "mysql"
sqlite://           → "sqlite"
sqlite3://          → "sqlite"
*.db (suffix)       → "sqlite"
(no prefix)         → "postgres" (default)
```

When adding driver-specific SQL, check `db.DriverName()` against `"mysql"`, `"sqlite"`, `"sqlite3"`, or default to PostgreSQL.

### Key Driver Differences

| Feature | PostgreSQL | MySQL | SQLite |
|---------|-----------|-------|--------|
| Boolean type | `BOOLEAN` | `TINYINT(1)` | `INTEGER` |
| Timestamp default | `now()::TIMESTAMP` | `CURRENT_TIMESTAMP(3)` | `strftime('%Y-%m-%d %H:%M:%f', 'now')` |
| ID default | `uuid_generate_v4()::text` | (app-generated) | (app-generated) |
| JSON storage | `text` | `JSON` | `text` |
| ALTER TABLE IF NOT EXISTS | ✅ `ADD COLUMN IF NOT EXISTS` | ❌ check `information_schema` | ❌ check `PRAGMA table_info` |
| Index IF NOT EXISTS | ✅ | ❌ check `information_schema.statistics` | ✅ |
| Table metadata | `information_schema.tables` | `information_schema.tables` | `sqlite_master` |
| Column info | `pg_attribute` | `information_schema.columns` | `PRAGMA table_info` |
| Index info | `pg_indexes` | `information_schema.statistics` | `sqlite_master` (type='index') |
| Column comments | `COMMENT ON COLUMN` | `MODIFY ... COMMENT` | Not supported |

### Where Driver Checks Matter

- **`migrations/`** — Every migration with DDL must branch on `db.DriverName()`.
- **`daos/record_table_sync.go`** — Table creation and schema sync use driver-specific column types.
- **`daos/table.go`** — `HasTable`, `TableColumns`, `TableInfo`, `TableIndexes` all branch per driver.
- **`models/schema/schema_field.go:ColDefinition()`** — Returns driver-specific column type strings.
- **`daos/view.go`** — View creation/deletion (currently Postgres/MySQL only; SQLite views work but less tested).

## MCP Implementation

The MCP server is hand-written (no external SDK) as a JSON-RPC 2.0 implementation with zero additional dependencies.

### Architecture

- **`mcp/server.go`** — Core JSON-RPC router. Handles `initialize`, `tools/list`, `tools/call`, `resources/list`, `resources/read`, `ping`. Tool schemas are defined inline (lines ~180-270).
- **`mcp/tools.go`** — 8 tool handler functions. Uses `daos.Dao` for data access and `search.Provider` for filtering.
- **`mcp/resources.go`** — 2 resource handlers. Settings resource is sanitized (no secrets).
- **`mcp/auth.go`** — Two auth modes:
  - Standard JWT admin tokens (parsed via `security.ParseUnverifiedJWT`).
  - MCP-specific tokens (prefix `mcp_`): looked up in `_pb_mcp_tokens_` collection, checked for `active` flag and `expiresAt`.
- **`mcp/transport_sse.go`** — Three HTTP endpoints bound in `apis/base.go`:
  - `GET /api/mcp/sse` — SSE event stream
  - `POST /api/mcp/message` — Traditional message endpoint
  - `POST /api/mcp/stream` — Streamable HTTP (single request/response)
- **`mcp/transport_stdio.go`** — Line-delimited JSON-RPC over stdin/stdout.

### MCP Token Management

- **REST API:** `apis/mcp_token.go` — `GET/POST /api/mcp-tokens`, `DELETE /api/mcp-tokens/:id`, `POST /api/mcp-tokens/generate`. All require admin auth.
- **Collection:** `_pb_mcp_tokens_` (created by migration `1704067200_mcp_tokens.go`). Fields: `name`, `token`, `description`, `active`, `expiresAt`.
- **Token format:** `mcp_` + 48 random characters = 52 chars total. Full value shown only once at creation; list API masks to first 8 chars.
- **Admin UI:** `ui/src/components/settings/PageMCPTokens.svelte` — accessible at `/settings/mcp-tokens`.

## Coding Conventions

### Adding a New Migration

1. Create `migrations/<unix_timestamp>_<description>.go`.
2. Use `AppMigrations.Register(up, down)` in `init()`.
3. Branch on `db.DriverName()` for any DDL (see Driver Differences table above).
4. For SQLite column existence checks, use `PRAGMA table_info` instead of `information_schema`.

### Adding a New MCP Tool

1. Register the tool name in `mcp/server.go` tool schema list (around line 180-270).
2. Add a handler method in `mcp/tools.go` following the existing pattern:
   ```go
   func (s *Server) toolMyNewTool(args map[string]interface{}) (*ToolCallResult, error) {
       // parse args
       // use s.app.Dao() for data access
       // return &ToolCallResult{Content: []Content{{Type: "text", Text: result}}}
   }
   ```
3. Register the handler in `mcp/tools.go` init-like setup.

### Modifying Table Schema

- Update `daos/record_table_sync.go:SyncRecordTableSchema()` for new column types.
- Update `models/schema/schema_field.go:ColDefinition()` if adding a new field type.
- Always add driver-specific branches for `"sqlite"/"sqlite3"`, `"mysql"`, and default (postgres).

### Working with DateTime

- Use `types.DateTime` (wraps `time.Time`).
- `.Time()` accessor returns `time.Time`.
- `.IsZero()` checks for zero value.
- `types.NowDateTime()` creates current time.
- `DateTime.Scan()` handles `string`, `time.Time`, `int64` inputs.
- Default layout: `"2006-01-02 15:04:05.000"`.

### Record Operations

- Use `dao.FindRecordById(collectionId, recordId)` to fetch.
- Use `dao.SaveRecord(record)` to create/update.
- Use `dao.DeleteRecord(record)` to delete.
- Use `record.Set(key, value)` to set field values.
- Use `record.GetString(key)` / `record.Get(key)` to read field values.

## Maintenance Notes

- **Linter:** `golangci-lint` (config in `golangci.yml`).
- **CI/CD:** GitHub Actions in `.github/workflows/` (Build Check and GoReleaser).
- **Versioning:** Version is set via `ldflags` in `.goreleaser.yaml`.
- **Vendor directory:** `/vendor/` must not be deleted. All dependencies are vendored.
- **Testing:** `go test ./tools/... ./models/... ./daos/...` — tests use `modernc.org/sqlite` (pure Go).
- **Static analysis:** `go vet ./...` should pass with no warnings.
