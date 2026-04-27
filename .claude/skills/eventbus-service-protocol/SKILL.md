---
name: eventbus-service-protocol
description: >
  Define new EventBus gRPC services and generate multi-language SDKs from protobuf.
  Use this skill when: adding a new gRPC service to the LightRAG EventBus, modifying
  existing service RPC methods or message types, regenerating SDK code after service
  proto changes, or troubleshooting service SDK generation. Services are direct gRPC
  APIs (unlike Topics which use scatter-gather pub/sub).
---

# EventBus Service Protocol — Define Service + Generate SDK

## Overview

The LightRAG EventBus supports two namespaces:

| Namespace | Pattern | Use Case |
|-----------|---------|----------|
| **Topics** (`proto/topics/`) | Scatter-gather pub/sub | Event-driven pipelines (chunking, embedding, search) |
| **Services** (`proto/services/`) | Direct gRPC API | Stateful services (session store, config center) |

This skill covers **Services**. For Topics, use `eventbus-topic-protocol`.

Adding a new service is a **2-step process**.

---

## Directory Structure

```
go-eventbus/proto/services/
├── session.proto              # SessionStore — 会话管理服务
├── config.proto               # (future) ConfigCenter
└── {service_name}.proto       # 每个文件 = 一个 gRPC 服务
```

**规则**：
- **文件名 = 服务名**（`session.proto`, `config.proto`, ...）
- **package** = `lightrag.services.{service_name}.v1`
- **go_package** = `.../services/{service_name};{service_name}`
- 每个 proto 文件定义一个 `service` 块 + 相关的 request/response messages

---

## Step 1: Define Proto Service

### Which File

- 已有服务直接修改对应 proto 文件
- 新服务 → 创建 `proto/services/{service_name}.proto`

### Proto File Template

```protobuf
syntax = "proto3";

package lightrag.services.{service_name}.v1;
option go_package = "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/services/{service_name};{service_name}";

// =============================================================================
// {ServiceName} Service — 服务描述
// =============================================================================

// --- 核心数据类型 ---

message Foo {
    string id = 1;                // 唯一标识
    // ...
}

// --- Request / Response ---

message GetFooRequest {
    string id = 1;                // 要查询的 ID
}

message GetFooResponse {
    Foo foo = 1;                  // 查询结果
}

// =============================================================================
// gRPC Service 定义
// =============================================================================

service FooService {
    // 方法描述
    rpc GetFoo(GetFooRequest) returns (GetFooResponse);
    // 更多 RPC ...
}
```

### Naming Convention

- Service 名：`PascalCase` + `Service` 后缀，如 `SessionService`, `ConfigService`
- Request/Response：`XxxRequest` / `XxxResponse`（不用 `Input`/`Output`，那是 Topics 的约定）
- 业务数据类型：直接用领域名称，如 `Session`, `Message`, `Config`

### Field Comments = Descriptions

Proto field comments are extracted at runtime as field descriptions in the API.
Always add a comment after each field:

```protobuf
message Session {
    string session_id = 1;                    // 会话唯一标识
    string tenant_id = 2;                     // 租户 ID
    map<string, string> context = 4;          // 会话上下文 KV
    int64 created_at = 6;                     // 创建时间 (Unix ms)
}
```

---

## Step 2: Register Go Types

在 `go-eventbus/server/service_registry.go` 的 `allServiceMessages` init() 中添加新服务的所有 message types：

```go
(*topicspb.GetFooRequest)(nil),
(*topicspb.GetFooResponse)(nil),
(*topicspb.Foo)(nil),
```

注册后，server 启动时会自动：
1. 扫描 `allServiceMessages` 中注册的类型
2. 按 proto 文件分组
3. 从 FileDescriptor 中提取 `service` 定义和 RPC methods
4. 构建 `ServiceSchema` 供 API 查询

**不需要**手动定义 methods —— service descriptor 自动提供。

---

## Step 3: Generate SDKs

```bash
cd go-eventbus
./scripts/generate_protos.sh
```

服务代码生成到：

| Language | Output Directory | Notes |
|----------|-----------------|-------|
| Go | `sdk/v1/go/services/{name}/` | 含 gRPC stubs (`*_grpc.pb.go`) |
| Python | `sdk/v1/python/services/` | 含 `_pb2_grpc.py` |
| TypeScript | `sdk/v1/node/src/services/` | 含 service client/server types |

---

## Implementing a Service

### Go Service Implementation

```go
package main

import (
    "context"
    "google.golang.org/grpc"
    sessionpb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/services/session"
)

type sessionServer struct {
    sessionpb.UnimplementedSessionServiceServer
}

func (s *sessionServer) CreateSession(ctx context.Context, req *sessionpb.CreateSessionRequest) (*sessionpb.Session, error) {
    // 实现逻辑
    return &sessionpb.Session{SessionId: "xxx", TenantId: req.TenantId}, nil
}

func main() {
    lis, _ := net.Listen("tcp", ":50052")
    srv := grpc.NewServer()
    sessionpb.RegisterSessionServiceServer(srv, &sessionServer{})
    srv.Serve(lis)
}
```

### Python Service Implementation

```python
import grpc
from concurrent import futures
from services import session_pb2, session_pb2_grpc

class SessionService(session_pb2_grpc.SessionServiceServicer):
    def CreateSession(self, request, context):
        return session_pb2.Session(
            session_id="xxx",
            tenant_id=request.tenant_id,
        )

server = grpc.server(futures.ThreadPoolExecutor())
session_pb2_grpc.add_SessionServiceServicer_to_server(SessionService(), server)
server.add_insecure_port('[::]:50052')
server.start()
```

---

## Services vs Topics — 何时用哪个

| 考量 | Service (gRPC) | Topic (EventBus) |
|------|---------------|------------------|
| 调用模式 | 点对点请求/响应 | 发布/订阅广播 |
| 状态 | 有状态（可维护连接） | 无状态 |
| 负载均衡 | gRPC 内置 | FIRST strategy |
| 结果合并 | 不适用 | APPEND/REPLACE strategy |
| 典型场景 | Session Store, Config, Auth | 管道处理 (chunking, embedding, search) |
| 适用条件 | 需要 CRUD / 持久连接 / 流式 | 事件驱动 / 多消费者 / 扇出 |

**经验法则**：
- 如果你需要 **CRUD 操作**或**有状态管理** → Service
- 如果你需要**扇出给多个消费者**或**管道编排** → Topic

---

## Auto-Discovery Details

Service schemas 自动暴露到 HTTP API：

```
GET /api/services/schemas        → 所有服务的 schema 列表
GET /api/services/schemas/{name} → 单个服务的 schema
```

前端 dashboard 自动展示，无需修改前端代码（首次添加服务目录时需要）。

Discovery 规则：
1. 扫描 `allServiceMessages` 中注册的所有 message types
2. 按 proto 文件分组
3. 从 `protoreflect.FileDescriptor` 中提取 `ServiceDescriptor`
4. 遍历每个 service 的 methods，提取 input/output message names
5. 提取 proto 注释作为 description

---

## Checklist for New Service

- [ ] 创建或修改 `proto/services/{service_name}.proto`
- [ ] 定义 `service` 块 + 所有 RPC methods
- [ ] 定义 Request/Response messages + 业务数据类型
- [ ] 添加字段注释（中文优先，英文兜底）
- [ ] 在 `service_registry.go` 的 `allServiceMessages` 中注册所有 message types
- [ ] 运行 `generate_protos.sh`
- [ ] 验证 Go 编译：`cd go-eventbus && go build .`
- [ ] 验证前端编译：`cd eventbus-dashboard && npx vite build`
- [ ] 实现服务端代码（Go/Python gRPC server）
- [ ] 提交 proto + 生成代码 + 实现代码
