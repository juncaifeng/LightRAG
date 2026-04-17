---
name: eventbus-go-sdk-usage
description: >
  Guide business developers to use the LightRAG EventBus Go SDK.
  Use this skill when: writing a Go subscriber service, publishing events
  to a topic, integrating with the EventBus gRPC API, or understanding
  how topic data models (Input/Output) connect to the transport layer.
---

# LightRAG EventBus — Go SDK 使用指南

## SDK 获取

```go
import (
    pb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go"
    topicspb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics"
)
```

| 包 | 导入路径 | 内容 |
|----|---------|------|
| `pb` (eventbus) | `github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go` | gRPC 客户端、信封类型 |
| `topicspb` (topics) | `github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics` | Topic 数据模型 |

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

订阅（Subscribe）和响应（Respond）是**两个独立 RPC**。

## Topic 数据模型

Topic 的输入输出定义在 protobuf 中，生成的 SDK 就是数据模型。查看字段直接看 Go struct：

```go
// 查看 EmbeddingInput 有哪些字段
var input topicspb.EmbeddingInput
input.Texts  // []string — 待嵌入的文本列表

// 查看 ChunkingInput 有哪些字段
var chunkInput topicspb.ChunkingInput
chunkInput.Content              // string
chunkInput.ChunkTokenSize       // int32
chunkInput.ChunkOverlapTokenSize // int32
```

## 写一个订阅者

订阅 `rag.insert.embedding` topic 的完整示例：

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
        SubscriberId: "my-embedder",
        Topic:        "rag.insert.embedding",
        Capabilities: []string{"openai"},
    }
    client.RegisterSubscriber(ctx, regReq)

    // 2. 订阅事件流（阻塞）
    stream, _ := client.Subscribe(ctx, regReq)

    for {
        envelope, _ := stream.Recv()
        go handleEvent(client, envelope)
    }
}

func handleEvent(client pb.EventBusClient, envelope *pb.EventEnvelope) {
    start := time.Now()

    // 3. 反序列化 Topic Input
    var input topicspb.EmbeddingInput
    proto.Unmarshal(envelope.Inputs["result"], &input)

    // 4. 业务逻辑
    output := doEmbedding(input.Texts)

    // 5. 序列化 Topic Output
    outputBytes, _ := proto.Marshal(output)

    // 6. 响应
    client.Respond(context.Background(), &pb.SubscriberReply{
        CorrelationId: envelope.CorrelationId,
        SubscriberId:  "my-embedder",
        Outputs:       map[string][]byte{"result": outputBytes},
        Strategy:      pb.SubscriberReply_FIRST,
        Weight:        10,
        LatencyMs:     int32(time.Since(start).Milliseconds()),
        Health:        pb.SubscriberReply_HEALTHY,
    })
}
```

## 发布一个事件

```go
func publishEmbedding(ctx context.Context, client pb.EventBusClient, texts []string) (*topicspb.EmbeddingOutput, error) {
    inputBytes, _ := proto.Marshal(&topicspb.EmbeddingInput{Texts: texts})

    envelope := &pb.EventEnvelope{
        Topic:         "rag.insert.embedding",
        CorrelationId: uuid.NewString(),
        SourceService: "lightrag-core",
        Inputs:        map[string][]byte{"result": inputBytes},
    }

    reply, _ := client.PublishAndWait(ctx, envelope)

    var output topicspb.EmbeddingOutput
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

## Topic 速查

### Insert

| Topic | Input | Output |
|-------|-------|--------|
| `rag.insert.chunking` | `topicspb.ChunkingInput` | `topicspb.ChunkingOutput` |
| `rag.insert.embedding` | `topicspb.EmbeddingInput` | `topicspb.EmbeddingOutput` |
| `rag.insert.ocr` | `topicspb.OcrInput` | `topicspb.OcrOutput` |

### Query

| Topic | Input | Output |
|-------|-------|--------|
| `rag.query.keyword_extraction` | `topicspb.KeywordExtractionInput` | `topicspb.KeywordExtractionOutput` |
| `rag.query.query_expansion` | `topicspb.QueryExpansionInput` | `topicspb.QueryExpansionOutput` |
| `rag.query.vector_search` | `topicspb.VectorSearchInput` | `topicspb.VectorSearchOutput` |
| `rag.query.kg_search` | `topicspb.KgSearchInput` | `topicspb.KgSearchOutput` |
| `rag.query.rerank` | `topicspb.RerankInput` | `topicspb.RerankOutput` |
| `rag.query.response` | `topicspb.ResponseInput` | `topicspb.ResponseOutput` |

## 合并策略

| Strategy | 行为 | 适用 |
|----------|------|------|
| `FIRST` | 首个到达生效 | 嵌入、分块、OCR |
| `APPEND` | 追加所有结果 | 向量搜索、KG 搜索、查询扩展 |
| `REPLACE` | 权重高覆盖低 | 重排序 |
| `IGNORE` | 旁路不影响主流程 | 审计、敏感词检测 |

## 常见问题

**Q: `Inputs` 的 key 为什么是 `"result"`？**
约定：每个 topic 只有一个输入输出，统一用 `"result"`。

**Q: 怎么知道 topic 有哪些字段？**
导入 `topicspb` 包，直接看 Go struct。每个 topic 的 `XxxInput` / `XxxOutput` 就是完整的字段定义。

**Q: 多个订阅者注册同一 topic 会怎样？**
平台同时发给所有订阅者，根据 `Strategy` 和 `Weight` 聚合结果。
