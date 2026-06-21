# MCP（Model Context Protocol）

PostgreBase 内置 MCP 服务器，通过标准化 JSON-RPC 2.0 协议让 AI 工具直接操作你的数据。

## 概述

MCP（Model Context Protocol）让 Claude Desktop、Cursor、Windsurf 等 AI 工具直接读写你的数据库记录。PostgreBase 原生实现了这个协议 —— 无需外部 MCP 服务器。

## 传输模式

### SSE（Server-Sent Events）— HTTP

运行 `./pb serve` 后可用。连接地址：

```
GET  http://localhost:8090/api/mcp/sse      # SSE 事件流
POST http://localhost:8090/api/mcp/message   # 发送 JSON-RPC 请求
POST http://localhost:8090/api/mcp/stream    # Streamable HTTP（单次请求/响应）
```

### Stdio — CLI

以独立进程运行 MCP 服务器，通过 stdin/stdout 通信：

```bash
./pb mcp --dataDsn "sqlite://./dev.db" --mcp-token "YOUR_TOKEN"

# 或禁用认证（不推荐用于生产）
./pb mcp --dataDsn "sqlite://./dev.db" --mcp-no-auth
```

| 参数 | 说明 |
|------|------|
| `--mcp-token` | Admin JWT token 或 MCP 专用 token |
| `--mcp-no-auth` | 禁用认证（仅限开发环境） |

## Claude Desktop 配置

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

## MCP Token

在生产环境中，建议通过 Admin UI 的 **Settings → MCP Tokens** 创建专用 MCP Token（`mcp_` 前缀）。

### Token 管理 API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/mcp-tokens` | GET | 列出所有 token（已脱敏） |
| `/api/mcp-tokens` | POST | 创建新 token |
| `/api/mcp-tokens/generate` | POST | 生成自定义 token |
| `/api/mcp-tokens/:id` | DELETE | 撤销 token |

所有端点需要管理员身份验证。

### Token 属性

- **格式：** `mcp_` + 48 个随机字符 = 共 52 字符
- **与 Admin JWT Token 独立** —— 可单独撤销
- **支持可选过期时间**
- **创建时仅显示一次完整值**
- **列表 API 显示前 8 个字符**

## 可用工具

| 工具 | 说明 |
|------|------|
| `list_collections` | 列出所有集合 |
| `get_collection` | 获取集合的 Schema 和设置 |
| `list_records` | 列出记录（支持分页、过滤、排序） |
| `get_record` | 通过 ID 获取单条记录 |
| `create_record` | 创建新记录 |
| `update_record` | 更新已有记录 |
| `delete_record` | 删除记录 |
| `search_records` | 使用 PostgreBase 过滤表达式搜索记录 |

## 可用资源

| URI | 说明 |
|-----|------|
| `postgrebase://collections` | 所有集合及其 Schema |
| `postgrebase://settings` | 应用设置（已脱敏，不含敏感信息） |
