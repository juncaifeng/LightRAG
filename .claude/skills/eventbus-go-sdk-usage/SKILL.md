---
name: eventbus-go-sdk-usage
description: >
  Guide business developers to use the LightRAG EventBus Go SDK.
  Use this skill when: writing a Go subscriber service, publishing events
  to a topic, integrating with the EventBus gRPC API, or understanding
  how topic data models (Input/Output) connect to the transport layer.
---

# LightRAG EventBus — Go SDK 使用指南

## Topic 名 → 一切推导规则

只需要知道 topic 名（如 `index.retriever.retrieve`），即可推导出所有需要的信息：

```
topic: index.retriever.retrieve
       ──── ──────── ───────
       domain pipeline stage
         ↓       ↓       ↓
proto 路径: go-eventbus/proto/topics/{domain}/{pipeline}.proto
           → proto/topics/index/retriever.proto

Go 类型:   PascalCase(stage) + "Input" / "Output"
           → RetrieveInput / RetrieveOutput

Go import: 固定路径（扁平包，所有 topic 共享）
           → github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics
```

**完整映射表：**

| Topic | Proto 文件 | Go Input | Go Output |
|-------|-----------|----------|-----------|
| `rag.insert.chunking` | `proto/topics/rag/insert.proto` | `ChunkingInput` | `ChunkingOutput` |
| `rag.insert.embedding` | `proto/topics/rag/insert.proto` | `EmbeddingInput` | `EmbeddingOutput` |
| `rag.insert.ocr` | `proto/topics/rag/insert.proto` | `OcrInput` | `OcrOutput` |
| `rag.query.keyword_extraction` | `proto/topics/rag/query.proto` | `KeywordExtractionInput` | `KeywordExtractionOutput` |
| `rag.query.query_expansion` | `proto/topics/rag/query.proto` | `QueryExpansionInput` | `QueryExpansionOutput` |
| `rag.query.vector_search` | `proto/topics/rag/query.proto` | `VectorSearchInput` | `VectorSearchOutput` |
| `rag.query.kg_search` | `proto/topics/rag/query.proto` | `KgSearchInput` | `KgSearchOutput` |
| `rag.query.rerank` | `proto/topics/rag/query.proto` | `RerankInput` | `RerankOutput` |
| `rag.query.response` | `proto/topics/rag/query.proto` | `ResponseInput` | `ResponseOutput` |
| `index.builder.index_build` | `proto/topics/index/builder.proto` | `IndexBuildInput` | `IndexBuildOutput` |
| `index.retriever.retrieve` | `proto/topics/index/retriever.proto` | `RetrieveInput` | `RetrieveOutput` |

**反向推导（查看参数）：**
- 给定 topic `index.retriever.retrieve` → stage = `retrieve` → Go 类型 = `RetrieveInput`
- 查看参数：`topicspb.RetrieveInput` 的 Go struct 字段，或直接读 proto 文件 `proto/topics/index/retriever.proto`

**判断有没有这个 topic：**
- stage = `retrieve` → PascalCase = `Retrieve` → 检查有没有 `RetrieveInput` 和 `RetrieveOutput` 两个成对的 message
- 或者直接读 `proto/topics/index/retriever.proto` 文件确认

## SDK 导入

```go
import (
    pb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go"        // gRPC 客户端、信封类型
    topicspb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics" // Topic 数据模型（所有 topic 扁平在一个包）
)
```

## 架构

```
业务服务 (Subscriber)               EventBus 平台 (Server)
       │                                   │
       │── RegisterSubscriber ────────────>│
       │                                   │
       │── Subscribe (stream) ────────────>│
       │<── EventEnvelope ─────────────────│  收到任务
       │                                   │
       │   处理业务逻辑...                   │
       │                                   │
       │── Respond (SubscriberReply) ─────>│  返回结果
```

## 查看 Topic 字段

方式一：看 Go struct（推荐，IDE 有补全）
```go
var input topicspb.RetrieveInput
input.IndexName      // string — 目标索引名称
input.Query          // string — 检索查询文本
input.TopK           // int32  — 返回结果数量
// ...
```

方式二：读 proto 文件（看注释最全）
```
proto/topics/index/retriever.proto
```

## 写一个订阅者

示例：订阅 `index.retriever.retrieve`

```go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/protobuf/proto"
    "github.com/google/uuid"

    pb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go"
    topicspb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics"
)

func main() {
    conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
    defer conn.Close()

    client := pb.NewEventBusClient(conn)
    ctx := context.Background()

    // 1. 注册
    regReq := &pb.RegisterRequest{
        SubscriberId: "my-retriever",
        Topic:        "index.retriever.retrieve",  // topic 名
        Capabilities: []string{"meilisearch"},
    }
    client.RegisterSubscriber(ctx, regReq)

    // 2. 订阅事件流
    stream, _ := client.Subscribe(ctx, regReq)

    for {
        envelope, _ := stream.Recv()
        go handleEvent(client, envelope)
    }
}

func handleEvent(client pb.EventBusClient, envelope *pb.EventEnvelope) {
    start := time.Now()

    // 3. 反序列化 — 从 topic 名推导：RetrieveInput
    var input topicspb.RetrieveInput
    proto.Unmarshal(envelope.Inputs["result"], &input)

    // 4. 使用 input.IndexName, input.Query, input.ScoreThreshold ...
    output := doRetrieve(input)

    // 5. 序列化 — RetrieveOutput
    outputBytes, _ := proto.Marshal(output)

    // 6. 响应
    client.Respond(context.Background(), &pb.SubscriberReply{
        CorrelationId: envelope.CorrelationId,
        SubscriberId:  "my-retriever",
        Outputs:       map[string][]byte{"result": outputBytes},
        Strategy:      pb.SubscriberReply_APPEND,
        Weight:        10,
        LatencyMs:     int32(time.Since(start).Milliseconds()),
        Health:        pb.SubscriberReply_HEALTHY,
    })
}

func doRetrieve(input topicspb.RetrieveInput) *topicspb.RetrieveOutput {
    // 业务逻辑...
    return &topicspb.RetrieveOutput{
        Results:   []*topicspb.RetrieveResult{},
        TotalHits: 0,
        IndexName: input.IndexName,
    }
}
```

## 发布一个事件

```go
func publishRetrieve(ctx context.Context, client pb.EventBusClient, indexName, query string) (*topicspb.RetrieveOutput, error) {
    inputBytes, _ := proto.Marshal(&topicspb.RetrieveInput{
        IndexName:       indexName,
        Query:           query,
        TopK:            20,
        SemanticRatio:   0.5,
        ScoreThreshold:  0.65,
    })

    envelope := &pb.EventEnvelope{
        Topic:         "index.retriever.retrieve",
        CorrelationId: uuid.NewString(),
        SourceService: "query-expansion",
        Inputs:        map[string][]byte{"result": inputBytes},
    }

    reply, _ := client.PublishAndWait(ctx, envelope)

    var output topicspb.RetrieveOutput
    proto.Unmarshal(reply.Outputs["result"], &output)
    return &output, nil
}
```

## 序列化约定

| 操作 | 方式 |
|------|------|
| 序列化 Input | `proto.Marshal(&topicspb.XxxInput{...})` |
| 反序列化 Input | `proto.Unmarshal(envelope.Inputs["result"], &input)` |
| 序列化 Output | `proto.Marshal(&topicspb.XxxOutput{...})` |
| 反序列化 Output | `proto.Unmarshal(reply.Outputs["result"], &output)` |
| Map key | 固定 `"result"` |

## 合并策略

| Strategy | 行为 | 适用 |
|----------|------|------|
| `FIRST` | 首个到达生效 | 嵌入、分块、OCR、索引构建 |
| `APPEND` | 追加所有结果 | 向量搜索、KG 搜索、查询扩展、检索 |
| `REPLACE` | 权重高覆盖低 | 重排序 |
| `IGNORE` | 旁路不影响主流程 | 审计、敏感词检测 |

## 常见问题

**Q: 我只知道 topic 名，怎么找到对应的 Go 类型？**
用推导规则：`index.retriever.retrieve` → stage = `retrieve` → PascalCase = `Retrieve` → `RetrieveInput` / `RetrieveOutput`。

**Q: 怎么知道 topic 有哪些字段？**
方式一：看 `topicspb.RetrieveInput` 的 Go struct（IDE 补全）。
方式二：读 `proto/topics/index/retriever.proto`（看注释最全）。

**Q: 多个订阅者注册同一 topic 会怎样？**
平台同时发给所有订阅者，根据 `Strategy` 和 `Weight` 聚合结果。

**Q: `Inputs` 的 key 为什么是 `"result"`？**
约定：每个 topic 只有一个输入输出，统一用 `"result"`。
