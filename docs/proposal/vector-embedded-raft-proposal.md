# 向量能力内嵌与 Raft 同步方案

> 状态: Draft
> 目标: 将向量能力直接内嵌到 `pb serve`，单实例正常运行，多实例时通过 Raft 同步向量元数据和无 Redis 场景下的内存缓存。

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
9. PostgreSQL、MySQL、SQLite 都支持向量与缓存的 Raft 同步；其中 SQLite 作为主数据库时，也支持把主业务库纳入 Raft 复制范围。

## 3. 非目标

- 不再引入独立 `pb vec` 命令。
- 不把向量检索做成单独服务。
- 不把 Redis 作为集群唯一依赖。

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
- 协调 embedding 任务和检索路由

## 5. Raft 负责什么

Raft 负责向量数据库的全表内容同步，把向量库本身当成需要复制的状态机状态。

这条复制链路对所有数据库模式都启用，但作用范围不同：

- SQLite：同步向量库全表内容、可同步缓存状态，以及主业务库状态
- PostgreSQL：同步向量层状态与可同步缓存状态
- MySQL：同步向量层状态与可同步缓存状态

主业务库是否纳入 Raft，取决于当前主数据库类型。SQLite 模式下纳入，PostgreSQL 和 MySQL 不纳入。

同步内容包括：

- 集群成员列表
- leader 选举结果
- 项目级配置快照
- 向量索引全表内容
- 内存缓存的可同步状态
- 任务分配与重试状态

同步向量表时应按分片、分页或日志追加方式传输，避免单条大事务阻塞复制链路。

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
