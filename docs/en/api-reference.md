# API Reference

PostgreBase provides a RESTful API that is 100% compatible with PocketBase.

## Base URL

```
http://127.0.0.1:8090/api/
```

## Authentication

### Admin Authentication

Admin endpoints require a valid JWT token in the `Authorization` header:

```
Authorization: Bearer <admin_jwt_token>
```

Obtain a token via:

```
POST /api/admins/auth-via-email
{
  "identity": "admin@example.com",
  "password": "your_password"
}
```

### Record Authentication

For user-facing auth:

```
POST /api/collections/{collection}/auth-via-email
{
  "identity": "user@example.com",
  "password": "password"
}
```

## Collections

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/collections` | List all collections |
| GET | `/api/collections/{id}` | Get collection details |
| POST | `/api/collections` | Create collection (admin) |
| PATCH | `/api/collections/{id}` | Update collection (admin) |
| DELETE | `/api/collections/{id}` | Delete collection (admin) |

## Records (CRUD)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/collections/{collection}/records` | List records |
| GET | `/api/collections/{collection}/records/{id}` | Get record |
| POST | `/api/collections/{collection}/records` | Create record |
| PATCH | `/api/collections/{collection}/records/{id}` | Update record |
| DELETE | `/api/collections/{collection}/records/{id}` | Delete record |

### Query Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `page` | Page number | `page=1` |
| `perPage` | Records per page | `perPage=50` |
| `sort` | Sort field and direction | `sort=-created` |
| `filter` | Filter expression | `filter=status="active"` |
| `expand` | Expand relations | `expand=author,comments` |
| `fields` | Select specific fields | `fields=id,title,created` |

### Filter Syntax

```
status = "active"
created > "2024-01-01"
title ~ "search term" && status != "draft"
price >= 10 && price <= 100
```

## Realtime (SSE)

Subscribe to record changes:

```
GET /api/realtime
```

Then subscribe to a specific collection:

```json
{
  "action": "subscribe",
  "collection": "posts"
}
```

## MCP Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/mcp/sse` | SSE event stream |
| POST | `/api/mcp/message` | Send JSON-RPC request |
| POST | `/api/mcp/stream` | Streamable HTTP |

## MCP Token Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/mcp-tokens` | List tokens (admin) |
| POST | `/api/mcp-tokens` | Create token (admin) |
| POST | `/api/mcp-tokens/generate` | Generate token (admin) |
| DELETE | `/api/mcp-tokens/:id` | Revoke token (admin) |

## Health Check

```
GET /api/health
```

Returns `200 OK` when the server is running.
