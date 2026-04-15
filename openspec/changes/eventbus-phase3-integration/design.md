# Design: eventbus-phase3-integration

## 架构：双引擎与本地短路模型

在 Phase 3 中，为了达成在 `lightrag.py` / `operate.py` 中的平滑替换，我们设计一套基于抽象接口的 `HookDispatcher`，该引擎将具备两种运行模式：

1. **GrpcEventBusDispatcher**：如果用户配置了外部的 Go Event Bus 地址，LightRAG 将其实例化，执行网络上的 `PublishAndWait`。
2. **LocalMemoryDispatcher**（默认保底）：在没有外部微服务时，这套 Dispatcher 在单机进程内模拟 Publish-Subscribe 模型，直接调用挂载在内存中的 Python 原生函数，延迟为 0。

## 适配层设计

### 1. `LocalSubscriberAdapter`
我们要将现有的 `lightrag/operate.py` 里的原生函数，包裹为一个标准的订阅者。
- **职责**：将原生函数的调用结果（如抽取的 Entities），组装成符合 `SubscriberReply` 规范的对象，赋予 `MergeStrategy.APPEND` 及对应的 `weight`。
- **设计示例**：
```python
class NativeChunkingSubscriber(LocalSubscriberAdapter):
    def __init__(self, topic="rag.insert.chunking"):
        self.topic = topic
        self.strategy = MergeStrategy.APPEND

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        # 调用 lightrag 原生的分块函数
        chunks = await chunking_by_token_size(envelope.inputs["raw_text"])
        # 返回标准的 Reply，用于本地短路或通过 gRPC 发出
        return SubscriberReply(outputs={"chunks": serialize(chunks)}, strategy=self.strategy)
```

### 2. `HookDispatcher` 与打桩位置

在 LightRAG 的主干中（如 `lightrag.py` 和 `operate.py`），我们将原来的直接调用替换为通过 `Dispatcher` 抛出事件：

**重构前（原样）**：
```python
# operate.py
chunks = await chunking_by_token_size(doc_content)
entities = await extract_entities(chunks, llm_func)
```

**重构后（打桩）**：
```python
# operate.py
# 主服务对到底谁来执行“分块”绝对无知。
envelope = EventEnvelope(topic="rag.insert.chunking", inputs={"raw_text": doc_content})
reply = await dispatcher.publish_and_wait(envelope)

chunks = deserialize(reply.outputs["chunks"])

# 提取实体阶段同理，喊一嗓子
envelope_ext = EventEnvelope(topic="rag.insert.entity_extraction", inputs={"chunks": serialize(chunks)})
reply_ext = await dispatcher.publish_and_wait(envelope_ext)
```

## 生命周期与拓扑注册

当 `LightRAG` 实例被创建时（`__init__` 阶段）：
1. 读取配置（比如是否有 `EVENT_BUS_URL`）。
2. 初始化 `HookDispatcher`。
3. 主动将内置的 `NativeChunkingSubscriber`、`NativeEntityExtractorSubscriber` 注册到本地内存中（如果是 Grpc 模式，则向 Go 总线发起 `RegisterSubscriber` 调用并将自身作为订阅者连接上去）。

## 瘦事件机制与数据序列化

由于 `operate.py` 中的上下文传递往往包含巨大的字典和文本块：
- 在 Phase 3 初期，我们将定义 `serialize/deserialize` 辅助函数将 `dict` 转为 `bytes` 以兼容 `outputs: map<string, bytes>`。
- 当传递长文本且部署为分布式微服务时，将启用 **瘦事件机制**：仅传递 `doc_id`，强制远程订阅者利用内部共享存储读取具体内容。但在 `LocalMemoryDispatcher` 模式下，直接传递内存引用即可，避免不必要的序列化拷贝。