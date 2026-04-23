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
Go 类型:   PascalCase(stage) + "Input" / "Output"
           → RetrieveInput / RetrieveOutput

Go import: 固定路径（扁平包，所有 topic 共享）
           → github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics
```

**完整映射表：**

| Topic | Go Input | Go Output |
|-------|----------|-----------|
| `rag.insert.chunking` | `ChunkingInput` | `ChunkingOutput` |
| `embedding.embed.embedding` | `EmbeddingInput` | `EmbeddingOutput` |
| `rag.insert.ocr` | `OcrInput` | `OcrOutput` |
| `rag.insert.load.text` | `LoadTextInput` | `LoadTextOutput` |
| `rag.insert.load.pdf` | `LoadPdfInput` | `LoadPdfOutput` |
| `rag.insert.load.docx` | `LoadDocxInput` | `LoadDocxOutput` |
| `rag.query.keyword_extraction` | `KeywordExtractionInput` | `KeywordExtractionOutput` |
| `rag.query.query_expansion` | `QueryExpansionInput` | `QueryExpansionOutput` |
| `rag.query.vector_search` | `VectorSearchInput` | `VectorSearchOutput` |
| `rag.query.kg_search` | `KgSearchInput` | `KgSearchOutput` |
| `rag.query.rerank` | `RerankInput` | `RerankOutput` |
| `rag.query.response` | `ResponseInput` | `ResponseOutput` |
| `index.builder.index_build` | `IndexBuildInput` | `IndexBuildOutput` |
| `index.retriever.retrieve` | `RetrieveInput` | `RetrieveOutput` |
| `llm.completion.complete` | `CompleteInput` | `CompleteOutput` |
| `kg.merge.entity` | `EntityMergeInput` | `EntityMergeOutput` |
| `kg.merge.relation` | `RelationMergeInput` | `RelationMergeOutput` |

### 验证 SDK 是否包含目标 Topic 类型

推导规则告诉你类型名，但 SDK 可能还没发布。**动手写代码前先验证**：

```bash
go doc github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics | grep "IndexBuild"
```

- 有输出 → 类型已发布，直接用
- 无输出 → SDK 尚未包含该 topic，联系 SDK 维护者 push 最新 proto 并发版
- 或者拉指定 commit：`go get github.com/juncaifeng/LightRAG/go-eventbus@{commit_hash}`

> Proto 文件位于 SDK 仓库 (`go-eventbus/proto/topics/{domain}/{pipeline}.proto`)，业务项目不需要本地管理 proto，直接 import `topicspb` 包即可。

### 查看 Topic 字段

```bash
# 列出 RetrieveInput 的所有字段
go doc github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics RetrieveInput
```

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

## 最小可运行模板

以下模板是可直接编译运行的骨架，只差业务逻辑。所有 error 分支和重连逻辑都已包含。

```go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	pb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go"
	topicspb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics"
)

func main() {
	addr := "localhost:50051"

	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("connect to %s: %v", addr, err)
	}
	defer conn.Close()

	client := pb.NewEventBusClient(conn)
	ctx := context.Background()

	// 1. 注册
	regReq := &pb.RegisterRequest{
		SubscriberId: "my-retriever",
		Topic:        "index.retriever.retrieve",
		Capabilities: []string{"meilisearch"},
	}
	if _, err := client.RegisterSubscriber(ctx, regReq); err != nil {
		log.Fatalf("register: %v", err)
	}
	log.Printf("registered as %s on topic %s", regReq.SubscriberId, regReq.Topic)

	// 2. 订阅事件流（阻塞循环，断开后需外层重连）
	stream, err := client.Subscribe(ctx, regReq)
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	for {
		envelope, err := stream.Recv()
		if err == io.EOF {
			log.Println("stream closed by server")
			return
		}
		if err != nil {
			log.Fatalf("recv error: %v", err)
		}
		go handleEvent(ctx, client, envelope)
	}
}

func handleEvent(ctx context.Context, client pb.EventBusClient, envelope *pb.EventEnvelope) {
	start := time.Now()

	// 3. 反序列化 Input
	var input topicspb.RetrieveInput
	if err := proto.Unmarshal(envelope.Inputs["result"], &input); err != nil {
		log.Printf("unmarshal input: %v", err)
		respondError(client, envelope, err)
		return
	}

	// 4. 业务逻辑
	output := doRetrieve(ctx, client, input)

	// 5. 序列化 Output
	outputBytes, err := proto.Marshal(output)
	if err != nil {
		log.Printf("marshal output: %v", err)
		respondError(client, envelope, err)
		return
	}

	// 6. 响应
	if _, err := client.Respond(ctx, &pb.SubscriberReply{
		CorrelationId: envelope.CorrelationId,
		SubscriberId:  "my-retriever",
		Outputs:       map[string][]byte{"result": outputBytes},
		Strategy:      pb.SubscriberReply_APPEND,
		Weight:        10,
		LatencyMs:     int32(time.Since(start).Milliseconds()),
		Health:        pb.SubscriberReply_HEALTHY,
	}); err != nil {
		log.Printf("respond: %v", err)
	}
}

func respondError(client pb.EventBusClient, envelope *pb.EventEnvelope, err error) {
	client.Respond(context.Background(), &pb.SubscriberReply{
		CorrelationId: envelope.CorrelationId,
		SubscriberId:  "my-retriever",
		Strategy:      pb.SubscriberReply_FIRST,
		Health:        pb.SubscriberReply_UNHEALTHY,
		ErrorCode:     "INTERNAL",
		ErrorMessage:  err.Error(),
	})
}

func doRetrieve(ctx context.Context, client pb.EventBusClient, input topicspb.RetrieveInput) *topicspb.RetrieveOutput {
	// 你的业务逻辑放这里
	return &topicspb.RetrieveOutput{
		Results:   []*topicspb.RetrieveResult{},
		TotalHits: 0,
		IndexName: input.IndexName,
	}
}
```

## 发布一个事件（PublishAndWait）

```go
func publishRetrieve(ctx context.Context, client pb.EventBusClient, indexName, query string) (*topicspb.RetrieveOutput, error) {
	inputBytes, err := proto.Marshal(&topicspb.RetrieveInput{
		IndexName:      indexName,
		Query:          query,
		TopK:           20,
		SemanticRatio:  0.5,
		ScoreThreshold: 0.65,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	reply, err := client.PublishAndWait(ctx, &pb.EventEnvelope{
		Topic:         "index.retriever.retrieve",
		CorrelationId: uuid.NewString(),
		SourceService: "query-expansion",
		Inputs:        map[string][]byte{"result": inputBytes},
	})
	if err != nil {
		return nil, fmt.Errorf("publish_and_wait: %w", err)
	}

	// 检查聚合健康状态
	if reply.Health != pb.SubscriberReply_HEALTHY {
		return nil, fmt.Errorf("reply unhealthy: %s", reply.ErrorMessage)
	}

	var output topicspb.RetrieveOutput
	if err := proto.Unmarshal(reply.Outputs["result"], &output); err != nil {
		return nil, fmt.Errorf("unmarshal output: %w", err)
	}
	return &output, nil
}
```

## Subscriber 内嵌套调用另一个 Topic

实际场景：retriever subscriber 收到请求后，如果 `semantic_ratio > 0` 且没有预计算向量，需要内部调用 `embedding.embed.embedding` 获取向量。

```go
func handleRetrieve(ctx context.Context, client pb.EventBusClient, input topicspb.RetrieveInput) *topicspb.RetrieveOutput {
	// 如果需要语义检索但没有预计算向量，先调 embedding
	if input.SemanticRatio > 0 && len(input.QueryVector) == 0 {
		vector, err := callEmbedding(ctx, client, input.Query, input.Embedder)
		if err != nil {
			log.Printf("embedding fallback failed: %v", err)
			// 可以降级为纯文本检索，或返回错误
		} else {
			input.QueryVector = vector
		}
	}

	return doRetrieve(ctx, client, input)
}

func callEmbedding(ctx context.Context, client pb.EventBusClient, text string, embedder string) ([]float32, error) {
	inputBytes, _ := proto.Marshal(&topicspb.EmbeddingInput{Texts: []string{text}})

	reply, err := client.PublishAndWait(ctx, &pb.EventEnvelope{
		Topic:         "embedding.embed.embedding",
		CorrelationId: uuid.NewString(),
		SourceService: "index-retriever",  // SourceService 用于链路追踪
		Inputs:        map[string][]byte{"result": inputBytes},
	})
	if err != nil {
		return nil, err
	}

	var output topicspb.EmbeddingOutput
	proto.Unmarshal(reply.Outputs["result"], &output)

	if len(output.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return output.Embeddings[0].Values, nil
}
```

**注意事项：**
- 嵌套调用使用同一个 gRPC conn 是安全的（HTTP/2 多路复用）
- `SourceService` 设为当前服务名，用于链路追踪和调试
- 注意外层请求的超时控制：嵌套调用前检查 `ctx.Deadline()`
- 避免循环调用：A → B → A

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
| `FIRST` | 首个到达生效 | 嵌入、分块、OCR |
| `APPEND` | 追加所有结果 | 向量搜索、KG 搜索、查询扩展、检索、索引构建（批量结果） |
| `REPLACE` | 权重高覆盖低 | 重排序 |
| `IGNORE` | 旁路不影响主流程 | 审计、敏感词检测 |

## SubscriberReply 关键字段

| 字段 | 用途 |
|------|------|
| `CorrelationId` | 必须原样回传，用于匹配请求和响应 |
| `SubscriberId` | 当前订阅者 ID |
| `Outputs` | 结果数据，key 固定 `"result"` |
| `Strategy` | 聚合策略（FIRST/APPEND/REPLACE/IGNORE） |
| `Weight` | 权重，影响 REPLACE 策略的优先级 |
| `Health` | `HEALTHY` / `UNHEALTHY`，标记本订阅者是否正常 |
| `ErrorCode` | 错误码（如 `"INTERNAL"`），仅 Health=UNHEALTHY 时填 |
| `ErrorMessage` | 错误详情，仅 Health=UNHEALTHY 时填 |
| `LatencyMs` | 处理耗时毫秒数，用于监控 |

## 常见问题

**Q: 我只知道 topic 名，怎么找到对应的 Go 类型？**
用推导规则：`index.retriever.retrieve` → stage = `retrieve` → PascalCase = `Retrieve` → `RetrieveInput` / `RetrieveOutput`。然后 `go doc ... topics RetrieveInput` 验证。

**Q: 怎么知道 topic 有哪些字段？**
`go doc github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics RetrieveInput`

**Q: 多个订阅者注册同一 topic 会怎样？**
平台同时发给所有订阅者，根据 `Strategy` 和 `Weight` 聚合结果。

**Q: `Inputs` 的 key 为什么是 `"result"`？**
约定：每个 topic 只有一个输入输出，统一用 `"result"`。

**Q: `go handleEvent(client, envelope)` 并发安全吗？**
`envelope` 是独立的 protobuf message（每次 Recv 返回新对象），可以安全跨 goroutine 使用。但不要在 goroutine 间共享修改同一个 envelope。

**Q: 如何优雅重启？**
生产环境建议外层包一个重连循环：`Subscribe` 失败或 `io.EOF` 后等待退避再重新注册+订阅。收到 OS signal 时用 `conn.Close()` 关闭连接，当前在处理的 goroutine 自然结束。
