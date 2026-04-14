# Design: eventbus-phase2

## 设计概述

Phase 2 旨在通过 Go 语言快速落地基于 `lightrag_eventbus.proto` 契约的独立 Event Bus，验证 **“单向流数据分发”** 与 **“订阅者驱动的智能合并”**。LightRAG 作为发布者发起请求，Go 总线负责动态路由、分发并聚合各语言（Dummy Subscriber）的响应结果，最终实现零硬编码的 RAG 流水线扩展。

## 架构职责与数据流

### 1. Go Event Bus 核心引擎
作为中枢神经，主要实现三个核心模块：
- **控制平面（Registry）**：基于 `RegisterSubscriber`，维护并发安全的内存路由表 `map[topic][]SubscriberInfo`。
- **数据分发平面（Dispatcher）**：通过 `Subscribe` 建立 gRPC 单向 Server-streaming。当收到 `PublishAndWait` 请求时，开启协程通过流向下游广播 `EventEnvelope`。
- **聚合响应平面（Gather & Merge）**：
  - 使用 `Respond` 接收订阅者的 `SubscriberReply`。
  - 使用 Go 的 `select` 与 `context.WithDeadline` 对 `deadline_timestamp` 进行严苛的超时控制。
  - **合并算法**：在规定时间内收齐响应，或超时触发强制结算。遵循订阅者声明的 `MergeStrategy`（`APPEND` 拼接列表, `REPLACE` 根据 `weight` 决断覆盖, `IGNORE` 跳过）。

### 2. Dummy Subscriber (验证示例)
使用 Go 或 Python 实现一个简单的订阅者：
- 启动时调用 `RegisterSubscriber` 声明监听 `rag.query.query_expansion`。
- 连接 `Subscribe` 监听流，收到包含查询的包后，模拟耗时计算，并通过独立的 `Respond` RPC 返回带 `APPEND` 策略的 `SubscriberReply`（如同义词扩充）。

### 3. LightRAG Core Python 适配层
- 通过 `grpcio-tools` 生成 Python Stub。
- 封装 `EventBusClient`，负责组装 `EventEnvelope`，调用 `PublishAndWait`，并从返回的 `SubscriberReply` 中解析合并后的输出（outputs）供 RAG 下游流程使用。

## 核心实现机制 (The Scatter-Gather Pattern)

1. **Scatter (分散发包)**：
   收到 `PublishAndWait` 时，生成一个局部的 `GatherTask`，包含一个用于接收订阅者结果的 `chan *SubscriberReply`。根据路由表向所有匹配 Topic 的存活订阅者的 gRPC 流写入数据。

2. **Wait (超时控制)**：
   基于 `deadline_timestamp` 构建 Go 的 `context.Context`，主协程进入 `select` 阻塞等待，一旦 `ctx.Done()` 触发，立即停止等待剩余订阅者，进入合并阶段。

3. **Gather (结果合并算法)**：
   对已收集到的响应执行机械合并：
   - 维护一个暂存的 `map[string]bytes` 用于存放最终 outputs。
   - 遍历每个响应：
     - 若 `strategy == IGNORE`：直接丢弃。
     - 若 `strategy == REPLACE`：比较当前缓存的最高 weight，若当前响应 weight 更高，则整体覆盖 outputs。
     - 若 `strategy == APPEND`：对 outputs 相同 key 的 bytes 执行追加操作（在 Phase 2，假定 payload 格式为 JSON 数组或预定的 Protobuf repeated bytes 以简化合并逻辑）。

## 可观测性指标收集 (Phase 2.5 前置准备)

为后续生产观测（Phase 2.5）提供决策依据，Go Event Bus 需要内置或通过 Prometheus 输出核心指标：
- **延迟分布**：记录 `PublishAndWait` 端到端的 P50, P90, P99 耗时分布（毫秒）。
- **资源监控**：使用 Go pprof 暴露内存占用与 Goroutine 数量，供压测时分析。
- **健康日志**：当接收到订阅者返回的 `OVERLOADED` 或 `UNHEALTHY` 状态时，必须打印带有 `correlation_id` 与 `subscriber_id` 的显式 Warning 日志。