# Development Guide

## Building from Source

### Prerequisites

- Go 1.18+
- Node.js (for Admin UI development)

### Build Server

```bash
git clone https://github.com/zhenruyan/postgrebase.git
cd postgrebase
go build -o pb ./build/
```

Static build (no CGO):

```bash
CGO_ENABLED=0 go build -o pb ./build/
```

### Build Admin UI

If you modify the Admin UI, rebuild the embedded assets before compiling:

```bash
cd ui
npm install
npm run build
cd ..
go build -o pb ./build/
```

## Running Tests

```bash
go test ./tools/... ./models/... ./daos/...
```

Tests use `modernc.org/sqlite` (pure Go) — no external database required.

## Static Analysis

```bash
go vet ./...
```

Linter configuration is in `golangci.yml`.

## Project Structure

```
postgrebase/
├── build/           # Server entry point (main.go)
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

## Adding a New Migration

1. Create `migrations/<unix_timestamp>_<description>.go`.
2. Use `AppMigrations.Register(up, down)` in `init()`.
3. Branch on `db.DriverName()` for any DDL.

### Driver-Specific SQL

When adding driver-specific SQL, check `db.DriverName()` against:

- `"mysql"` — MySQL
- `"sqlite"` / `"sqlite3"` — SQLite
- Default — PostgreSQL

### SQLite Column Checks

Use `PRAGMA table_info` instead of `information_schema` for SQLite.

## Adding a New MCP Tool

1. Register the tool name in `mcp/server.go` tool schema list.
2. Add a handler method in `mcp/tools.go`:

```go
func (s *Server) toolMyNewTool(args map[string]interface{}) (*ToolCallResult, error) {
    // parse args
    // use s.app.Dao() for data access
    return &ToolCallResult{Content: []Content{{Type: "text", Text: result}}}, nil
}
```

3. Register the handler in the tools dispatch map.

## Modifying Table Schema

- Update `daos/record_table_sync.go:SyncRecordTableSchema()` for new column types.
- Update `models/schema/schema_field.go:ColDefinition()` if adding a new field type.
- Always add driver-specific branches for SQLite, MySQL, and PostgreSQL.

## Working with DateTime

Use `types.DateTime` (wraps `time.Time`):

```go
now := types.NowDateTime()
t := now.Time()        // returns time.Time
zero := types.DateTime{}
zero.IsZero()          // true
```

Default layout: `"2006-01-02 15:04:05.000"`

## Record Operations

```go
// Fetch
record, err := dao.FindRecordById(collectionId, recordId)

// Set values
record.Set("title", "New Title")

// Read values
title := record.GetString("title")

// Save (create or update)
err := dao.SaveRecord(record)

// Delete
err := dao.DeleteRecord(record)
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Vendor Directory

The `/vendor/` directory contains all Go dependencies and **must not be deleted**. All dependencies are vendored for offline builds.

## CI/CD

- **GitHub Actions** in `.github/workflows/` (Build Check and GoReleaser)
- **Version** is set via `ldflags` in `.goreleaser.yaml`
