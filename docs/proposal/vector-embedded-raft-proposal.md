# 向量能力内嵌与 SQLite Raft 同步方案

> 状态: Draft
> 目标: 将向量能力直接内嵌到 `pb serve`，单实例正常运行，多实例时通过 Raft 同步向量元数据、无 Redis 场景下的内存缓存，以及 SQLite 集群模式下的主业务库状态。

> 当前实现阶段:
> - 已实现通用 replicated operation envelope，并保持现有 vector operation 兼容。
> - 已实现 strict operation 语义：SQLite 强一致写入在 follower 转发 leader 失败时返回错误，不再本地 fallback apply。
> - 已接入 SQLite 集群模式下的 collection create/update/delete/import，使 `_collections` 元数据和 `SyncRecordTableSchema` 触发的物理表结构变更通过同一复制入口 apply。
> - 已接入 admin create/update/delete，使首次注册 admin 也会进入 SQLite strict replicated operation。
> - 已接入普通 record create/update/delete，使管理界面表数据 CRUD 进入 SQLite strict replicated operation。
> - strict SQLite operation 当前会按 peer 顺序同步复制，避免 collection/schema 与 record 写入乱序；完整多数派提交、失败重放和 log index 仍待补齐。
> - auth 辅助流程、MCP/Agent 直连 record 写入、settings/migrations 复制、完整 SQLite 快照安装仍属于后续阶段。

## 1. 背景

PostgreBase 的 Agent 方案已经要求：

- 项目边界明确
- 数据操作统一走 tool
- 写操作必须授权
- 向量能力使用 `sqlite-vec`（modernc）
- embedding 由配置的 `embedded` 模型自动生成

下一步需要把这些能力收敛到 `pb serve` 内部，而不是拆成独立 `vec` 进程。

## 2. 目标

1. 向量能力直接内嵌到 `pb serve`。
2. 单实例时正常启动，不引入集群协议开销。
3. 多实例时自动进入 Raft 集群模式。
4. 支持 `embedded` LLM 模型自动进行 embedding。
5. 后端使用 `sqlite-vec`（modernc）。
6. 没有 Redis 时，使用内存缓存模式，并通过 Raft 同步必要状态。
7. 后台管理需要提供监控状态面板。
8. 向量与缓存都必须遵守项目边界。
9. PostgreSQL、MySQL、SQLite 都支持向量与缓存的 Raft 同步；其中 SQLite 作为主数据库时，主业务库必须纳入同一个 Raft 复制状态机。

## 3. 非目标

- 不再引入独立 `pb vec` 命令。
- 不把向量检索做成单独服务。
- 不把 Redis 作为集群唯一依赖。
- 不把 PostgreSQL/MySQL 的外部数据库复制做成 PostgreBase 自己的数据库级复制；这两类数据库默认由外部数据库自身保证一致性。

## 4. 运行模式

### 4.1 单实例

```bash
pb serve --dataDsn "postgres://..." --dir ./pb_data
```

行为：

- 直接启动业务服务
- 本地维护 `sqlite-vec`
- 本地使用内存缓存或 Redis 缓存
- 直接执行 embedding 与检索

### 4.2 多实例

```bash
pb serve --dataDsn "postgres://..." --listen :8090
pb serve --dataDsn "postgres://..." --listen :8091
pb serve --dataDsn "postgres://..." --listen :8092
```

行为：

- 自动进入 Raft 集群模式
- 选主
- 同步向量元数据
- 同步无 Redis 场景下的内存缓存状态
- SQLite 主库模式下，同步 collection 元数据、业务表结构和业务数据写入
- 协调 embedding 任务和检索路由

## 5. Raft 负责什么

Raft 负责复制 PostgreBase 在多实例模式下必须保持一致的状态机。

向量库、任务队列、无 Redis 缓存状态都是这个状态机的一部分。SQLite 作为主数据库时，主业务库也必须成为同一个状态机的一部分，不能独立于向量复制链路之外。

这条复制链路对所有数据库模式都启用，但作用范围不同：

- SQLite：同步向量库全表内容、可同步缓存状态，以及主业务库状态
- PostgreSQL：同步向量层状态与可同步缓存状态
- MySQL：同步向量层状态与可同步缓存状态

主业务库是否纳入 Raft，取决于当前主数据库类型。SQLite 模式下必须纳入，PostgreSQL 和 MySQL 默认不纳入。

同步内容包括：

- 集群成员列表
- leader 选举结果
- 项目级配置快照
- SQLite 主库模式下的 collection 元数据
- SQLite 主库模式下的业务表 DDL 和索引/view 变更
- SQLite 主库模式下的 record 创建、更新、删除
- SQLite 主库模式下的迁移版本与系统表状态
- 向量索引全表内容
- 内存缓存的可同步状态
- 任务分配与重试状态

同步向量表时应按分片、分页或日志追加方式传输，避免单条大事务阻塞复制链路。

### 5.1 SQLite 主业务库复制边界

SQLite 集群模式下，每个节点都有自己的本地 SQLite 文件。如果只同步 vector/cache，不同步主业务库，节点之间会出现 collection 元数据、物理表结构、索引、视图和 record 数据漂移。

因此以下操作必须通过 Raft leader 提交，并按 log index 在所有节点顺序 replay：

- collection create/update/delete/import
- `SyncRecordTableSchema` 触发的 create table、rename table、add/drop/rename column
- collection indexes 创建和删除
- view collection 的 create/drop/recreate
- record create/update/delete
- admin create/update/delete
- settings/params 中会影响运行时行为的配置变更
- migrations 的应用状态
- vector task enqueue/dequeue、vector entry upsert/delete
- cache invalidation 事件

SQLite 模式下必须禁止 follower 本地写入。follower 收到写请求时只能转发给 leader；如果转发失败，应返回错误，不能本地 fallback apply。

### 5.2 单一有序日志

SQLite 主业务库、vector、embedding 队列和缓存失效事件必须共享同一个 Raft log。

原因是这些状态之间存在强顺序关系：

- collection 创建必须先于该 collection 的 record 写入。
- schema 更新必须先于依赖新字段的 record 写入。
- record 写入必须先于对应 embedding task 入队。
- record 写入或 schema 更新必须触发对应 collection 的 cache invalidation。

如果拆成多条复制链路，就会出现 follower 上先收到 record insert，后收到 create table，或者先生成 embedding，后应用 record 更新的问题。

### 5.3 状态机操作类型

Raft log 不应只承载 vector operation。建议引入更通用的 replicated operation：

- `schema.collection_upsert`
- `schema.collection_delete`
- `schema.collections_import`
- `record.create`
- `record.update`
- `record.delete`
- `admin.upsert`
- `admin.delete`
- `settings.update`
- `migration.apply`
- `vector.task_enqueue`
- `vector.task_dequeue`
- `vector.entry_upsert`
- `vector.entry_delete`
- `cache.invalidate`

SQLite schema 变更优先复制高层确定性操作，而不是让 follower 自己重新推导一次随机中间 DDL。leader 应在提交前生成所有必要的 id、时间戳、字段快照和操作参数，follower replay 时不得再生成新的随机值。

### 5.4 与现有表同步路径的关系

当前 collection 保存路径会持久化 `_collections` 元数据，并调用 `SyncRecordTableSchema` 同步真实业务表结构。

SQLite 集群模式下，这条路径必须运行在 Raft apply 阶段：

1. API、MCP 或 Agent 接收到写请求。
2. 非 leader 节点转发给 leader。
3. leader 完成权限校验、表单校验和操作归一化。
4. leader 生成 replicated operation 并提交 Raft。
5. 每个节点在 apply 阶段调用同一套 DAO/表同步逻辑。
6. apply 成功后触发 vector task 和 cache invalidation 的后续 operation。

这样可以继续复用现有业务内核，同时保证 SQLite 多节点的表结构和数据顺序一致。

### 5.5 快照与恢复

SQLite 主业务库纳入 Raft 后，快照不能只保存 vector manager 状态。

SQLite 模式下的节点快照至少包含：

- SQLite 主库文件的一致性快照
- 当前 applied log index / term
- vector runtime 状态
- embedding task 队列
- vector entries
- 可恢复的缓存元数据或缓存失效水位

新节点加入集群时必须先安装快照，再从快照对应的 log index 之后继续追日志。

### 5.6 实施阶段

分阶段实施，避免一次性改动过大：

1. 第一阶段：建立通用 replicated operation envelope，保留旧 vector operation 兼容；SQLite strict operation 不允许 follower 本地 fallback。
2. 第二阶段：接入 collection create/update/delete/import，使 collection 元数据和 `SyncRecordTableSchema` 触发的表结构变更进入同一个复制入口。
3. 第三阶段：接入 record create/update/delete，并处理文件上传、hooks、权限校验、embedding task 和 cache invalidation 的顺序关系。当前已完成管理界面普通 record CRUD 的复制接入，其他直连 `SaveRecord` 路径仍需继续收敛。
4. 第四阶段：接入 settings/params/migrations，确保影响运行时行为的系统状态一致。
5. 第五阶段：实现 SQLite 一致性快照安装和新节点 join 流程。

## 6. 向量存储

### 6.1 后端

使用 `sqlite-vec`（modernc）作为本地向量存储后端。

### 6.2 数据模型

至少包含：

- project_id
- source_type
- source_id
- source_field
- embedding_model
- vector
- content_hash
- created
- updated

### 6.3 触发点

embedding 自动触发于：

- record 创建
- record 更新
- 批量导入
- 文件入库后的结构化写入

## 7. embedding 策略

### 7.1 模型

配置的 `embedded` LLM 模型负责生成 embedding。

### 7.2 执行

1. 主服务接收写入。
2. 记录待 embedding 任务。
3. 单实例模式下本地执行。
4. 多实例模式下由 Raft 协调节点执行。
5. 结果写入本地 `sqlite-vec`。

## 8. 内存缓存同步

### 8.1 单实例

没有 Redis 时，单实例只使用本地内存缓存。

### 8.2 多实例

没有 Redis 时，多实例需要通过 Raft 同步必要的内存缓存状态。

建议同步项：

- 项目级配置快照
- 热 collection 元信息
- 向量索引热状态
- 会话短期状态
- 查询热缓存摘要

这不是完整分布式缓存，而是保证多实例行为一致的最小同步集。

## 9. 集群关系

### 9.1 节点模型

所有 `pb serve` 实例都是同级节点。

- 单实例时：无 leader/follower 概念
- 多实例时：Raft 产生 leader/follower

### 9.2 任务路由

路由按以下优先级：

1. `project_id`
2. `collection_id`
3. `embedding_model`
4. 节点负载

### 9.3 故障恢复

- leader 挂掉后自动选主
- 节点重启后自动恢复成员身份
- 向量索引可重建
- 缓存状态按 Raft 元数据恢复

## 10. 与 Agent 的关系

Agent 负责：

- 项目边界
- tool 调用
- 写操作授权
- 查询结果展示

向量内嵌方案负责：

- embedding
- 向量检索
- 多实例一致性
- 无 Redis 时的缓存同步

两者共享同一权限与项目边界模型。

## 11. 后台监控面板

后台管理需要提供统一的状态面板，按运行模式自动切换。

### 11.1 单机视图

单实例时展示基础监控信息：

- 服务在线状态
- CPU / 内存 / 磁盘
- 请求数 / 延迟 / 错误率
- 缓存命中率
- 向量索引数量
- embedding 队列长度
- 最近任务结果

### 11.2 集群视图

多实例 Raft 模式下展示集群内信息：

- 节点列表
- leader / follower 状态
- Raft 任期
- 集群成员变更记录
- 节点健康状态
- 节点负载
- 向量任务分配情况
- 缓存同步状态
- 项目分片分布

### 11.3 交互要求

- 单机模式不显示集群无关字段
- 集群模式默认展开 leader 和节点健康信息
- 所有指标按 project_id 支持下钻
- 状态面板与后台配置页共存，但独立于配置编辑区
