# 向量子服务方案

> 状态: Draft
> 目标: 在 PostgreBase 中新增独立的向量子服务 `pb vec`，支持多个实例注册到一个或多个主服务，后端使用 `sqlite-vec`（modernc）实现向量存储与检索。

## 1. 背景

Agent 方案里已经引入了项目边界、工具化数据操作和多模态输入。但如果要做语义检索、相似度搜索、RAG 或自动 embedding，就需要一个独立的向量服务层。

这层不应该混进普通关系型 CRUD，而应该作为单独的运行单元存在。

## 2. 目标

1. 新增 `pb vec` 命令作为向量子服务。
2. 主服务 `pb serve` 负责业务 API、Admin UI、数据库和项目管理。
3. 一个或多个 `pb vec` 实例可注册到一个主服务或多个主服务。
4. 支持 `embedded` LLM 模型自动生成 embedding。
5. 后端向量存储使用 `sqlite-vec`（modernc）实现。
6. 向量服务必须遵守项目边界和权限模型。

## 3. 非目标

- 不把向量能力塞进普通 record CRUD。
- 不把 embedding 写成手工批处理脚本。
- 不要求主服务直接承担全部向量计算。

## 4. 总体架构

### 4.1 进程拆分

#### 主服务

```bash
pb serve --dataDsn "postgres://..." --dir ./pb_data
```

职责：

- 管理项目、collection、record、文件和权限
- 接收向量子服务注册
- 下发待 embedding / 待检索任务
- 聚合向量服务结果

#### 向量子服务

```bash
pb vec --master=pb_server1,pb_server2
pb vec --master=pb_server1
pb vec --master=pb_server1,pb_server2,pb_server3
```

职责：

- 注册到一个或多个主服务
- 拉取项目范围内的 embedding 任务
- 使用本地 `sqlite-vec` 存储向量
- 提供相似度检索结果

### 4.2 多实例模式

允许多个向量子服务同时启动。

设计要求：

- 同一主服务可挂多个 vec worker
- 一个 vec worker 也可同时注册多个 master
- 任务分发要支持负载均衡和失败重试
- 检索请求要支持就近路由或聚合路由

## 5. 向量存储

### 5.1 后端实现

向量存储使用 `sqlite-vec`，并通过 modernc 体系保证无 CGO 构建兼容。

### 5.2 数据模型

向量数据建议至少包含：

- project_id
- source_type
- source_id
- source_field
- embedding_model
- vector
- content_hash
- created
- updated

### 5.3 边界规则

- 向量索引必须绑定 `project_id`
- 默认不跨项目检索
- 删除项目时，向量数据一并清理

## 6. 自动 embedding

### 6.1 embedding 来源

配置的 `embedded` LLM 模型负责生成 embedding。

要求：

- 可在后台配置
- 可按项目、collection、字段选择不同 embedding 模型
- 可回退到默认 embedding 模型

### 6.2 触发时机

embedding 自动触发于：

- record 创建
- record 更新
- 批量导入
- 文件入库后的结构化写入

### 6.3 执行策略

1. 主服务记录待 embedding 任务。
2. 向量子服务领取任务。
3. 子服务调用 `embedded` 模型生成 embedding。
4. 结果写入本地向量库。
5. 回传状态到主服务。

## 7. 注册协议

### 7.1 子服务注册

`pb vec` 启动后向主服务发送注册信息：

- service id
- host / endpoint
- capability
- supported models
- current load
- health status

### 7.2 心跳

子服务需要周期性心跳，主服务据此判断：

- 在线
- 忙碌
- 失联
- 可用容量

### 7.3 任务领取

主服务把 embedding / retrieval 任务下发给可用子服务，子服务执行后返回结果。

## 8. 集群关系设计

### 8.1 基本模型

vec worker 之间不做 peer-to-peer 复制，也不直接组成强一致集群。

采用的是：

- **主控平面**：由 master 维护 worker 注册、健康、租约和任务分发
- **执行平面**：worker 独立运行，处理自己领取到的任务

这样每个 worker 的关系是“同级执行节点”，不是“主从复制节点”。

### 8.2 worker 角色

每个 worker 都是对等的，职责相同：

- 接收注册
- 维持心跳
- 领取任务
- 写本地 `sqlite-vec`
- 返回结果

worker 之间不共享本地向量库，也不直接同步状态。

### 8.3 任务分配

任务分配建议按以下优先级：

1. `project_id` 固定路由到同一 worker，保证局部性
2. worker 忙碌时改派到同一 master 下的其他 worker
3. worker 失联后，任务租约过期自动重试

如果后面要做更强的局部性，可以把 `project_id` 再细分到 `collection_id` 或 `embedding_model` 维度。

### 8.4 一致性策略

不做跨 worker 的同步写一致性，避免把向量层做成数据库复制系统。

一致性由主服务和租约保证：

- 同一个任务只允许一个 worker 持有 lease
- lease 到期可重新领取
- 失败任务可重试
- 重试必须幂等，靠 `content_hash` / `source_id` 去重

### 8.5 故障与恢复

- worker 重启后重建本地 `sqlite-vec` 索引
- master 重启后 worker 自动重注册
- 某个 worker 宕机，不影响其他 worker
- 某个项目的向量数据如果只落在单 worker 上，恢复依赖重建任务，而不是复制副本

## 9. API 与 Tool 接口

### 8.1 业务侧接口

向量能力对外不暴露底层 SQL，而是通过 service/tool 调用：

- `vector.index`
- `vector.upsert`
- `vector.query`
- `vector.delete`
- `vector.rebuild`

### 8.2 检索返回

检索结果应返回：

- 命中内容
- 相似度分数
- 来源对象
- 项目 ID
- 可选的摘要片段

## 10. 配置方式

### 9.1 主服务配置

主服务只保留向量服务的接入配置，不直接保存向量计算逻辑。

### 9.2 子服务配置

`pb vec` 需要配置：

- master 列表
- worker id
- 本地存储目录
- embedding 模型引用
- 并发数
- 失败重试策略

## 11. 运维与扩展

### 10.1 多 master

一个 vec worker 可以同时注册多个 master，但必须有明确的优先级和隔离策略，避免串项目。

### 10.2 失败恢复

- master 重启后 worker 自动重注册
- worker 重启后恢复本地向量库
- 任务中断后可重试

### 10.3 可观测性

建议记录：

- 任务数
- embedding 耗时
- 检索耗时
- 向量库大小
- worker 在线状态

## 12. 与 Agent 方案的关系

Agent 方案负责：

- 交互
- 工具调用
- 项目边界
- 写操作授权

向量子服务方案负责：

- embedding
- 向量存储
- 语义检索
- 多实例分发

两者共享同一项目边界和权限模型，但运行职责分离。
