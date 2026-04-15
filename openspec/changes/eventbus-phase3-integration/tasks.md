# Tasks: eventbus-phase3-integration

## 交付物

- LightRAG 代码库中新增的 `lightrag/hooks` 包目录
- 重构后且经过跑通验证的 `lightrag/operate.py` 核心流程代码
- 一套兼容本地内存与 gRPC 通信的 `EventBusDispatcher` 接口实现

## 任务清单

- [x] 1. 创建基础的 Hook 接口定义
  - [x] 1.1 在 `lightrag/hooks/base.py` 中定义 `HookDispatcher` 协议接口。
  - [x] 1.2 定义 `LocalSubscriberAdapter` 类，包装 Python 函数使其返回符合 `MergeStrategy` 规范的实体对象。
- [x] 2. 实现 Local Memory Dispatcher（本地兜底模式）
  - [x] 2.1 在 `lightrag/hooks/local.py` 中编写基于内存字典和异步调用的事件总线。
  - [x] 2.2 实现内存环境下的智能合并逻辑（支持 APPEND, REPLACE 等）。
- [x] 3. 实现 Grpc Event Bus Dispatcher（微服务模式）
  - [x] 3.1 在 `lightrag/hooks/grpc_bus.py` 中封装在 Phase 2 里写好的测试客户端。
  - [x] 3.2 包装 `PublishAndWait` 方法，处理 gRPC 的连接与 Protobuf 数据结构的打包/解包。
- [x] 4. 将原生函数封装为 Adapter
  - [x] 4.1 在 `operate.py` 或同级模块中，把 `chunking_by_token_size` 包装成 `NativeChunkingSubscriber`。
  - [x] 4.2 把 `extract_entities` 和 `merge_nodes_and_edges` 等封装成原生的 `Subscriber` 并指定其合并策略和 Topic。
- [x] 5. 改造主引擎打桩 (Hook 植入)
  - [x] 5.1 修改 `LightRAG.__init__`，增加根据配置初始化 `Local` 或 `Grpc` Dispatcher 的逻辑，并注册所有原生的 Adapter。
  - [x] 5.2 逐步替换 `operate.py` 中的 `insert` 与 `query` 链路中的硬编码调用，改为调用 `dispatcher.publish_and_wait()`。
- [x] 6. 回归与验证测试
  - [x] 6.1 在不配置 gRPC Event Bus 时，执行完整的 LightRAG Insert & Query 单测，确保行为和之前完全一致（100% 向后兼容）。
  - [x] 6.2 开启外部的 Go Event Bus 与 Dummy 同义词订阅者，跑通一次微服务下的真实 Query 扩词场景。