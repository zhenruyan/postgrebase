# API 参考

PostgreBase 提供 100% 兼容 PocketBase 的 RESTful API。

## 基础 URL

```
http://127.0.0.1:8090/api/
```

## 认证

### 管理员认证

管理员端点需要在 `Authorization` 头中携带有效的 JWT Token：

```
Authorization: Bearer <admin_jwt_token>
```

通过以下方式获取 Token：

```
POST /api/admins/auth-via-email
{
  "identity": "admin@example.com",
  "password": "your_password"
}
```

### 用户认证

用户认证：

```
POST /api/collections/{collection}/auth-via-email
{
  "identity": "user@example.com",
  "password": "password"
}
```

## 集合

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/collections` | 列出所有集合 |
| GET | `/api/collections/{id}` | 获取集合详情 |
| POST | `/api/collections` | 创建集合（管理员） |
| PATCH | `/api/collections/{id}` | 更新集合（管理员） |
| DELETE | `/api/collections/{id}` | 删除集合（管理员） |

## 记录（CRUD）

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/collections/{collection}/records` | 列出记录 |
| GET | `/api/collections/{collection}/records/{id}` | 获取记录 |
| POST | `/api/collections/{collection}/records` | 创建记录 |
| PATCH | `/api/collections/{collection}/records/{id}` | 更新记录 |
| DELETE | `/api/collections/{collection}/records/{id}` | 删除记录 |

### 查询参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `page` | 页码 | `page=1` |
| `perPage` | 每页记录数 | `perPage=50` |
| `sort` | 排序字段和方向 | `sort=-created` |
| `filter` | 过滤表达式 | `filter=status="active"` |
| `expand` | 展开关联 | `expand=author,comments` |
| `fields` | 选择特定字段 | `fields=id,title,created` |

### 过滤语法

```
status = "active"
created > "2024-01-01"
title ~ "search term" && status != "draft"
price >= 10 && price <= 100
```

## 实时订阅（SSE）

订阅记录变更：

```
GET /api/realtime
```

然后订阅特定集合：

```json
{
  "action": "subscribe",
  "collection": "posts"
}
```

## MCP 端点

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/mcp/sse` | SSE 事件流 |
| POST | `/api/mcp/message` | 发送 JSON-RPC 请求 |
| POST | `/api/mcp/stream` | Streamable HTTP |

## MCP Token 管理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/mcp-tokens` | 列出 token（管理员） |
| POST | `/api/mcp-tokens` | 创建 token（管理员） |
| POST | `/api/mcp-tokens/generate` | 生成 token（管理员） |
| DELETE | `/api/mcp-tokens/:id` | 撤销 token（管理员） |

## 健康检查

```
GET /api/health
```

服务器运行时返回 `200 OK`。
