# 内嵌 Agent 平台方案

> 状态: Draft
> 目标: 在 PostgreBase 内嵌一个可配置的 Agent Runtime，支持多厂商/多模型、图片与文档输入，并将建表、灌数、查询全部收敛到注册 tool 的统一流程。

## 1. 背景

PostgreBase 当前已经具备数据库抽象、MCP 工具、REST API、Admin UI 和多数据库支持。下一步不应该把 AI 能力做成“额外接一层 LLM 调用”，而是把 Agent 变成平台内的原生执行单元：

- AI 负责理解意图、规划步骤、选择工具。
- 所有结构化数据操作必须走 tool，不允许 Agent 直接拼 SQL。
- 多模态输入统一归一到消息内容，供 Agent 处理图片、PDF、Word、表格等内容。
- 同一套 Agent 能力既可用于 UI，也可用于 API，也可用于 MCP 或自动化任务。

## 2. 目标

1. 支持配置多个厂商和多个模型。
2. 支持同一 Agent 会话内切换模型，或按任务选择模型。
3. 支持图片、PDF、文档、表格等输入。
4. 支持项目内“建表”但不暴露 SQL，统一走注册工具。
5. 支持“灌入数据”和“查询数据”全部走注册工具。
6. 复用 `github.com/startvibecoding/vibecoding/agent` 作为外层 Agent SDK。

## 3. 非目标

- 不在 Agent 层开放任意 SQL 执行。
- 不把数据库 DDL/DML 直接写进 prompt。
- 不把各厂商差异散落在业务代码里。
- 不重做现有 REST / MCP / DAO 体系，而是在其上加一层 Agent 编排。

## 4. 总体架构

### 4.1 三层结构

1. **Agent 层**
   - 用 vibecoding 的 `agent.Builder` 创建运行实例。
   - 负责对话、规划、工具调用、事件流。

2. **Tool 层**
   - 注册统一工具集：
     - `schema.create_table`
     - `schema.add_field`
     - `schema.update_field`
     - `schema.drop_field`
     - `data.insert`
     - `data.update`
     - `data.query`
     - `data.delete`
     - `file.ingest`
     - `file.extract`
     - `dataset.preview`
   - Tool 是唯一的数据操作入口。

3. **DAO / Core 层**
   - tool 内部调用现有 `Dao`、`Record`、`Collection`、`Migration`、`File` 逻辑。
   - 统一校验、事务、审计、权限、错误包装。

### 4.2 核心原则

- Agent 只做决策，不直接持久化。
- Tool 只做受控能力，不暴露底层 SQL。
- Schema 变更必须走规范化流程，保证 Postgres / MySQL / SQLite 一致。

## 5. Agent SDK 接入方式

### 5.1 采用 vibecoding/agent 作为运行时

建议在 PostgreBase 中新增一个嵌入式 agent 包装层，复用如下能力：

- `agent.NewBuilder()`
- `WithProvider(...)` / `WithProviderByName(...)`
- `WithModel(...)`
- `WithMode(...)`
- `WithTools(...)`
- `WithApprovalHandler(...)`
- `Run(...)` / `RunWithMessages(...)`

### 5.2 Provider 接入策略

每个厂商做成一个 provider 适配器，统一映射到 Agent SDK 的 `agent.Provider` 接口。

Provider 需要支持：

- 多模型列表 `Models()`
- 模型兼容信息 `ModelCompat`
- 文本输入
- 图片输入
- 文档输入
- 工具调用流

推荐配置形式：

```jsonc
{
  "agents": {
    "providers": [
      {
        "id": "openai-main",
        "vendor": "openai",
        "baseURL": "https://api.openai.com/v1",
        "apiKey": "env:OPENAI_API_KEY",
        "models": ["gpt-4.1", "gpt-4o-mini"]
      },
      {
        "id": "deepseek-main",
        "vendor": "deepseek",
        "baseURL": "https://api.deepseek.com",
        "apiKey": "env:DEEPSEEK_API_KEY",
        "models": ["deepseek-chat", "deepseek-reasoner"]
      }
    ],
    "defaultProvider": "openai-main",
    "defaultModel": "gpt-4.1"
  }
}
```

## 6. 多模态输入

### 6.1 输入类型

- **图片**：上传后转为 `ImageContent` 或 provider 可识别的多模态消息。
- **PDF / 文档**：先做抽取，再把正文、标题、表格结构分段送入 Agent。
- **表格**：解析为结构化行列，必要时附带原始文件片段。
- **混合输入**：同一轮请求可包含文本 + 图片 + 文档摘要。

### 6.2 推荐处理链路

1. 文件先进入现有 file subsystem。
2. 生成 `fileId`、mime type、hash、大小、来源。
3. 对可解析文档做预处理：
   - PDF -> 文字 + 页码
   - DOCX -> 标题层级 + 段落 + 表格
   - XLSX -> sheet / row / column 结构
4. 结果以 message content blocks 形式喂给 Agent。
5. Agent 只看“内容”，不直接接触底层文件系统。

## 7. 统一 tool 流程

### 7.1 建表不能走 SQL

“建表”不提供 `CREATE TABLE` 原文执行入口，而是提供结构化工具：

- `schema.create_table`
- `schema.add_field`
- `schema.update_field`
- `schema.drop_field`
- `schema.create_index`
- `schema.set_relation`

工具内部做：

- 字段合法性校验
- 驱动兼容转换
- 迁移/同步流程
- 回滚或补偿

### 7.2 灌入数据走 tool

数据导入不直接批量插 SQL，而是：

- `data.insert`
- `data.bulk_insert`
- `file.ingest`

支持模式：

- CSV / XLSX 映射到 collection
- 文档抽取结果映射到 collection
- AI 解析结果批量写入

### 7.3 查询数据走 tool

查询统一走：

- `data.query`
- `data.get`
- `dataset.preview`

查询参数必须是结构化对象：

- collection
- filters
- sort
- pagination
- field projection

不允许 Agent 拼接 SQL 片段。

## 8. Tool 注册机制

### 8.1 注册表

建议新增一个 tool registry，用于集中管理工具定义、参数 schema、执行器、权限与日志。

注册条目至少包含：

- name
- description
- input schema
- executor
- required permissions
- audit category

### 8.2 统一执行流程

1. Agent 产生 tool call。
2. Registry 校验 tool 是否存在。
3. 校验权限、项目范围、collection 范围。
4. 进入事务或分步执行器。
5. 写审计日志。
6. 返回标准化 tool result。

### 8.3 好处

- 工具能力可复用到 UI、MCP、HTTP、批处理。
- 便于后续加审批、配额、审计、回放。
- 便于隔离高风险操作，如 schema 变更。

## 9. 运行模式

### 9.1 项目内 Agent

在每个项目下提供一个 Agent 配置：

- 默认 provider
- 默认 model
- 允许的工具集合
- 文件输入策略
- 是否允许 schema 变更
- 审批策略

### 9.2 会话模型

建议支持三类会话：

- **一次性任务**：上传资料，完成建表/导入/查询后结束。
- **项目长期会话**：记住上下文，持续优化 schema 和数据。
- **受控自动化会话**：在后台定时执行 tool 流程。

## 10. UI / API 建议

### 10.1 UI

新增一个 Agent 工作区：

- 左侧：会话和项目
- 中间：对话流和 tool 结果
- 右侧：provider / model / tools / files / schema 状态

### 10.2 API

建议新增：

- `POST /api/agents/sessions`
- `POST /api/agents/sessions/:id/messages`
- `GET /api/agents/providers`
- `GET /api/agents/models`
- `POST /api/agents/tools/:name`

文件输入则复用现有上传能力，再把 fileId 交给 Agent。

## 11. 落地拆分

### Phase 1

- 接入 vibecoding/agent runtime。
- 实现 provider/model 配置。
- 打通文本对话 + tool call。

### Phase 2

- 接入图片和文档输入。
- 加入文件抽取管线。
- 统一 tool registry。

### Phase 3

- 实现 schema tool 集。
- 实现 data import/query tool 集。
- 加入审批、审计、回放。

### Phase 4

- 补齐 UI 工作区。
- 开放项目级 Agent 配置。
- 扩展 MCP / HTTP / 批处理共用同一套工具层。

## 12. 风险点

- 多厂商多模型会带来请求格式差异，需要放进 provider adapter。
- 建表/改表属于高风险操作，必须有审批和回滚策略。
- 文档解析质量会直接影响结果，需要做分层抽取和可追溯输出。
- 工具集如果不收口，后续会变成“半 SQL 半 Agent”的混合态，必须守住统一流程。

## 13. 结论

这个方案的核心不是“给 PostgreBase 接一个聊天窗口”，而是把 Agent 变成平台的编排层：

- 模型是可配置的。
- 内容是多模态的。
- 数据操作是工具化的。
- Schema 变更是受控的。

这样 PostgreBase 才能真正形成“AI 原生的无代码 API 开发平台”闭环。
