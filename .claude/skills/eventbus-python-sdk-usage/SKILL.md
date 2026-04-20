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

## Topic 名 → 一切推导规则

只需要知道 topic 名（如 `index.retriever.retrieve`），即可推导出所有需要的信息：

```
topic: index.retriever.retrieve
       ──── ──────── ───────
       domain pipeline stage
         ↓       ↓       ↓
proto 路径: go-eventbus/proto/topics/{domain}/{pipeline}.proto
           → proto/topics/index/retriever.proto

Python import: from topics.{domain}.{pipeline}_pb2 import {PascalCase}Input, {PascalCase}Output
           → from topics.index.retriever_pb2 import RetrieveInput, RetrieveOutput
```

**完整映射表：**

| Topic | Proto 文件 | Python import |
|-------|-----------|---------------|
| `rag.insert.chunking` | `proto/topics/rag/insert.proto` | `from topics.rag.insert_pb2 import ChunkingInput` |
| `rag.insert.embedding` | `proto/topics/rag/insert.proto` | `from topics.rag.insert_pb2 import EmbeddingInput` |
| `rag.insert.ocr` | `proto/topics/rag/insert.proto` | `from topics.rag.insert_pb2 import OcrInput` |
| `rag.query.keyword_extraction` | `proto/topics/rag/query.proto` | `from topics.rag.query_pb2 import KeywordExtractionInput` |
| `rag.query.query_expansion` | `proto/topics/rag/query.proto` | `from topics.rag.query_pb2 import QueryExpansionInput` |
| `rag.query.vector_search` | `proto/topics/rag/query.proto` | `from topics.rag.query_pb2 import VectorSearchInput` |
| `rag.query.kg_search` | `proto/topics/rag/query.proto` | `from topics.rag.query_pb2 import KgSearchInput` |
| `rag.query.rerank` | `proto/topics/rag/query.proto` | `from topics.rag.query_pb2 import RerankInput` |
| `rag.query.response` | `proto/topics/rag/query.proto` | `from topics.rag.query_pb2 import ResponseInput` |
| `index.builder.index_build` | `proto/topics/index/builder.proto` | `from topics.index.builder_pb2 import IndexBuildInput` |
| `index.retriever.retrieve` | `proto/topics/index/retriever.proto` | `from topics.index.retriever_pb2 import RetrieveInput` |
| `llm.completion.complete` | `proto/topics/llm/completion.proto` | `from topics.llm.completion_pb2 import CompleteInput` |
| `kg.merge.entity` | `proto/topics/kg/merge.proto` | `from topics.kg.merge_pb2 import EntityMergeInput` |
| `kg.merge.relation` | `proto/topics/kg/merge.proto` | `from topics.kg.merge_pb2 import RelationMergeInput` |

**Python 推导公式：**
- import 模块 = `topics.{domain}.{pipeline}_pb2`
- 类型名 = `PascalCase(stage)` + `Input`/`Output`

**注意**：Python 的 `topics` 包保留了子目录结构（`topics.rag.*` 和 `topics.index.*`），而 Go 是扁平的（`topics.*`）。

## SDK 获取

### 1. Topic 数据模型（protobuf 生成）

```bash
cd go-eventbus
uv add --editable sdk/v1/python
```

```python
from lightrag_eventbus_pb2 import EventEnvelope, SubscriberReply, RegisterRequest

# 按推导规则导入 — topic = index.retriever.retrieve
from topics.index.retriever_pb2 import RetrieveInput, RetrieveOutput

# 或 topic = rag.insert.embedding
from topics.rag.insert_pb2 import EmbeddingInput, EmbeddingOutput
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

class MyRetrieverSubscriber(LocalSubscriberAdapter):
    def __init__(self, retriever_func):
        super().__init__(
            topic="index.retriever.retrieve",  # topic 名
            subscriber_id="my-retriever",
            strategy=MergeStrategy.APPEND,
            weight=10,
        )
        self.retriever_func = retriever_func

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        # 1. 从 envelope.inputs 获取输入（按 proto 字段名）
        index_name = envelope.inputs.get("index_name", "")
        query = envelope.inputs.get("query", "")
        top_k = envelope.inputs.get("top_k", 20)
        score_threshold = envelope.inputs.get("score_threshold", 0.0)

        # 2. 业务逻辑
        results, total_hits = await self.retriever_func(
            index_name=index_name, query=query, top_k=top_k,
            score_threshold=score_threshold,
        )

        # 3. 返回结果
        return SubscriberReply(
            outputs={"results": results, "total_hits": total_hits, "index_name": index_name},
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
    "index.retriever.retrieve",
    MyRetrieverSubscriber(my_retriever_func),
)
```

## 发布事件

```python
from lightrag.hooks.base import EventEnvelope

envelope = EventEnvelope(
    topic="index.retriever.retrieve",
    inputs={
        "index_name": "thesaurus",
        "query": "deep learning",
        "top_k": 20,
        "semantic_ratio": 0.5,
        "score_threshold": 0.65,
    },
    source_service="query-expansion",
)

result = await dispatcher.publish_and_wait(envelope)

# result.outputs = {"results": [...], "total_hits": 42, "index_name": "thesaurus"}
```

## 查看 Topic 字段

方式一：看 proto 源文件（注释最全）
```
proto/topics/index/retriever.proto
```

方式二：导入生成的 protobuf 类型查看字段描述
```python
from topics.index.retriever_pb2 import RetrieveInput
print(RetrieveInput.DESCRIPTOR.fields_by_name.keys())
# dict_keys(['index_name', 'query', 'top_k', 'semantic_ratio', 'query_vector', 'embedder', 'score_threshold', 'filter_expr'])
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

## 合并策略

| Strategy | 值 | 行为 | 适用 |
|----------|---|------|------|
| `MergeStrategy.FIRST` | 0 | 首个到达生效 | 嵌入、分块、OCR、索引构建 |
| `MergeStrategy.APPEND` | — | 追加所有结果 | 向量搜索、KG 搜索、查询扩展、检索 |
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

**Q: 我只知道 topic 名，怎么找到对应的 Python 类型？**
用推导规则：`index.retriever.retrieve` → `from topics.index.retriever_pb2 import RetrieveInput, RetrieveOutput`

**Q: Python 和 Go 的 import 路径为什么不同？**
Python 保留了 proto 子目录结构（`topics.rag.*` / `topics.index.*`），Go 是扁平包（`topics.*`）。对业务开发者来说，Python 的按目录分包更清晰。

**Q: 怎么知道 topic 有哪些字段？**
方式一：读 `proto/topics/{domain}/{pipeline}.proto`（注释最全）。
方式二：`RetrieveInput.DESCRIPTOR.fields_by_name.keys()` 查看所有字段名。

**Q: `envelope.inputs` 的 key 从哪来？**
由发布方决定，对应 Topic Input 的字段名。例如 `RetrieveInput` 的 `query` 字段，key 就是 `"query"`。
