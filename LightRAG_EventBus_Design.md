# LightRAG 扩展架构演进设计：基于 gRPC 的事件总线 (Event Bus) 模式

## 1. 架构背景与痛点
目前 LightRAG 的 RAG 流水线（如扩词、向量召回、图谱召回、重排等）主要采用 Python 函数硬编码调用。这种方式在引入外部异构计算能力（如使用 Go/Rust 开发的同义词服务、重排服务）时，存在以下痛点：
- **代码侵入性高**：需要修改核心代码或配置表来适配外部服务。
- **主服务认知负担重**：主服务（Core Engine）需要感知并管理所有外部扩展服务（知道它们的地址、处理它们的超时与容灾）。
- **扩展性瓶颈**：难以做到服务的“热插拔”与真正的物理隔离。

## 2. 核心设计理念：完全订阅式 (Pub/Sub) 架构
为实现“零代码修改”与“能力持续扩展”，我们决定将系统改造为 **基于事件总线 (Event Bus) 的微服务架构**。
核心思想：**主服务对扩展服务绝对无知 (Absolute Ignorance)**。主服务只负责在特定生命周期抛出事件（如：“我要扩词”），而不关心是谁、有多少个服务在处理这个事件。

### 2.1 架构图

```text
                               ┌───────────────────────┐
                               │ LightRAG Core Engine  │
                               │ (只保留最纯粹的编排逻辑) │
                               └──────────┬────────────┘
                                          │ 1. 广播事件: `rag.stage.query_expansion`
                                          │    Payload: {query: "AI"}
                                          ▼
 ┌────────────────────────────────────────────────────────────────────────────┐
 │                            gRPC Event Bus (事件总线)                       │
 │  (维护着动态注册表：监听外部服务的心跳与注册状态)                              │
 └──────────┬─────────────────────────────┬──────────────────────────┬────────┘
            │ 2. 并发分发                 │                          │
            ▼                             ▼                          ▼
 ┌────────────────────┐        ┌────────────────────┐     ┌────────────────────┐
 │ LightRAG 默认实现    │        │  同义词服务 (Go)     │     │ 拼写纠错服务 (Rust)  │
 │ (伪装为本地订阅者)    │        │ (主动注册、订阅该事件)│     │ (主动注册、订阅该事件)│
 └──────────┬─────────┘        └──────────┬─────────┘     └──────────┬─────────┘
            │ 3. 返回原生结果             │ 3. 返回 ["人工智能"]       │ 3. 没纠错，返回空
            ▼                             ▼                          ▼
 ┌────────────────────────────────────────────────────────────────────────────┐
 │                            gRPC Event Bus (事件总线)                       │
 │  (收集所有订阅者的结果，根据策略去重、合并)                                   │
 └──────────┬─────────────────────────────────────────────────────────────────┘
            │ 4. 返回合并后的上下文
            ▼
 ┌───────────────────────┐
 │ LightRAG Core Engine  │
 │ (收到结果，进入下一阶段) │
 └───────────────────────┘
```

## 3. 核心机制：订阅者驱动的智能合并 (Subscriber-driven Merge)

为了确保 Event Bus 的“绝对纯粹”，**Event Bus 不应该包含任何针对特定 RAG 阶段的硬编码合并逻辑（如扩词用并集，分块用替换等）**。所有的控制权必须下放给扩展服务本身。

### 3.1 生产级 Protobuf 契约设计

这份契约设计具备了高并发流水线所需的流式处理、熔断降级、和背压调度能力。

```protobuf
syntax = "proto3";
package lightrag.eventbus.v1;

// 1. 服务注册与服务发现 (控制平面)
message RegisterRequest {
    string subscriber_id = 1;           // 订阅者唯一ID (如: "rust-semantic-chunker-node1")
    string topic = 2;                   // 订阅的阶段 (如: "rag.insert.chunking")
    repeated string capabilities = 3;   // 细粒度能力声明 (如: ["markdown", "pdf"])
    int32 max_concurrency = 4;          // 声明最大并发处理能力 (用于背压)
    int32 expected_latency_ms = 5;      // 预期处理延迟 (用于总线调度参考)
}

message RegisterResponse {
    bool success = 1;
    string message = 2;
}

// 2. 核心数据传输载体 (数据平面)
message EventEnvelope {
    string topic = 1;                   // 路由 Topic
    string correlation_id = 2;          // 全局唯一的请求标识 (匹配 Request/Reply)
    string trace_id = 3;                // OpenTelemetry 追踪 ID
    int64 deadline_timestamp = 4;       // 绝对截止时间戳 (Unix Epoch)，用于精确超时控制
    int32 priority = 5;                 // 优先级 (0: Normal, 1: High)
    string source_service = 6;          // 发起者标识 (如: "lightrag-core-api")
    
    map<string, bytes> inputs = 10;     // 业务上下文负载
    map<string, string> metadata = 11;  // 附加元数据
}

message SubscriberReply {
    string correlation_id = 1;          // 必须与请求中的 ID 对应
    string subscriber_id = 2;           // 回包者标识
    
    map<string, bytes> outputs = 10;    // 处理后的增量结果
    
    // ✨ 灵魂设计：订阅者主动决定合并策略
    enum MergeStrategy {
        APPEND = 0;    // 追加 (适合扩词、抽取实体)
        REPLACE = 1;   // 替换/覆盖 (适合更高优的算法，如专有分块算法覆盖默认算法)
        IGNORE = 2;    // 旁路/忽略 (适合审计、敏感词拦截)
    }
    MergeStrategy strategy = 11;
    int32 weight = 12;                  // 冲突解决权重
    bool partial_result = 13;           // 流式标识: 是否只是部分结果 (支持渐进式返回)
    
    // 运维与可观测性
    int32 latency_ms = 20;              // 订阅者内部处理耗时
    string error_code = 21;             // 标准错误码 (成功为空，如 RATE_LIMIT)
    string error_message = 22;          // 错误详情
    map<string, bytes> metadata = 23;   // 订阅者回传元数据 (如 Token 消耗)
}

// 3. gRPC 接口定义
service EventBus {
    // 订阅者启动时注册
    rpc RegisterSubscriber(RegisterRequest) returns (RegisterResponse);
    
    // 双向流通信：订阅者连上后，总线通过流下发 EventEnvelope，订阅者处理完通过流推回 SubscriberReply
    rpc SubscribeStream(stream SubscriberReply) returns (stream EventEnvelope);
    
    // 主服务(LightRAG)调用接口：发布事件并等待合并结果
    rpc PublishAndWait(EventEnvelope) returns (SubscriberReply);
}
```

### 3.2 运行机制推演 (以文档分块为例)

1. **主服务发广播**：LightRAG 核心引擎发布 `rag.insert.chunking` 事件。
2. **总线盲发**：Event Bus 将文本并发推给 A（默认的 Python Token 切分器）和 B（外部用 Rust 写的语义切分器）。
3. **订阅者表态**：
   - A 返回切分结果，并声明：`strategy=APPEND, weight=10`。
   - B 返回切分结果，并强势声明：`strategy=REPLACE, weight=100`。
4. **总线机械式合并**：Event Bus 不懂业务，只遵循元数据协议。它看到 B 的 `REPLACE` 且权重更高，直接丢弃 A 的结果，只将 B 的结果返回给主服务。

这种设计让 Event Bus 成为一个纯粹的、普适的中间件，甚至可以复用于其他非 RAG 场景，实现了终极的架构解耦。

## 5. 落地与演进路线图 (The Execution Plan)

我们采取 **"The Strangler Fig Pattern" (绞杀者模式)** 的变种，分三阶段平滑演进：

### Phase 1: 契约优先 (API First)
- 敲定并冻结 `lightrag-eventbus.proto` 设计。
- 在 LightRAG Python 项目中生成 gRPC Stub。

### Phase 2: Go 快速验证 (MVP)
- **目标**：以极低的心智负担验证双向流、Scatter-Gather 并发调度及合并策略。
- **动作**：用 Go 实现内存级的订阅路由表和超时控制，跑通 `Python -> Go Event Bus -> Dummy Subscriber -> Python` 的闭环。

### Phase 3: Rust 生产化固化
- **目标**：追求极致性能，提供无 GC 抖动的极低尾延迟 (P99)。
- **动作**：在契约稳定的基础上，使用 Rust (`tokio` + `tonic`) 重写 Event Bus。利用无锁数据结构优化高并发调度，无缝替换 Go 版本二进制文件，正式投入生产。

## 4. 方案优势总结
1. **彻底解耦**：新增能力（如同义词、术语过滤）只需要开发新服务并启动，无需修改 LightRAG 任何代码或配置。
2. **极佳的可扩展性与异构支持**：允许团队用最合适的语言（Go 处理高并发，Rust 处理重计算）扩展 RAG 流水线。
3. **平滑演进与向后兼容**：通过“本地订阅者”机制，完美兼容现有的纯 Python 用户群体，实现优雅降级。