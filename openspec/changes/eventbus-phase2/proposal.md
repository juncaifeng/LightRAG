# Proposal: eventbus-phase2

## 背景 (Context)

在 Phase 1 中，我们已经完成了 `lightrag-eventbus.proto` 契约的设计与冻结，确立了“主服务绝对无知”、“单向流数据分发”、“独立响应回传”以及“订阅者驱动的合并策略”等核心架构原则。同时规划了基于 Go 语言验证 MVP（最小可行性产品），并在未来视生产观测指标（P99 延迟、内存等）决定是否由 Rust 接管底层的平滑演进路径。

Phase 2 的核心目标是：**将纸面上的 Protobuf 契约转化为真正可运行的 Go Event Bus 基础设施，并跑通 LightRAG 的通信闭环。**

## 目标 (Goals)

- 产出一个独立的、基于 Go 实现的轻量级 gRPC Event Bus (Aggregating Broker)
- 实现基于 Protobuf 的核心机制：动态路由表维护、单向流 Subscribe 分发、独立 Respond 接收
- 实现基于超时控制（deadline_timestamp）和 `APPEND`/`REPLACE` 等合并策略的 `PublishAndWait` Scatter-Gather 引擎
- 实现 Python 端的调用适配器（Client），使 LightRAG Core 能够发出测试事件并获取结果
- 实现一个用作验证的 Dummy Subscriber（例如“同义词扩充”假服务）以验证闭环链路
- 构建完整的链路耗时、CPU 和内存的基础可观测埋点，为 Phase 2.5 提供决策依据

## 非目标 (Non-Goals)

- 本阶段**不包含**修改 LightRAG 真实的生产逻辑（例如替换现有的实体抽取或向量检索核心代码），仅在边缘或测试入口打桩验证。
- 本阶段不实现复杂的背压（Backpressure）与自动熔断恢复逻辑，仅实现基础超时与健康状态上报日志。
- 不使用 Rust 编写任何组件。

## 关键原则 (Key Principles)

- **Go 极简哲学**：保持 Event Bus 的轻量化，充分利用 goroutine 和 channel 实现 Scatter-Gather，避免过度引入沉重的第三方消息队列框架。
- **协议严格对齐**：完全遵照 Phase 1 产出的 `lightrag_eventbus.proto` 契约，不允许在 Go 代码中 hardcode 针对特定 RAG 阶段的合并逻辑。
- **可观测性先行**：在处理高并发分发的关键节点，必须留下明确的 trace_id 链路日志与 P50/P99 延迟打点。