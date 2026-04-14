# Tasks: eventbus-phase2

## 交付物

- Go 版本的轻量级 gRPC Event Bus 二进制产物及源代码
- 一个用于验证的 Dummy Subscriber（Go 或 Python）
- LightRAG Core 中的 Python gRPC Client 适配层代码

## 任务清单

- [x] 1. 生成并验证 Protobuf 代码
  - [x] 1.1 使用 `protoc` 与 `protoc-gen-go-grpc` 为 Go 服务端生成桩代码
  - [x] 1.2 使用 `grpcio-tools` 为 Python 客户端生成存根
- [x] 2. 实现 Go Event Bus 核心机制
  - [x] 2.1 实现 `RegisterSubscriber`，管理内存并发安全的路由表 (`map[string][]chan<- *EventEnvelope`)
  - [x] 2.2 实现 `Subscribe`，建立单向 Server-streaming，监听分发通道并写入下游
  - [x] 2.3 实现 `Respond` 接口，接收异步结果并基于 `correlation_id` 将 `SubscriberReply` 发回给对应的 Gather 任务
- [x] 3. 实现 PublishAndWait Scatter-Gather 引擎
  - [x] 3.1 基于 `deadline_timestamp` 构建 `context.WithDeadline`
  - [x] 3.2 开启 goroutine fan-out 到所有匹配 Topic 的订阅者流中
  - [x] 3.3 实现阻塞等待 (`select`) 与机械合并算法（依据 `APPEND`, `REPLACE`, `weight` 进行结果聚合）
- [x] 4. 建立验证闭环
  - [x] 4.1 开发一个独立的 Go/Python Dummy Subscriber（例如模拟同义词扩充，注册到 `rag.query.query_expansion`，返回 `APPEND` 策略）
  - [x] 4.2 在 LightRAG 项目中编写简单的测试脚本（Client），发送 `EventEnvelope` 并打印出聚合结果
- [x] 5. 补充可观测性
  - [x] 5.1 增加 `PublishAndWait` 请求的 P50/P99 耗时计算与基础日志打印
  - [x] 5.2 开启 pprof 的 HTTP 端口供压测时分析内存和 CPU