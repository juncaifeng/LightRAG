---
name: eventbus-python-sdk-usage
description: >
  Guide business developers to use the LightRAG EventBus Python SDK.
  Use this skill when: writing a Python subscriber, publishing events,
  registering local subscribers, or integrating with the EventBus gRPC API.
  Covers: local subscriber adapter, gRPC dispatcher, topic data models,
  and the envelope/reply contract.
---

# LightRAG EventBus — Python SDK 使用指南

## SDK 获取

### 1. Topic 数据模型（protobuf 生成）

```bash
cd go-eventbus
uv add --editable sdk/v1/python
```

```python
from lightrag_eventbus_pb2 import EventEnvelope, SubscriberReply, RegisterRequest
from topics.insert_pb2 import ChunkingInput, EmbeddingInput, OcrInput
from topics.query_pb2 import KeywordExtractionInput, VectorSearchInput, ...
```

### 2. 平台通信层（本地适配器）

位于 `lightrag/hooks/`，是 LightRAG 核心的一部分：

```python
from lightrag.hooks.base import (
    HookDispatcher,
    LocalSubscriberAdapter,
    EventEnvelope,
    SubscriberReply,
    MergeStrategy,
)
from lightrag.hooks import GrpcEventBusDispatcher
```

## 架构

```
Python 服务                          EventBus 平台（Go）
    │                                    │
    │  本地模式：register_local_subscriber │→ 直接在进程内处理
    │                                    │
    │  gRPC 模式：                        │
    │  publish_and_wait ────────────────→│→ 散射给所有订阅者
    │←───────────────────────────────────│← 聚合结果返回
    │                                    │
```

**两种使用模式**：
- **本地订阅者**：在 Python 进程内直接处理，零网络开销
- **gRPC 订阅者**：独立进程，通过 gRPC 连接 EventBus 平台

## 写一个本地订阅者

继承 `LocalSubscriberAdapter`，实现 `process` 方法：

```python
from lightrag.hooks.base import (
    LocalSubscriberAdapter,
    EventEnvelope,
    SubscriberReply,
    MergeStrategy,
)

class MyEmbeddingSubscriber(LocalSubscriberAdapter):
    def __init__(self, embedding_func):
        super().__init__(
            topic="rag.insert.embedding",
            subscriber_id="my-embedder",
            strategy=MergeStrategy.FIRST,
            weight=10,
        )
        self.embedding_func = embedding_func

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        # 1. 从 envelope.inputs 获取输入
        texts = envelope.inputs.get("texts", [])

        # 2. 业务逻辑
        embeddings = await self.embedding_func(texts)

        # 3. 返回结果
        return SubscriberReply(
            outputs={"embeddings": embeddings},
            strategy=self.strategy,
            weight=self.weight,
            correlation_id=envelope.correlation_id,
        )
```

### 注册订阅者

```python
from lightrag.hooks import GrpcEventBusDispatcher

dispatcher = GrpcEventBusDispatcher(target="localhost:50051")

# 注册本地订阅者
dispatcher.register_local_subscriber(
    "rag.insert.embedding",
    MyEmbeddingSubscriber(my_embedding_func),
)
```

## 发布事件

```python
from lightrag.hooks.base import EventEnvelope

envelope = EventEnvelope(
    topic="rag.insert.embedding",
    inputs={"texts": ["hello", "world"]},
    source_service="lightrag-core",
)

result = await dispatcher.publish_and_wait(envelope)

# result.outputs = {"embeddings": [[0.1, 0.2, ...], [0.3, 0.4, ...]]}
# result.strategy, result.weight, result.latency_ms
```

## 数据约定

Python 侧的 `envelope.inputs` 和 `reply.outputs` 是**普通 Python dict**，不需要手动 protobuf 序列化。

| 操作 | 方式 |
|------|------|
| 获取输入 | `envelope.inputs.get("field_name")` |
| 返回输出 | `SubscriberReply(outputs={"field_name": value})` |
| 传递列表 | 直接传 Python list |
| 传递复杂对象 | dict 或 JSON 可序列化的对象 |

gRPC 模式下，dict 自动转为 JSON bytes 编码传输。

## Topic 数据模型

Topic 的输入输出定义在 protobuf 中，Python 侧查看方式：

```python
# 查看 Topic Input 字段 — 导入生成的 protobuf 类型
from topics.insert_pb2 import ChunkingInput, EmbeddingInput, OcrInput
from topics.query_pb2 import (
    KeywordExtractionInput, KeywordExtractionOutput,
    QueryExpansionInput, QueryExpansionOutput,
    VectorSearchInput, VectorSearchOutput,
    KgSearchInput, KgSearchOutput,
    RerankInput, RerankOutput,
    ResponseInput, ResponseOutput,
)

# 字段定义在 proto 文件中
# go-eventbus/proto/topics/insert.proto
# go-eventbus/proto/topics/query.proto
```

## Topic 速查

### Insert

| Topic | Input | Output |
|-------|-------|--------|
| `rag.insert.chunking` | `ChunkingInput` | `ChunkingOutput` |
| `rag.insert.embedding` | `EmbeddingInput` | `EmbeddingOutput` |
| `rag.insert.ocr` | `OcrInput` | `OcrOutput` |

### Query

| Topic | Input | Output |
|-------|-------|--------|
| `rag.query.keyword_extraction` | `KeywordExtractionInput` | `KeywordExtractionOutput` |
| `rag.query.query_expansion` | `QueryExpansionInput` | `QueryExpansionOutput` |
| `rag.query.vector_search` | `VectorSearchInput` | `VectorSearchOutput` |
| `rag.query.kg_search` | `KgSearchInput` | `KgSearchOutput` |
| `rag.query.rerank` | `RerankInput` | `RerankOutput` |
| `rag.query.response` | `ResponseInput` | `ResponseOutput` |

## 合并策略

| Strategy | 值 | 行为 | 适用 |
|----------|---|------|------|
| `MergeStrategy.FIRST` | 0 | 首个到达生效 | 嵌入、分块、OCR |
| `MergeStrategy.REPLACE` | 1 | 权重高覆盖低 | 重排序 |
| `MergeStrategy.IGNORE` | 2 | 旁路不影响主流程 | 审计、敏感词检测 |

> 注意：Python 侧用 `MergeStrategy.FIRST` 等常量，Go 侧用 `SubscriberReply_FIRST`。

## 已有订阅者参考

`lightrag/hooks/adapters.py` 中已有以下内置订阅者可参考：

| 类名 | Topic | 功能 |
|------|-------|------|
| `NativeChunkingSubscriber` | `rag.insert.chunking` | Token 分块 |
| `NativeEmbeddingSubscriber` | `rag.insert.embedding` | 向量嵌入 |
| `NativeKeywordExtractionSubscriber` | `rag.query.keyword_extraction` | 关键词提取 |
| `NativeQueryExpansionSubscriber` | `rag.query.query_expansion` | 查询扩展（直通） |
| `NativeVectorSearchSubscriber` | `rag.query.vector_search` | 向量检索 |
| `NativeKGSearchSubscriber` | `rag.query.kg_search` | 知识图谱检索 |
| `NativeRerankSubscriber` | `rag.query.rerank` | 重排序（直通） |
| `NativeResponseSubscriber` | `rag.query.response` | LLM 响应生成 |

## 常见问题

**Q: 本地订阅者和 gRPC 订阅者有什么区别？**
本地订阅者在同一进程内，`inputs` 是普通 Python dict。gRPC 订阅者是独立进程，通过 gRPC 通信，dict 自动 JSON 编码。

**Q: 怎么知道 topic 有哪些字段？**
导入 `topicspb`（Go）或 `topics/insert_pb2`（Python）查看生成的数据模型。也可以看 proto 源文件。

**Q: `envelope.inputs` 的 key 从哪来？**
由发布方决定，对应 Topic Input 的字段名。例如 `EmbeddingInput` 的 `texts` 字段，key 就是 `"texts"`。
