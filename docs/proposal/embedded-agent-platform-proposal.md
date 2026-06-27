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
6. 支持向量数据库类型，后端基于 `sqlite-vec`（modernc）实现。
7. 支持配置 `embedded` 的 LLM 模型自动进行 embedding。
8. 复用 `github.com/startvibecoding/vibecoding/agent` 作为外层 Agent SDK。

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

### 4.3 统一业务内核

Tool 走的逻辑和网页端 API 走的逻辑必须一致，不能出现两套实现。

要求是：

- Web API、Tool、后续可能的 CLI/批处理，都调用同一套 service/use-case。
- 业务规则、权限校验、字段映射、默认值、事务边界都只保留一份。
- Tool 只是 service 的一个调用入口，Web API 也是同一个 service 的另一个入口。
- 如果某个能力需要调整，改 service 即可，同时影响网页端和 tool 端。

### 4.4 项目边界约束

Agent 的运行必须约束在 PostgreBase 的项目层。

这里的“项目边界”不是抽象概念，而是明确落到项目 ID：

- Agent 只能读取和操作当前 `project_id` 下的 collection、record、file 及其关联资源。
- 所有 tool、API、后台任务都必须显式携带或继承 `project_id`。
- 查询和写入都要先做 project scope 过滤，避免跨项目访问。
- 现有 `collection.project` 字段可以直接作为第一层边界依据。
- 若某个资源未绑定项目，默认不允许 Agent 直接操作。

### 4.2 核心原则

- Agent 只做决策，不直接持久化。
- Tool 只做受控能力，不暴露底层 SQL。
- Schema 变更必须走规范化流程，保证 Postgres / MySQL / SQLite 一致。
- 向量能力是独立的数据类型，不混入普通关系型表的 CRUD 流程。

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

### 5.1.1 依赖前置条件

如果 `github.com/startvibecoding/vibecoding/agent` 的现有能力不足，优先改造 `vibecoding` 仓库，再回到 PostgreBase 接入。

这是前置约束，不作为 PostgreBase 里临时绕过的理由。

当前从接口层面已经能看出的潜在缺口包括：

- **多模态消息粒度不够细**：现有 `ContentBlock` 有 `text` / `image` / `thinking` / `toolCall`，但没有显式的 `document` / `file` 语义，需要上层自己封装。
- **运行时模型切换不够直接**：Builder 以单一 `Provider + Model` 为主，任务过程中如果要按步骤切模型，可能需要扩展 Agent 配置或会话级路由。
- **文件解析不是 SDK 责任**：图片和文档输入需要在 Agent 外先做抽取，再以统一内容块喂入。

如果后续评审确认这些限制会影响产品目标，应先在 `vibecoding/agent` 内补齐能力，再让 PostgreBase 复用。

### 5.2 Provider 接入策略

每个厂商做成一个 provider 适配器，统一映射到 Agent SDK 的 `agent.Provider` 接口。

Provider 需要支持：

- 多模型列表 `Models()`
- 模型兼容信息 `ModelCompat`
- 文本输入
- 图片输入
- 文档输入
- 工具调用流

### 5.3 后台配置与持久化

LLM 厂商配置不放在启动配置文件里处理，而是放到后台管理界面里维护，并持久化到数据库配置表。

这意味着：

- provider、model、baseURL、apiKey、启用状态、默认模型等，都由后台管理配置。
- 启动时只读取数据库配置，不从本地 config 文件解析厂商列表。
- 配置变更后立即生效，或通过轻量热加载刷新缓存。
- 配置表需要支持审计和权限控制，避免普通用户修改。

示意结构：

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

实际落库时可拆成独立的 provider / model 配置表，而不是依赖静态配置文件。

## 6. 多模态输入

### 6.1 输入类型

- **图片**：由模型原生视觉能力解析，上传后直接作为图像内容输入 Agent。
- **文档**：当前方案先不支持文档解析。
- **混合输入**：同一轮请求可包含文本 + 图片。

### 6.2 推荐处理链路

1. 文件先进入现有 file subsystem。
2. 生成 `fileId`、mime type、hash、大小、来源。
3. 图片直接进入模型视觉输入链路。
4. 结果以 message content blocks 形式喂给 Agent。
5. Agent 只看“内容”，不直接接触底层文件系统。

### 6.3 向量输入

向量能力用于检索、相似度搜索和语义组织，不作为 Agent 的主输入格式。

- 向量库类型：`sqlite-vec`（modernc）
- embedding 来源：配置的 `embedded` 模型
- embedding 触发：在写入、导入、更新时自动生成
- 向量查询：通过单独的向量查询 tool 或检索 service 完成

向量索引和普通业务数据保持解耦，但必须共享同一项目边界和权限模型。

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

查询结果还应支持可视化输出，按需要渲染为图表：

- 折线图
- 柱状图
- 饼图
- 指标卡
- 表格

实现上建议由查询 tool 返回结构化数据 + 推荐图表类型，再由前端用 `echarts` 渲染，不把图表逻辑塞进 Agent 本身。

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

工具注册保持静态化是一个特性，不是限制。
这样可以：

- 提高 token 缓存命中率
- 减少每轮上下文中的 tool schema 波动
- 让模型更稳定地学习可用工具集合
- 降低工具定义频繁变化带来的 prompt 抖动

### 8.2 统一执行流程

1. Agent 产生 tool call。
2. Registry 校验 tool 是否存在。
3. 校验权限、`project_id`、collection 范围。
4. 对写操作先走授权检查。
5. 进入事务或分步执行器。
6. 写审计日志。
7. 返回标准化 tool result。

### 8.3 写操作授权

Agent 的写入性操作必须授权，不能默认放行。

需要授权的操作包括但不限于：

- schema 创建、修改、删除
- collection 创建、修改、删除
- record 创建、更新、删除
- 批量导入
- 文件入库后触发的持久化写入

建议策略：

- 读操作默认允许在项目边界内执行。
- 写操作默认进入 pending 状态。
- 由 UI、API 或审批策略显式授权后再执行。
- 可以按 tool 名称、collection、风险等级、用户身份做细粒度授权。

### 8.4 好处

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

会话展示规则：

- `session_id` 只作为内部逻辑标识。
- 用户第一次输入后，由 LLM 为当前 session 生成一次名称。
- 名称生成后，UI 以及大部分对外展示都显示 session 名称，不再直接显示 session id。
- session 名称只生成一次，后续沿用，除非用户显式重命名。

## 10. UI / API 建议

### 10.1 UI

新增一个 Agent 工作区：

- 左侧：会话和项目
- 中间：对话流和 tool 结果
- 右侧：provider / model / tools / files / schema 状态

会话列表默认展示 session 名称，未命名前可短暂显示 `session_id` 作为占位。

聊天框结构线框：

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Project: p_12345   Session: s_98765   Provider: openai   Model: gpt-4.1     │
├───────────────┬───────────────────────────────────────────┬──────────────────┤
│ Projects      │ Conversation                               │ Inspector        │
│               │                                           │                  │
│ • Project A   │  [system] 任务目标 / 约束                   │  Provider        │
│ • Project B   │  [user]    上传图片并生成表结构             │  - openai        │
│ • Project C   │  [assistant] 规划中...                     │  - deepseek      │
│               │                                           │                  │
│ Sessions      │  [tool]    schema.create_table             │  Model           │
│ • s_98765     │  [tool result] created table users          │  - gpt-4.1       │
│ • s_98766     │                                           │  - gpt-4o-mini   │
│               │  [assistant] 需要授权写入                  │                  │
│ Tools         │                                           │  Scope           │
│ • query       │  [approval] Allow write? [Approve][Deny]   │  project_id      │
│ • insert      │                                           │  collections     │
│ • update      │  ───────────────────────────────────────   │  auth policy     │
│ • schema.*    │  Message input...                           │                  │
│               │  [Attach Image] [Choose Tool] [Send]        │  Files           │
└───────────────┴───────────────────────────────────────────┴──────────────────┘
```

查询结果展示建议补一块图表预览区，和表格同级切换：

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Query Result                                                                 │
├──────────────────────────────┬───────────────────────────────────────────────┤
│ Table                         │ Chart                                         │
│ ┌──────────┬──────────┐      │  [Line | Bar | Pie | Metric]                 │
│ │ date     │ amount   │      │  ┌─────────────────────────────────────────┐  │
│ │ 2026-01  │ 120      │      │  │                echarts                 │  │
│ │ 2026-02  │ 180      │      │  │      series / axes / legend / tooltip   │  │
│ └──────────┴──────────┘      │  └─────────────────────────────────────────┘  │
└──────────────────────────────┴───────────────────────────────────────────────┘
```

### 10.2 API

建议新增：

- `POST /api/agents/sessions`
- `POST /api/agents/sessions/:id/messages`
- `GET /api/agents/providers`
- `GET /api/agents/models`
- `POST /api/agents/tools/:name`

这些接口不应该直接编排 DAO，而是转发到与 tool 共用的 service 层，确保网页端和 tool 的行为、错误码、权限规则、默认值保持一致。
此外，所有接口必须先解析 `project_id`，再进入同一套项目级 service，不能绕过项目边界直接访问全局数据。

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
