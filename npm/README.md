# PostgreBase

🚀 AI原生的无代码API开发平台 — 支持 PostgreSQL、MySQL、SQLite

AI-Native No-Code API Platform with built-in MCP server, supporting PostgreSQL, MySQL, and SQLite.

## Installation

```bash
npm install -g postgrebase-installer
```

## Quick Start

```bash
# SQLite (local development)
postgrebase serve --dataDsn "sqlite://./data.db"

# PostgreSQL (production)
postgrebase serve --dataDsn "postgres://user:pass@localhost:5432/db?sslmode=disable"

# MySQL
postgrebase serve --dataDsn "mysql://user:pass@tcp(localhost:3306)/db"

# With Redis cache
postgrebase serve --dataDsn "postgres://..." --redisDsn "redis://localhost:6379/0"
```

## Features

- 🗄️ **Multi-Database**: PostgreSQL, MySQL, SQLite support
- 🤖 **MCP Server**: Built-in Model Context Protocol server for AI tools
- ⚡ **REST API**: Instant RESTful API for your collections
- 🔄 **Realtime**: Server-Sent Events (SSE) subscriptions
- 🎛️ **Admin UI**: Built-in admin dashboard
- 🔐 **Auth**: Built-in authentication with JWT tokens
- 📁 **File Storage**: Local and S3-compatible storage
- 🚀 **High Performance**: Designed for clustered environments

## MCP (Model Context Protocol)

PostgreBase includes a built-in MCP server for AI tools like Claude Desktop, Cursor, and Windsurf:

```bash
# Start MCP server (stdio mode)
postgrebase mcp --dataDsn "sqlite://./dev.db" --mcp-no-auth

# With authentication
postgrebase mcp --dataDsn "postgres://..." --mcp-token "YOUR_TOKEN"
```

### Claude Desktop Configuration

```json
{
  "mcpServers": {
    "postgrebase": {
      "command": "postgrebase",
      "args": ["mcp", "--dataDsn", "sqlite:///path/to/dev.db", "--mcp-no-auth"]
    }
  }
}
```

## Configuration

| Flag | Description | Example |
|------|-------------|---------|
| `--dataDsn` | Database connection string | `postgres://user:pass@host:5432/db` |
| `--redisDsn` | Redis connection string (optional) | `redis://localhost:6379/0` |
| `--dir` | Data directory for file uploads | `./pb_data` |
| `--debug` | Enable debug mode | _(flag)_ |

## More Information

- **GitHub**: [github.com/zhenruyan/postgrebase](https://github.com/zhenruyan/postgrebase)
- **Documentation**: [README](https://github.com/zhenruyan/postgrebase/blob/main/README.md)

## Uninstall

```bash
npm uninstall -g postgrebase-installer
```

## License

MIT — Based on [PocketBase](https://pocketbase.io) by [Gani Georgiev](https://github.com/ganigeorgiev)
