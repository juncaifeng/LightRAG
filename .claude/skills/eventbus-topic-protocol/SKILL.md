---
name: eventbus-topic-protocol
description: >
  Define new EventBus topics and generate multi-language SDKs from protobuf.
  Use this skill when: adding a new topic to the LightRAG EventBus, modifying
  existing topic Input/Output messages, regenerating SDK code after proto changes,
  or troubleshooting SDK generation issues. The topic registry auto-discovers
  topics from proto descriptors — no manual registry updates needed.
---

# EventBus Topic Protocol — Add Topic + Generate SDK

## Overview

The LightRAG EventBus uses **protobuf as the single source of truth** for topic data contracts.
Topic schemas are auto-discovered at runtime by Go backend from proto message descriptors.

Adding a new topic is a **3-step process**. Modifying an existing topic is **2 steps**.

## Quick Reference

| Step | Action | File(s) |
|------|--------|---------|
| 1 | Define `XxxInput` + `XxxOutput` messages in proto | `go-eventbus/proto/topics/insert.proto` or `query.proto` |
| 2 | Register the Go types in `topic_registry.go` init() | `go-eventbus/server/topic_registry.go` |
| 3 | Run SDK generation script | `go-eventbus/scripts/generate_protos.sh` |

For **modifying** an existing topic, skip step 2.

---

## Step 1: Define Proto Messages

### Naming Convention (MUST follow)

```
XxxInput  +  XxxOutput  =  one topic
```

- **`XxxInput`** — the request message (what the publisher sends)
- **`XxxOutput`** — the response message (what the subscriber returns)
- **`Xxx`** — CamelCase stage name, e.g. `Chunking`, `Embedding`, `VectorSearch`
- **Nested types** — any helper messages that don't end in `Input`/`Output` are ignored by the auto-discovery

### Topic Name Derivation

The system auto-derives the topic name from:

```
proto file name  +  message name prefix
    ↓                    ↓
insert.proto    →    "insert"
ChunkingInput   →    "chunking"
    ↓
rag.insert.chunking
```

### Which File to Use

- **`insert.proto`** — pipeline `rag.insert.*` topics (chunking, embedding, ocr, etc.)
- **`query.proto`** — pipeline `rag.query.*` topics (keyword_extraction, vector_search, etc.)

If adding a new pipeline (e.g. `rag.train.*`), create a new proto file.

### Field Comments = Descriptions

Proto field comments are extracted at runtime as field descriptions in the API.
Always add a comment after each field:

```protobuf
message SummarizeInput {
    string text = 1;                    // 待摘要的文本内容
    int32 max_length = 2;               // 最大摘要长度 (默认 200)
}
```

### Example: Adding `rag.query.summarize`

In `query.proto`, append:

```protobuf
// ==========================================
// rag.query.summarize
// ==========================================

message SummarizeInput {
    string text = 1;                    // 待摘要的文本内容
    int32 max_length = 2;               // 最大摘要长度 (默认 200)
}

message SummarizeOutput {
    string summary = 1;                 // 生成的摘要
    int32 token_count = 2;              // 摘要 token 数
}
```

---

## Step 2: Register Go Types

**Only needed for NEW topics.** For modifications to existing topics, skip this step.

Open `go-eventbus/server/topic_registry.go` and add to `allProtoMessages` init():

```go
(*topicspb.SummarizeInput)(nil),
(*topicspb.SummarizeOutput)(nil),
```

If the default merge strategy `APPEND` doesn't fit, add an override in `topicStrategyOverrides`:

```go
"rag.query.summarize": "FIRST",
```

### Strategy Guide

| Strategy | When to use |
|----------|-------------|
| `FIRST` | Single subscriber wins — chunking, embedding, ocr, keyword extraction |
| `APPEND` | Merge all subscribers' results — vector search, kg search, query expansion |
| `REPLACE` | Last subscriber wins — rerank |

---

## Step 3: Generate SDKs

```bash
cd go-eventbus
./scripts/generate_protos.sh
```

This generates code for all languages:

| Language | Output Directory | Generated Files |
|----------|-----------------|-----------------|
| Go | `sdk/v1/go/topics/` | `insert.pb.go`, `query.pb.go` |
| Python | `sdk/v1/python/topics/` | `insert_pb2.py`, `query_pb2.py` |
| Rust | `sdk/v1/rust/src/` | via tonic-build (cargo) |
| TypeScript | `sdk/v1/node/src/topics/` | `insert.ts`, `query.ts` |
| Java | `sdk/v1/java/src/main/java/` | via protoc-gen-grpc-java |

### Prerequisites

- **Go**: `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`
- **Python**: `pip install grpcio-tools`
- **Rust**: `cargo` (uses tonic-build)
- **TypeScript**: `npx` (uses ts-proto)
- **Java**: `protoc-gen-grpc-java` plugin

### After Generation

1. **Go**: `cd go-eventbus && go build .` — verify compilation
2. **TypeScript**: check `sdk/v1/node/src/topics/` for generated types
3. **Python**: check `sdk/v1/python/topics/` for generated modules

---

## Auto-Discovery Details

The Go backend (`topic_registry.go`) uses these rules:

1. **Scan** all proto message types registered in `allProtoMessages`
2. **Filter** messages ending in `Input` (skip nested types like `ChunkItem`)
3. **Pair** each `XxxInput` with matching `XxxOutput`
4. **Derive** pipeline from proto file name (`insert.proto` → `insert`)
5. **Derive** stage from message name (`ChunkingInput` → `chunking` via CamelCase→snake_case)
6. **Extract** field definitions from proto descriptor (name, type, comments)
7. **Topic name** = `rag.{pipeline}.{stage}`

The frontend dashboard auto-displays all topics from `GET /api/topics/schemas` — no frontend changes needed.

---

## Checklist for New Topic

- [ ] Define `XxxInput` and `XxxOutput` in correct proto file
- [ ] Add field comments (Chinese preferred, English fallback)
- [ ] Add nested helper messages if needed (don't name them `*Input`/`*Output`)
- [ ] Register Go types in `topic_registry.go` init()
- [ ] Add strategy override if not defaulting to `APPEND`
- [ ] Run `generate_protos.sh`
- [ ] Verify Go compilation: `cd go-eventbus && go build .`
- [ ] Verify frontend build: `cd eventbus-dashboard && npx vite build`
- [ ] Commit proto + generated code together
