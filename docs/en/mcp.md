# MCP (Model Context Protocol)

PostgreBase includes a built-in MCP server that allows AI tools to interact with your data via a standardized JSON-RPC 2.0 protocol.

## Overview

MCP (Model Context Protocol) enables AI tools like Claude Desktop, Cursor, and Windsurf to directly read and write your database records. PostgreBase implements this protocol natively — no external MCP server needed.

## Transport Modes

### SSE (Server-Sent Events) — HTTP

Available when running `./pb serve`. Connect to:

```
GET  http://localhost:8090/api/mcp/sse      # SSE event stream
POST http://localhost:8090/api/mcp/message   # Send JSON-RPC requests
POST http://localhost:8090/api/mcp/stream    # Streamable HTTP (single request/response)
```

### Stdio — CLI

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

## Claude Desktop Configuration

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

## MCP Tokens

For production use, create dedicated MCP tokens (prefixed with `mcp_`) from the Admin UI under **Settings → MCP Tokens**.

### Token Management API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/mcp-tokens` | GET | List all tokens (masked) |
| `/api/mcp-tokens` | POST | Create a new token |
| `/api/mcp-tokens/generate` | POST | Generate a token with custom settings |
| `/api/mcp-tokens/:id` | DELETE | Revoke a token |

All endpoints require admin authentication.

### Token Properties

- **Format:** `mcp_` + 48 random characters = 52 chars total
- **Separate from admin JWT tokens** — can be revoked individually
- **Optional expiration dates**
- **Full value shown only once** at creation time
- **List API masks** to first 8 characters

## Available Tools

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

## Available Resources

| URI | Description |
|-----|-------------|
| `postgrebase://collections` | All collections with their schemas |
| `postgrebase://settings` | Application settings (sanitized, no secrets) |
