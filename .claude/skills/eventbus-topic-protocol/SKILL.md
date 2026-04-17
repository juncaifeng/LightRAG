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
| 1 | Define `XxxInput` + `XxxOutput` messages in proto | `go-eventbus/proto/topics/{domain}/{pipeline}.proto` |
| 2 | Register the Go types in `topic_registry.go` init() | `go-eventbus/server/topic_registry.go` |
| 3 | Run SDK generation script | `go-eventbus/scripts/generate_protos.sh` |

For **modifying** an existing topic, skip step 2.

---

## Directory Structure

```
go-eventbus/proto/topics/
├── rag/                          # domain = "rag"
│   ├── insert.proto              # pipeline = "insert"  → rag.insert.chunking
│   └── query.proto               # pipeline = "query"   → rag.query.vector_search
├── index/                        # domain = "index"
│   ├── builder.proto             # pipeline = "builder"  → index.builder.xxx
│   └── retriever.proto           # pipeline = "retriever" → index.retriever.xxx
└── {domain}/                     # future domains...
    └── {pipeline}.proto
```

**规则**：
- **目录名 = domain**（`rag/`, `index/`, `analytics/`, ...）
- **文件名 = pipeline**（`insert.proto`, `builder.proto`, ...）
- **topic 名 = `{directory}.{filename}.{stage}`**

| 路径 | 推导结果 |
|------|---------|
| `rag/insert.proto` + `ChunkingInput` | `rag.insert.chunking` |
| `rag/query.proto` + `VectorSearchInput` | `rag.query.vector_search` |
| `index/builder.proto` + `IndexBuildInput` | `index.builder.index_build` |
| `index/retriever.proto` + `RetrieveInput` | `index.retriever.retrieve` |

---

## Step 1: Define Proto Messages

### Naming Convention (MUST follow)

```
XxxInput  +  XxxOutput  =  one topic
```

- **`XxxInput`** — the request message (what the publisher sends)
- **`XxxOutput`** — the response message (what the subscriber returns)
- **`Xxx`** — CamelCase stage name, e.g. `Chunking`, `Embedding`, `IndexBuild`
- **Nested types** — any helper messages that don't end in `Input`/`Output` are ignored by the auto-discovery

### Which Directory and File

1. **选择 domain 目录**：这个 topic 属于哪个领域？
   - `rag/` — RAG 管道（文档插入、查询处理）
   - `index/` — 索引构建与检索
   - 新领域 → 创建新目录

2. **选择 pipeline 文件**：这个 topic 属于哪个管道？
   - 已有文件直接追加 message
   - 新管道 → 创建新 `.proto` 文件

### Each Proto File MUST Contain

```protobuf
syntax = "proto3";

package lightrag.eventbus.topics.v1;           // 固定不变
option go_package = ".../topics;topics";       // 固定不变

// ==========================================
// {domain}.{pipeline}.{stage}
// ==========================================

message XxxInput {
    // fields with comments
}

message XxxOutput {
    // fields with comments
}
```

> 所有 topic proto 共用同一个 package `lightrag.eventbus.topics.v1`，domain/pipeline 从文件路径推导。

### Field Comments = Descriptions

Proto field comments are extracted at runtime as field descriptions in the API.
Always add a comment after each field:

```protobuf
message IndexBuildInput {
    string document_id = 1;                     // 文档唯一标识
    string content = 2;                         // 文档原始内容
    map<string, string> metadata = 3;           // 附加元数据
}

message IndexBuildOutput {
    string index_id = 2;                        // 生成的索引 ID
    int32 entry_count = 3;                      // 索引条目数量
}
```

---

## Step 2: Register Go Types

**Only needed for NEW topics.** For modifications to existing topics, skip this step.

Open `go-eventbus/server/topic_registry.go` and add to `allProtoMessages` init():

```go
(*topicspb.IndexBuildInput)(nil),
(*topicspb.IndexBuildOutput)(nil),
```

If the default merge strategy `APPEND` doesn't fit, add an override in `topicStrategyOverrides`:

```go
"index.builder.index_build": "FIRST",
```

### Strategy Guide

| Strategy | When to use |
|----------|-------------|
| `FIRST` | Single subscriber wins — chunking, embedding, ocr, index build |
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
| Go | `sdk/v1/go/topics/` | `*.pb.go` |
| Python | `sdk/v1/python/topics/` | `*_pb2.py` |
| Rust | `sdk/v1/rust/src/` | via tonic-build (cargo) |
| TypeScript | `sdk/v1/node/src/topics/` | `*.ts` |
| Java | `sdk/v1/java/src/main/java/` | via protoc-gen-grpc-java |

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
4. **Derive** pipeline from proto file path's filename (`insert.proto` → `insert`)
5. **Derive** domain from proto file path's parent directory (`rag/insert.proto` → `rag`)
6. **Derive** stage from message name (`ChunkingInput` → `chunking` via CamelCase→snake_case)
7. **Topic name** = `{domain}.{pipeline}.{stage}`

The frontend dashboard auto-displays all topics from `GET /api/topics/schemas` — no frontend changes needed.

---

## Checklist for New Topic

- [ ] Create or use existing `proto/topics/{domain}/{pipeline}.proto`
- [ ] Define `XxxInput` and `XxxOutput` messages
- [ ] Add field comments (Chinese preferred, English fallback)
- [ ] Add nested helper messages if needed (don't name them `*Input`/`*Output`)
- [ ] Register Go types in `topic_registry.go` init()
- [ ] Add strategy override if not defaulting to `APPEND`
- [ ] Run `generate_protos.sh`
- [ ] Verify Go compilation: `cd go-eventbus && go build .`
- [ ] Verify frontend build: `cd eventbus-dashboard && npx vite build`
- [ ] Commit proto + generated code together
